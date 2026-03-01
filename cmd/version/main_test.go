package main

import (
	"os"
	"testing"

	"foonly.dev/foonver/internal/version"

	"github.com/Masterminds/semver/v3"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		want     string
		wantErr  bool
	}{
		{
			name:     "package.json valid",
			filename: "package.json",
			content:  `{"name": "test", "version": "1.2.3"}`,
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "version.json valid",
			filename: "version.json",
			content:  `{"version": "v2.0.0-rc.1"}`,
			want:     "v2.0.0-rc.1",
			wantErr:  false,
		},
		{
			name:     "version.toml valid double quotes",
			filename: "version.toml",
			content:  `version = "1.0.0"`,
			want:     "1.0.0",
			wantErr:  false,
		},
		{
			name:     "version.toml valid single quotes",
			filename: "version.toml",
			content:  `version = '1.0.0'`,
			want:     "1.0.0",
			wantErr:  false,
		},
		{
			name:     "version.txt valid",
			filename: "version.txt",
			content:  "  1.2.3-beta.1  \n",
			want:     "1.2.3-beta.1",
			wantErr:  false,
		},
		{
			name:     "version.md valid header",
			filename: "version.md",
			content:  "# Project Name\n\nVersion 1.2.3\n",
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "version.md valid badge",
			filename: "version.md",
			content:  "![version](https://img.shields.io/badge/version-v2.0.0-blue)",
			want:     "2.0.0-blue",
			wantErr:  false,
		},
		{
			name:     "unsupported file",
			filename: "version.xml",
			content:  "<version>1.0.0</version>",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := version.ExtractVersion(tt.filename, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineNextVersionTarget(t *testing.T) {
	current, _ := semver.NewVersion("1.2.3")

	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{
			name:    "major",
			target:  "major",
			want:    "2.0.0",
			wantErr: false,
		},
		{
			name:    "minor",
			target:  "minor",
			want:    "1.3.0",
			wantErr: false,
		},
		{
			name:    "patch",
			target:  "patch",
			want:    "1.2.4",
			wantErr: false,
		},
		{
			name:    "specific version",
			target:  "2.0.0-rc.1",
			want:    "2.0.0-rc.1",
			wantErr: false,
		},
		{
			name:    "invalid specific version",
			target:  "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := version.DetermineNextVersion(current, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineNextVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("determineNextVersion() got = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestUpdateVersionFile(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		content    string
		oldVersion string
		newVersion string
		want       string
		wantErr    bool
	}{
		{
			name:       "json update",
			filename:   "package.json",
			content:    `{"version": "1.2.3"}`,
			oldVersion: "1.2.3",
			newVersion: "1.2.4",
			want:       `{"version": "1.2.4"}`,
			wantErr:    false,
		},
		{
			name:       "toml update",
			filename:   "version.toml",
			content:    `version = "1.2.3"`,
			oldVersion: "1.2.3",
			newVersion: "2.0.0",
			want:       `version = "2.0.0"`,
			wantErr:    false,
		},
		{
			name:       "txt update",
			filename:   "version.txt",
			content:    "v1.2.3\n",
			oldVersion: "v1.2.3",
			newVersion: "v1.3.0",
			want:       "v1.3.0\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file to safely test file writing
			f, err := os.CreateTemp("", "*_"+tt.filename)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())

			err = version.UpdateVersionFile(f.Name(), tt.oldVersion, tt.newVersion, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("updateVersionFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got, err := os.ReadFile(f.Name())
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tt.want {
					t.Errorf("updateVersionFile() got = %v, want %v", string(got), tt.want)
				}
			}
		})
	}
}
