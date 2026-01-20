package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// MacOSNetwork implements network-only isolation using Seatbelt.
// It allows full filesystem access but restricts network to proxy only.
type MacOSNetwork struct{}

// Name returns the sandbox name.
func (m *MacOSNetwork) Name() string {
	return "macos-network"
}

// IsSandboxed returns true since this provides network isolation.
func (m *MacOSNetwork) IsSandboxed() bool {
	return true
}

// Available checks if sandbox-exec is available on the system.
func (m *MacOSNetwork) Available() bool {
	return runtime.GOOS == "darwin" && commandExists("sandbox-exec")
}

// Execute runs the script with network-only restriction.
func (m *MacOSNetwork) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	profile := m.generateProfile(cfg)

	// Write profile to temp file
	profileFile, err := os.CreateTemp("", "buns-network-*.sb")
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox profile: %w", err)
	}
	defer func() { _ = os.Remove(profileFile.Name()) }()

	if _, err := profileFile.WriteString(profile); err != nil {
		_ = profileFile.Close()
		return nil, fmt.Errorf("failed to write sandbox profile: %w", err)
	}
	_ = profileFile.Close()

	// Build command: sandbox-exec -f profile bun run script.ts args...
	args := append([]string{"-f", profileFile.Name()}, BuildBunArgs(cfg)...)

	cmd := exec.CommandContext(ctx, "sandbox-exec", args...)

	// Setup I/O and environment
	stdout, stderr := SetupCommand(cmd, cfg)

	// Add NODE_PATH
	cmd.Env = BuildEnvWithNodePath(cmd.Env, cfg.NodeModules)

	// Add memory limit hint (soft limit only - Seatbelt doesn't support resource limits)
	cmd.Env = BuildEnvWithMemoryLimit(cmd.Env, cfg.MemoryMB)

	err = cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// generateProfile creates a Seatbelt profile that only restricts network.
func (m *MacOSNetwork) generateProfile(cfg *Config) string {
	var profile bytes.Buffer

	profile.WriteString("(version 1)\n")
	profile.WriteString("(allow default)\n\n")

	// Block all network by default
	profile.WriteString(";; Block all network except proxy\n")
	profile.WriteString("(deny network*)\n\n")

	// If network is enabled, allow proxy connections
	if cfg.Network && cfg.ProxyPort > 0 {
		profile.WriteString(";; Allow proxy connections (localhost only)\n")
		profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxyPort))
		if cfg.ProxySOCKS5Port > 0 {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxySOCKS5Port))
		}
		profile.WriteString("(allow network-outbound (remote unix-socket))\n")
	}

	return profile.String()
}
