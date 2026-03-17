package commands

import (
	"fmt"

	"foonly.dev/foonver/internal/changelog"
	"github.com/spf13/cobra"
)

var changelogCommand = &cobra.Command{
	Use:   "changelog",
	Short: "Generate a changelog from git commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		md, err := changelog.GenerateMarkdown()
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), md)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(changelogCommand)
}
