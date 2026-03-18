// Package main provides the entry point for the foonver CLI utility,
// handling command-line arguments and orchestrating the versioning process.
package main

import (
	"foonly.dev/foonver/internal/commands"
)

func main() {
	commands.Execute()
}
