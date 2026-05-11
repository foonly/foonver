package changelog

import (
	"strings"
	"testing"
)

func TestFindVerRegex(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"0.0.0", true},
		{"1.2.3", true},
		{"v1.2.3", true},
		{"v0.11.0", true},
		{"10.20.30", true},
		{"v1.2.3-beta.1", true},
		{"1.2.3-rc.20", true},
		{"feat: add something", false},
		{"fix: 1.2.3", false},
		{"version 1.2.3", false},
		{"v1.2", false},
		{"1.2", false},
		{"not a version", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := findVer.MatchString(tt.msg)
			if got != tt.want {
				t.Errorf("findVer.MatchString(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestMessageFiltering(t *testing.T) {
	tests := []struct {
		msg  string
		want bool // true if it should be filtered out
	}{
		{"feat: add something", false},
		{"fix: bug", false},
		{"v1.2.3", true},
		{"1.2.3", true},
		{"chore: bump version [skip ci]", true},
		{"docs: update readme [skip action]", true},
		{"Merge branch 'main'", true},
		{"v0.11.1", true},
		{"just a message", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			// This matches the logic in filteredCommits
			isFiltered := false
			if strings.HasPrefix(tt.msg, "Merge ") {
				isFiltered = true
			} else if strings.Contains(tt.msg, "[skip ci]") || strings.Contains(tt.msg, "[skip action]") {
				isFiltered = true
			} else if findVer.MatchString(tt.msg) {
				isFiltered = true
			}

			if isFiltered != tt.want {
				t.Errorf("Filtering for %q = %v, want %v", tt.msg, isFiltered, tt.want)
			}
		})
	}
}
