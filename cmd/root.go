package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "tedge-oscar",
	Short: "Manage thin-edge.io flows and OCI images",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(flowsCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (overrides default)")
}
