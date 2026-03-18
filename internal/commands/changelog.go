package commands

import (
	"fmt"

	"foonly.dev/foonver/internal/changelog"
	"github.com/spf13/cobra"
)

var flagNext string

var changelogCommand = &cobra.Command{
	Use:   "changelog",
	Short: "Generate a changelog from git commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		md, err := changelog.GenerateMarkdown(flagNext)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), md)
		return nil
	},
}

func init() {
	changelogCommand.Flags().StringVar(&flagNext, "next", "", "Next version name (e.g. v1.0.0 or Unreleased)")
	rootCmd.AddCommand(changelogCommand)
}
