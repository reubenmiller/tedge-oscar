package imagepull

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/registryauth"
)

func PullImage(cfg *config.Config, imageRef string) error {
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
	store, err := file.New(cfg.ImageDir)
	if err != nil {
		return fmt.Errorf("failed to open image dir: %w", err)
	}
	repo, err := remote.NewRepository(repoRef)
	if err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}
	// Use shared registry auth logic
	client, _, _, _, err := registryauth.GetAuthenticatedClient(cfg, repoRef, "")
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}
	if client != nil {
		repo.Client = client
	}
	// Pull the image and get the manifest descriptor
	desc, err := oras.Copy(context.Background(), repo, ref, store, "", oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("oras pull failed: %w", err)
	}

	// Save the manifest JSON to the image folder
	manifestBytes, err := store.Fetch(context.Background(), desc)
	if err == nil {
		repoDir := repoRef
		if i := strings.LastIndex(repoRef, "/"); i != -1 {
			repoDir = repoRef[i+1:]
		}
		manifestPath := filepath.Join(cfg.ImageDir, repoDir, "manifest.json")
		var data []byte
		// manifestBytes is already io.ReadCloser
		rc := manifestBytes
		defer rc.Close()
		data, err = io.ReadAll(rc)
		if err != nil {
			data = nil
		}
		if data != nil {
			// Try to add version if not present
			var manifest map[string]interface{}
			if err := json.Unmarshal(data, &manifest); err == nil {
				ann, ok := manifest["annotations"].(map[string]interface{})
				if !ok {
					ann = make(map[string]interface{})
				}
				if _, hasVersion := ann["org.opencontainers.image.version"]; !hasVersion && ref != "" {
					ann["org.opencontainers.image.version"] = ref
				}
				manifest["annotations"] = ann
				if newData, err := json.MarshalIndent(manifest, "", "  "); err == nil {
					data = newData
				}
			}
			_ = os.WriteFile(manifestPath, data, 0644)
		}
	}

	return nil
}
