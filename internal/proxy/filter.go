package proxy

import (
	"strings"
)

// DomainFilter handles domain allowlisting.
type DomainFilter struct {
	allowedDomains  []string
	wildcardDomains []string // Domains starting with *.
	allowAll        bool
}

// NewDomainFilter creates a new domain filter.
func NewDomainFilter() *DomainFilter {
	return &DomainFilter{
		allowedDomains:  []string{},
		wildcardDomains: []string{},
	}
}

// AllowAll allows all domains (disables filtering).
func (f *DomainFilter) AllowAll() {
	f.allowAll = true
}

// AddAllowed adds a domain to the allow list.
// Supports wildcards like *.github.com
func (f *DomainFilter) AddAllowed(domain string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return
	}

	if strings.HasPrefix(domain, "*.") {
		// Wildcard domain - store the suffix
		f.wildcardDomains = append(f.wildcardDomains, domain[1:]) // Store ".github.com"
	} else {
		f.allowedDomains = append(f.allowedDomains, domain)
	}
}

// IsAllowed checks if a domain is allowed.
func (f *DomainFilter) IsAllowed(host string) bool {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	// Allow all mode
	if f.allowAll {
		return true
	}

	// Check exact matches
	for _, allowed := range f.allowedDomains {
		if host == allowed {
			return true
		}
	}

	// Check wildcard matches
	for _, suffix := range f.wildcardDomains {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}
