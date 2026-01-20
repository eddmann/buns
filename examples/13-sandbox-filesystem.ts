#!/usr/bin/env buns
// Example:
//   echo "hello" > /tmp/buns-test.txt
//   buns 13-sandbox-filesystem.ts --sandbox --allow-read /tmp --allow-write /tmp

const inputFile = "/tmp/buns-test.txt";
const outputFile = "/tmp/buns-output.txt";

// Read
try {
  const content = await Bun.file(inputFile).text();
  console.log("Read:", content.trim());
} catch (e) {
  console.log(`Read blocked (create input: echo "hello" > ${inputFile})`);
}

// Write
try {
  await Bun.write(outputFile, `written at ${Date.now()}\n`);
  console.log("Write: OK");
} catch (e) {
  console.log("Write blocked");
}
