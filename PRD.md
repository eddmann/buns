# buns MVP - Build Specification

Build a CLI tool called `buns` that runs TypeScript/JavaScript scripts with inline npm dependencies and automatic Bun version management.

---

## Prior Art

buns brings patterns established in other ecosystems to TypeScript/JavaScript:

| Concept | Inspiration | buns Equivalent |
|---------|-------------|-----------------|
| Inline script dependencies | PEP 723, uv inline scripts | `// buns` metadata block |
| Ephemeral tool execution | npx, bunx, uvx | Not needed (use `bunx`) |

**References:**
- [PEP 723](https://peps.python.org/pep-0723/) - Inline script metadata specification
- [uv inline scripts](https://docs.astral.sh/uv/guides/scripts/#declaring-script-dependencies) - uv's PEP 723 implementation
- [bunx](https://bun.sh/docs/cli/bunx) - Bun's built-in package runner (handles tool execution)

---

## Commands

### `buns run <script.ts> [-- args...]`

Run a TypeScript/JavaScript script, installing any declared dependencies first.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bun` | string | "" | Bun version constraint (overrides script) |
| `--packages` | string | "" | Comma-separated packages to add |
| `-v, --verbose` | bool | false | Show detailed output |
| `-q, --quiet` | bool | false | Suppress buns output |

**Stdin support:** `echo 'console.log("hi")' | buns run -`

**Default behavior:** `buns script.ts` is shorthand for `buns run script.ts`

### `buns cache list|clean|dir`

- `list` - show cached Bun builds and dependencies
- `clean` - remove dependency cache (default)
- `clean --bun` - remove Bun builds
- `clean --deps` - remove dependencies
- `clean --all` - remove everything
- `dir` - print cache path

### `buns version`

Print version string.

---

## Script Metadata Format

Scripts declare dependencies in a `// buns` TOML comment block:

```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.1"
// packages = ["zod@^3.0", "chalk@^5.0"]

import { z } from "zod";
import chalk from "chalk";

const schema = z.object({ name: z.string() });
console.log(chalk.green("Hello, buns!"));
```

**Parsing rules:**
1. Find line matching `// buns` (with optional whitespace)
2. Read subsequent lines starting with `//`
3. Strip `// ` prefix from each line
4. Parse accumulated text as TOML
5. Stop at first non-comment line

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `bun` | string | Version constraint (semver) |
| `packages` | string[] | npm packages as `name@constraint` |

---

## Bun Binary Management

### Resolution Order

1. Check cache for matching version
2. Download from official Bun releases
3. **Never use system Bun** - always use managed binaries for consistency

### Available Versions (Dynamic Index)

Fetched from GitHub Releases API and cached locally:

```
GET https://api.github.com/repos/oven-sh/bun/releases
```

**Index cache structure:**
```
~/.buns/index/
├── bun-versions.json    # Parsed available versions
└── fetched_at           # Timestamp file
```

**Cache TTL:** 24 hours (re-fetch if older)

### Download URL

```
https://github.com/oven-sh/bun/releases/download/bun-v{version}/bun-{os}-{arch}.zip
```

- `{os}`: `darwin` or `linux`
- `{arch}`: `x64` or `aarch64`

### Version Resolution

Given constraint like `>=1.1` or `^1.2`:
1. Load cached versions (fetch if stale/missing)
2. Parse using semver library
3. Filter available versions against constraint
4. Return highest matching version
5. If no match → error: `Error: no Bun version satisfies '{constraint}'`

### Platform Detection

- OS: `runtime.GOOS` → "darwin" or "linux"
- Arch: `runtime.GOARCH` → "x64" (amd64) or "aarch64" (arm64)

---

## npm Registry API

### Endpoint

```
GET https://registry.npmjs.org/{package}
```

### Response (abbreviated)

```json
{
  "name": "zod",
  "dist-tags": {
    "latest": "3.24.1"
  },
  "versions": {
    "3.24.1": {
      "name": "zod",
      "version": "3.24.1",
      "dist": {
        "tarball": "https://registry.npmjs.org/zod/-/zod-3.24.1.tgz"
      }
    }
  }
}
```

### Version Resolution

1. If no version specified: use `dist-tags.latest`
2. Parse constraint, filter versions, return highest match

---

## Dependency Management

### Installation

Generate `package.json`:
```json
{
  "dependencies": {
    "zod": "^3.0",
    "chalk": "^5.0"
  }
}
```

Run:
```bash
bun install --frozen-lockfile=false
```

### Caching

**Cache key:** SHA-256 of sorted, lowercase package list

**Structure:**
```
~/.buns/deps/{hash}/
├── package.json
├── bun.lockb
└── node_modules/
```

**Cache hit:** Check if `node_modules/.bin` exists or `node_modules` is non-empty

---

## Execution Flow

### Script (`buns run`)

```
1. If "-", read stdin to temp file
2. Parse script for // buns metadata
3. Merge --packages with metadata packages
4. Resolve Bun version (--bun > metadata > default)
5. Get Bun binary (cache > download)
6. If packages exist:
   a. Hash package list
   b. Check ~/.buns/deps/{hash}/
   c. If miss: bun install
7. Execute:
   NODE_PATH={deps_dir}/node_modules bun run script.ts args...
8. Return script's exit code
```

---

## Cache Structure

```
~/.buns/
├── bun/{version}/bun                # Bun binaries
├── deps/{hash}/                     # Script dependencies
│   ├── package.json
│   ├── bun.lockb
│   └── node_modules/
└── index/                           # Cached version info
    ├── bun-versions.json
    └── fetched_at
```

---

## Example Usage

```bash
# Run script with inline deps
$ cat script.ts
#!/usr/bin/env buns
// buns
// packages = ["chalk@^5.0"]

import chalk from "chalk";
console.log(chalk.green("Hello!"));

$ buns script.ts
Hello!

# Stdin
$ echo 'console.log(Bun.version)' | buns run -
1.2.3

# Override Bun version
$ buns run --bun="^1.1" script.ts

# Add runtime package
$ buns run --packages=lodash@^4.0 script.ts
```

---

## Tech Stack

- Language: Go 1.22+
- CLI: github.com/spf13/cobra
- Semver: github.com/Masterminds/semver/v3
- TOML: github.com/BurntSushi/toml
- Progress: github.com/schollz/progressbar/v3

---

## Error Cases

```
Error: script not found: {path}
Error: package not found: {name}
Error: no Bun version satisfies '{constraint}'
Error: failed to download Bun: {details}
Error: failed to install dependencies: {details}
```

Exit codes: 0 = success, 1 = buns error, N = script exit code (passthrough)

---

## Design Decisions

1. **Implementation language:** Go (single static binary, easy cross-compilation)
2. **No system Bun:** Always use managed binaries for consistency
3. **No tool command:** `bunx` already handles ephemeral tool execution
4. **Shebang support:** `#!/usr/bin/env buns` works naturally
5. **TypeScript config:** Bun handles TS natively, no tsconfig needed for scripts
