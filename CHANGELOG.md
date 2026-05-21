# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-05-21

### Added

- Added `--typecheck` for `buns run` and the `buns <script.ts>` shorthand to run TypeScript checking before execution
- Added isolated typecheck dependency caches under `~/.buns/typecheck`
- Added `buns cache list` visibility and `buns cache clean --typecheck` support for typecheck caches
- Added a typechecking example that validates external package types

### Fixed

- Fixed macOS sandbox execution with Bun 1.3.14+ by allowing Bun to read system ICU data required at startup

## [0.0.6] - 2026-02-06

### Added

- Linux ARM64 (linux-arm64) build support for release binaries
- Interactive sandbox section on the landing page

## [0.0.5] - 2026-01-20

### Changed

- Replaced `version` subcommand with `--version` flag for standard CLI conventions
- Improved `run` command help text to show metadata syntax examples

## [0.0.4] - 2026-01-20

### Added

- Changelog generation with release information derived from git history

## [0.0.3] - 2026-01-20

### Added

- Sandboxing and network isolation for script execution
- ASCII logo displayed on the help screen
- Configurable binary path option in test suite

## [0.0.2] - 2026-01-16

### Added

- Optimized release build with stripped binaries and version info via ldflags

## [0.0.1] - 2026-01-16

### Added

- Initial implementation of buns CLI for running TypeScript/JavaScript scripts with inline npm dependencies
- Automatic Bun runtime version management and downloading
- Script metadata parsing via `// buns` TOML comment blocks
- Cache management for Bun binaries and dependencies
- CI/CD workflows and installation scripts
- GitHub Pages documentation site

[1.0.0]: https://github.com/eddmann/buns/compare/v0.0.6...v1.0.0
[0.0.6]: https://github.com/eddmann/buns/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/eddmann/buns/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/eddmann/buns/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/eddmann/buns/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/eddmann/buns/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/eddmann/buns/releases/tag/v0.0.1
