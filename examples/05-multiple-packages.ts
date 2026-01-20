#!/usr/bin/env buns
// buns
// packages = ["dayjs@^1.0", "chalk@^5.0"]

import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import chalk from "chalk";

dayjs.extend(relativeTime);

const now = dayjs();
console.log(chalk.blue("Current time:"), now.format("YYYY-MM-DD HH:mm:ss"));
console.log(chalk.blue("Day of week:"), now.format("dddd"));
console.log(chalk.blue("From now:"), now.add(7, "day").fromNow());
