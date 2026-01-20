#!/usr/bin/env buns
// Example:
//   buns 11-sandbox-offline.ts --offline

try {
  const response = await fetch("https://httpbin.org/get", {
    signal: AbortSignal.timeout(5000),
  });
  const data = await response.json();
  console.log(`Network allowed - origin: ${data.origin}`);
} catch (e) {
  console.log("Network blocked (expected with --offline)");
}
