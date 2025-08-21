package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/imagepull"
)

var (
	// saveImageRef removed, now positional
	saveTarballPath string
	saveCompress    bool
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a flow image as a tarball",
	Long:  `Save a flow image from a registry to a local tarball (optionally compressed).`,
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
		if saveTarballPath == "" {
			return fmt.Errorf("--output is required")
		}
		tmpDir, err := os.MkdirTemp("", "tedge-oscar-save-*")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		if err := imagepull.PullImage(cfg, imageRef, tmpDir, saveTarballPath, saveCompress); err != nil {
			return err
		}
		fmt.Printf("Image saved to %s\n", saveTarballPath)
		return nil
	},
}

func init() {
	// imageRef is now positional
	saveCmd.Flags().StringVarP(&saveTarballPath, "output", "o", "", "Path to output tarball (e.g. image.tar or image.tar.gz)")
	saveCmd.Flags().BoolVar(&saveCompress, "compress", false, "Compress tarball using gzip")
}
