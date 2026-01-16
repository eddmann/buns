package bun

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/schollz/progressbar/v3"
)

// Downloader handles downloading Bun binaries
type Downloader struct {
	cacheDir string
	verbose  bool
	quiet    bool
}

// NewDownloader creates a new downloader
func NewDownloader(cacheDir string, verbose, quiet bool) *Downloader {
	return &Downloader{
		cacheDir: cacheDir,
		verbose:  verbose,
		quiet:    quiet,
	}
}

// GetBinary returns the path to the Bun binary, downloading if necessary
func (d *Downloader) GetBinary(version *semver.Version) (string, error) {
	binPath := d.binaryPath(version)

	// Check if already cached
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	// Download
	if err := d.download(version); err != nil {
		return "", err
	}

	return binPath, nil
}

// download fetches and extracts the Bun binary
func (d *Downloader) download(version *semver.Version) error {
	url := d.downloadURL(version)

	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "bun-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download with progress bar
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Bun: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Bun: HTTP %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if !d.quiet {
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			fmt.Sprintf("Downloading Bun %s", version.Original()),
		)
		reader = io.TeeReader(resp.Body, bar)
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		return fmt.Errorf("failed to download Bun: %w", err)
	}

	// Extract
	if err := d.extract(tmpFile.Name(), version); err != nil {
		return fmt.Errorf("failed to extract Bun: %w", err)
	}

	return nil
}

// extract unpacks the zip and moves the binary to the cache
func (d *Downloader) extract(zipPath string, version *semver.Version) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Find the bun binary in the zip
	var bunFile *zip.File
	for _, f := range r.File {
		// The binary is at bun-{os}-{arch}/bun
		if strings.HasSuffix(f.Name, "/bun") || f.Name == "bun" {
			bunFile = f
			break
		}
	}

	if bunFile == nil {
		return fmt.Errorf("bun binary not found in archive")
	}

	// Create the cache directory
	versionDir := filepath.Join(d.cacheDir, version.Original())
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return err
	}

	// Extract the binary
	binPath := filepath.Join(versionDir, "bun")
	outFile, err := os.OpenFile(binPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := bunFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return err
	}

	return nil
}

// downloadURL returns the GitHub release URL for the given version
func (d *Downloader) downloadURL(version *semver.Version) string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go's arch names to Bun's
	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "aarch64"
	}

	// Bun uses "darwin" for macOS (same as Go)
	return fmt.Sprintf(
		"https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s-%s.zip",
		version.Original(),
		os,
		arch,
	)
}

// binaryPath returns the expected path to the cached binary
func (d *Downloader) binaryPath(version *semver.Version) string {
	return filepath.Join(d.cacheDir, version.Original(), "bun")
}

// IsCached checks if a version is already downloaded
func (d *Downloader) IsCached(version *semver.Version) bool {
	_, err := os.Stat(d.binaryPath(version))
	return err == nil
}
