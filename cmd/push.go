package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a flow image to an OCI registry",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Pushing flow image (OCI artifact)...")
		// TODO: Implement oras push logic
	},
}

func init() {
	imagesCmd.AddCommand(pushCmd)
}
