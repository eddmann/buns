#!/usr/bin/env buns
// Example:
//   API_KEY=secret123 DEBUG=1 buns 14-sandbox-env.ts --sandbox --allow-env API_KEY,DEBUG

const vars = ["API_KEY", "DEBUG", "HOME", "PATH"];

for (const name of vars) {
  const value = process.env[name];
  const status = value !== undefined ? "set" : "not set";
  console.log(`${name}: ${status}`);
}
