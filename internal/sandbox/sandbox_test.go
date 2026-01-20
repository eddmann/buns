package sandbox

import (
	"runtime"
	"testing"
)

func TestDetect_fullSandbox(t *testing.T) {
	sb := Detect(true)

	if sb == nil {
		t.Fatal("Detect(true) returned nil")
	}

	// Should always return something (at least None)
	if sb.Name() == "" {
		t.Error("sandbox Name() is empty")
	}

	// Available should not panic
	_ = sb.Available()

	// IsSandboxed depends on platform
	switch runtime.GOOS {
	case "darwin":
		// On macOS, should return MacOS if sandbox-exec is available
		if sb.Available() && sb.Name() != "macos" && sb.Name() != "none" {
			t.Errorf("expected macos or none sandbox on darwin, got %s", sb.Name())
		}
	case "linux":
		// On Linux, should return bubblewrap, nsjail, or none
		validNames := map[string]bool{"bubblewrap": true, "nsjail": true, "none": true}
		if !validNames[sb.Name()] {
			t.Errorf("unexpected sandbox name on linux: %s", sb.Name())
		}
	default:
		// On other platforms, should return None
		if sb.Name() != "none" {
			t.Errorf("expected none sandbox on %s, got %s", runtime.GOOS, sb.Name())
		}
	}
}

func TestDetect_networkOnly(t *testing.T) {
	sb := Detect(false)

	if sb == nil {
		t.Fatal("Detect(false) returned nil")
	}

	// Should always return something
	if sb.Name() == "" {
		t.Error("sandbox Name() is empty")
	}

	switch runtime.GOOS {
	case "darwin":
		// On macOS, should return MacOSNetwork if available
		validNames := map[string]bool{"macos-network": true, "none": true}
		if !validNames[sb.Name()] {
			t.Errorf("expected macos-network or none on darwin, got %s", sb.Name())
		}
	case "linux":
		// On Linux, should return LinuxNetwork or none
		validNames := map[string]bool{"linux-network": true, "none": true}
		if !validNames[sb.Name()] {
			t.Errorf("expected linux-network or none on linux, got %s", sb.Name())
		}
	default:
		if sb.Name() != "none" {
			t.Errorf("expected none on %s, got %s", runtime.GOOS, sb.Name())
		}
	}
}

func TestNone_Interface(t *testing.T) {
	var sb Sandbox = &None{}

	if sb.Name() != "none" {
		t.Errorf("Name() = %q, want %q", sb.Name(), "none")
	}

	if sb.IsSandboxed() {
		t.Error("IsSandboxed() should return false")
	}

	if !sb.Available() {
		t.Error("Available() should always return true")
	}
}

func TestCommandExists(t *testing.T) {
	// "sh" should exist on all Unix systems
	if !commandExists("sh") {
		t.Error("sh should exist")
	}

	// Random non-existent command
	if commandExists("this-command-definitely-does-not-exist-12345") {
		t.Error("non-existent command should not exist")
	}
}
