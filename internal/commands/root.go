package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagQuiet   bool
	flagVerbose bool
	flagDebug   bool
)

var rootCmd = &cobra.Command{
	Use:   "foonver",
	Short: "Version Management Utility",
	Long:  "foonver is a lightweight CLI utility for automated Semantic Versioning (SemVer) management.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		Version := "foo"
		fmt.Printf("Version: %s\n\n", Version)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable additional informational output")
	rootCmd.PersistentFlags().BoolVarP(&flagDebug, "debug", "d", false, "enable detailed debug output")

	rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose", "debug")

	// Hide the default completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
