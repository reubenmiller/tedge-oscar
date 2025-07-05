package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	Use:   "deploy",
	Short: "Deploy a flow instance",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deploying flow instance...")
		// TODO: Implement deploy logic
	},
}

func init() {
	instancesCmd.AddCommand(listInstancesCmd)
	instancesCmd.AddCommand(deployCmd)
	flowsCmd.AddCommand(instancesCmd)
}
