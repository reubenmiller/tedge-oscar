package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/imagepull"
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
		if err := imagepull.PullImage(cfg, imageRef); err != nil {
			return err
		}
		fmt.Printf("Image %s pulled to %s\n", imageRef, cfg.ImageDir)
		return nil
	},
}
