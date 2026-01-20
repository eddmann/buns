package sandbox

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// None is a no-op sandbox that provides no isolation
// It is used as a fallback when no sandbox is available or requested
type None struct{}

// Name returns the sandbox name
func (n *None) Name() string {
	return "none"
}

// IsSandboxed returns false since this sandbox provides no isolation
func (n *None) IsSandboxed() bool {
	return false
}

// Available always returns true since no external tools are required
func (n *None) Available() bool {
	return true
}

// Execute runs the script without any sandbox isolation
// No environment filtering - inherits full environment from parent
func (n *None) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args := BuildBunArgs(cfg)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Setup I/O only (no env filtering for none sandbox)
	var stdout, stderr bytes.Buffer
	if cfg.Stdin != nil {
		cmd.Stdin = cfg.Stdin
	}
	if cfg.Stdout != nil {
		cmd.Stdout = cfg.Stdout
	} else {
		cmd.Stdout = &stdout
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	// Inherit full environment, just add NODE_PATH and memory hint
	cmd.Env = os.Environ()
	if cfg.NodeModules != "" {
		cmd.Env = append(cmd.Env, "NODE_PATH="+cfg.NodeModules)
	}
	cmd.Env = BuildEnvWithMemoryLimit(cmd.Env, cfg.MemoryMB)

	cmd.Dir = cfg.WorkDir

	err := cmd.Run()
	return BuildResult(err, cfg, &stdout, &stderr)
}
