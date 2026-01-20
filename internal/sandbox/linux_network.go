package sandbox

import (
	"context"
	"os/exec"
)

// LinuxNetwork implements network-only isolation using unshare
type LinuxNetwork struct{}

// Name returns the sandbox name
func (l *LinuxNetwork) Name() string {
	return "linux-network"
}

// IsSandboxed returns true since this provides network isolation
func (l *LinuxNetwork) IsSandboxed() bool {
	return true
}

// Available checks if unshare is available on the system
func (l *LinuxNetwork) Available() bool {
	return commandExists("unshare")
}

// Execute runs the script with network-only isolation
func (l *LinuxNetwork) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	var cmd *exec.Cmd

	if !cfg.Network {
		// Full network isolation - no proxy needed
		cmd = l.buildOfflineCommand(ctx, cfg)
	} else if cfg.ProxySocketPath != "" {
		// Network through proxy with socat bridge
		cmd = l.buildProxyCommand(ctx, cfg)
	} else {
		// No isolation needed, fall back to direct execution
		args := BuildBunArgs(cfg)
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}

	// Setup I/O and environment
	stdout, stderr := SetupCommand(cmd, cfg)

	// Add NODE_PATH
	cmd.Env = BuildEnvWithNodePath(cmd.Env, cfg.NodeModules)

	// Add memory limit hint (soft limit only - unshare doesn't support rlimits)
	cmd.Env = BuildEnvWithMemoryLimit(cmd.Env, cfg.MemoryMB)

	err := cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildOfflineCommand creates a command for completely offline execution
func (l *LinuxNetwork) buildOfflineCommand(ctx context.Context, cfg *Config) *exec.Cmd {
	bunArgs := BuildBunArgs(cfg)

	// unshare --net creates a new network namespace with no network access
	args := []string{"--net", "--map-root-user", "--"}
	args = append(args, bunArgs...)

	return exec.CommandContext(ctx, "unshare", args...)
}

// buildProxyCommand creates a command with network isolation but proxy access
func (l *LinuxNetwork) buildProxyCommand(ctx context.Context, cfg *Config) *exec.Cmd {
	bunCmd := BuildBunCommand(cfg)
	script := BuildSocatBridgeCommand(cfg.ProxySocketPath, bunCmd)

	// Use unshare to create isolated network, then run the bridge script
	args := []string{"--net", "--map-root-user", "sh", "-c", script}

	return exec.CommandContext(ctx, "unshare", args...)
}
