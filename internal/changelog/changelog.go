package changelog

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GenerateMarkdown builds a changelog in Markdown from git tags and commits.
//
// Output format:
//
//	# Changelog
//
//	## v1.2.0
//	- abc1234 Commit subject
//	- def5678 Another commit
//
//	## v1.1.0
//	- ...
//
// Tags are discovered using creation date order and rendered newest-first.
// Each tag section includes commits between previousTag..tag.
// The first tag includes all commits reachable from that tag.
func GenerateMarkdown() (string, error) {
	tags, err := getTags()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# Changelog\n\n")

	// No tags: fall back to full history.
	if len(tags) == 0 {
		commits, err := getCommits("")
		if err != nil {
			return "", err
		}
		b.WriteString("## Unreleased\n\n")
		if len(commits) == 0 {
			b.WriteString("- No commits found\n")
		} else {
			for _, c := range commits {
				b.WriteString("- " + c + "\n")
			}
		}
		return b.String(), nil
	}

	// tags are oldest -> newest; render newest -> oldest
	for i := len(tags) - 1; i >= 0; i-- {
		tag := tags[i]

		var revRange string
		if i == 0 {
			// First tag: include all commits up to this tag.
			revRange = tag
		} else {
			prev := tags[i-1]
			revRange = fmt.Sprintf("%s..%s", prev, tag)
		}

		commits, err := getCommits(revRange)
		if err != nil {
			return "", err
		}

		// Filter out commits where the subject exactly matches the tag name.
		var filtered []string
		for _, c := range commits {
			parts := strings.SplitN(c, " ", 2)
			if len(parts) == 2 && parts[1] == tag {
				continue
			}
			filtered = append(filtered, c)
		}

		b.WriteString("## " + tag + "\n\n")
		if len(filtered) == 0 {
			b.WriteString("- No changes\n\n")
			continue
		}

		for _, c := range filtered {
			b.WriteString("- " + c + "\n")
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func getTags() ([]string, error) {
	out, err := runGit("tag", "--sort=creatordate")
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	lines := splitNonEmptyLines(out)
	return lines, nil
}

func getCommits(revRange string) ([]string, error) {
	args := []string{"log", "--pretty=format:%h %s"}
	if strings.TrimSpace(revRange) != "" {
		args = append(args, revRange)
	}

	out, err := runGit(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for range %q: %w", revRange, err)
	}

	return splitNonEmptyLines(out), nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}

	return stdout.String(), nil
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
