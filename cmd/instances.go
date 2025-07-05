package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-oscar/internal/config"
	"github.com/thin-edge/tedge-oscar/internal/imagepull"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage flow instances",
}

var listInstancesCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed flow instances",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing flow instances...")
		// TODO: Implement logic to list flow instances
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
			fmt.Printf("Image %s does not container the expected entrypoint. path=%s\n", filterPath)
			return err
		}

		tomlPath := filepath.Join(deployDir, instanceName+".toml")
		data := struct {
			InputTopics []string `toml:"input_topics"`
			Stages      []struct {
				Filter           string `toml:"filter"`
				TickEverySeconds int    `toml:"tick_every_seconds,omitempty"`
			} `toml:"stages"`
		}{
			InputTopics: topics,
			Stages: []struct {
				Filter           string `toml:"filter"`
				TickEverySeconds int    `toml:"tick_every_seconds,omitempty"`
			}{
				{Filter: filterPath, TickEverySeconds: tick},
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
		fmt.Printf("Instance %s deployed at %s\n", instanceName, tomlPath)
		return nil
	},
}

func init() {
	instancesCmd.AddCommand(listInstancesCmd)
	instancesCmd.AddCommand(deployCmd)
	deployCmd.Flags().Int("tick", 0, "Tick interval in seconds (optional)")
	flowsCmd.AddCommand(instancesCmd)
}
