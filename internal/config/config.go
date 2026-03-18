package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"

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
	Push      bool   `mapstructure:"push"`
	Prefix    string `mapstructure:"prefix"`
	Verbosity Level  `mapstructure:"verbosity"`
	Parser    string `mapstructure:"parser"`
	Changelog bool   `mapstructure:"changelog"`
	File      string `mapstructure:"file"`
	Info      GitInfo
}

var Conf Config

func Init() {
	viper.SetConfigName("foonver")

	viper.AddConfigPath("/etc/foonver")
	viper.AddConfigPath(xdg.ConfigHome)
	viper.AddConfigPath(Conf.Info.RootDir)

	viper.SetDefault("push", false)
	viper.SetDefault("prefix", "v")
	viper.SetDefault("verbosity", "normal")
	viper.SetDefault("parser", "all")
	viper.SetDefault("changelog", false)
	viper.SetDefault("file", "CHANGELOG.md")

	// Find and read the config file
	err := viper.ReadInConfig()

	if err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFound) {
			fmt.Printf("No config file found, using defaults\n")
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
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
		fmt.Printf("Error unmarshalling config: %v\n", err)
	}

	processFlags()
}

func processFlags() {
	// This function can be used to process command-line flags and override config values if needed.
	if viper.IsSet("push") && viper.GetBool("push") {
		Conf.Push = true
	}
	if viper.IsSet("no-push") && viper.GetBool("no-push") {
		Conf.Push = false
	}

	if viper.IsSet("changelog") && viper.GetBool("changelog") {
		Conf.Changelog = true
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
