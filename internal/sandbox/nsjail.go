package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Nsjail implements full sandbox using Google's nsjail
type Nsjail struct{}

// Name returns the sandbox name
func (n *Nsjail) Name() string {
	return "nsjail"
}

// IsSandboxed returns true since this provides full isolation
func (n *Nsjail) IsSandboxed() bool {
	return true
}

// Available checks if nsjail is available on the system
func (n *Nsjail) Available() bool {
	return commandExists("nsjail")
}

// Execute runs the script within nsjail sandbox
func (n *Nsjail) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args, err := n.buildArgs(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build nsjail args: %w", err)
	}

	cmd := exec.CommandContext(ctx, "nsjail", args...)

	// Setup I/O and environment
	stdout, stderr := SetupCommand(cmd, cfg)

	// Add NODE_PATH
	cmd.Env = BuildEnvWithNodePath(cmd.Env, cfg.NodeModules)

	err = cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildArgs constructs nsjail command arguments
func (n *Nsjail) buildArgs(cfg *Config) ([]string, error) {
	var args []string

	// Mode: once (run once and exit)
	args = append(args, "--mode", "o")

	// Run as nobody user (65534) for privilege dropping
	args = append(args, "--user", "65534")
	args = append(args, "--group", "65534")

	// Quiet mode (reduce nsjail output)
	args = append(args, "--quiet")

	// Resource limits
	if cfg.Timeout > 0 {
		args = append(args, "--time_limit", strconv.Itoa(int(cfg.Timeout.Seconds())))
	}
	if cfg.MemoryMB > 0 {
		args = append(args, "--rlimit_as", strconv.Itoa(cfg.MemoryMB))
	}
	if cfg.CPUSeconds > 0 {
		args = append(args, "--rlimit_cpu", strconv.Itoa(cfg.CPUSeconds))
	}

	// Additional resource limits
	args = append(args,
		"--rlimit_fsize", "50",   // Max file size 50MB
		"--rlimit_nofile", "128", // Max open files
		"--rlimit_nproc", "10",   // Max processes
	)

	// Network isolation
	if !cfg.Network {
		// Full network isolation
		args = append(args, "--clone_newnet")
	} else if cfg.ProxySocketPath != "" {
		// Need network access through proxy
		// Disable network namespace cloning to allow proxy access
		args = append(args, "--disable_clone_newnet")
	}

	// System directories (read-only)
	systemDirs := []string{
		"/usr",
		"/lib",
		"/lib64",
		"/bin",
		"/sbin",
	}

	for _, dir := range systemDirs {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "-R", dir)
		}
	}

	// Device access
	args = append(args,
		"-R", "/dev/null",
		"-R", "/dev/urandom",
		"-R", "/dev/random",
	)

	// Proc filesystem
	args = append(args, "--mount_proc")

	// Timezone data
	timezonePaths := []string{
		"/usr/share/zoneinfo",
		"/etc/localtime",
	}
	for _, path := range timezonePaths {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "-R", path)
		}
	}

	// DNS resolution (if network enabled)
	if cfg.Network {
		dnsFiles := []string{
			"/etc/resolv.conf",
			"/etc/hosts",
			"/etc/services",
			"/etc/nsswitch.conf",
		}
		for _, path := range dnsFiles {
			if _, err := os.Stat(path); err == nil {
				args = append(args, "-R", path)
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
				args = append(args, "-R", dir)
			}
		}
	}

	// Bun binary
	bunPath, err := ResolvePath(cfg.BunBinary)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve bun path: %w", err)
	}
	bunDir := filepath.Dir(bunPath)
	args = append(args, "-R", bunDir)

	// Script file
	scriptPath, err := ResolvePath(cfg.ScriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script path: %w", err)
	}
	scriptDir := filepath.Dir(scriptPath)
	args = append(args, "-R", scriptDir)

	// Working directory (set CWD but don't mount - requires explicit --allow-write)
	if cfg.WorkDir != "" {
		workDir, err := ResolvePath(cfg.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve work dir: %w", err)
		}
		args = append(args, "--cwd", workDir)
	}

	// Node modules
	if cfg.NodeModules != "" {
		nodeModulesPath, err := ResolvePath(cfg.NodeModules)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve node_modules: %w", err)
		}
		depsDir := filepath.Dir(nodeModulesPath)
		args = append(args, "-R", depsDir)
	}

	// Additional readable paths
	for _, path := range cfg.ReadablePaths {
		resolved, err := ResolvePath(path)
		if err != nil {
			continue
		}
		args = append(args, "-R", resolved)
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
		args = append(args, "-B", resolved)
	}

	// Temp directory (isolated tmpfs - no host access)
	args = append(args, "--tmpfsmount", "/tmp")

	// Pass environment variables
	env := FilterEnv(cfg.AllowedEnvVars)
	env = append(env, cfg.Env...)
	env = BuildEnvWithNodePath(env, cfg.NodeModules)
	for _, e := range env {
		args = append(args, "-E", e)
	}

	// Add the command to run
	args = append(args, "--")
	args = append(args, BuildBunArgs(cfg)...)

	return args, nil
}
