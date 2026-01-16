#!/usr/bin/env buns
// buns
// packages = ["chalk@^5.0"]

import chalk from "chalk";

console.log(chalk.green("Success:"), "buns installed chalk automatically!");
console.log(chalk.blue("Info:"), "No package.json needed");
console.log(chalk.yellow("Tip:"), "Dependencies are cached for reuse");
