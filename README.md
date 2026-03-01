# Foonver

`foonver` is a lightweight CLI utility for automated Semantic Versioning (SemVer) management. Inspired by the functionality of `npm version`, it simplifies the release workflow by automating version bumping, file updates, and Git lifecycle operations.

## Features

- **Automated Versioning**: Intelligently determines the next version by analyzing Git commit history since the last tag.
- **Multi-format Support**: Discovers and updates version strings in `package.json`, `version.json`, `version.toml`, `version.txt`, and `version.md`.
- **Git Integration**:
  - Enforces a clean working directory before proceeding.
  - Creates a dedicated version-bump commit.
  - Generates a Git tag corresponding to the new version.
- **Format Preservation**: Attempts to maintain the styling and formatting of JSON/TOML files during updates.

## Installation

Ensure you have [Go](https://go.dev/) (1.25.0 or later) installed.

```bash
# Clone the repository
git clone https://github.com/foonly-dev/foonver.git
cd foonver

# Build and install to ~/.local/bin/
make install
```

## Usage

Run `foonver` from the root of a Git repository containing one of the supported version files.

### Manual Bumping

Explicitly specify the bump type or a target version:

```bash
foonver major   # 1.0.0 -> 2.0.0
foonver minor   # 1.0.0 -> 1.1.0
foonver patch   # 1.0.0 -> 1.0.1
foonver 1.2.3   # Set version specifically to 1.2.3
```

### Automatic Bumping (Recommended)

When run without arguments, `foonver` analyzes commit messages since the last tag to decide the increment:

```bash
foonver
```

**Automatic Logic:**

1. **Major**: If any commit message contains "breaking change".
2. **Minor**: If any commit message starts with `feat:`, `feature`, or `new feature`.
3. **Patch**: Default behavior for other changes (e.g., `fix:`, `docs:`, etc.).

## Supported Version Files

The tool scans the project root for files in this order of precedence:

1. `package.json`
2. `version.json`
3. `version.toml`
4. `version.txt`
5. `version.md`

## Development

The project includes a `Makefile` for standard development tasks:

- `make build`: Compiles the binary to `bin/version`.
- `make test`: Runs the test suite.
- `make format`: Formats Go source code.
- `make clean`: Removes build artifacts.

## License

Refer to the project for licensing details.
