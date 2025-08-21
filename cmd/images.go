package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/thin-edge/tedge-oscar/internal/artifact"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/util"
)

var imagesCmd = &cobra.Command{
	Use:     "images",
	Short:   "Manage flow images as OCI artifacts",
	Aliases: []string{"image"},
	Example: `# List all images
$ tedge-oscar flows images list

# Pull an image from a registry
$ tedge-oscar flows images pull ghcr.io/thin-edge/connectivity-counter:1.0

# Push an image to a registry
$ tedge-oscar flows images push ghcr.io/thin-edge/connectivity-counter:1.0
`,
}

var listImagesCmd = &cobra.Command{
	Use:     "list",
	Short:   "List flow images in the image_dir",
	Aliases: []string{"ls"},
	Example: `tedge-oscar flows images list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		selectCols, err := cmd.Flags().GetString("select")
		if err != nil {
			return err
		}
		var colNames []string
		if selectCols != "" {
			colNames = strings.Split(selectCols, ",")
		} else {
			colNames = []string{"image", "version", "digest", "imageDir"}
		}

		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}

		outputFormat, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		imageDir := cfg.ImageDir
		unexpandedImageDir := cfg.UnexpandedImageDir
		if imageDir == "" {
			return fmt.Errorf("image_dir not set in config")
		}
		// Don't fail if directory does not exist
		entries, err := os.ReadDir(imageDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to read image_dir. Check the permissions of the folder. %w", err)
			}
		}
		rows := [][]string{}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			imageDir := filepath.Join(imageDir, entry.Name())
			manifestPath := filepath.Join(imageDir, "manifest.json")
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
			rowMap := map[string]string{
				"image":    artifact.TrimVersion(entry.Name()),
				"version":  version,
				"digest":   digest,
				"imageDir": imageDir,
			}
			row := make([]string, len(colNames))
			for i, col := range colNames {
				row[i] = rowMap[col]
			}
			rows = append(rows, row)
		}
		if len(rows) == 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "No images found in image_dir (%s).\n", unexpandedImageDir)
			return nil
		}

		if outputFormat == "jsonl" || outputFormat == "json" {
			for _, row := range rows {
				obj := map[string]string{}
				for i, col := range colNames {
					obj[col] = row[i]
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				if err := enc.Encode(obj); err != nil {
					return err
				}
			}
			return nil
		}
		if outputFormat == "tsv" {
			for _, row := range rows {
				fmt.Fprintln(cmd.OutOrStdout(), strings.Join(row, "\t"))
			}
			return nil
		}
		maxWidth := 0
		tablePadding := 2 // left + right border
		columnPadding := 2
		if w, _, err := terminalSize(); err == nil {
			maxWidth = w - tablePadding
		} else {
			maxWidth = 120 // fallback
		}
		colWidths := make([]int, len(colNames))
		for i := range colNames {
			colWidths[i] = len(colNames[i]) + columnPadding
		}
		for _, row := range rows {
			for i, cell := range row {
				if l := len(cell); l > colWidths[i] {
					colWidths[i] = l + columnPadding
				}
			}
		}
		total := len(colNames) - 1 // for separators
		for _, w := range colWidths {
			total += w
		}
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
		table := tablewriter.NewTable(cmd.OutOrStdout())
		table.Header(colHeaders...)
		table.Bulk(filteredRows)
		table.Render()
		return nil
	},
}

func init() {
	defaultOutput := "jsonl"
	if util.Isatty(os.Stdout.Fd()) {
		defaultOutput = "table"
	}
	listImagesCmd.Flags().StringP("output", "o", defaultOutput, "Output format: table|jsonl|tsv")
	listImagesCmd.Flags().String("select", "", "Comma separated list of columns to display (e.g. image,version,digest)")
	_ = listImagesCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "jsonl", "tsv"}, cobra.ShellCompDirectiveNoFileComp
	})
	imagesCmd.AddCommand(listImagesCmd)
	imagesCmd.AddCommand(saveCmd)
	imagesCmd.AddCommand(removeImageCmd)
}
