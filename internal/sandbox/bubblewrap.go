package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Bubblewrap implements full sandbox using Linux bubblewrap (bwrap)
type Bubblewrap struct{}

// Name returns the sandbox name
func (b *Bubblewrap) Name() string {
	return "bubblewrap"
}

// IsSandboxed returns true since this provides full isolation
func (b *Bubblewrap) IsSandboxed() bool {
	return true
}

// Available checks if bwrap is available on the system
func (b *Bubblewrap) Available() bool {
	return commandExists("bwrap")
}

// Execute runs the script within bubblewrap sandbox
func (b *Bubblewrap) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args, err := b.buildArgs(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build bwrap args: %w", err)
	}

	cmd := exec.CommandContext(ctx, "bwrap", args...)

	// Setup I/O and environment
	stdout, stderr := SetupCommand(cmd, cfg)

	// Add NODE_PATH
	cmd.Env = BuildEnvWithNodePath(cmd.Env, cfg.NodeModules)

	// Add memory limit hint (soft limit only - bubblewrap doesn't support rlimits)
	cmd.Env = BuildEnvWithMemoryLimit(cmd.Env, cfg.MemoryMB)

	err = cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildArgs constructs bubblewrap command arguments
func (b *Bubblewrap) buildArgs(cfg *Config) ([]string, error) {
	var args []string

	// Namespace isolation
	args = append(args,
		"--unshare-user",
		"--unshare-pid",
		"--unshare-uts",
		"--unshare-cgroup",
		"--unshare-net", // Always isolate network
		"--die-with-parent",
		"--new-session",
	)

	// Create a minimal root filesystem
	args = append(args, "--tmpfs", "/")

	// Device access
	args = append(args,
		"--dev", "/dev",
		"--ro-bind", "/dev/null", "/dev/null",
		"--ro-bind", "/dev/urandom", "/dev/urandom",
		"--ro-bind", "/dev/random", "/dev/random",
	)

	// Proc filesystem
	args = append(args, "--proc", "/proc")

	// System directories (read-only)
	systemDirs := []string{
		"/usr",
		"/lib",
		"/lib64",
		"/bin",
		"/sbin",
		"/etc/alternatives",
		"/etc/ld.so.cache",
		"/etc/ld.so.conf",
		"/etc/ld.so.conf.d",
	}

	for _, dir := range systemDirs {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}

	// Timezone data
	timezoneDirs := []string{
		"/usr/share/zoneinfo",
		"/etc/localtime",
	}
	for _, path := range timezoneDirs {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--ro-bind", path, path)
		}
	}

	// DNS resolution (if network enabled via proxy)
	if cfg.Network {
		dnsFiles := []string{
			"/etc/resolv.conf",
			"/etc/hosts",
			"/etc/services",
			"/etc/nsswitch.conf",
		}
		for _, path := range dnsFiles {
			if _, err := os.Stat(path); err == nil {
				args = append(args, "--ro-bind", path, path)
			}
		}

		// SSL certificates
		certDirs := []string{
			"/etc/ssl",
			"/etc/pki",
			"/etc/ca-certificates",
			"/usr/share/ca-certificates",
		}
		for _, dir := range certDirs {
			if _, err := os.Stat(dir); err == nil {
				args = append(args, "--ro-bind", dir, dir)
			}
		}
	}

	// Bun binary
	bunPath, err := ResolvePath(cfg.BunBinary)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve bun path: %w", err)
	}
	// Bind the bun binary directory
	bunDir := filepath.Dir(bunPath)
	args = append(args, "--ro-bind", bunDir, bunDir)

	// Script file
	scriptPath, err := ResolvePath(cfg.ScriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script path: %w", err)
	}
	// Bind the script directory
	scriptDir := filepath.Dir(scriptPath)
	args = append(args, "--ro-bind", scriptDir, scriptDir)

	// Working directory (set CWD but don't mount - requires explicit --allow-write)
	if cfg.WorkDir != "" {
		workDir, err := ResolvePath(cfg.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve work dir: %w", err)
		}
		args = append(args, "--chdir", workDir)
	}

	// Node modules
	if cfg.NodeModules != "" {
		nodeModulesPath, err := ResolvePath(cfg.NodeModules)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve node_modules: %w", err)
		}
		// Bind the deps directory (parent of node_modules)
		depsDir := filepath.Dir(nodeModulesPath)
		args = append(args, "--ro-bind", depsDir, depsDir)
	}

	// Additional readable paths
	for _, path := range cfg.ReadablePaths {
		resolved, err := ResolvePath(path)
		if err != nil {
			continue
		}
		args = append(args, "--ro-bind", resolved, resolved)
	}

	// Additional writable paths
	for _, path := range cfg.WritablePaths {
		resolved, err := ResolvePath(path)
		if err != nil {
			// Create the path if it doesn't exist
			if err := os.MkdirAll(path, 0755); err != nil {
				continue
			}
			resolved = path
		}
		args = append(args, "--bind", resolved, resolved)
	}

	// Temp directory (isolated tmpfs - no host access)
	args = append(args, "--tmpfs", "/tmp")

	// Proxy socket mount (if using proxy)
	if cfg.Network && cfg.ProxySocketPath != "" {
		// Mount proxy socket
		args = append(args, "--ro-bind", cfg.ProxySocketPath, "/tmp/proxy.sock")
	}

	// Add the command to run
	// If we need network through proxy, wrap with socat bridge
	if cfg.Network && cfg.ProxySocketPath != "" {
		// Create a shell script that sets up socat and runs bun
		bunCmd := BuildBunCommand(cfg)
		script := BuildSocatBridgeCommand("/tmp/proxy.sock", bunCmd)
		args = append(args, "/bin/sh", "-c", script)
	} else {
		bunArgs := BuildBunArgs(cfg)
		args = append(args, bunArgs...)
	}

	return args, nil
}
