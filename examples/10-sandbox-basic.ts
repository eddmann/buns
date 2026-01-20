#!/usr/bin/env buns
// Example:
//   buns 10-sandbox-basic.ts --sandbox --memory 64 --timeout 10 --cpu 5

console.log("Bun Version:", Bun.version);
console.log("Script path:", import.meta.path);
console.log(
  "Memory usage:",
  Math.round((process.memoryUsage().heapUsed / 1024 / 1024) * 100) / 100,
  "MB"
);
