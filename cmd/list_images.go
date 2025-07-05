package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listImagesCmd = &cobra.Command{
	Use:   "list",
	Short: "List available flow images",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing flow images...")
		// TODO: Implement logic to list OCI images
	},
}

func init() {
	imagesCmd.AddCommand(listImagesCmd)
}
