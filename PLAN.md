# Foonver - Version Management Utility

## Overview

`foonver` is a standalone binary utility designed to replicate and extend the functionality of the `npm version` command. It automates the process of bumping project versions based on Semantic Versioning (SemVer), updating local version tracking files, and creating the corresponding Git commits and tags.

## Core Workflows

### 1. Pre-flight Checks

Before performing any operations, the utility must ensure the environment is safe:

- **Git Repository Check**: Verify the current directory is inside a Git repository.
- **Clean Working Directory**: Run `git status --porcelain` to ensure there are no uncommitted changes. If the working tree is dirty, exit immediately with a descriptive error message (e.g., "Git working directory not clean. Commit or stash changes first.").

### 2. File Discovery & Version Extraction

The tool needs to find the current version. It will scan the project root for the following files (in order of precedence or failing if multiple conflicting files exist):

1.  `package.json` (Parse JSON, extract `"version"`)
2.  `version.json` (Parse JSON, extract `"version"`)
3.  `version.toml` (Parse TOML, extract `version` key)
4.  `version.txt` (Read raw text, strip whitespace)
5.  `version.md` (Extract version string, typically from a standard header or badge)

_Note: The extracted version must be validated as a compliant SemVer string (e.g., `1.2.3` or `v1.2.3`)._

### 3. Determining the Next Version

The utility accepts an optional argument to determine how the version should be bumped:

- **Explicit Target (Argument provided)**:
  - `major`: Increment the major version (e.g., `1.2.3` -> `2.0.0`).
  - `minor`: Increment the minor version (e.g., `1.2.3` -> `1.3.0`).
  - `patch`: Increment the patch version (e.g., `1.2.3` -> `1.2.4`).
  - `<specific-version>` (e.g., `2.1.0`): Validate as SemVer and use exactly this version.

- **Automatic Target (No argument provided)**:
  - Fetch the latest Git tag (`git describe --tags --abbrev=0`).
  - If no previous tag exists, read the entire commit history.
  - Get the commit log from the previous tag to `HEAD` (`git log <last-tag>..HEAD --oneline`).
  - **Rule Evaluation (Highest precedence wins)**:
    1.  **Major**: If any commit message starts with (or contains) `"breaking change"`, increment the MAJOR version.
    2.  **Minor**: If any commit message starts with `"feat:"`, `"feature"`, or `"new feature"`, increment the MINOR version.
    3.  **Patch**: If neither of the above conditions are met, increment the PATCH version.

### 4. File Update

Once the new version is calculated:

- Open the discovered version file.
- Safely replace the old version string with the new version string.
  - For JSON/TOML: Use structured reading/writing to avoid wiping out formatting, or use a targeted Regex replacement if the file is simple.
  - For TXT/MD: Replace the matching SemVer string.

### 5. Git Operations

After successfully updating the file, automate the source control steps:

1.  **Add**: `git add <updated-version-file>`
2.  **Commit**: `git commit -m "<new-version>"` (e.g., `v1.3.0`)
3.  **Tag**: `git tag <new-version>` (Optionally prefix with `v` if the project convention uses it, such as `v1.3.0`).

## Edge Cases & Error Handling

- **Missing Version File**: If none of the supported files are found, exit with an error instructing the user to create one.
- **Invalid SemVer**: If the current file contains a malformed version, exit with a parse error.
- **Git Failure**: If `git add`, `commit`, or `tag` fails, attempt to rollback the file changes or provide clear instructions on how to recover.
- **Empty Commit History**: If running in automatic mode and there are no new commits since the last tag, exit and notify the user that there is nothing to bump.

## Implementation Details (Go)

- **CLI Framework**: Use the standard `flag` library or `spf13/cobra` for parsing the optional arguments and providing a `--help` menu.
- **SemVer Parsing**: Utilize an established library like `golang.org/x/mod/semver` or `github.com/Masterminds/semver/v3` to safely parse, increment, and format versions.
- **Subprocess Execution**: Use `os/exec` for interacting with Git (`git status`, `git log`, etc.).

## Future Enhancements (Optional)

- **Pre/Post Hooks**: Support running shell commands before or after the version bump (e.g., `npm run build` or `make test`).
- **Custom Commit Messages**: Allow users to pass a flag like `-m "Release %s"` to customize the Git commit message.
- **Dry Run**: Add a `--dry-run` flag to simulate the version bump and Git operations without actually modifying files or history.
