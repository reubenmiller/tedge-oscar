package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

var pullCmd = &cobra.Command{
	Use:   "pull [image]",
	Short: "Pull a flow image from an OCI registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		imageRef := args[0]

		// Split imageRef into repo and reference (tag or digest)
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

		// Set credentials if available in config
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
				// Set a custom HTTP client with basic auth
				repo.Client = &basicAuthClient{username: username, password: password}
			}
		}

		if _, err := oras.Copy(context.Background(), repo, ref, store, "", oras.DefaultCopyOptions); err != nil {
			return fmt.Errorf("oras pull failed: %w", err)
		}
		fmt.Printf("Image %s pulled to %s\n", imageRef, cfg.ImageDir)
		return nil
	},
}

// basicAuthClient implements remote.Client for basic auth

type basicAuthClient struct {
	username string
	password string
}

func (c *basicAuthClient) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.username, c.password)
	return http.DefaultClient.Do(req)
}
