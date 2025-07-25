package cmd

import (
	"fmt"

	"github.com/reubenmiller/tedge-oscar/internal/config"
	"github.com/reubenmiller/tedge-oscar/internal/imagepush"
	"github.com/reubenmiller/tedge-oscar/internal/registryauth"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:     "push [image]",
	Short:   "Push a flow image to an OCI registry",
	Example: `tedge-oscar flows images push ghcr.io/reubenmiller/connectivity-counter:1.0 --file flow.json --file README.md`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set debugHTTP based on logLevel
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
		ociType, _ := cmd.Flags().GetString("type")
		if ociType == "" {
			ociType = "application/vnd.tedge.flow.v1"
		}
		files, _ := cmd.Flags().GetStringArray("file")
		if len(files) == 0 {
			return fmt.Errorf("at least one --file must be specified to include in the artifact")
		}
		rootDir, _ := cmd.Flags().GetString("root")
		if rootDir == "" {
			rootDir = "."
		}
		if err := imagepush.PushImage(cfg, imageRef, ociType, files, rootDir); err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Image %s pushed to registry as type %s with files: %v (root: %s)\n", imageRef, ociType, files, rootDir)
		return nil
	},
}

func init() {
	pushCmd.Flags().String("type", "", "OCI artifact type (default: application/vnd.tedge.flow.v1)")
	pushCmd.Flags().StringArray("file", nil, "File(s) to include in the artifact (repeatable)")
	pushCmd.Flags().String("root", ".", "Root directory for path preservation inside the artifact (default: current working directory)")
	imagesCmd.AddCommand(pushCmd)
}
