# Implementation Plan: Comprehensive Example Suite for buns

## Summary

Create a progressive example suite for buns following the same pattern as phpx-2. The suite demonstrates all buns features from simple scripts to real-world applications, using numbered files (`01-hello-world.ts` through `09-cli-app.ts`) with a README documenting each example.

## Current State

The `examples/` directory has two basic examples:
- `hello.ts` - Basic chalk usage
- `with-bun-version.ts` - Zod with Bun version constraint

These will be replaced with a structured, progressive suite.

## Design Decisions

### Example Count and Structure

**9 examples** (vs 11 for phpx) because buns doesn't have:
- PHP extension tiers (common/bulk) - Bun handles everything natively
- PHP-specific version features to demonstrate

The progression:
1. **01-03**: No dependencies (basic TypeScript, Bun APIs, CLI args)
2. **04-05**: Package dependencies (single, multiple)
3. **06**: Bun version constraints
4. **07-09**: Real-world scenarios (HTTP, JSON processing, CLI app)

### Package Choices

| Example | Package(s) | Rationale |
|---------|------------|-----------|
| 04 | `chalk@^5.0` | Console colors, very common, ES module |
| 05 | `dayjs@^1.0`, `chalk@^5.0` | Date handling + colors, multiple deps |
| 06 | `zod@^3.0` | Schema validation with Bun constraint |
| 07 | None (uses fetch) | HTTP via Bun's native fetch API |
| 08 | None (uses Bun APIs) | JSON processing via Bun's native APIs |
| 09 | `@clack/prompts@^0.9` | Modern interactive CLI prompts |

### TypeScript vs JavaScript

All examples use `.ts` extension since:
- Bun runs TypeScript natively
- TypeScript is the primary use case for buns
- Matches existing examples in the repo

### Metadata Block Style

Follow existing examples using TOML in comments:
```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["chalk@^5.0"]
```

## Files to Create

### 1. `examples/01-hello-world.ts`
```typescript
#!/usr/bin/env buns
console.log("Hello from buns!");
```
- Simplest possible script, no metadata block needed
- Demonstrates basic execution

### 2. `examples/02-bun-version.ts`
```typescript
#!/usr/bin/env buns
console.log("Bun version:", Bun.version);
console.log("Platform:", process.platform);
console.log("Arch:", process.arch);
```
- Shows runtime info using Bun globals
- No external dependencies

### 3. `examples/03-cli-arguments.ts`
```typescript
#!/usr/bin/env buns
console.log("Arguments received:");
process.argv.slice(2).forEach((arg, i) => {
  console.log(`  [${i}] ${arg}`);
});
console.log(`\nTotal: ${process.argv.length - 2} arguments`);
```
- CLI argument handling via process.argv
- Usage: `buns 03-cli-arguments.ts -- arg1 arg2 --flag`

### 4. `examples/04-single-package.ts`
```typescript
#!/usr/bin/env buns
// buns
// packages = ["chalk@^5.0"]

import chalk from "chalk";

console.log(chalk.green("Success:"), "buns installed chalk automatically!");
console.log(chalk.blue("Info:"), "No package.json needed");
console.log(chalk.yellow("Tip:"), "Dependencies are cached for reuse");
```
- First metadata block usage
- Single popular package

### 5. `examples/05-multiple-packages.ts`
```typescript
#!/usr/bin/env buns
// buns
// packages = ["dayjs@^1.0", "chalk@^5.0"]

import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import chalk from "chalk";

dayjs.extend(relativeTime);

const now = dayjs();
console.log(chalk.blue("Current time:"), now.format("YYYY-MM-DD HH:mm:ss"));
console.log(chalk.blue("Day of week:"), now.format("dddd"));
console.log(chalk.blue("From now:"), now.add(7, "day").fromNow());
```
- Multiple package declaration
- dayjs mirrors Carbon from phpx examples

### 6. `examples/06-bun-constraint.ts`
```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["zod@^3.0"]

import { z } from "zod";

// Using modern Bun APIs that require 1.0+
const UserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
  age: z.number().positive(),
});

const result = UserSchema.safeParse({
  name: "Alice",
  email: "alice@example.com",
  age: 30,
});

if (result.success) {
  console.log("Valid user:", result.data);
} else {
  console.error("Validation failed:", result.error.issues);
}
console.log("Bun version:", Bun.version);
```
- Demonstrates Bun version constraints
- Shows Zod for schema validation

### 7. `examples/07-http-client.ts`
```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"

// Using Bun's native fetch - no external packages needed
console.log("Fetching random user from API...\n");

const response = await fetch("https://randomuser.me/api/");
const data = await response.json();

const user = data.results[0];
console.log("Name:", user.name.first, user.name.last);
console.log("Email:", user.email);
console.log("Country:", user.location.country);
```
- HTTP requests using native fetch
- No external packages - showcases Bun's built-in capabilities
- Same API as phpx-2's example 09

### 8. `examples/08-json-processing.ts`
```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"

// Process JSON from stdin or file using Bun's native APIs
const input = process.argv[2] ?? "-";

let json: string;
if (input === "-") {
  // Read from stdin
  json = await Bun.stdin.text();
} else {
  const file = Bun.file(input);
  if (!(await file.exists())) {
    console.error(`Error: File not found: ${input}`);
    process.exit(1);
  }
  json = await file.text();
}

try {
  const data = JSON.parse(json);
  console.log("Parsed JSON:");
  console.log(JSON.stringify(data, null, 2));
} catch (e) {
  console.error("Error: Invalid JSON:", (e as Error).message);
  process.exit(1);
}
```
- stdin and file input handling
- Bun.file() and Bun.stdin APIs
- Error handling to stderr with exit codes
- Usage: `echo '{"name":"test"}' | buns 08-json-processing.ts -- -`

### 9. `examples/09-cli-app.ts`
```typescript
#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["@clack/prompts@^0.9"]

import * as p from "@clack/prompts";

p.intro("buns CLI Example");

const name = await p.text({
  message: "What is your name?",
  placeholder: "World",
  defaultValue: "World",
});

if (p.isCancel(name)) {
  p.cancel("Cancelled");
  process.exit(0);
}

const shout = await p.confirm({
  message: "Shout the greeting?",
  initialValue: false,
});

if (p.isCancel(shout)) {
  p.cancel("Cancelled");
  process.exit(0);
}

let message = `Hello, ${name}!`;
if (shout) {
  message = message.toUpperCase();
}

p.outro(message);
```
- Full interactive CLI application
- Uses @clack/prompts (modern alternative to inquirer)
- Mirrors phpx-2's example 11 functionality

### 10. `examples/README.md`
Comprehensive documentation following phpx-2 structure:
- Quick start
- Example table with descriptions
- Running examples section
- Script metadata format
- CLI flags reference
- Cache management
- Stdin support

## Files to Delete

- `examples/hello.ts` - Replaced by 01-hello-world.ts and 04-single-package.ts
- `examples/with-bun-version.ts` - Replaced by 06-bun-constraint.ts

## Risks and Considerations

1. **@clack/prompts compatibility**: Modern package, should work with Bun 1.0+. If issues arise, could fall back to `prompts` or `enquirer`.

2. **stdin handling**: Bun's `Bun.stdin.text()` may behave differently in pipe vs TTY contexts. Example 08 handles this.

3. **fetch API availability**: Requires Bun 0.6+ but we constrain to 1.0+ anyway.

4. **dayjs plugins**: Some dayjs plugins may have import issues with ESM. The relativeTime plugin is well-tested.

## Testing the Examples

After implementation, each example should be manually tested:
```bash
cd examples
buns 01-hello-world.ts
buns 02-bun-version.ts
buns 03-cli-arguments.ts -- arg1 arg2 --flag
buns 04-single-package.ts
buns 05-multiple-packages.ts
buns 06-bun-constraint.ts
buns 07-http-client.ts
echo '{"name":"test","count":42}' | buns 08-json-processing.ts -- -
buns 09-cli-app.ts
```
