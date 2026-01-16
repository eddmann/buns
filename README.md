# buns

Run TypeScript scripts with inline dependencies. No package.json needed.

Inspired by [PEP 723](https://peps.python.org/pep-0723/) and [uv's inline scripts](https://docs.astral.sh/uv/guides/scripts/#declaring-script-dependencies), buns brings the same workflow to TypeScript/JavaScript with automatic Bun version management.

## Features

- **Inline dependencies** - Declare npm packages in script comments, installed automatically
- **Bun version management** - Pin specific Bun versions per script, downloaded on demand
- **Dependency caching** - Same packages across scripts share a single cache
- **Zero config** - No package.json, no tsconfig, just run

## Installation

### From Source

```bash
git clone https://github.com/edwardsmale/buns
cd buns
make build
make install  # Installs to ~/.local/bin
```

## Quick Start

1. Create a script with inline dependencies:

```typescript
#!/usr/bin/env buns
// buns
// packages = ["chalk@^5.0"]

import chalk from "chalk";

console.log(chalk.green("Hello from buns!"));
```

2. Run it:

```bash
buns script.ts
```

That's it. Dependencies are installed automatically on first run and cached for reuse.

## Script Metadata

Declare dependencies in a `// buns` TOML comment block:

```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["zod@^3.0", "chalk@^5.0"]

import { z } from "zod";
import chalk from "chalk";
```

| Field | Type | Description |
|-------|------|-------------|
| `bun` | string | Bun version constraint (semver) |
| `packages` | string[] | npm packages as `name@constraint` |

## Command Reference

### `buns run <script.ts>`

Run a TypeScript/JavaScript script with inline dependencies.

```bash
buns script.ts                      # Shorthand for buns run
buns run script.ts -- --flag value  # Pass args to script
echo 'console.log("hi")' | buns -   # Read from stdin
```

| Flag | Description |
|------|-------------|
| `--bun <constraint>` | Override Bun version |
| `--packages <pkgs>` | Add comma-separated packages |
| `-v, --verbose` | Show detailed output |
| `-q, --quiet` | Suppress buns output |

### `buns cache`

Manage cached Bun binaries and dependencies.

```bash
buns cache list   # Show cached items
buns cache dir    # Print cache path
buns cache clean  # Remove dependency cache
```

| Flag | Description |
|------|-------------|
| `--bun` | Remove Bun binaries |
| `--deps` | Remove dependencies (default) |
| `--index` | Remove version index |
| `--all` | Remove everything |

## How It Works

```
Script → Parse metadata → Resolve Bun version → Install deps → Execute
                              ↓                      ↓
                         ~/.buns/bun/          ~/.buns/deps/
```

1. **Parse** - Extract `// buns` TOML block from script
2. **Resolve Bun** - Match version constraint against GitHub releases, download if needed
3. **Install deps** - Hash package list, check cache, run `bun install` if miss
4. **Execute** - Run script with `NODE_PATH` pointing to cached node_modules

## Cache Structure

```
~/.buns/
├── bun/{version}/bun     # Bun binaries
├── deps/{hash}/          # Dependencies (package.json, node_modules)
└── index/                # Cached Bun version list (24h TTL)
```

## Examples

See the [examples/](examples/) directory for progressive examples from hello world to full CLI apps.

## Development

```bash
git clone https://github.com/edwardsmale/buns
cd buns
make build       # Build binary
make test        # Run tests
make lint        # Run linters
make can-release # CI gate (test + lint)
```

## License

MIT
