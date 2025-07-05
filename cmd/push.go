package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/imagepush"
)

var pushCmd = &cobra.Command{
	Use:     "push [image]",
	Short:   "Push a flow image to an OCI registry",
	Example: `tedge-oscar flows images push ghcr.io/reubenmiller/connectivity-counter:1.0 --file flow.json --file README.md`,
	Args:    cobra.ExactArgs(1),
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
		ociType, _ := cmd.Flags().GetString("type")
		if ociType == "" {
			ociType = "application/vnd.tedge.flow"
		}
		files, _ := cmd.Flags().GetStringArray("file")
		if len(files) == 0 {
			return fmt.Errorf("at least one --file must be specified to include in the artifact")
		}
		if err := imagepush.PushImage(cfg, imageRef, ociType, files); err != nil {
			return err
		}
		fmt.Printf("Image %s pushed to registry as type %s with files: %v\n", imageRef, ociType, files)
		return nil
	},
}

func init() {
	pushCmd.Flags().String("type", "", "OCI artifact type (default: application/vnd.tedge.flow)")
	pushCmd.Flags().StringArray("file", nil, "File(s) to include in the artifact (repeatable)")
	imagesCmd.AddCommand(pushCmd)
}
