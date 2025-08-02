package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reubenmiller/tedge-oscar/internal/artifact"
	"github.com/reubenmiller/tedge-oscar/internal/config"
	"github.com/reubenmiller/tedge-oscar/internal/imagepull"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load [source]",
	Short: "Load a flow image from a tarball (local file or URL)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		outputDir, _ := cmd.Flags().GetString("output-dir")
		if outputDir == "" {
			cfgPath := configPath
			if cfgPath == "" {
				cfgPath = config.DefaultConfigPath()
			}
			cfg, err := config.LoadConfig(cfgPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			name, err := artifact.ParseName(source, false)
			if err != nil {
				return err
			}
			outputDir = filepath.Join(cfg.ImageDir, name)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Loading image from %s to %s\n", source, outputDir)
		if err := imagepull.LoadTarballImage(source, outputDir); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Image loaded to %s\n", outputDir)
		return nil
	},
}

func init() {
	loadCmd.Flags().String("output-dir", "", "Directory to download the artifact contents to (default: config image_dir)")
	imagesCmd.AddCommand(loadCmd)
}
