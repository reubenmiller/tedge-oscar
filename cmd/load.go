package cmd

import (
	"fmt"
	"os"

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
			outputDir = "./image"
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
	loadCmd.Flags().String("output-dir", "./image", "Directory to extract the image contents to")
	imagesCmd.AddCommand(loadCmd)
}
