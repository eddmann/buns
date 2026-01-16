package metadata

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Metadata
		wantErr bool
	}{
		{
			name: "full metadata block",
			content: `#!/usr/bin/env buns
// buns
// bun = ">=1.1"
// packages = ["zod@^3.0", "chalk@^5.0"]

import { z } from "zod";
`,
			want: &Metadata{
				Bun:      ">=1.1",
				Packages: []string{"zod@^3.0", "chalk@^5.0"},
			},
		},
		{
			name: "packages only",
			content: `// buns
// packages = ["lodash@^4.0"]

console.log("hi");
`,
			want: &Metadata{
				Packages: []string{"lodash@^4.0"},
			},
		},
		{
			name: "bun version only",
			content: `// buns
// bun = "^1.2"

console.log("hi");
`,
			want: &Metadata{
				Bun: "^1.2",
			},
		},
		{
			name:    "no metadata block",
			content: `console.log("no deps");`,
			want:    &Metadata{},
		},
		{
			name: "empty metadata block",
			content: `// buns

console.log("empty block");
`,
			want: &Metadata{},
		},
		{
			name: "metadata with shebang",
			content: `#!/usr/bin/env buns
// buns
// packages = ["express@^4.0"]

import express from "express";
`,
			want: &Metadata{
				Packages: []string{"express@^4.0"},
			},
		},
		{
			name: "multiple packages",
			content: `// buns
// bun = ">=1.0"
// packages = [
//   "zod@^3.0",
//   "chalk@^5.0",
//   "lodash@^4.0"
// ]

import stuff from "stuff";
`,
			want: &Metadata{
				Bun:      ">=1.0",
				Packages: []string{"zod@^3.0", "chalk@^5.0", "lodash@^4.0"},
			},
		},
		{
			name: "stops at non-comment line",
			content: `// buns
// packages = ["a@1.0"]
const x = 1;
// packages = ["b@2.0"]
`,
			want: &Metadata{
				Packages: []string{"a@1.0"},
			},
		},
		{
			name: "whitespace variations",
			content: `  // buns
  // bun = ">=1.1"
  // packages = ["test@^1.0"]

code here
`,
			want: &Metadata{
				Bun:      ">=1.1",
				Packages: []string{"test@^1.0"},
			},
		},
		{
			name: "invalid TOML",
			content: `// buns
// this is not valid = [toml
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
