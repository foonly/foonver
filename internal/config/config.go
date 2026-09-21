package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/adrg/xdg"
	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var AppVersion = "dev"

// Level represents the verbosity level of the application.
type Level int

const (
	// Quiet suppresses all non-essential output.
	Quiet Level = iota
	// Normal is the default verbosity level.
	Normal
	// Verbose enables additional informational output.
	Verbose
	// Debug enables detailed debug output.
	Debug
)

// String returns the human-readable name of the verbosity level.
func (l Level) String() string {
	switch l {
	case Quiet:
		return "quiet"
	case Normal:
		return "normal"
	case Verbose:
		return "verbose"
	case Debug:
		return "debug"
	default:
		return "unknown"
	}
}

// UnmarshalText implements encoding.TextUnmarshaler to allow loading from strings.
func (l *Level) UnmarshalText(text []byte) error {
	switch s := string(text); s {
	case "quiet":
		*l = Quiet
	case "normal":
		*l = Normal
	case "verbose":
		*l = Verbose
	case "debug":
		*l = Debug
	default:
		return fmt.Errorf("unknown verbosity level: %s", s)
	}
	return nil
}

type GitInfo struct {
	Ok        bool
	Clean     bool
	HasRemote bool
	RootDir   string
}

type Config struct {
	DryRun             bool     `mapstructure:"dry-run"`
	Push               bool     `mapstructure:"push"`
	Prefix             string   `mapstructure:"prefix"`
	Remote             string   `mapstructure:"remote"`
	Verbosity          Level    `mapstructure:"verbosity"`
	Parser             string   `mapstructure:"parser"`
	Changelog          bool     `mapstructure:"changelog"`
	ChangelogFile      string   `mapstructure:"changelog-file"`
	ChangelogFormat    string   `mapstructure:"changelog-format"`
	ChangelogStart     string   `mapstructure:"changelog-start"`
	ChangelogEnd       string   `mapstructure:"changelog-end"`
	VersionSync        []string `mapstructure:"version-sync"`
	ReleaseNotes       string   `mapstructure:"release-notes"`
	PrintVersion       bool     `mapstructure:"print-version"`
	JSON               bool     `mapstructure:"json"`
	CommitMessage      string   `mapstructure:"commit-message"`
	CommitSuffix       string   `mapstructure:"commit-suffix"`
	VersionFile        string   `mapstructure:"version-file"`
	Prerelease         string   `mapstructure:"prerelease"`
	Promote            bool     `mapstructure:"promote"`
	IncludePrereleases bool     `mapstructure:"include-prereleases"`
	Info               GitInfo
}

var Conf Config

func Init() {
	viper.SetConfigName(".foonver")

	viper.AddConfigPath(Conf.Info.RootDir)
	viper.AddConfigPath(xdg.ConfigHome)
	viper.AddConfigPath("/etc/foonver")

	viper.SetDefault("dry-run", false)
	viper.SetDefault("push", false)
	viper.SetDefault("prefix", "v")
	viper.SetDefault("remote", "origin")
	viper.SetDefault("verbosity", "normal")
	viper.SetDefault("parser", "all")
	viper.SetDefault("changelog", false)
	viper.SetDefault("changelog-file", "CHANGELOG.md")
	viper.SetDefault("changelog-format", "markdown")
	viper.SetDefault("changelog-start", "")
	viper.SetDefault("changelog-end", "")
	viper.SetDefault("version-sync", []string{})
	viper.SetDefault("release-notes", "")
	viper.SetDefault("print-version", false)
	viper.SetDefault("json", false)
	viper.SetDefault("commit-message", "")
	viper.SetDefault("commit-suffix", "")
	viper.SetDefault("version-file", "")
	viper.SetDefault("prerelease", "")
	viper.SetDefault("promote", false)
	viper.SetDefault("include-prereleases", false)

	// Find and read the config file (.foonver first, fallback to foonver)
	err := viper.ReadInConfig()
	if err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFound) {
			viper.SetConfigName("foonver")
			err = viper.ReadInConfig()
			if err != nil {
				if errors.As(err, &configNotFound) {
					fmt.Fprintf(os.Stderr, "No config file found, using defaults\n")
				} else {
					fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
					os.Exit(1)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Use a DecoderHook to support the TextUnmarshaler interface
	err = viper.Unmarshal(&Conf, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			func(f reflect.Type, t reflect.Type, data any) (any, error) {
				if f.Kind() != reflect.String || t != reflect.TypeFor[Level]() {
					return data, nil
				}
				var l Level
				if err := l.UnmarshalText([]byte(data.(string))); err != nil {
					return nil, err
				}
				return l, nil
			},
		),
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config: %v\n", err)
	}

	processFlags()
	checkDuplicateConfigs()
}

func checkDuplicateConfigs() {
	usedFile := viper.ConfigFileUsed()
	if usedFile == "" {
		return
	}
	base := filepath.Base(usedFile)
	if strings.HasPrefix(base, ".") {
		dir := filepath.Dir(usedFile)
		legacyNames := []string{"foonver.toml", "foonver.yaml", "foonver.yml", "foonver.json"}
		for _, leg := range legacyNames {
			legPath := filepath.Join(dir, leg)
			if _, err := os.Stat(legPath); err == nil {
				if Conf.Verbosity != Quiet && !Conf.PrintVersion && !Conf.JSON {
					fmt.Fprintf(os.Stderr, "Warning: Both %s and %s exist. Using %s and ignoring %s.\n", base, leg, base, leg)
				}
				break
			}
		}
	}
}

func processFlags() {
	// This function can be used to process command-line flags and override config values if needed.
	if viper.IsSet("dry-run") && viper.GetBool("dry-run") {
		Conf.DryRun = true
	}

	if viper.IsSet("push") && viper.GetBool("push") {
		Conf.Push = true
	}
	if viper.IsSet("no-push") && viper.GetBool("no-push") {
		Conf.Push = false
	}

	if viper.IsSet("remote") {
		Conf.Remote = viper.GetString("remote")
	}

	if viper.IsSet("changelog") && viper.GetBool("changelog") {
		Conf.Changelog = true
	}

	if viper.IsSet("changelog-file") {
		Conf.ChangelogFile = viper.GetString("changelog-file")
	} else if viper.IsSet("file") {
		Conf.ChangelogFile = viper.GetString("file")
	}

	if viper.IsSet("changelog-format") {
		Conf.ChangelogFormat = viper.GetString("changelog-format")
	}

	if viper.IsSet("changelog-start") {
		Conf.ChangelogStart = viper.GetString("changelog-start")
	}

	if viper.IsSet("changelog-end") {
		Conf.ChangelogEnd = viper.GetString("changelog-end")
	}

	if viper.IsSet("version-sync") {
		Conf.VersionSync = viper.GetStringSlice("version-sync")
	}

	if viper.IsSet("release-notes") {
		Conf.ReleaseNotes = viper.GetString("release-notes")
	}

	if viper.IsSet("print-version") && viper.GetBool("print-version") {
		Conf.PrintVersion = true
	}

	if viper.IsSet("json") && viper.GetBool("json") {
		Conf.JSON = true
	}

	if viper.IsSet("commit-message") {
		Conf.CommitMessage = viper.GetString("commit-message")
	}

	if viper.IsSet("commit-suffix") {
		Conf.CommitSuffix = viper.GetString("commit-suffix")
	}

	if viper.IsSet("version-file") {
		Conf.VersionFile = viper.GetString("version-file")
	}

	if viper.IsSet("prerelease") {
		Conf.Prerelease = viper.GetString("prerelease")
	}

	if viper.IsSet("promote") && viper.GetBool("promote") {
		Conf.Promote = true
	}

	if viper.IsSet("include-prereleases") && viper.GetBool("include-prereleases") {
		Conf.IncludePrereleases = true
	}

	if viper.IsSet("quiet") && viper.GetBool("quiet") {
		Conf.Verbosity = Quiet
	} else if viper.IsSet("normal") && viper.GetBool("normal") {
		Conf.Verbosity = Normal
	} else if viper.IsSet("verbose") && viper.GetBool("verbose") {
		Conf.Verbosity = Verbose
	} else if viper.IsSet("debug") && viper.GetBool("debug") {
		Conf.Verbosity = Debug
	}

}
