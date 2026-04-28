# Changelog

### 0.11.1 (2026-04-28)

#### Continuous Integration

- github: add version_sync to release workflow (d2645f4)

## v0.11.0 (2026-04-28)

#### Features

- cli: add version sync functionality for external files (0e305cd)

#### Continuous Integration

- github: remove version_sync from release workflow (46be0ff)
- github: update pkgname for AUR deployment (772c0b3)

## v0.10.0 (2026-04-28)

#### Features

- version: add support for synchronizing version strings in arbitrary files (1db3c6b)

#### Documentation

- github-actions: prepend v prefix to release tag name (43db53f)

#### Continuous Integration

- aur: automate publishing to AUR on release (116da19)
- release: prefix release tag with v (bb4dd46)

### v0.9.1 (2026-04-27)

### 0.9.1 (2026-04-27)

#### Documentation

- update repository and action paths to foonly organization (ac4196c)

#### Continuous Integration

- github: automate release workflow using foonver (34535cb)

### Misc
- v0.9.1 (8ce59a6)

## v0.9.0 (2026-04-27)

#### Features

- add machine-readable CLI flags and enhance GitHub Action (f8f4442)

### v0.8.2 (2026-04-10)

#### Bug Fixes

- Change some error messages to print to stderr instead of stdout (dde878f)
- changelog: Add blank line after group titles (ec13aaa)

### v0.8.1 (2026-04-08)

#### Bug Fixes

- changelog: Add blank line after group titles (c4f1b1f)
- ver: Remove redundant "(default)" from auto command short description (2e6703a)

## v0.8.0 (2026-04-06)

#### Features

- cmd: Ensure repository exists before execution (dea8423)
- versioning: Add foonver.toml configuration (d9a6428)
- root: Removed "auto" as default. (4fb63e9)
- version: add dry-run flag and execution plan (da333c0)

#### Bug Fixes

- version: Ignore unused return value in test (786c036)

#### Documentation

- Update README with new usage details and configuration (1bacdbc)

#### Maintenance

- Add PLAN.md to .gitignore (ecb736f)

### v0.7.1 (2026-03-19)

#### Bug Fixes

- version: Handle auto-detected version bumps correctly (b1b78b7)

## v0.7.0 (2026-03-19)

#### Features

- version: Default to auto versioning when no version is specified (786f3cd)
- action: Add composite action for foonver (df58257)

#### Bug Fixes

- git: Standardize error message casing (a4ba453)
- version: Log auto-detected version bump (9169e7b)
- version: Allow empty action for auto versioning (ba992a6)

#### Build System

- Update module path and rename command (9660f4c)

## v0.6.0 (2026-03-18)

#### Features

- commands: add config command to display current settings (d0a90d0)
- changelog: include release date for nextVersion when generating changelog group (4d96758)

#### Maintenance

- changelog: adjust Markdown heading levels used in generated changelog (3161f6f)
- plan: remove PLAN.md (6c21bfe)

## v0.5.0 (2026-03-18)

#### Features

- changelog: generate and write changelog and include it in release commits (21196c1)
- changelog: group commits by conventional types and use git package for tag/commit retrieval (79bf73a)

#### Documentation

- readme: add changelog integration and --changelog usage (9347f3d)

## v0.4.0 (2026-03-18)

#### Features

- changelog: add git-based changelog generator and CLI command (52fb03e)

#### Refactor

- commands: run preflight checks in version command and remove startup debug prints (df526d4)

## v0.3.0 (2026-03-18)

#### Features

- version: implement support for different parsers. (b98e0d9)

### v0.2.2 (2026-03-18)

#### Features

- config: add version prefix and move git info to config (03c6d7f)

### 0.2.1 (2026-03-09)

#### Maintenance

- root: remove git info print statement (0792cab)

## 0.2.0 (2026-03-09)

#### Features

- git: add remote detection and tag push support (600331b)
- cli: add configuration system and persistent flags (86ccb66)
- implement cobra commands (07edb48)
- added cobra for command processing WIP (f9f7c77)

#### Bug Fixes

- git: trim whitespace from git root path (980e368)
- version: use git project root for version discovery (7916bc0)

#### Build System

- rename binary to foonver and add README (874610b)

## 0.1.0 (2026-03-01)

#### Features

- Drew the rest of the owl. (0567ada)

