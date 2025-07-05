package imagepush

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/registryauth"
)

func PushImage(cfg *config.Config, imageRef string, ociType string, files []string) error {
	var err error
	repoRef, ref := imageRef, ""
	if i := strings.LastIndex(imageRef, ":"); i > strings.LastIndex(imageRef, "/") {
		repoRef = imageRef[:i]
		ref = imageRef[i+1:]
	} else if i := strings.LastIndex(imageRef, "@"); i > strings.LastIndex(imageRef, "/") {
		repoRef = imageRef[:i]
		ref = imageRef[i+1:]
	}
	if repoRef == imageRef || ref == "" {
		return fmt.Errorf("image reference must include a tag or digest, e.g. ghcr.io/user/repo:tag or @sha256:<hash>")
	}
	memStore := memory.New()
	var descriptors []ocispec.Descriptor
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", f, err)
		}
		mediaType := "application/octet-stream"
		var contentReader *bytes.Reader
		if strings.HasPrefix(repoRef, "ghcr.io/") {
			// ghcr.io expects image layers to have this media type and be gzipped
			mediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			if _, err := gz.Write(data); err != nil {
				return fmt.Errorf("failed to gzip file %s: %w", f, err)
			}
			gz.Close()
			data = buf.Bytes()
			contentReader = bytes.NewReader(data)
		} else {
			if strings.HasSuffix(f, ".json") {
				mediaType = "application/json"
			} else if strings.HasSuffix(f, ".toml") {
				mediaType = "application/toml"
			} else if strings.HasSuffix(f, ".mjs") {
				mediaType = "application/javascript"
			}
			contentReader = bytes.NewReader(data)
		}
		d := ocispec.Descriptor{
			MediaType:   mediaType,
			Digest:      digest.FromBytes(data),
			Size:        int64(len(data)),
			Annotations: map[string]string{"org.opencontainers.image.title": filepath.Base(f)},
		}
		if err := memStore.Push(context.Background(), d, contentReader); err != nil {
			return fmt.Errorf("failed to add file %s to store: %w", f, err)
		}
		descriptors = append(descriptors, d)
	}
	// Always create a minimal config blob
	configBytes := []byte(`{"architecture":"amd64","os":"linux","created_by":"tedge-oscar"}`)
	configDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.config.v1+json",
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	if err := memStore.Push(context.Background(), configDesc, bytes.NewReader(configBytes)); err != nil {
		return fmt.Errorf("failed to add config to store: %w", err)
	}
	var packVersion oras.PackManifestVersion
	artifactType := ociType
	if strings.HasPrefix(repoRef, "ghcr.io/") {
		packVersion = oras.PackManifestVersion1_0
		artifactType = ""
	} else {
		packVersion = oras.PackManifestVersion1_1
	}
	packOpts := oras.PackManifestOptions{
		ConfigDescriptor: &configDesc,
		Layers:           descriptors,
	}
	manifestDesc, err := oras.PackManifest(context.Background(), memStore, packVersion, artifactType, packOpts)
	if err != nil {
		return fmt.Errorf("failed to pack manifest: %w", err)
	}
	// Tag the manifest in the memory store with the user-supplied tag and its own digest
	if err := memStore.Tag(context.Background(), manifestDesc, ref); err != nil {
		return fmt.Errorf("failed to tag manifest in memory store: %w", err)
	}
	if err := memStore.Tag(context.Background(), manifestDesc, manifestDesc.Digest.String()); err != nil {
		return fmt.Errorf("failed to tag manifest digest in memory store: %w", err)
	}
	// Prepare remote repository and authentication
	var repo *remote.Repository
	repo, err = remote.NewRepository(repoRef)
	if err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}
	// Use shared registry auth logic
	scope := ""
	if strings.HasPrefix(repoRef, "ghcr.io/") {
		ownerRepo := strings.TrimPrefix(repoRef, "ghcr.io/")
		scope = "repository:" + ownerRepo + ":push,pull"
	}
	client, _, _, _, err := registryauth.GetAuthenticatedClient(cfg, repoRef, scope)
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}
	if client != nil {
		repo.Client = client
	}
	// Push the manifest and its blobs to the remote repository using the manifest digest as the source reference
	copyOpts := oras.DefaultCopyOptions
	_, err = oras.Copy(context.Background(), memStore, manifestDesc.Digest.String(), repo, ref, copyOpts)
	if err != nil {
		return fmt.Errorf("oras push failed: %w", err)
	}
	return nil
}
