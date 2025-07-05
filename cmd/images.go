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
	Use:     "images",
	Short:   "Manage flow images as OCI artifacts",
	Aliases: []string{"image"},
	Example: `# List all images
$ tedge-oscar flows images list

# Pull an image from a registry
$ tedge-oscar flows images pull ghcr.io/reubenmiller/connectivity-counter:1.0

# Push an image to a registry
$ tedge-oscar flows images push ghcr.io/reubenmiller/connectivity-counter:1.0
`,
}

var listImagesCmd = &cobra.Command{
	Use:     "list",
	Short:   "List pulled images in the image_dir",
	Aliases: []string{"ls"},
	Example: `tedge-oscar flows images list`,
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
		unexpandedImageDir := cfg.UnexpandedImageDir
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
			digest := "<unknown>"
			if f, err := os.Open(manifestPath); err == nil {
				var manifest map[string]interface{}
				if err := json.NewDecoder(f).Decode(&manifest); err == nil {
					if ann, ok := manifest["annotations"].(map[string]interface{}); ok {
						if v, ok := ann["org.opencontainers.image.version"].(string); ok {
							version = v
						}
					}
					if d, ok := manifest["config"].(map[string]interface{}); ok {
						if dgst, ok := d["digest"].(string); ok {
							digest = dgst
						}
					}
					if d, ok := manifest["digest"].(string); ok && d != "" {
						digest = d
					}
				}
				f.Close()
			}
			rows = append(rows, []string{entry.Name(), version, digest})
		}
		if len(rows) == 0 {
			fmt.Printf("No images found in image_dir (%s).\n", unexpandedImageDir)
			return nil
		}
		// Dynamically fit columns to terminal width
		colNames := []string{"IMAGE", "VERSION", "DIGEST"}
		maxWidth := 0
		if w, _, err := terminalSize(); err == nil {
			maxWidth = w
		} else {
			maxWidth = 120 // fallback
		}
		colWidths := make([]int, len(colNames))
		for i := range colNames {
			colWidths[i] = len(colNames[i])
		}
		for _, row := range rows {
			for i, cell := range row {
				if l := len(cell); l > colWidths[i] {
					colWidths[i] = l
				}
			}
		}
		total := len(colNames) - 1 // for separators
		for _, w := range colWidths {
			total += w
		}
		// Remove columns from right until fits
		keep := len(colNames)
		for total > maxWidth && keep > 1 {
			keep--
			total -= colWidths[keep] + 1
		}
		filteredColNames := colNames[:keep]
		filteredRows := [][]string{}
		for _, row := range rows {
			filteredRows = append(filteredRows, row[:keep])
		}
		colHeaders := make([]any, len(filteredColNames))
		for i, v := range filteredColNames {
			colHeaders[i] = v
		}
		table := tablewriter.NewTable(os.Stdout)
		table.Header(colHeaders...)
		table.Bulk(filteredRows)
		table.Render()
		return nil
	},
}

func init() {
	imagesCmd.AddCommand(listImagesCmd)
}
