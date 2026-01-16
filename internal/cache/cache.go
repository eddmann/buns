package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Cache manages the buns cache directory
type Cache struct {
	baseDir string
}

// New creates a new cache manager
func New(baseDir string) *Cache {
	return &Cache{baseDir: baseDir}
}

// Default returns a cache using the default directory (~/.buns)
func Default() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return New(filepath.Join(home, ".buns")), nil
}

// BaseDir returns the base cache directory
func (c *Cache) BaseDir() string {
	return c.baseDir
}

// BunDir returns the directory for Bun binaries
func (c *Cache) BunDir() string {
	return filepath.Join(c.baseDir, "bun")
}

// DepsDir returns the directory for script dependencies
func (c *Cache) DepsDir() string {
	return filepath.Join(c.baseDir, "deps")
}

// IndexDir returns the directory for cached index data
func (c *Cache) IndexDir() string {
	return filepath.Join(c.baseDir, "index")
}

// DepsDirForHash returns the directory for a specific dependency hash
func (c *Cache) DepsDirForHash(hash string) string {
	return filepath.Join(c.DepsDir(), hash)
}

// HashPackages creates a cache key from a list of packages
func HashPackages(packages []string) string {
	// Sort and lowercase for consistent hashing
	sorted := make([]string, len(packages))
	copy(sorted, packages)
	sort.Strings(sorted)

	for i, p := range sorted {
		sorted[i] = strings.ToLower(p)
	}

	joined := strings.Join(sorted, "\n")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}

// IsDepsHit checks if dependencies are cached for the given hash
func (c *Cache) IsDepsHit(hash string) bool {
	nodeModules := filepath.Join(c.DepsDirForHash(hash), "node_modules")
	info, err := os.Stat(nodeModules)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Check that node_modules is non-empty
	entries, err := os.ReadDir(nodeModules)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// EnsureDirs creates all necessary cache directories
func (c *Cache) EnsureDirs() error {
	dirs := []string{
		c.BunDir(),
		c.DepsDir(),
		c.IndexDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// CleanBun removes all cached Bun binaries
func (c *Cache) CleanBun() error {
	return os.RemoveAll(c.BunDir())
}

// CleanDeps removes all cached dependencies
func (c *Cache) CleanDeps() error {
	return os.RemoveAll(c.DepsDir())
}

// CleanIndex removes the cached index
func (c *Cache) CleanIndex() error {
	return os.RemoveAll(c.IndexDir())
}

// CleanAll removes the entire cache
func (c *Cache) CleanAll() error {
	return os.RemoveAll(c.baseDir)
}

// ListBunVersions returns all cached Bun versions
func (c *Cache) ListBunVersions() ([]string, error) {
	entries, err := os.ReadDir(c.BunDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	return versions, nil
}

// ListDepsHashes returns all cached dependency hashes
func (c *Cache) ListDepsHashes() ([]string, error) {
	entries, err := os.ReadDir(c.DepsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var hashes []string
	for _, entry := range entries {
		if entry.IsDir() {
			hashes = append(hashes, entry.Name())
		}
	}
	return hashes, nil
}

// Size returns the total size of the cache in bytes
func (c *Cache) Size() (int64, error) {
	var size int64
	err := filepath.Walk(c.baseDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return size, err
}
