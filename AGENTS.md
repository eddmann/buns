# AGENTS.md

## Project Overview

Go CLI tool (`buns`) for running TypeScript/JavaScript scripts with inline npm dependencies declared in comments. Uses Bun runtime with automatic version management.

## Setup

```bash
# Install dependencies
make deps

# Build binary
make build
```

## Common Commands

| Task | Command |
|------|---------|
| Build (dev) | `make build` |
| Build (release) | `make build-release VERSION=x.x.x` |
| Test | `make test` |
| Lint | `make lint` |
| CI gate | `make can-release` |
| Install locally | `make install` |

**Build flags:**
- `build` - Development build with debug symbols
- `build-release` - Optimized build with `-s -w` (stripped), `-trimpath`, `CGO_ENABLED=0`, and version/commit/time injected via ldflags

## Code Conventions

- Go 1.24 required
- Package names: single word, lowercase (`cache`, `metadata`, `exec`)
- Receiver names: single letter matching type (`(c *Cache)`, `(r *Runner)`)
- No `Get` prefix on methods returning values
- Sentinel errors: `Err` prefix (`ErrNoMatchingVersion`)
- Constants: MixedCaps, not SCREAMING_SNAKE
- Imports: stdlib first, blank line, third-party
- Error handling: early return guards, wrap with `fmt.Errorf("context: %w", err)`
- Tests: table-driven with `t.Run()`, use stubs not mocks
- CLI: Cobra framework, commands in `internal/cli/`, flags in `init()`

**Directory structure:**
```
cmd/buns/main.go     # Entry point (thin, delegates to cli.Execute())
internal/
├── bun/             # Bun version resolution and download
├── cache/           # Cache directory management
├── cli/             # Cobra commands (root, run, cache, version)
├── exec/            # Script execution orchestration
├── index/           # Bun version index caching
├── metadata/        # Script TOML metadata parsing
└── npm/             # npm registry lookups
```

## Tests & CI

- Test files co-located: `*_test.go` next to source
- Run all tests: `make test` (wraps `go test ./...`)
- Linting: `make lint` (golangci-lint with 5m timeout)
- CI runs on push/PR to `main` via `.github/workflows/test.yml`
- CI gate: `make can-release` must pass (test + lint)

## PR & Workflow Rules

- Branch: work on feature branches, PR to `main`
- Commit format: Conventional Commits (`feat:`, `fix:`, `chore:`)
- Commit body: bullet points with `-` prefix describing changes
- No PR/issue templates configured
- Manual release: `workflow_dispatch` on `release.yml` with version input

## Security & Gotchas

**Never commit:**
- `bin/` directory (build output)
- `buns` binary
- `coverage.out`, `coverage.html`
- `.env` files (none currently used)

**Common mistakes:**
- Forgetting `make deps` before `make lint` (golangci-lint not installed)
- Tests may skip if network unavailable (registry/index tests)
- Go 1.24 required (check with `go version`)

**Non-obvious rules:**
- `.golangci.yml` has `tests: false` - linter skips test files
- Cache directory: `~/.buns/` (bun binaries, deps, index)
- Script metadata format uses `// buns` TOML comment blocks
