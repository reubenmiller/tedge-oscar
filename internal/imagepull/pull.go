package imagepull

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/thin-edge/tedge-oscar/internal/config"
)

type basicAuthClient struct {
	username string
	password string
}

func (c *basicAuthClient) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.username, c.password)
	return http.DefaultClient.Do(req)
}

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
	u, err := url.Parse("https://" + repoRef)
	if err == nil {
		reg := u.Host
		var username, password string
		if cred := cfg.FindCredential(reg); cred != nil {
			username = cred.Username
			password = cred.Password
		} else {
			username, password, _ = config.LoadDockerCredentials(reg)
		}
		if username != "" && password != "" {
			repo.Client = &basicAuthClient{username: username, password: password}
		}
	}
	if _, err := oras.Copy(context.Background(), repo, ref, store, "", oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("oras pull failed: %w", err)
	}
	return nil
}
