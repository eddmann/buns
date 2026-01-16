package npm

import (
	"testing"
)

func TestParsePackageSpec(t *testing.T) {
	tests := []struct {
		spec           string
		wantName       string
		wantConstraint string
	}{
		{"zod", "zod", ""},
		{"zod@^3.0", "zod", "^3.0"},
		{"zod@3.24.1", "zod", "3.24.1"},
		{"@types/node", "@types/node", ""},
		{"@types/node@^20.0", "@types/node", "^20.0"},
		{"@org/package@1.0.0", "@org/package", "1.0.0"},
		{"lodash@>=4.0.0", "lodash", ">=4.0.0"},
		{"express@~4.18.0", "express", "~4.18.0"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			name, constraint := parsePackageSpec(tt.spec)
			if name != tt.wantName {
				t.Errorf("parsePackageSpec(%s) name = %s, want %s", tt.spec, name, tt.wantName)
			}
			if constraint != tt.wantConstraint {
				t.Errorf("parsePackageSpec(%s) constraint = %s, want %s", tt.spec, constraint, tt.wantConstraint)
			}
		})
	}
}

func TestRegistry_ResolveVersion(t *testing.T) {
	// Integration test - requires network
	r := NewRegistry()

	t.Run("resolve latest zod", func(t *testing.T) {
		name, version, err := r.ResolveVersion("zod")
		if err != nil {
			t.Skip("Network unavailable")
		}
		if name != "zod" {
			t.Errorf("expected name 'zod', got %s", name)
		}
		if version == "" {
			t.Error("expected a version")
		}
	})

	t.Run("resolve zod with constraint", func(t *testing.T) {
		name, version, err := r.ResolveVersion("zod@^3.0")
		if err != nil {
			t.Skip("Network unavailable")
		}
		if name != "zod" {
			t.Errorf("expected name 'zod', got %s", name)
		}
		// Version should start with 3.
		if len(version) < 2 || version[0] != '3' {
			t.Errorf("expected version starting with 3, got %s", version)
		}
	})

	t.Run("scoped package", func(t *testing.T) {
		name, version, err := r.ResolveVersion("@types/node@^20.0")
		if err != nil {
			t.Skip("Network unavailable")
		}
		if name != "@types/node" {
			t.Errorf("expected name '@types/node', got %s", name)
		}
		if version == "" {
			t.Error("expected a version")
		}
	})

	t.Run("nonexistent package", func(t *testing.T) {
		_, _, err := r.ResolveVersion("this-package-definitely-does-not-exist-12345")
		if err == nil {
			t.Error("expected error for nonexistent package")
		}
	})
}
