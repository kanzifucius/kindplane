package main

import (
	"os"

	"github.com/kanzi/kindplane/internal/cmd"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, Commit, BuildTime)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
