# GitHub Actions Integration

`foonver` is designed to be easily integrated into GitHub Actions to automate your release workflow. It can automatically determine the next version, update files, generate a changelog, and provide the metadata needed to create a GitHub Release.

## Using the Official Action

The easiest way to use `foonver` in your workflow is via the composite action provided in this repository.

### Example Workflow: Automatic Releases

This workflow triggers on every push to the `main` branch. It uses `foonver` to determine if a version bump is needed based on the commit history. If a bump is detected, it creates a GitHub Release with the generated release notes.

```yaml
name: Release

on:
  push:
    branches:
      - main

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write # Required to create tags and releases
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # foonver needs history to determine the next version

      - name: Run foonver
        id: foonver
        uses: foonly-dev/foonver@v0.9.0
        with:
          command: "auto"
          changelog: "true"
          push: "true"

      - name: Create GitHub Release
        if: steps.foonver.outputs.is_bumped == 'true'
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ steps.foonver.outputs.version }}
          name: Release ${{ steps.foonver.outputs.version }}
          body: ${{ steps.foonver.outputs.release_notes }}
          draft: false
          prerelease: false
```

## Inputs

| Input       | Description                                                                | Default  |
| ----------- | -------------------------------------------------------------------------- | -------- |
| `version`   | Version of the `foonver` binary to download.                               | `latest` |
| `command`   | The foonver command to run (`auto`, `patch`, `minor`, `major`).            | `auto`   |
| `changelog` | Whether to update the project's `CHANGELOG.md`.                            | `true`   |
| `push`      | Whether to push the created commit and tag to the remote.                  | `false`  |
| `dry_run`   | If `true`, simulates the bump and provides outputs without changing files. | `false`  |
| `args`      | Additional raw CLI arguments to pass to `foonver`.                         | `""`     |

## Outputs

| Output          | Description                                                              |
| --------------- | ------------------------------------------------------------------------ |
| `version`       | The calculated new version string (e.g., `1.2.3`).                       |
| `is_bumped`     | Returns `true` if a new version was actually created, `false` otherwise. |
| `release_notes` | Markdown formatted release notes for just the latest version.            |

## Advanced CLI Usage in Actions

If you prefer to use the CLI directly (e.g., if you've already installed the binary), you can use the machine-readable flags:

### Getting the New Version

```bash
NEW_VER=$(foonver auto --print-version)
echo "The next version is $NEW_VER"
```

### Exporting Release Notes to a File

```bash
foonver auto --release-notes ./REL_NOTES.md
```

### Parsing the Execution Plan with JQ

```bash
# Get the type of the first planned step
STEP_TYPE=$(foonver auto --json --dry-run | jq -r '.steps[0].type')
```

## Best Practices

1. **Fetch Depth**: Always use `fetch-depth: 0` with `actions/checkout`. `foonver` relies on Git tags and history to calculate the correct version bump. If the history is truncated, it may calculate an incorrect version.
2. **Permissions**: Ensure your workflow has `contents: write` permissions so that `foonver` can push tags and the release step can create the GitHub Release.
3. **CI Config**: You can store your preferred `foonver` settings in a `foonver.toml` in your repository root to keep your workflow files clean.
