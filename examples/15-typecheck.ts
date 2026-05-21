#!/usr/bin/env buns
// buns
// packages = ["zod@^3.0"]

import { z } from "zod";

const UserSchema = z.object({
  name: z.string(),
  age: z.number(),
});

type User = z.infer<typeof UserSchema>;

const user: User = {
  name: "Ada",
  age: 37,
};

// Try changing age to "37" and run with --typecheck to see tsc fail.
console.log("Typed user:", UserSchema.parse(user));
