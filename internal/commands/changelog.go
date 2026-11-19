package commands

import (
	"fmt"
	"os"

	"github.com/foonly/foonver/internal/changelog"
	"github.com/foonly/foonver/internal/config"
	"github.com/spf13/cobra"
)

var flagNext string
var flagLatest bool
var flagFormat string
var flagWrite bool
var flagForce bool

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

		if flagForce && !flagWrite {
			return fmt.Errorf("--force can only be used together with --write")
		}

		if flagWrite {
			config.Conf.ChangelogFormat = format
			if flagForce {
				config.Conf.ChangelogForce = true
			}
			filePath, content, err := changelog.RenderChangelog(flagNext)
			if err != nil {
				return err
			}
			if config.Conf.DryRun {
				fmt.Fprintf(cmd.ErrOrStderr(), "Dry run: would write %s\n", filePath)
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write changelog to %s: %w", filePath, err)
			}
			if config.Conf.Verbosity != config.Quiet {
				fmt.Fprintf(cmd.ErrOrStderr(), "Updated %s\n", filePath)
			}
			return nil
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
	changelogCommand.Flags().BoolVarP(&flagWrite, "write", "w", false, "Update the changelog file instead of printing (keeps existing sections of released versions)")
	changelogCommand.Flags().BoolVar(&flagForce, "force", false, "With --write: regenerate every section, discarding manual edits")
	changelogCommand.MarkFlagsMutuallyExclusive("write", "latest")
	rootCmd.AddCommand(changelogCommand)
}
