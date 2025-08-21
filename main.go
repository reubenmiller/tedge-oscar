package main

import "github.com/thin-edge/tedge-oscar/cmd"

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
