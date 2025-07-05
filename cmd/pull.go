package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull a flow image from an OCI registry",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Pulling flow image (OCI artifact)...")
		// TODO: Implement oras pull logic
	},
}
