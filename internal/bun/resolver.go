package bun

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

var (
	ErrNoMatchingVersion = errors.New("no Bun version satisfies constraint")
)

// VersionSource provides available Bun versions
type VersionSource interface {
	GetVersions() ([]*semver.Version, error)
}

// Resolver handles Bun version resolution
type Resolver struct {
	source VersionSource
}

// NewResolver creates a new resolver with the given version source
func NewResolver(source VersionSource) *Resolver {
	return &Resolver{source: source}
}

// Resolve finds the highest Bun version matching the constraint
func (r *Resolver) Resolve(constraint string) (*semver.Version, error) {
	versions, err := r.source.GetVersions()
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, errors.New("no Bun versions available")
	}

	// If no constraint, return the latest version
	if constraint == "" {
		return versions[0], nil
	}

	// Parse the constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint '%s': %w", constraint, err)
	}

	// Find the highest matching version (versions are already sorted descending)
	for _, v := range versions {
		if c.Check(v) {
			return v, nil
		}
	}

	return nil, fmt.Errorf("%w: '%s'", ErrNoMatchingVersion, constraint)
}

// ResolveExact finds an exact version or returns an error
func (r *Resolver) ResolveExact(version string) (*semver.Version, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version '%s': %w", version, err)
	}

	versions, err := r.source.GetVersions()
	if err != nil {
		return nil, err
	}

	for _, available := range versions {
		if available.Equal(v) {
			return available, nil
		}
	}

	return nil, fmt.Errorf("%w: '%s'", ErrNoMatchingVersion, version)
}
