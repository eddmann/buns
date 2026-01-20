#!/usr/bin/env buns
// buns
// bun = ">=1.0"

// Examples:
//   echo '{"key": "value"}' | buns 08-json-processing.ts
//   buns 08-json-processing.ts data.json

const input = process.argv[2] ?? "-";

let json: string;
if (input === "-") {
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
