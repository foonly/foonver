package commands

import (
	"github.com/foonly/foonver/internal/config"
	"github.com/foonly/foonver/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagQuiet        bool
	flagNormal       bool
	flagVerbose      bool
	flagDebug        bool
	flagPush         bool
	flagNoPush       bool
	flagChangelog    bool
	flagDryRun       bool
	flagReleaseNotes string
	flagPrintVersion bool
	flagJSON         bool
	flagSync         []string
)

var rootCmd = &cobra.Command{
	Use:   "foonver",
	Short: "Version Management Utility",
	Long:  "foonver is a lightweight CLI utility for automated Semantic Versioning (SemVer) management.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_ = git.EnsureRepo()
		config.Init()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&flagNormal, "normal", "n", false, "set normal informational output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable additional informational output")
	rootCmd.PersistentFlags().BoolVarP(&flagDebug, "debug", "d", false, "enable detailed debug output")
	rootCmd.PersistentFlags().BoolVar(&flagPush, "push", false, "push new versions to the remote repository")
	rootCmd.PersistentFlags().BoolVar(&flagNoPush, "no-push", false, "don't push new versions to the remote repository")
	rootCmd.PersistentFlags().BoolVarP(&flagChangelog, "changelog", "c", false, "automatically update CHANGELOG.md")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "simulate the full flow without changing files, tags, or git state")
	rootCmd.PersistentFlags().StringVar(&flagReleaseNotes, "release-notes", "", "write delta changelog to this file")
	rootCmd.PersistentFlags().BoolVar(&flagPrintVersion, "print-version", false, "only print the new version number")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output result as JSON")
	rootCmd.PersistentFlags().StringSliceVar(&flagSync, "sync", []string{}, "files to synchronize version in")

	rootCmd.MarkFlagsMutuallyExclusive("quiet", "normal", "verbose", "debug")
	rootCmd.MarkFlagsMutuallyExclusive("push", "no-push")

	viper.BindPFlag("push", rootCmd.PersistentFlags().Lookup("push"))
	viper.BindPFlag("no-push", rootCmd.PersistentFlags().Lookup("no-push"))
	viper.BindPFlag("changelog", rootCmd.PersistentFlags().Lookup("changelog"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("normal", rootCmd.PersistentFlags().Lookup("normal"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("release-notes", rootCmd.PersistentFlags().Lookup("release-notes"))
	viper.BindPFlag("print-version", rootCmd.PersistentFlags().Lookup("print-version"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
	viper.BindPFlag("version-sync", rootCmd.PersistentFlags().Lookup("sync"))

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
