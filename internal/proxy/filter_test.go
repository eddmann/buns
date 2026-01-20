package proxy

import "testing"

// createFilter is a helper to create a filter with allowed domains
func createFilter(allowed []string) *DomainFilter {
	f := NewDomainFilter()
	if len(allowed) == 0 {
		f.AllowAll()
	} else {
		for _, domain := range allowed {
			f.AddAllowed(domain)
		}
	}
	return f
}

func TestDomainFilter_IsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		allowed  []string
		host     string
		expected bool
	}{
		{
			name:     "empty filter allows all",
			allowed:  []string{},
			host:     "example.com",
			expected: true,
		},
		{
			name:     "exact match allowed",
			allowed:  []string{"api.github.com"},
			host:     "api.github.com",
			expected: true,
		},
		{
			name:     "exact match with port",
			allowed:  []string{"api.github.com"},
			host:     "api.github.com:443",
			expected: true,
		},
		{
			name:     "case insensitive",
			allowed:  []string{"API.GitHub.com"},
			host:     "api.github.com",
			expected: true,
		},
		{
			name:     "not in allow list",
			allowed:  []string{"api.github.com"},
			host:     "evil.com",
			expected: false,
		},
		{
			name:     "wildcard subdomain match",
			allowed:  []string{"*.github.com"},
			host:     "api.github.com",
			expected: true,
		},
		{
			name:     "wildcard matches nested subdomain",
			allowed:  []string{"*.github.com"},
			host:     "api.v2.github.com",
			expected: true,
		},
		{
			name:     "wildcard does not match base domain",
			allowed:  []string{"*.github.com"},
			host:     "github.com",
			expected: false,
		},
		{
			name:     "multiple allowed hosts",
			allowed:  []string{"api.github.com", "httpbin.org"},
			host:     "httpbin.org",
			expected: true,
		},
		{
			name:     "mixed exact and wildcard",
			allowed:  []string{"api.github.com", "*.example.com"},
			host:     "test.example.com",
			expected: true,
		},
		{
			name:     "IPv4 address",
			allowed:  []string{"127.0.0.1"},
			host:     "127.0.0.1:8080",
			expected: true,
		},
		{
			name:     "IPv6 address with brackets",
			allowed:  []string{"[::1]"},
			host:     "[::1]:8080",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := createFilter(tt.allowed)
			result := f.IsAllowed(tt.host)
			if result != tt.expected {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestDomainFilter_AllowAll(t *testing.T) {
	t.Run("calling AllowAll enables allow all mode", func(t *testing.T) {
		f := NewDomainFilter()
		f.AllowAll()
		if !f.IsAllowed("anything.com") {
			t.Error("expected IsAllowed to return true after AllowAll()")
		}
	})

	t.Run("filter with domains does not allow all", func(t *testing.T) {
		f := NewDomainFilter()
		f.AddAllowed("example.com")
		if f.IsAllowed("other.com") {
			t.Error("expected IsAllowed to return false for non-allowed domain")
		}
	})
}

func TestDomainFilter_AddAllowed(t *testing.T) {
	t.Run("adds exact domain", func(t *testing.T) {
		f := NewDomainFilter()
		f.AddAllowed("example.com")
		if !f.IsAllowed("example.com") {
			t.Error("expected domain to be allowed after AddAllowed")
		}
	})

	t.Run("adds wildcard domain", func(t *testing.T) {
		f := NewDomainFilter()
		f.AddAllowed("*.example.com")
		if !f.IsAllowed("sub.example.com") {
			t.Error("expected subdomain to be allowed after AddAllowed with wildcard")
		}
	})

	t.Run("ignores empty string", func(t *testing.T) {
		f := NewDomainFilter()
		f.AddAllowed("")
		f.AddAllowed("   ")
		// Should not allow anything since no valid domains added
		if f.IsAllowed("example.com") {
			t.Error("expected empty strings to be ignored")
		}
	})
}
