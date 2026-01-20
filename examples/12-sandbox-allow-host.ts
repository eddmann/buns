#!/usr/bin/env buns
// Example:
//   buns 12-sandbox-allow-host.ts --allow-host httpbin.org

const urls = [
  "https://httpbin.org/get",
  "https://api.github.com/zen",
];

for (const url of urls) {
  const host = new URL(url).hostname;
  try {
    const response = await fetch(url, { signal: AbortSignal.timeout(5000) });
    if (response.status === 403) {
      console.log(`[blocked] ${host}`);
    } else {
      console.log(`[allowed] ${host}`);
    }
  } catch (e) {
    console.log(`[blocked] ${host}`);
  }
}
