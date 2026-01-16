package cache

import (
	"os"
	"path/filepath"
	"testing"
)

const SHA256HexLength = 64

func TestHashPackages(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		compare  []string
		wantSame bool
	}{
		{
			name:     "produces deterministic hash",
			packages: []string{"zod@^3.0", "chalk@^5.0"},
			compare:  []string{"zod@^3.0", "chalk@^5.0"},
			wantSame: true,
		},
		{
			name:     "produces same hash regardless of order",
			packages: []string{"chalk@^5.0", "zod@^3.0"},
			compare:  []string{"zod@^3.0", "chalk@^5.0"},
			wantSame: true,
		},
		{
			name:     "produces same hash regardless of case",
			packages: []string{"ZOD@^3.0"},
			compare:  []string{"zod@^3.0"},
			wantSame: true,
		},
		{
			name:     "produces different hash for different packages",
			packages: []string{"zod@^3.0"},
			compare:  []string{"chalk@^5.0"},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashPackages(tt.packages)
			hash2 := HashPackages(tt.compare)

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("HashPackages(%v) = %s, HashPackages(%v) = %s, want same",
					tt.packages, hash1, tt.compare, hash2)
			}

			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("HashPackages(%v) = HashPackages(%v), want different",
					tt.packages, tt.compare)
			}
		})
	}

	t.Run("produces 64 character hex string", func(t *testing.T) {
		hash := HashPackages([]string{"test@^1.0"})

		if len(hash) != SHA256HexLength {
			t.Errorf("len(hash) = %d, want %d", len(hash), SHA256HexLength)
		}
	})
}

func TestCache_Paths(t *testing.T) {
	c := New("/tmp/test-buns")

	if c.BaseDir() != "/tmp/test-buns" {
		t.Errorf("unexpected base dir: %s", c.BaseDir())
	}

	if c.BunDir() != "/tmp/test-buns/bun" {
		t.Errorf("unexpected bun dir: %s", c.BunDir())
	}

	if c.DepsDir() != "/tmp/test-buns/deps" {
		t.Errorf("unexpected deps dir: %s", c.DepsDir())
	}

	if c.IndexDir() != "/tmp/test-buns/index" {
		t.Errorf("unexpected index dir: %s", c.IndexDir())
	}

	hash := "abc123"
	if c.DepsDirForHash(hash) != "/tmp/test-buns/deps/abc123" {
		t.Errorf("unexpected deps hash dir: %s", c.DepsDirForHash(hash))
	}
}

func TestCache_IsDepsHit(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	hash := "test-hash"

	t.Run("returns false for missing directory", func(t *testing.T) {
		hit := c.IsDepsHit(hash)

		if hit {
			t.Error("got true, want false")
		}
	})

	t.Run("returns false for empty node_modules", func(t *testing.T) {
		depsDir := c.DepsDirForHash(hash)
		os.MkdirAll(filepath.Join(depsDir, "node_modules"), 0755)

		hit := c.IsDepsHit(hash)

		if hit {
			t.Error("got true, want false")
		}
	})

	t.Run("returns true when node_modules has packages", func(t *testing.T) {
		depsDir := c.DepsDirForHash(hash)
		os.MkdirAll(filepath.Join(depsDir, "node_modules", "some-package"), 0755)

		hit := c.IsDepsHit(hash)

		if !hit {
			t.Error("got false, want true")
		}
	})
}

func TestCache_EnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(filepath.Join(tmpDir, "buns-test"))

	if err := c.EnsureDirs(); err != nil {
		t.Errorf("EnsureDirs failed: %v", err)
	}

	// Check directories exist
	dirs := []string{c.BunDir(), c.DepsDir(), c.IndexDir()}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestCache_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.EnsureDirs()

	// Create some files
	os.WriteFile(filepath.Join(c.BunDir(), "test"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(c.DepsDir(), "test"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(c.IndexDir(), "test"), []byte("test"), 0644)

	t.Run("CleanBun", func(t *testing.T) {
		c.CleanBun()
		if _, err := os.Stat(c.BunDir()); !os.IsNotExist(err) {
			t.Error("bun dir should be removed")
		}
	})

	t.Run("CleanDeps", func(t *testing.T) {
		c.CleanDeps()
		if _, err := os.Stat(c.DepsDir()); !os.IsNotExist(err) {
			t.Error("deps dir should be removed")
		}
	})

	t.Run("CleanIndex", func(t *testing.T) {
		c.CleanIndex()
		if _, err := os.Stat(c.IndexDir()); !os.IsNotExist(err) {
			t.Error("index dir should be removed")
		}
	})
}

func TestCache_ListBunVersions(t *testing.T) {
	tmpDir := t.TempDir()
	c := New(tmpDir)
	c.EnsureDirs()

	// Create some version dirs
	os.MkdirAll(filepath.Join(c.BunDir(), "1.1.34"), 0755)
	os.MkdirAll(filepath.Join(c.BunDir(), "1.1.33"), 0755)

	versions, err := c.ListBunVersions()
	if err != nil {
		t.Errorf("ListBunVersions failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}
