package npm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	RegistryURL = "https://registry.npmjs.org"
)

// PackageInfo represents npm package metadata
type PackageInfo struct {
	Name     string                    `json:"name"`
	DistTags map[string]string         `json:"dist-tags"`
	Versions map[string]PackageVersion `json:"versions"`
}

// PackageVersion represents a specific version's metadata
type PackageVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Registry handles npm registry lookups
type Registry struct{}

// NewRegistry creates a new npm registry client
func NewRegistry() *Registry {
	return &Registry{}
}

// ResolveVersion resolves a package spec (name@constraint) to a concrete version
func (r *Registry) ResolveVersion(packageSpec string) (string, string, error) {
	name, constraint := parsePackageSpec(packageSpec)

	info, err := r.fetchPackage(name)
	if err != nil {
		return "", "", err
	}

	version, err := r.resolveConstraint(info, constraint)
	if err != nil {
		return "", "", err
	}

	return name, version, nil
}

// ValidatePackage checks if a package exists
func (r *Registry) ValidatePackage(name string) error {
	_, err := r.fetchPackage(name)
	return err
}

// fetchPackage retrieves package info from npm registry
func (r *Registry) fetchPackage(name string) (*PackageInfo, error) {
	url := fmt.Sprintf("%s/%s", RegistryURL, name)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned %d for %s", resp.StatusCode, name)
	}

	var info PackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %w", err)
	}

	return &info, nil
}

// resolveConstraint finds the best version matching the constraint
func (r *Registry) resolveConstraint(info *PackageInfo, constraint string) (string, error) {
	// No constraint means latest
	if constraint == "" {
		if latest, ok := info.DistTags["latest"]; ok {
			return latest, nil
		}
		return "", fmt.Errorf("no latest version found for %s", info.Name)
	}

	// Try to parse as semver constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		// Maybe it's an exact version
		if _, ok := info.Versions[constraint]; ok {
			return constraint, nil
		}
		return "", fmt.Errorf("invalid version constraint '%s': %w", constraint, err)
	}

	// Collect and sort all versions
	var versions []*semver.Version
	for vStr := range info.Versions {
		v, err := semver.NewVersion(vStr)
		if err != nil {
			continue // Skip invalid versions
		}
		// Skip prereleases unless explicitly requested
		if v.Prerelease() != "" && !strings.Contains(constraint, "-") {
			continue
		}
		versions = append(versions, v)
	}

	// Sort descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	// Find highest matching
	for _, v := range versions {
		if c.Check(v) {
			return v.Original(), nil
		}
	}

	return "", fmt.Errorf("no version of %s satisfies '%s'", info.Name, constraint)
}

// parsePackageSpec splits "name@constraint" into name and constraint
func parsePackageSpec(spec string) (name, constraint string) {
	// Handle scoped packages (@org/name@version)
	if strings.HasPrefix(spec, "@") {
		// Find the second @ which separates name from version
		idx := strings.LastIndex(spec, "@")
		if idx > 0 && idx != strings.Index(spec, "@") {
			return spec[:idx], spec[idx+1:]
		}
		return spec, ""
	}

	// Regular package (name@version)
	if idx := strings.Index(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}

	return spec, ""
}
