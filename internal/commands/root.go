package commands

import (
	"fmt"
	"os"

	"foonly.dev/foonver/internal/config"
	"foonly.dev/foonver/internal/git"
	"foonly.dev/foonver/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagQuiet   bool
	flagVerbose bool
	flagDebug   bool
	flagPush    bool
	flagNoPush  bool
)

var rootCmd = &cobra.Command{
	Use:   "foonver",
	Short: "Version Management Utility",
	Long:  "foonver is a lightweight CLI utility for automated Semantic Versioning (SemVer) management.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := git.RunPreflightChecks()
		config.Init()

		fmt.Printf("push=%v\n", config.Conf.Push)
		fmt.Printf("verbosity=%v\n", config.Conf.Verbosity)
		fmt.Printf("parser=%v\n", config.Conf.Parser)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
	RunE: version.RunVersion,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "normal", "n", false, "set normal informational output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable additional informational output")
	rootCmd.PersistentFlags().BoolVarP(&flagDebug, "debug", "d", false, "enable detailed debug output")
	rootCmd.PersistentFlags().BoolVar(&flagPush, "push", false, "push new versions to the remote repository")
	rootCmd.PersistentFlags().BoolVar(&flagNoPush, "no-push", false, "don't push new versions to the remote repository")

	rootCmd.MarkFlagsMutuallyExclusive("quiet", "normal", "verbose", "debug")
	rootCmd.MarkFlagsMutuallyExclusive("push", "no-push")

	viper.BindPFlag("push", rootCmd.PersistentFlags().Lookup("push"))
	viper.BindPFlag("no-push", rootCmd.PersistentFlags().Lookup("no-push"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("normal", rootCmd.PersistentFlags().Lookup("normal"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
