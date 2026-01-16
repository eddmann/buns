#!/usr/bin/env buns
// buns
// bun = ">=1.0"

// Using Bun's native fetch - no external packages needed
console.log("Fetching random user from API...\n");

const response = await fetch("https://randomuser.me/api/");
const data = await response.json();

const user = data.results[0];
console.log("Name:", user.name.first, user.name.last);
console.log("Email:", user.email);
console.log("Country:", user.location.country);
