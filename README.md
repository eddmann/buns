# buns

![buns](docs/heading.png)

Run TypeScript scripts with inline dependencies. No package.json needed.

Inspired by [PEP 723](https://peps.python.org/pep-0723/) and [uv's inline scripts](https://docs.astral.sh/uv/guides/scripts/#declaring-script-dependencies), buns brings the same workflow to TypeScript/JavaScript with automatic Bun version management.

## Features

- **Inline dependencies** - Declare npm packages in script comments, installed automatically
- **Bun version management** - Pin specific Bun versions per script, downloaded on demand
- **Dependency caching** - Same packages across scripts share a single cache
- **Sandboxing & isolation** - Run scripts with controlled filesystem, network, and resource limits
- **Zero config** - No package.json, no tsconfig, just run

## Installation

### Homebrew (Recommended)

```bash
brew install eddmann/tap/buns
```

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/eddmann/buns/main/install.sh | sh
```

### Download Binary

```bash
# macOS (Apple Silicon)
curl -L https://github.com/eddmann/buns/releases/latest/download/buns-macos-arm64 -o buns
chmod +x buns && sudo mv buns /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/eddmann/buns/releases/latest/download/buns-macos-x64 -o buns
chmod +x buns && sudo mv buns /usr/local/bin/

# Linux (x64)
curl -L https://github.com/eddmann/buns/releases/latest/download/buns-linux-x64 -o buns
chmod +x buns && sudo mv buns /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/eddmann/buns
cd buns
make build-release VERSION=0.1.0
make install  # Installs to ~/.local/bin
```

## Quick Start

**1. Run a simple script**

```bash
buns script.ts
```

**2. Add inline dependencies**

```typescript
#!/usr/bin/env buns
// buns
// packages = ["chalk@^5.0"]

import chalk from "chalk";

console.log(chalk.green("Hello from buns!"));
```

**3. Run it**

```bash
buns script.ts
# Hello from buns!
```

**4. Pin a Bun version**

```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["zod@^3.0"]

import { z } from "zod";
console.log("Bun version:", Bun.version);
```

**5. Pipe from stdin**

```bash
echo 'console.log(Bun.version)' | buns run -
```

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

| Field      | Type     | Description                       |
| ---------- | -------- | --------------------------------- |
| `bun`      | string   | Bun version constraint (semver)   |
| `packages` | string[] | npm packages as `name@constraint` |

## Command Reference

### buns run

Run a TypeScript/JavaScript script with inline dependencies.

```bash
buns run <script.ts> [-- args...]
buns <script.ts> [-- args...]  # Shorthand
```

| Flag           | Short | Description                               |
| -------------- | ----- | ----------------------------------------- |
| `--bun`        |       | Bun version constraint (overrides script) |
| `--packages`   |       | Comma-separated packages to add           |
| `--verbose`    | `-v`  | Show detailed output                      |
| `--quiet`      | `-q`  | Suppress buns output                      |
| `--sandbox`    |       | Enable sandboxing (restricts filesystem)  |
| `--offline`    |       | Block all network access                  |
| `--allow-host` |       | Allow network to specific hosts           |
| `--allow-read` |       | Additional readable paths                 |
| `--allow-write`|       | Additional writable paths                 |
| `--allow-env`  |       | Environment variables to pass             |
| `--memory`     |       | Memory limit in MB (default: 128)                   |
| `--timeout`    |       | Execution timeout in seconds (default: 30)          |
| `--cpu`        |       | CPU time limit in seconds, Linux only (default: 30) |

### buns cache

Manage the buns cache.

```bash
buns cache list              # Show cached items
buns cache clean             # Remove dependency cache (default)
buns cache clean --bun       # Remove Bun binaries
buns cache clean --deps      # Remove dependencies
buns cache clean --index     # Remove version index
buns cache clean --all       # Remove everything
buns cache dir               # Print cache path
```

### buns version

Print version information.

## Examples

The `examples/` directory contains 14 progressive examples demonstrating all buns features - from basic script execution through inline dependencies, Bun version constraints, CLI apps, and sandboxing.

```bash
# Simple dependency usage
buns examples/04-single-package.ts

# Full CLI app with @clack/prompts
buns examples/09-cli-app.ts

# Sandboxed execution with resource limits
buns examples/10-sandbox-basic.ts --sandbox --memory 64 --timeout 10

# Network filtering (allow specific hosts)
buns examples/12-sandbox-allow-host.ts --allow-host httpbin.org
```

See [examples/README.md](examples/README.md) for the complete list and detailed usage instructions.

## How It Works

```
Script → Parse metadata → Resolve Bun version → Download Bun (if needed)
                                              → Install dependencies (if needed)
                                              → Execute script
```

**Bun binaries** are downloaded from [oven-sh/bun releases](https://github.com/oven-sh/bun/releases) - official pre-built binaries for all platforms.

**Dependencies** are installed via Bun into content-addressed cache directories at `~/.buns/deps/{hash}/`.

## Cache Structure

```
~/.buns/
├── bun/{version}/bun     # Bun binaries
├── deps/{hash}/          # Script dependencies (node_modules)
└── index/                # Version index (24h TTL)
```

## Security & Sandboxing

buns supports sandboxed execution to restrict script capabilities.

### Sandbox Modes

**Filesystem sandbox** (`--sandbox`): Restricts filesystem access to script directory, dependencies, and explicit paths.

**Network sandbox** (`--offline`): Blocks all network access.

**Host filtering** (`--allow-host`): Allows network only to specified hosts.

### Resource Limits

```bash
buns script.ts --sandbox --memory 64 --timeout 10 --cpu 5
```

| Flag        | Default | Description                          |
| ----------- | ------- | ------------------------------------ |
| `--memory`  | 128     | Memory limit in MB                   |
| `--timeout` | 30      | Wall-clock timeout (seconds)         |
| `--cpu`     | 30      | CPU time limit (seconds, Linux only) |

Resource enforcement depends on available tooling:
- **nsjail** (Linux): Hard memory and CPU limits enforced via cgroups
- **bubblewrap** (Linux): Timeout and basic isolation
- **macOS/fallback**: `--memory` sets `BUN_JSC_forceRAMSize` as a GC hint; `--cpu` has no effect

### Filesystem Access

```bash
# Allow read from /data, write to /tmp
buns script.ts --sandbox --allow-read /data --allow-write /tmp
```

By default, sandboxed scripts can only read their script file and dependencies. Use `--allow-read` and `--allow-write` to grant access to additional paths.

### Environment Variables

```bash
# Only pass specific env vars
API_KEY=secret buns script.ts --sandbox --allow-env API_KEY,DEBUG
```

In sandbox mode, environment variables are filtered. Use `--allow-env` to pass specific variables to the script.

### Platform Support

- **macOS**: Uses `sandbox-exec` with custom profiles
- **Linux**: Uses `bubblewrap` or `nsjail` for full sandbox, `unshare` for network-only (auto-detected)

## Development

```bash
git clone https://github.com/eddmann/buns
cd buns
make test                           # Run tests
make lint                           # Run linters
make build                          # Build binary (dev, with debug symbols)
make build-release VERSION=x.x.x    # Build binary (release, optimized)
make install                        # Install to ~/.local/bin
```

## Credits

- [Bun](https://bun.sh/) - Fast JavaScript runtime
- [oven-sh/bun releases](https://github.com/oven-sh/bun/releases) - Bun binary distribution

## License

MIT License - see [LICENSE](LICENSE) for details.
