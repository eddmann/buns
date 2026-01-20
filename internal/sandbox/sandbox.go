package sandbox

import (
	"context"
	"os/exec"
	"runtime"
)

// Sandbox is the interface for script execution isolation
type Sandbox interface {
	// Name returns the sandbox implementation name
	Name() string

	// IsSandboxed returns true if this sandbox provides actual isolation
	IsSandboxed() bool

	// Available returns true if this sandbox can run on the current system
	Available() bool

	// Execute runs the script within the sandbox
	Execute(ctx context.Context, cfg *Config) (*Result, error)
}

// Result contains execution outcome
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Detect returns the best available sandbox for the platform
// If fullSandbox is true, it attempts to find a full filesystem+process isolation sandbox
// Otherwise, it looks for network-only isolation
func Detect(fullSandbox bool) Sandbox {
	if fullSandbox {
		return detectFullSandbox()
	}
	return detectNetworkSandbox()
}

// detectFullSandbox returns the best full sandbox for the platform
func detectFullSandbox() Sandbox {
	switch runtime.GOOS {
	case "darwin":
		sb := &MacOS{}
		if sb.Available() {
			return sb
		}
	case "linux":
		// Try bubblewrap first, then nsjail
		bwrap := &Bubblewrap{}
		if bwrap.Available() {
			return bwrap
		}
		nsjail := &Nsjail{}
		if nsjail.Available() {
			return nsjail
		}
	}
	return &None{}
}

// detectNetworkSandbox returns a network-only sandbox for the platform
func detectNetworkSandbox() Sandbox {
	switch runtime.GOOS {
	case "darwin":
		sb := &MacOSNetwork{}
		if sb.Available() {
			return sb
		}
	case "linux":
		sb := &LinuxNetwork{}
		if sb.Available() {
			return sb
		}
	}
	return &None{}
}

// commandExists checks if a command is available in PATH
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
