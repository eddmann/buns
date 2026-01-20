#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["@clack/prompts@^0.9"]

// Example:
//   buns 09-cli-app.ts

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
