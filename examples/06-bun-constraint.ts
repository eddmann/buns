#!/usr/bin/env buns
// buns
// bun = ">=1.0"
// packages = ["zod@^3.0"]

import { z } from "zod";

const UserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
  age: z.number().positive(),
});

const result = UserSchema.safeParse({
  name: "Alice",
  email: "alice@example.com",
  age: 30,
});

if (result.success) {
  console.log("Valid user:", result.data);
} else {
  console.error("Validation failed:", result.error.issues);
}
console.log("Bun version:", Bun.version);
