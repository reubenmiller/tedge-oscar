package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var configPath string
var logLevel string

var rootCmd = &cobra.Command{
	Use:   "tedge-oscar",
	Short: "Manage thin-edge.io flows and OCI images",
	Example: `# List all images
$ tedge-oscar images list

# Pull an image from a registry
$ tedge-oscar images pull ghcr.io/thin-edge/connectivity-counter:1.0

# Deploy a flow instance
$ tedge-oscar instances deploy myinstance ghcr.io/thin-edge/connectivity-counter:1.0 --topics te/device/main///m/+
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(flowsCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (overrides default)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set the log level (debug, info, warn, error)")
	_ = rootCmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})
}
