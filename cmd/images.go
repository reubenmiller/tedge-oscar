package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/thin-edge/tedge-oscar/internal/config"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage flow images as OCI artifacts",
}

var listImagesCmd = &cobra.Command{
	Use:   "list",
	Short: "List pulled images in the image_dir",
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
		entries, err := os.ReadDir(imageDir)
		if err != nil {
			return fmt.Errorf("failed to read image_dir: %w", err)
		}
		rows := [][]string{}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(imageDir, entry.Name(), "manifest.json")
			version := "<unknown>"
			if f, err := os.Open(manifestPath); err == nil {
				var manifest struct {
					Annotations map[string]string `json:"annotations"`
				}
				if err := json.NewDecoder(f).Decode(&manifest); err == nil {
					if v, ok := manifest.Annotations["org.opencontainers.image.version"]; ok {
						version = v
					}
				}
				f.Close()
			}
			rows = append(rows, []string{entry.Name(), version})
		}
		if len(rows) == 0 {
			fmt.Println("No images found in image_dir.")
			return nil
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"IMAGE", "VERSION"})
		table.SetRowLine(true)
		table.SetAutoWrapText(false)
		for _, row := range rows {
			table.Append(row)
		}
		table.Render()
		return nil
	},
}

func init() {
	imagesCmd.AddCommand(pullCmd)
	imagesCmd.AddCommand(pushCmd)
	imagesCmd.AddCommand(listImagesCmd)
}
