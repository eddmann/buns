package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	GitHubReleasesURL = "https://api.github.com/repos/oven-sh/bun/releases"
	CacheTTL          = 24 * time.Hour
)

var (
	ErrNoCache     = errors.New("no cached index available")
	versionRegex   = regexp.MustCompile(`^bun-v(\d+\.\d+\.\d+)$`)
)

// Index manages the cached Bun version index
type Index struct {
	cacheDir string
}

// GitHubRelease represents a release from GitHub API
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// New creates a new Index with the given cache directory
func New(cacheDir string) *Index {
	return &Index{cacheDir: cacheDir}
}

// GetVersions returns available Bun versions, fetching from GitHub if cache is stale
func (idx *Index) GetVersions() ([]*semver.Version, error) {
	versions, err := idx.loadCachedVersions()
	if err == nil && !idx.isCacheStale() {
		return versions, nil
	}

	// Fetch from GitHub
	versions, err = idx.fetchVersions()
	if err != nil {
		// If fetch fails but we have cached versions, use them
		if cached, cacheErr := idx.loadCachedVersions(); cacheErr == nil {
			return cached, nil
		}
		return nil, fmt.Errorf("failed to fetch Bun index from GitHub: %w\nRun with network access to initialize the index cache", err)
	}

	// Cache the versions (non-fatal if it fails)
	_ = idx.cacheVersions(versions)

	return versions, nil
}

// fetchVersions fetches available versions from GitHub releases
func (idx *Index) fetchVersions() ([]*semver.Version, error) {
	req, err := http.NewRequest("GET", GitHubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "buns-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var versions []*semver.Version
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}

		matches := versionRegex.FindStringSubmatch(release.TagName)
		if len(matches) != 2 {
			continue
		}

		v, err := semver.NewVersion(matches[1])
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	// Sort descending (highest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	return versions, nil
}

// loadCachedVersions loads versions from the cache file
func (idx *Index) loadCachedVersions() ([]*semver.Version, error) {
	data, err := os.ReadFile(idx.versionsFile())
	if err != nil {
		return nil, ErrNoCache
	}

	var versionStrings []string
	if err := json.Unmarshal(data, &versionStrings); err != nil {
		return nil, err
	}

	var versions []*semver.Version
	for _, vs := range versionStrings {
		v, err := semver.NewVersion(vs)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// cacheVersions saves versions to the cache file
func (idx *Index) cacheVersions(versions []*semver.Version) error {
	if err := os.MkdirAll(idx.cacheDir, 0755); err != nil {
		return err
	}

	var versionStrings []string
	for _, v := range versions {
		versionStrings = append(versionStrings, v.Original())
	}

	data, err := json.Marshal(versionStrings)
	if err != nil {
		return err
	}

	if err := os.WriteFile(idx.versionsFile(), data, 0644); err != nil {
		return err
	}

	// Update timestamp
	return os.WriteFile(idx.timestampFile(), []byte(time.Now().Format(time.RFC3339)), 0644)
}

// isCacheStale checks if the cache is older than CacheTTL
func (idx *Index) isCacheStale() bool {
	data, err := os.ReadFile(idx.timestampFile())
	if err != nil {
		return true
	}

	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return true
	}

	return time.Since(t) > CacheTTL
}

func (idx *Index) versionsFile() string {
	return filepath.Join(idx.cacheDir, "bun-versions.json")
}

func (idx *Index) timestampFile() string {
	return filepath.Join(idx.cacheDir, "fetched_at")
}
