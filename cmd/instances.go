package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/imagepull"
	"golang.org/x/term"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage flow instances",
}

var listInstancesCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed flow instances",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			return
		}
		deployDir := cfg.DeployDir
		if deployDir == "" {
			deployDir = os.Getenv("DEPLOY_DIR")
		}
		if deployDir == "" {
			deployDir = filepath.Join(filepath.Dir(cfg.ImageDir), "deployments")
		}
		files, err := os.ReadDir(deployDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read deploy dir: %v\n", err)
			return
		}
		// Use the unexpanded deployDir from config for display
		unexpandedDeployDir := config.DefaultConfigPath()
		if configPath != "" {
			// Try to read the unexpanded deployDir from the config file directly
			var rawCfg map[string]interface{}
			if _, err := toml.DecodeFile(configPath, &rawCfg); err == nil {
				if v, ok := rawCfg["deploy_dir"]; ok {
					if s, ok := v.(string); ok && s != "" {
						unexpandedDeployDir = s
					}
				}
			}
		}
		if unexpandedDeployDir == "" {
			unexpandedDeployDir = "$DEPLOY_DIR"
		}
		// Get the unexpanded image_dir from the config file for display
		unexpandedImageDir := cfg.ImageDir
		if configPath != "" {
			var rawCfg map[string]interface{}
			if _, err := toml.DecodeFile(configPath, &rawCfg); err == nil {
				if v, ok := rawCfg["image_dir"]; ok {
					if s, ok := v.(string); ok && s != "" {
						unexpandedImageDir = s
					}
				}
			}
		}
		// Prepare all rows first
		rows := [][]string{}
		colNames := []string{"NAME", "PATH", "TOPICS", "IMAGE"}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") {
				continue
			}
			name := strings.TrimSuffix(file.Name(), ".toml")
			path := filepath.Join(unexpandedDeployDir, file.Name())
			type stage struct {
				Filter string `toml:"filter"`
			}
			var data struct {
				InputTopics []string `toml:"input_topics"`
				Stages      []stage  `toml:"stages"`
			}
			topics := ""
			image := "<invalid>"
			if _, err := toml.DecodeFile(filepath.Join(deployDir, file.Name()), &data); err == nil && len(data.Stages) > 0 {
				topics = strings.Join(data.InputTopics, ", ")
				// If the image path starts with the expanded imageDir, replace with unexpanded
				imgPath := data.Stages[0].Filter
				if strings.HasPrefix(imgPath, cfg.ImageDir) && unexpandedImageDir != "" {
					rel, err := filepath.Rel(cfg.ImageDir, imgPath)
					if err == nil {
						imgPath = filepath.Join(unexpandedImageDir, rel)
					}
				}
				image = imgPath
			}
			rows = append(rows, []string{name, path, topics, image})
		}
		// Determine which columns fit in one row
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
		// Prepare filtered columns
		filteredColNames := colNames[:keep]
		filteredRows := [][]string{}
		for _, row := range rows {
			filteredRows = append(filteredRows, row[:keep])
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(filteredColNames)
		table.SetRowLine(true)
		table.SetAutoWrapText(false)
		for _, row := range filteredRows {
			table.Append(row)
		}
		table.Render()
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy [instance_name] [image] [topics...]",
	Short: "Deploy a flow instance",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}

		instanceName := args[0]
		imageRef := args[1]
		topics := args[2:]
		tick := 0
		if cmd.Flags().Changed("tick") {
			tick, _ = cmd.Flags().GetInt("tick")
		}
		deployDir := cfg.DeployDir
		if deployDir == "" {
			deployDir = os.Getenv("DEPLOY_DIR")
		}
		if deployDir == "" {
			deployDir = filepath.Join(filepath.Dir(cfg.ImageDir), "deployments")
		}
		if err := os.MkdirAll(deployDir, 0755); err != nil {
			return err
		}

		// Extract repository part from image reference (remove tag/digest)
		repoDir := imageRef
		if i := strings.IndexAny(imageRef, ":@"); i != -1 {
			repoDir = filepath.Base(imageRef[:i])
		}
		repoDir = strings.TrimSuffix(repoDir, "/")
		imagePath := filepath.Join(cfg.ImageDir, repoDir)
		filterPath := filepath.Join(imagePath, "dist/main.mjs")

		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			fmt.Printf("Image %s not found locally. Pulling...\n", imageRef)
			if err := imagepull.PullImage(cfg, imageRef); err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
		}

		if _, err := os.Stat(filterPath); os.IsNotExist(err) {
			fmt.Printf("Image %s does not contain the expected entrypoint. path=%s\n", imageRef, filterPath)
			return err
		}

		tomlPath := filepath.Join(deployDir, instanceName+".toml")
		type stageTOML struct {
			Filter           string `toml:"filter"`
			TickEverySeconds *int   `toml:"tick_every_seconds,omitempty"`
		}
		stages := []stageTOML{}
		var tickPtr *int
		if tick > 0 {
			tickPtr = &tick
		}
		stages = append(stages, stageTOML{
			Filter:           filterPath,
			TickEverySeconds: tickPtr,
		})
		data := struct {
			InputTopics []string    `toml:"input_topics"`
			Stages      []stageTOML `toml:"stages"`
		}{
			InputTopics: topics,
			Stages:      stages,
		}
		f, err := os.Create(tomlPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := toml.NewEncoder(f).Encode(data); err != nil {
			return err
		}
		fmt.Printf("Instance %s deployed at %s\n", instanceName, tomlPath)
		return nil
	},
}

var removeInstanceCmd = &cobra.Command{
	Use:   "remove [instance_name]",
	Short: "Remove a deployed flow instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}
		deployDir := cfg.DeployDir
		if deployDir == "" {
			deployDir = os.Getenv("DEPLOY_DIR")
		}
		if deployDir == "" {
			deployDir = filepath.Join(filepath.Dir(cfg.ImageDir), "deployments")
		}
		instanceName := args[0]
		// Find the matching file by instance name (basename without .toml)
		var matchFile string
		entries, err := os.ReadDir(deployDir)
		if err != nil {
			return fmt.Errorf("failed to read deploy dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
				continue
			}
			if strings.TrimSuffix(entry.Name(), ".toml") == instanceName {
				matchFile = filepath.Join(deployDir, entry.Name())
				break
			}
		}
		if matchFile == "" {
			return fmt.Errorf("instance '%s' not found in %s", instanceName, deployDir)
		}
		if err := os.Remove(matchFile); err != nil {
			return fmt.Errorf("failed to remove instance file: %w", err)
		}
		fmt.Printf("Instance %s removed (%s)\n", instanceName, matchFile)
		return nil
	},
}

func init() {
	instancesCmd.AddCommand(listInstancesCmd)
	instancesCmd.AddCommand(deployCmd)
	instancesCmd.AddCommand(removeInstanceCmd)
	deployCmd.Flags().Int("tick", 0, "Tick interval in seconds (optional)")
	flowsCmd.AddCommand(instancesCmd)
}

// Helper to get terminal width
func terminalSize() (width int, height int, err error) {
	fd := int(os.Stdout.Fd())
	w, h, err := term.GetSize(fd)
	return w, h, err
}
