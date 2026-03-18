# Changelog

## 0.6.0 (2026-03-18)

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

