package cmd

import "github.com/spf13/cobra"

var flowsCmd = &cobra.Command{
	Use:     "flows",
	Aliases: []string{"flow"},
	Short:   "Manage flows and images",
}

func init() {
	flowsCmd.AddCommand(imagesCmd)
}
