package main

import (
	"fmt"
	"os"
	"strings"

	"foonly.dev/foonver/internal/git"
	"foonly.dev/foonver/internal/version"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("foonver - Version Management Utility")
		fmt.Printf("Version: %s\n\n", Version)
		fmt.Println("Usage: foonver [major|minor|patch|<specific-version>]")
		os.Exit(0)
	}

	if err := git.RunPreflightChecks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fileName, currentVersion, fileContent, err := version.DiscoverVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found version %s in %s\n", currentVersion.Original(), fileName)

	target := ""
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	nextVersion, err := version.DetermineNextVersion(currentVersion, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining next version: %v\n", err)
		os.Exit(1)
	}

	nextVersionStr := nextVersion.String()
	if strings.HasPrefix(currentVersion.Original(), "v") {
		nextVersionStr = "v" + nextVersionStr
	}

	if currentVersion.String() == nextVersion.String() {
		fmt.Println("Version is already up to date.")
		os.Exit(0)
	}

	fmt.Printf("Bumping version from %s to %s\n", currentVersion.Original(), nextVersionStr)

	if err := version.UpdateVersionFile(fileName, currentVersion.Original(), nextVersionStr, fileContent); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating version file: %v\n", err)
		os.Exit(1)
	}

	if err := git.CommitAndTag(fileName, nextVersionStr); err != nil {
		fmt.Fprintf(os.Stderr, "Git error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully bumped version to %s\n", nextVersionStr)
}
