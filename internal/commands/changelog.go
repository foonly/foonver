package commands

import (
	"fmt"

	"github.com/foonly/foonver/internal/changelog"
	"github.com/foonly/foonver/internal/config"
	"github.com/spf13/cobra"
)

var flagNext string
var flagLatest bool
var flagFormat string

var changelogCommand = &cobra.Command{
	Use:   "changelog",
	Short: "Generate a changelog from git commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.Conf.Info.Ok {
			return fmt.Errorf("not inside a Git repository")
		}

		format := flagFormat
		if format == "" {
			format = config.Conf.ChangelogFormat
		}

		content, err := changelog.Generate(format, flagNext, flagLatest)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	},
}

func init() {
	changelogCommand.Flags().StringVar(&flagNext, "next", "", "Next version name (e.g. v1.0.0 or Unreleased)")
	changelogCommand.Flags().BoolVar(&flagLatest, "latest", false, "Only output the latest version section")
	changelogCommand.Flags().StringVarP(&flagFormat, "format", "f", "", "Changelog format (markdown, wordpress)")
	rootCmd.AddCommand(changelogCommand)
}
