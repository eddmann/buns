# buns Examples

Progressive examples demonstrating all buns features.

## Quick Start

```bash
cd examples
buns 01-hello-world.ts
```

## Examples

| # | File | Description |
|---|------|-------------|
| 01 | `01-hello-world.ts` | Simplest script - no dependencies |
| 02 | `02-bun-version.ts` | Display Bun/platform info |
| 03 | `03-cli-arguments.ts` | Pass arguments to scripts |
| 04 | `04-single-package.ts` | Single dependency (chalk) |
| 05 | `05-multiple-packages.ts` | Multiple dependencies (dayjs + chalk) |
| 06 | `06-bun-constraint.ts` | Require Bun >=1.0 with zod |
| 07 | `07-http-client.ts` | HTTP requests with native fetch |
| 08 | `08-json-processing.ts` | Process JSON from stdin/file |
| 09 | `09-cli-app.ts` | Full CLI app with @clack/prompts |

## Running Examples

```bash
# Basic scripts (no dependencies)
buns 01-hello-world.ts
buns 02-bun-version.ts
buns 03-cli-arguments.ts -- arg1 arg2 --flag

# Scripts with dependencies
buns 04-single-package.ts
buns 05-multiple-packages.ts

# Bun version constraints
buns 06-bun-constraint.ts

# Real-world examples
buns 07-http-client.ts
echo '{"name":"test","count":42}' | buns 08-json-processing.ts -- -
buns 09-cli-app.ts
```

## Script Metadata

Declare dependencies in a `// buns` comment block:

```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["zod@^3.0", "chalk@^5.0"]

import { z } from "zod";
import chalk from "chalk";

// Your code here...
```

| Field | Type | Description |
|-------|------|-------------|
| `bun` | string | Bun version constraint (semver) |
| `packages` | string[] | npm packages as `name@constraint` |

## CLI Flags

### For `buns run`

| Flag | Description |
|------|-------------|
| `--bun` | Override Bun version constraint |
| `--packages` | Add packages (comma-separated) |
| `-v, --verbose` | Show detailed output |
| `-q, --quiet` | Suppress buns output |

### Examples

```bash
# Override Bun version
buns --bun="^1.1" script.ts

# Add runtime packages
buns --packages=lodash@^4.0 script.ts

# Verbose mode (see what buns is doing)
buns -v script.ts

# Quiet mode (only script output)
buns -q script.ts
```

## Cache Management

```bash
# Show cached items
buns cache list

# Print cache directory
buns cache dir

# Remove dependency cache (default)
buns cache clean

# Remove specific caches
buns cache clean --bun      # Remove Bun binaries
buns cache clean --deps     # Remove dependencies
buns cache clean --index    # Remove version index
buns cache clean --all      # Remove everything
```

## Stdin Support

Read TypeScript from stdin:

```bash
echo 'console.log(Bun.version)' | buns run -
```
