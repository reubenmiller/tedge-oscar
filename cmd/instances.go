package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/olekukonko/tablewriter"
	"github.com/reubenmiller/tedge-oscar/internal/artifact"
	"github.com/reubenmiller/tedge-oscar/internal/config"
	"github.com/reubenmiller/tedge-oscar/internal/imagepull"
	"github.com/reubenmiller/tedge-oscar/internal/util"
	"github.com/reubenmiller/tedge-oscar/pkg/maputil"
	"github.com/reubenmiller/tedge-oscar/pkg/types/flows"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage flow instances",
	Example: `# List all deployed flow instances
$ tedge-oscar flows instances list

# Deploy a new instance
$ tedge-oscar flows instances deploy myinstance ghcr.io/reubenmiller/connectivity-counter:1.0 --topics te/device/main///m/+

# Remove an instance
$ tedge-oscar flows instances remove myinstance
`,
}

var listInstancesCmd = &cobra.Command{
	Use:     "list",
	Short:   "List deployed flow instances",
	Aliases: []string{"ps", "ls"},
	Example: `# List all deployed flow instances
$ tedge-oscar flows instances list`,
	SilenceUsage: true, // Do not show help on runtime errors
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		cfgPath := configPath
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
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
			return fmt.Errorf("failed to read deploy dir: %w", err)
		}
		// Use the unexpanded deployDir from config for display
		unexpandedDeployDir := cfg.UnexpandedDeployDir
		if unexpandedDeployDir == "" {
			unexpandedDeployDir = "$DEPLOY_DIR"
		}
		// Get the unexpanded image_dir from the config for display
		unexpandedImageDir := cfg.UnexpandedImageDir
		if unexpandedImageDir == "" {
			unexpandedImageDir = "$IMAGE_DIR"
		}
		// Prepare all rows first
		rows := [][]string{}
		colNames := []string{"name", "path", "topics", "image", "imageVersion"}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") {
				continue
			}
			name := strings.TrimSuffix(file.Name(), ".toml")
			path := filepath.Join(unexpandedDeployDir, file.Name())
			var data flows.InstanceFile
			topics := ""
			image := "<invalid>"
			imageVersion := "<unknown>"
			if _, err := toml.DecodeFile(filepath.Join(deployDir, file.Name()), &data); err == nil && len(data.Steps) > 0 {
				topics = strings.Join(data.Input.MQTT.Topics, ", ")
				// If the image path starts with the expanded imageDir, replace with unexpanded
				imgPath := data.Steps[0].Script
				if strings.HasPrefix(imgPath, cfg.ImageDir) && unexpandedImageDir != "" {
					rel, err := filepath.Rel(cfg.ImageDir, imgPath)
					if err == nil {
						imgPath = filepath.Join(unexpandedImageDir, rel)
					}
				}
				image = imgPath
				// Try to get image version from manifest.json
				manifestPath := ""
				if strings.HasPrefix(data.Steps[0].Script, cfg.ImageDir) {
					// e.g. /Users/you/.tedge/images/imagename/dist/main.mjs
					imgDir := filepath.Dir(filepath.Dir(data.Steps[0].Script))
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
				imgDir := filepath.Base(filepath.Dir(filepath.Dir(data.Steps[0].Script)))
				if imgDir != "." && imgDir != "/" && imgDir != "" {
					imageName = artifact.TrimVersion(imgDir)
				}
			}
			rows = append(rows, []string{name, path, topics, imageName, imageVersion})
		}
		if len(rows) == 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "No flow instances are currently deployed.")
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
				_ = enc.Encode(obj)
			}
			return nil
		}
		// Determine which columns fit in one row
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
		// Remove columns from right until fits
		keep := len(colNames)
		for total > maxWidth && keep > 1 {
			keep--
			total -= (colWidths[keep] + 1)
		}
		// Prepare filtered columns
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

var deployCmd = &cobra.Command{
	Use:     "deploy [instance_name] [image]",
	Short:   "Deploy a flow instance",
	Aliases: []string{"run"},
	Example: `# Deploy a new instance using a specific image and topic
$ tedge-oscar flows instances deploy myinstance ghcr.io/reubenmiller/connectivity-counter:1.0 --topics te/device/main///m/+`,
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
		name, err := artifact.ParseName(imageRef, false)
		if err != nil {
			return err
		}

		imagePath := filepath.Join(cfg.ImageDir, name)
		scriptPath := filepath.Join(imagePath, "dist/main.mjs")
		fmt.Fprintf(cmd.ErrOrStderr(), "script path: %s\n", scriptPath)

		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Image %s not found locally. Pulling...\n", imageRef)
			if err := imagepull.PullImage(cfg, imageRef, cfg.ImageDir); err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
		}

		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Image %s does not contain the expected entrypoint. path=%s\n", imageRef, scriptPath)
			return err
		}

		tomlPath := filepath.Join(deployDir, instanceName+".toml")
		// Look for the first existing TOML config file in priority order
		var imageFlowDefinitionPath string
		for _, candidate := range []string{"flow.toml", "pipeline.toml"} {
			candidatePath := filepath.Join(imagePath, candidate)
			if _, err := os.Stat(candidatePath); err == nil {
				imageFlowDefinitionPath = candidatePath
				break
			}
		}
		if _, err := os.Stat(imageFlowDefinitionPath); err == nil {
			// Load flow definition as a map to preserve all fields
			var m map[string]interface{}
			if _, err := toml.DecodeFile(imageFlowDefinitionPath, &m); err != nil {
				return fmt.Errorf("failed to parse %s: %w", imageFlowDefinitionPath, err)
			}
			// Always update topics from CLI using a helper to set nested keys
			if err := maputil.SetNestedMapValue(m, []string{"input", "mqtt", "topics"}, topics); err != nil {
				return fmt.Errorf("failed to set input.mqtt.topics: %w", err)
			}
			// If tick is set, update all steps with tick value
			if stepsRaw, ok := m["steps"]; ok {
				var newSteps []map[string]interface{}
				switch steps := stepsRaw.(type) {
				case []map[string]interface{}:
					newSteps = make([]map[string]interface{}, len(steps))
					for i, stepMap := range steps {
						stepMap["script"] = scriptPath
						if tick > 0 {
							stepMap["tick_every_seconds"] = tick
						}
						newSteps[i] = stepMap
					}
				case []interface{}:
					newSteps = make([]map[string]interface{}, len(steps))
					for i, s := range steps {
						if stageMap, ok := s.(map[string]interface{}); ok {
							stageMap["script"] = scriptPath
							if tick > 0 {
								stageMap["tick_every_seconds"] = tick
							}
							newSteps[i] = stageMap
						}
					}
				}
				m["steps"] = newSteps
			}
			f, err := os.Create(tomlPath)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := toml.NewEncoder(f).Encode(m); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Instance %s deployed at %s\n", instanceName, tomlPath)
			return nil
		} else {
			// Fallback: create minimal config
			var tickPtr *int
			if tick > 0 {
				tickPtr = &tick
			}
			data := map[string]interface{}{
				"input_topics": topics,
				"steps": []map[string]interface{}{
					{
						"script":             scriptPath,
						"tick_every_seconds": tickPtr,
					},
				},
			}
			f, err := os.Create(tomlPath)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := toml.NewEncoder(f).Encode(data); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Instance %s deployed at %s\n", instanceName, tomlPath)
			return nil
		}
	},
}

var removeInstanceCmd = &cobra.Command{
	Use:     "remove [instance_name]",
	Short:   "Remove a deployed flow instance",
	Aliases: []string{"rm"},
	Example: `# Remove a deployed instance
$ tedge-oscar flows instances remove myinstance`,
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
		fmt.Fprintf(cmd.ErrOrStderr(), "Instance %s removed (%s)\n", instanceName, matchFile)
		return nil
	},
}

func init() {
	defaultOutput := "jsonl"
	if util.Isatty(os.Stdout.Fd()) {
		defaultOutput = "table"
	}
	listInstancesCmd.Flags().StringP("output", "o", defaultOutput, "Output format: table|jsonl")
	_ = listInstancesCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "jsonl"}, cobra.ShellCompDirectiveNoFileComp
	})
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
			"te/device/main/service/tedge-mapper-bridge-c8y/status/health\tbuilt-in bridge status",
			"te/device/main/service/mosquitto-c8y-bridge/status/health\tmosquitto bridge status",
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
