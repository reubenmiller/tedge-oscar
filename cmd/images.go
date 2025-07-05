package cmd

import "github.com/spf13/cobra"

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage flow images as OCI artifacts",
}

func init() {
	imagesCmd.AddCommand(pullCmd)
	imagesCmd.AddCommand(pushCmd)
}
