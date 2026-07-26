// Package main implements gh-sweep: a Bubble Tea TUI gh extension for cross-repository GitHub maintenance.
package main

import (
	"fmt"

	"github.com/KyleKing/gh-sweep/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Execute(fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date))
}
