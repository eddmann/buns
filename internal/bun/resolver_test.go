package bun

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
)

// stubVersionSource is a stub for testing version resolution
type stubVersionSource struct {
	versions []*semver.Version
	err      error
}

func (s *stubVersionSource) GetVersions() ([]*semver.Version, error) {
	return s.versions, s.err
}

func mustVersion(s string) *semver.Version {
	v, err := semver.NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestResolver_Resolve(t *testing.T) {
	source := &stubVersionSource{
		versions: []*semver.Version{
			mustVersion("1.1.34"),
			mustVersion("1.1.33"),
			mustVersion("1.1.32"),
			mustVersion("1.1.0"),
			mustVersion("1.0.0"),
		},
	}
	resolver := NewResolver(source)

	tests := []struct {
		name       string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "returns latest when no constraint",
			constraint: "",
			want:       "1.1.34",
		},
		{
			name:       "returns exact version when specified",
			constraint: "1.1.33",
			want:       "1.1.33",
		},
		{
			name:       "returns highest matching for caret constraint",
			constraint: "^1.1.0",
			want:       "1.1.34",
		},
		{
			name:       "returns highest patch for tilde constraint",
			constraint: "~1.1.32",
			want:       "1.1.34",
		},
		{
			name:       "returns highest for greater than or equal",
			constraint: ">=1.1.0",
			want:       "1.1.34",
		},
		{
			name:       "returns highest matching for greater than",
			constraint: ">1.1.32",
			want:       "1.1.34",
		},
		{
			name:       "returns matching version for less than",
			constraint: "<1.1.0",
			want:       "1.0.0",
		},
		{
			name:       "returns highest in range",
			constraint: ">=1.1.0, <1.1.34",
			want:       "1.1.33",
		},
		{
			name:       "returns error when no version matches",
			constraint: ">=2.0.0",
			wantErr:    true,
		},
		{
			name:       "returns error for invalid constraint",
			constraint: "not-a-version",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.constraint)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Original() != tt.want {
				t.Errorf("got %s, want %s", got.Original(), tt.want)
			}
		})
	}
}

func TestResolver_Resolve_returns_error_when_source_fails(t *testing.T) {
	source := &stubVersionSource{
		err: errors.New("network error"),
	}
	resolver := NewResolver(source)

	_, err := resolver.Resolve("")

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestResolver_Resolve_returns_error_when_no_versions_available(t *testing.T) {
	source := &stubVersionSource{
		versions: []*semver.Version{},
	}
	resolver := NewResolver(source)

	_, err := resolver.Resolve("")

	if err == nil {
		t.Error("expected error, got nil")
	}
}
