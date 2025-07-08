package main

import "github.com/reubenmiller/tedge-oscar/cmd"

var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "golang"
)

func main() {
	cmd.SetVersionInfo(version, commit, date, builtBy)
	cmd.Execute()
}
