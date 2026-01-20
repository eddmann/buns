package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// MacOS implements full filesystem and process isolation using Seatbelt.
type MacOS struct{}

// Name returns the sandbox name.
func (m *MacOS) Name() string {
	return "macos"
}

// IsSandboxed returns true since this provides full isolation.
func (m *MacOS) IsSandboxed() bool {
	return true
}

// Available checks if sandbox-exec is available on the system.
func (m *MacOS) Available() bool {
	return runtime.GOOS == "darwin" && commandExists("sandbox-exec")
}

// Execute runs the script within macOS Seatbelt sandbox.
func (m *MacOS) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	profile := m.generateProfile(cfg)

	// Write profile to temp file
	profileFile, err := os.CreateTemp("", "buns-sandbox-*.sb")
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

	cmd.Dir = cfg.WorkDir

	err = cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// generateProfile creates a minimal Seatbelt sandbox profile.
// Follows principle of least privilege - only allows what's strictly necessary.
func (m *MacOS) generateProfile(cfg *Config) string {
	var profile bytes.Buffer

	profile.WriteString("(version 1)\n")
	profile.WriteString("(deny default)\n\n")

	// ============================================================
	// PROCESS & SYSTEM OPERATIONS
	// ============================================================
	profile.WriteString(";; Process operations (required for Bun to run)\n")
	profile.WriteString("(allow process*)\n")
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(allow mach-lookup)\n")
	profile.WriteString("(allow signal (target self))\n\n")

	// ============================================================
	// MINIMAL READ ACCESS
	// ============================================================
	profile.WriteString(";; Root directory (required for path resolution)\n")
	profile.WriteString("(allow file-read* (literal \"/\"))\n\n")

	profile.WriteString(";; Minimal device access\n")
	profile.WriteString("(allow file-read* (literal \"/dev/null\"))\n")
	profile.WriteString("(allow file-read* (literal \"/dev/urandom\"))\n")
	profile.WriteString("(allow file-read* (literal \"/dev/random\"))\n\n")

	// Timezone support
	profile.WriteString(";; Timezone data\n")
	profile.WriteString("(allow file-read* (subpath \"/usr/share/zoneinfo\"))\n")
	profile.WriteString("(allow file-read* (subpath \"/var/db/timezone\"))\n")
	profile.WriteString("(allow file-read* (literal \"/etc/localtime\"))\n")
	profile.WriteString("(allow file-read* (literal \"/private/etc/localtime\"))\n\n")

	// DNS resolution (only if network enabled)
	if cfg.Network {
		profile.WriteString(";; DNS resolution (network enabled)\n")
		profile.WriteString("(allow file-read* (literal \"/etc/resolv.conf\"))\n")
		profile.WriteString("(allow file-read* (literal \"/private/etc/resolv.conf\"))\n")
		profile.WriteString("(allow file-read* (literal \"/etc/hosts\"))\n")
		profile.WriteString("(allow file-read* (literal \"/private/etc/hosts\"))\n\n")

		profile.WriteString(";; SSL certificates (required for HTTPS)\n")
		profile.WriteString("(allow file-read* (literal \"/etc\"))\n")
		profile.WriteString("(allow file-read* (subpath \"/private/etc/ssl\"))\n\n")
	}

	// Bun binary directory
	if cfg.BunBinary != "" {
		profile.WriteString(";; Bun binary\n")
		bunDir := filepath.Dir(cfg.BunBinary)
		m.addPathComponents(&profile, bunDir)
		profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n\n", SeatbeltEscape(bunDir)))
	}

	// Script file and directory
	if cfg.ScriptPath != "" {
		profile.WriteString(";; Script file\n")
		scriptDir := filepath.Dir(cfg.ScriptPath)
		m.addPathComponents(&profile, scriptDir)
		profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n\n", SeatbeltEscape(scriptDir)))
	}

	// Node modules directory
	if cfg.NodeModules != "" {
		profile.WriteString(";; Node modules (dependencies)\n")
		resolved, _ := ResolvePath(cfg.NodeModules)
		m.addPathComponents(&profile, resolved)
		profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n\n", SeatbeltEscape(resolved)))
	}

	// Additional readable paths from --allow-read flag
	if len(cfg.ReadablePaths) > 0 {
		profile.WriteString(";; Additional readable paths (--allow-read)\n")
		for _, path := range cfg.ReadablePaths {
			resolved, err := ResolvePath(path)
			if err != nil {
				continue
			}
			// If path is a symlink, allow both the symlink and resolved path
			m.addPathComponents(&profile, path)
			if path != resolved {
				profile.WriteString(fmt.Sprintf("(allow file-read* (literal \"%s\"))\n", SeatbeltEscape(path)))
				m.addPathComponents(&profile, resolved)
			}
			profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", SeatbeltEscape(resolved)))
		}
		profile.WriteString("\n")
	}

	// ============================================================
	// MINIMAL WRITE ACCESS
	// ============================================================
	profile.WriteString(";; Minimal write access\n")
	profile.WriteString("(allow file-write* (literal \"/dev/null\"))\n\n")

	// Additional writable paths from --allow-write flag
	if len(cfg.WritablePaths) > 0 {
		profile.WriteString(";; Additional writable paths (--allow-write)\n")
		for _, path := range cfg.WritablePaths {
			resolved, err := ResolvePath(path)
			if err != nil {
				continue
			}
			// If path is a symlink, allow both the symlink and resolved path
			m.addPathComponents(&profile, path)
			if path != resolved {
				profile.WriteString(fmt.Sprintf("(allow file-read* (literal \"%s\"))\n", SeatbeltEscape(path)))
				m.addPathComponents(&profile, resolved)
			}
			// Need both read and write access
			profile.WriteString(fmt.Sprintf("(allow file-write* (subpath \"%s\"))\n", SeatbeltEscape(resolved)))
			profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", SeatbeltEscape(resolved)))
		}
		profile.WriteString("\n")
	}

	// ============================================================
	// NETWORK ACCESS
	// ============================================================
	if cfg.Network {
		profile.WriteString(";; Network: proxy connections only\n")
		if cfg.ProxyPort > 0 {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxyPort))
		}
		if cfg.ProxySOCKS5Port > 0 {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxySOCKS5Port))
		}
		// Always allow Unix socket connections for proxy
		profile.WriteString("(allow network-outbound (remote unix-socket))\n")
	}

	return profile.String()
}

// addPathComponents adds file-read* literal permissions for each directory in the path.
func (m *MacOS) addPathComponents(profile *bytes.Buffer, path string) {
	components := []string{}
	current := path
	for current != "/" && current != "" {
		parent := filepath.Dir(current)
		if parent != current {
			components = append([]string{parent}, components...)
		}
		current = parent
	}
	for _, comp := range components {
		if comp != "/" {
			_, _ = fmt.Fprintf(profile, "(allow file-read* (literal \"%s\"))\n", SeatbeltEscape(comp))
		}
	}
}
