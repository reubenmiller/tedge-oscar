package cmd

import (
	"fmt"

	"github.com/reubenmiller/tedge-oscar/internal/config"
	"github.com/reubenmiller/tedge-oscar/internal/imagepull"
	"github.com/reubenmiller/tedge-oscar/internal/registryauth"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:     "pull [image]",
	Short:   "Pull a flow image from an OCI registry",
	Example: `tedge-oscar flows images pull ghcr.io/reubenmiller/connectivity-counter:1.0`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Enable debug HTTP if logLevel is debug
		registryauth.SetDebugHTTP(logLevel)
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		imageRef := args[0]
		if err := imagepull.PullImage(cfg, imageRef); err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Image %s pulled to %s\n", imageRef, cfg.ImageDir)
		return nil
	},
}

func init() {
	imagesCmd.AddCommand(pullCmd)
}
