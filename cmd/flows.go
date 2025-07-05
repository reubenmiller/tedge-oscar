package cmd

import "github.com/spf13/cobra"

var flowsCmd = &cobra.Command{
	Use:   "flows",
	Short: "Manage flows and images",
}

func init() {
	flowsCmd.AddCommand(imagesCmd)
	flowsCmd.AddCommand(deployCmd)
}
