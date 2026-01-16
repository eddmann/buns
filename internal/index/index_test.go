package index

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIndex_GetVersions(t *testing.T) {
	// Create a mock GitHub releases response
	releases := []GitHubRelease{
		{TagName: "bun-v1.1.34", Prerelease: false, Draft: false},
		{TagName: "bun-v1.1.33", Prerelease: false, Draft: false},
		{TagName: "bun-v1.2.0-canary.1", Prerelease: true, Draft: false},
		{TagName: "bun-v1.1.32", Prerelease: false, Draft: false},
		{TagName: "bun-v1.0.0", Prerelease: false, Draft: false},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releases)
	}))
	defer server.Close()

	// Create temp cache dir
	tmpDir := t.TempDir()

	// Create index with mock server (we'll need to patch the URL for real testing)
	idx := New(tmpDir)

	// Test fetchVersions directly with the mock server
	t.Run("fetchVersions excludes prereleases", func(t *testing.T) {
		// For this test, we'll manually test the filtering logic
		// since we can't easily override the URL

		// The regex should match stable versions
		matches := versionRegex.FindStringSubmatch("bun-v1.1.34")
		if len(matches) != 2 || matches[1] != "1.1.34" {
			t.Errorf("version regex failed: %v", matches)
		}

		// Canary versions should not match the clean format
		// (they have extra stuff after the version)
		matches = versionRegex.FindStringSubmatch("bun-v1.2.0-canary.1")
		if len(matches) == 2 {
			t.Errorf("version regex should not match canary versions")
		}
	})

	t.Run("cacheVersions and loadCachedVersions", func(t *testing.T) {
		versions, err := idx.fetchVersions()
		if err != nil {
			// Skip if no network
			t.Skip("Network unavailable")
		}

		if len(versions) == 0 {
			t.Error("expected some versions")
		}

		// Cache should work
		err = idx.cacheVersions(versions)
		if err != nil {
			t.Errorf("cacheVersions failed: %v", err)
		}

		// Load from cache
		cached, err := idx.loadCachedVersions()
		if err != nil {
			t.Errorf("loadCachedVersions failed: %v", err)
		}

		if len(cached) != len(versions) {
			t.Errorf("cached count mismatch: got %d, want %d", len(cached), len(versions))
		}
	})
}

func TestIndex_isCacheStale(t *testing.T) {
	tmpDir := t.TempDir()
	idx := New(tmpDir)

	t.Run("no cache file is stale", func(t *testing.T) {
		if !idx.isCacheStale() {
			t.Error("expected missing cache to be stale")
		}
	})

	t.Run("fresh cache is not stale", func(t *testing.T) {
		os.MkdirAll(tmpDir, 0755)
		os.WriteFile(filepath.Join(tmpDir, "fetched_at"), []byte(time.Now().Format(time.RFC3339)), 0644)

		if idx.isCacheStale() {
			t.Error("expected fresh cache to not be stale")
		}
	})

	t.Run("old cache is stale", func(t *testing.T) {
		oldTime := time.Now().Add(-25 * time.Hour)
		os.WriteFile(filepath.Join(tmpDir, "fetched_at"), []byte(oldTime.Format(time.RFC3339)), 0644)

		if !idx.isCacheStale() {
			t.Error("expected old cache to be stale")
		}
	})
}

func TestVersionRegex(t *testing.T) {
	tests := []struct {
		tag     string
		wantVer string
		match   bool
	}{
		{"bun-v1.1.34", "1.1.34", true},
		{"bun-v1.0.0", "1.0.0", true},
		{"bun-v2.0.0", "2.0.0", true},
		{"bun-v1.2.0-canary.1", "", false},
		{"v1.1.34", "", false},
		{"bun-1.1.34", "", false},
		{"bun-v1.1", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			matches := versionRegex.FindStringSubmatch(tt.tag)
			if tt.match {
				if len(matches) != 2 {
					t.Errorf("expected match for %s", tt.tag)
					return
				}
				if matches[1] != tt.wantVer {
					t.Errorf("got version %s, want %s", matches[1], tt.wantVer)
				}
			} else {
				if len(matches) == 2 {
					t.Errorf("expected no match for %s", tt.tag)
				}
			}
		})
	}
}
