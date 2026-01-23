package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/kanzi/kindplane/internal/cmd"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	// Pass version to cmd package for version update checking
	cmd.Version = Version

	if err := fang.Execute(
		context.Background(),
		cmd.RootCmd,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
