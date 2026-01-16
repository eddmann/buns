#!/usr/bin/env buns
console.log("Arguments received:");
process.argv.slice(2).forEach((arg, i) => {
  console.log(`  [${i}] ${arg}`);
});
console.log(`\nTotal: ${process.argv.length - 2} arguments`);
