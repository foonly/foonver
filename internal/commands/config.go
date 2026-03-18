package commands

import (
	"fmt"

	"foonly.dev/foonver/internal/config"
	"github.com/spf13/cobra"
)

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "Show the current settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Version: %v\n\n", config.AppVersion)
		fmt.Printf("Push: %v\n", config.Conf.Push)
		fmt.Printf("Prefix: %s\n", config.Conf.Prefix)
		fmt.Printf("Verbosity: %s\n", config.Conf.Verbosity)
		fmt.Printf("Parser: %s\n", config.Conf.Parser)
		fmt.Printf("Changelog: %v\n", config.Conf.Changelog)
		fmt.Printf("File: %s\n", config.Conf.File)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCommand)
}
