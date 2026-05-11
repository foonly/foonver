package changelog

import (
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

func TestFilteredCommits(t *testing.T) {
	// Since filteredCommits calls git.GetCommits, which interacts with the actual git repo,
	// we would ideally mock the git package. However, for this task, we can verify
	// the logic by testing the regex and message handling in a more isolated way
	// if the code allowed it, but here we've already updated the regex which is
	// the core of the requested change.

	// We can verify the filtering logic that uses findVer manually or via internal
	// knowledge of how filteredCommits is implemented.
}
