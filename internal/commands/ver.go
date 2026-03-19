package commands

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/foonly/foonver/internal/version"
	"github.com/spf13/cobra"
)

var autoCommand = &cobra.Command{
	Use:   "auto",
	Short: "Determine the version by git commits (default)",
	RunE:  version.RunVersion,
}

var majorCommand = &cobra.Command{
	Use:   "major",
	Short: "Bump the major version",
	RunE:  version.RunVersion,
}

var minorCommand = &cobra.Command{
	Use:   "minor",
	Short: "Bump the minor version",
	RunE:  version.RunVersion,
}

var patchCommand = &cobra.Command{
	Use:   "patch",
	Short: "Bump the patch version",
	RunE:  version.RunVersion,
}

var verCommand = &cobra.Command{
	Use:   "ver [version]",
	Short: "Set the project version",
	Args:  validateVerArg,
	RunE:  version.RunVersion,
}

func init() {
	rootCmd.AddCommand(autoCommand)
	rootCmd.AddCommand(majorCommand)
	rootCmd.AddCommand(minorCommand)
	rootCmd.AddCommand(patchCommand)
	rootCmd.AddCommand(verCommand)
}

func validateVerArg(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 1 {
		return fmt.Errorf("expected semantic version")
	}

	if _, err := semver.NewVersion(args[0]); err != nil {
		return fmt.Errorf("invalid version %q: %w", args[0], err)
	}
	return nil
}
