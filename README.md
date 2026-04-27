# Foonver

`foonver` is a lightweight CLI utility for automated Semantic Versioning (SemVer) management. Inspired by the functionality of `npm version`, it simplifies the release workflow by automating version bumping, file updates, and Git lifecycle operations.

## Features

- **Plan-First Workflow**: Preview exactly what will happen before any files are changed.
- **Automated Versioning**: Intelligently determines the next version by analyzing Git commit history since the last tag.
- **Multi-format Support**: Discovers and updates version strings in `package.json`, `version.json`, `version.toml`, `version.txt`, and `version.md`.
- **Git Integration**:
  - Validates repository state before mutating.
  - Creates dedicated version-bump commits and tags.
  - Optional automatic pushing of tags and commits.
- **Changelog Management**: Categorizes commits and automatically updates `CHANGELOG.md`.

## Installation

Ensure you have [Go](https://go.dev/) (1.25.0 or later) installed.

```bash
# Clone the repository
git clone https://github.com/foonly/foonver.git
cd foonver

# Build and install
make install
```

## Usage

Foonver requires a subcommand to perform a version bump.

### Automatic Bumping (Recommended)

The `auto` command analyzes all commit messages since the last version tag to calculate the next bump:

```bash
foonver auto
```

**Automatic Logic:**

1. **Major**: Triggered if any commit contains `BREAKING CHANGE:` or uses the `!` suffix (e.g., `feat!:`).
2. **Minor**: Triggered if any commit starts with `feat:`.
3. **Patch**: Default behavior for other changes (e.g., `fix:`, `docs:`, `chore:`).

### Manual Bumping

Explicitly specify the bump type or a target version:

```bash
foonver major   # 1.0.0 -> 2.0.0
foonver minor   # 1.0.0 -> 1.1.0
foonver patch   # 1.0.0 -> 1.0.1
foonver ver 1.2.3  # Set version specifically to 1.2.3
```

### Dry Run

Use the `--dry-run` flag to see the calculated version and the list of commits being considered without making any changes to your files or Git state. This works even on dirty repositories.

```bash
foonver auto --dry-run
```

## Configuration

Foonver looks for a `foonver.toml`, `foonver.yaml`, or `foonver.json` file in the following locations (in order):

1. The project root directory.
2. The XDG configuration home (usually `~/.config/foonver/`).
3. `/etc/foonver/`.

### Example Configuration (`foonver.toml`)

```toml
# Automatically push commits and tags to remote
push = false

# Prefix for git tags (e.g., v1.0.0)
prefix = "v"

# Output verbosity: quiet, normal, verbose, debug
verbosity = "normal"

# Commit parser: angular, generic, or all
parser = "all"

# Automatically update the changelog file
changelog = true

# The name of the changelog file
file = "CHANGELOG.md"
```

## Supported Version Files

The tool scans the project root for files in this order:

1. `package.json`
2. `version.json`
3. `version.toml`
4. `version.txt`
5. `version.md`

## Development

The project includes a `Makefile` for standard development tasks:

- `make build`: Compiles the binary to `bin/foonver`.
- `make test`: Runs the test suite.
- `make format`: Formats Go source code.
- `make clean`: Removes build artifacts.

## License

Refer to the project for licensing details.
