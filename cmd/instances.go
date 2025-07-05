package cmd

import (
	"encoding/json"
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
	Use:          "list",
	Short:        "List deployed flow instances",
	SilenceUsage: true, // Do not show help on runtime errors
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
		colNames := []string{"NAME", "PATH", "TOPICS", "IMAGE", "IMAGE_VERSION"}
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
			imageVersion := "<unknown>"
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
				// Try to get image version from manifest.json
				manifestPath := ""
				if strings.HasPrefix(data.Stages[0].Filter, cfg.ImageDir) {
					// e.g. /Users/you/.tedge/images/imagename/dist/main.mjs
					imgDir := filepath.Dir(filepath.Dir(data.Stages[0].Filter))
					manifestPath = filepath.Join(imgDir, "manifest.json")
				}
				if manifestPath != "" {
					if f, err := os.Open(manifestPath); err == nil {
						var manifest map[string]interface{}
						if err := json.NewDecoder(f).Decode(&manifest); err == nil {
							if ann, ok := manifest["annotations"].(map[string]interface{}); ok {
								if v, ok := ann["org.opencontainers.image.version"].(string); ok {
									imageVersion = v
								}
							}
						}
						f.Close()
					}
				}
			}
			// Only show the image name (not the path)
			imageName := "<invalid>"
			if image != "<invalid>" {
				// Try to extract the image directory name from the path
				imgDir := filepath.Base(filepath.Dir(filepath.Dir(data.Stages[0].Filter)))
				if imgDir != "." && imgDir != "/" && imgDir != "" {
					imageName = imgDir
				}
			}
			rows = append(rows, []string{name, path, topics, imageName, imageVersion})
		}
		if len(rows) == 0 {
			fmt.Println("No flow instances are currently deployed.")
			return
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
	Use:          "deploy [instance_name] [image]",
	Short:        "Deploy a flow instance",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true, // Do not show help on runtime errors
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Only complete for the image argument (second arg, index 1)
		if len(args) != 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
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
			if strings.HasPrefix(name, toComplete) {
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

		instanceName := args[0]
		imageRef := args[1]
		topics, err := cmd.Flags().GetStringArray("topics")
		if err != nil {
			return err
		}
		if len(topics) == 0 {
			return fmt.Errorf("at least one --topics value must be provided")
		}
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
	Use:          "remove [instance_name]",
	Short:        "Remove a deployed flow instance",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // Do not show help on runtime errors
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		deployDir := cfg.DeployDir
		if deployDir == "" {
			deployDir = os.Getenv("DEPLOY_DIR")
		}
		if deployDir == "" {
			deployDir = filepath.Join(filepath.Dir(cfg.ImageDir), "deployments")
		}
		entries, err := os.ReadDir(deployDir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		provided := make(map[string]struct{})
		for _, arg := range args {
			provided[arg] = struct{}{}
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".toml")
			if _, already := provided[name]; already {
				continue
			}
			if strings.HasPrefix(name, toComplete) {
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
	deployCmd.Flags().StringArray("topics", nil, "Input topics (repeatable, required)")
	deployCmd.MarkFlagRequired("topics")
	_ = deployCmd.RegisterFlagCompletionFunc("topics", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Common thin-edge.io MQTT topics
		commonTopics := []string{
			// main device values
			"te/device/main//\tRegistration (main device)",
			"te/device/main///m/+\tMeasurements (main device)",
			"te/device/main///e/+\tEvents (main device)",
			"te/device/main///a/+\tAlarms (main device)",
			"te/device/main///twin/+\tTwin (main device)",
			"te/device/main///cmd/+/+\tCommands (main device)",
			// all devices/services
			"te/+/+/+/+\tRegistration (all devices)",
			"te/+/+/+/+/m/+\tMeasurements (all devices)",
			"te/+/+/+/+/e/+\tEvents (all devices)",
			"te/+/+/+/+/a/+\tAlarms (all devices)",
			"te/+/+/+/+/twin/+\tTwin (all devices)",
			"te/+/+/+/+/cmd/+/+\tCommands (all devices)",
		}

		// TODO Add common suffixes to the given users options
		// commonSuffixes := []string{
		// 	"/m/",
		// 	"/e/",
		// 	"/a/",
		// 	"/twin/",
		// 	"/cmd/+/+",
		// }

		// if len(strings.Split(toComplete, "/")) == 5 {
		// 	for _, suffix := range commonSuffixes {
		// 		commonTopics = append(commonTopics, toComplete+suffix)
		// 	}
		// }

		var completions []string
		for _, topic := range commonTopics {
			if strings.HasPrefix(topic, toComplete) {
				completions = append(completions, topic)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	flowsCmd.AddCommand(instancesCmd)
}

// Helper to get terminal width
func terminalSize() (width int, height int, err error) {
	fd := int(os.Stdout.Fd())
	w, h, err := term.GetSize(fd)
	return w, h, err
}
