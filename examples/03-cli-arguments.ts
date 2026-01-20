#!/usr/bin/env buns
// Example:
//   buns 03-cli-arguments.ts -- arg1 arg2 --flag

console.log("Arguments received:");
process.argv.slice(2).forEach((arg, i) => {
  console.log(`  [${i}] ${arg}`);
});
console.log(`\nTotal: ${process.argv.length - 2} arguments`);
