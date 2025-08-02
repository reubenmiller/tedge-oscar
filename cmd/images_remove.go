package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reubenmiller/tedge-oscar/internal/config"
	"github.com/spf13/cobra"
)

var removeImageCmd = &cobra.Command{
	Use:     "remove [image_folder]",
	Short:   "Remove a flow image version (by folder name)",
	Aliases: []string{"rm"},
	Example: `tedge-oscar flows images remove myimage:1.0.0`,
	Args:    cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		imageDir := cfg.ImageDir
		if imageDir == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		entries, err := os.ReadDir(imageDir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if len(toComplete) == 0 || strings.HasPrefix(name, toComplete) {
				completions = append(completions, name)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}
		imageDir := cfg.ImageDir
		if imageDir == "" {
			return fmt.Errorf("image_dir not set in config")
		}
		folderName := args[0]
		fullPath := filepath.Join(imageDir, folderName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Image folder %s does not exist locally, skipping removal.\n", folderName)
			return nil
		}
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to remove image directory: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Image folder %s removed (%s)\n", folderName, fullPath)
		return nil
	},
}
