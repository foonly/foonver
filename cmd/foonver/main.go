// Package main provides the entry point for the foonver CLI utility,
// handling command-line arguments and orchestrating the versioning process.
package main

import (
	"os"

	"github.com/foonly/foonver/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
