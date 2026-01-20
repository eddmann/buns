package sandbox

import (
	"os"
	"strings"
	"testing"
)

func TestFilterEnv(t *testing.T) {
	// Set up some test env vars
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range originalEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	os.Clearenv()
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("HOME", "/home/test")
	os.Setenv("API_KEY", "secret")
	os.Setenv("DATABASE_URL", "postgres://...")
	os.Setenv("LC_ALL", "en_US.UTF-8")
	os.Setenv("XDG_CONFIG_HOME", "/home/test/.config")

	t.Run("filters to safe vars only", func(t *testing.T) {
		result := FilterEnv(nil)

		found := make(map[string]bool)
		for _, e := range result {
			parts := strings.SplitN(e, "=", 2)
			found[parts[0]] = true
		}

		// Safe vars should be included
		if !found["PATH"] {
			t.Error("PATH should be included")
		}
		if !found["HOME"] {
			t.Error("HOME should be included")
		}
		if !found["LC_ALL"] {
			t.Error("LC_ALL should be included (safe prefix)")
		}
		if !found["XDG_CONFIG_HOME"] {
			t.Error("XDG_CONFIG_HOME should be included (safe prefix)")
		}

		// Sensitive vars should be excluded
		if found["API_KEY"] {
			t.Error("API_KEY should not be included")
		}
		if found["DATABASE_URL"] {
			t.Error("DATABASE_URL should not be included")
		}
	})

	t.Run("includes explicitly allowed vars", func(t *testing.T) {
		result := FilterEnv([]string{"API_KEY"})

		found := make(map[string]bool)
		for _, e := range result {
			parts := strings.SplitN(e, "=", 2)
			found[parts[0]] = true
		}

		if !found["API_KEY"] {
			t.Error("API_KEY should be included when explicitly allowed")
		}
	})
}

func TestBuildBunArgs(t *testing.T) {
	cfg := &Config{
		BunBinary:  "/path/to/bun",
		ScriptPath: "/path/to/script.ts",
		ScriptArgs: []string{"--flag", "value"},
	}

	args := BuildBunArgs(cfg)

	expected := []string{"/path/to/bun", "run", "/path/to/script.ts", "--flag", "value"}
	if len(args) != len(expected) {
		t.Fatalf("got %d args, want %d", len(args), len(expected))
	}

	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildEnvWithNodePath(t *testing.T) {
	base := []string{"PATH=/usr/bin", "HOME=/home/test"}

	t.Run("adds NODE_PATH when provided", func(t *testing.T) {
		result := BuildEnvWithNodePath(base, "/path/to/node_modules")

		found := false
		for _, e := range result {
			if e == "NODE_PATH=/path/to/node_modules" {
				found = true
				break
			}
		}

		if !found {
			t.Error("NODE_PATH should be added")
		}
	})

	t.Run("returns base unchanged when empty", func(t *testing.T) {
		result := BuildEnvWithNodePath(base, "")

		if len(result) != len(base) {
			t.Error("should return base unchanged when nodeModules is empty")
		}
	})
}

func TestSeatbeltEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`with"quote`, `with\"quote`},
		{`with\backslash`, `with\\backslash`},
		{`both"and\`, `both\"and\\`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SeatbeltEscape(tt.input)
			if result != tt.expected {
				t.Errorf("SeatbeltEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	t.Run("resolves relative path", func(t *testing.T) {
		result, err := ResolvePath(".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should be absolute
		if !strings.HasPrefix(result, "/") {
			t.Errorf("expected absolute path, got %q", result)
		}
	})

	t.Run("handles non-existent path", func(t *testing.T) {
		result, err := ResolvePath("/non/existent/path/that/does/not/exist")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still return absolute path
		if result != "/non/existent/path/that/does/not/exist" {
			t.Errorf("expected absolute path for non-existent, got %q", result)
		}
	})
}
