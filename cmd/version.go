package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

// SetVersionInfo allows main.go to set version info injected by GoReleaser
func SetVersionInfo(v, c, d, b string) {
	version = v
	commit = c
	date = d
	builtBy = b
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the tedge-oscar CLI version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tedge-oscar version: %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("date: %s\n", date)
		fmt.Printf("built by: %s\n", builtBy)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
