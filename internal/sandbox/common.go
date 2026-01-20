package sandbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SafeEnvVars is a whitelist of environment variables considered safe to pass through
var SafeEnvVars = map[string]bool{
	"PATH":    true,
	"HOME":    true,
	"USER":    true,
	"SHELL":   true,
	"LANG":    true,
	"TERM":    true,
	"TZ":      true,
	"TMPDIR":  true,
	"TEMP":    true,
	"TMP":     true,
	"LOGNAME": true,
	"EDITOR":  true,
	"VISUAL":  true,
	"PAGER":   true,
}

// SafeEnvPrefixes are prefixes for environment variables considered safe
var SafeEnvPrefixes = []string{"LC_", "XDG_"}

// FilterEnv creates a filtered environment from the current environment
// It includes only safe vars and explicitly allowed vars
func FilterEnv(allowed []string) []string {
	allowedSet := make(map[string]bool)
	for _, v := range allowed {
		allowedSet[v] = true
	}

	var filtered []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]

		// Check if explicitly allowed
		if allowedSet[name] {
			filtered = append(filtered, env)
			continue
		}

		// Check if in safe list
		if SafeEnvVars[name] {
			filtered = append(filtered, env)
			continue
		}

		// Check if has safe prefix
		for _, prefix := range SafeEnvPrefixes {
			if strings.HasPrefix(name, prefix) {
				filtered = append(filtered, env)
				break
			}
		}
	}

	return filtered
}

// ShellEscape escapes a string for safe use in shell commands.
// Wraps in single quotes and escapes embedded single quotes.
func ShellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildBunArgs constructs the bun command arguments
func BuildBunArgs(cfg *Config) []string {
	args := []string{cfg.BunBinary, "run", cfg.ScriptPath}
	args = append(args, cfg.ScriptArgs...)
	return args
}

// BuildBunCommand constructs an escaped bun command string from config.
func BuildBunCommand(cfg *Config) string {
	bunArgs := BuildBunArgs(cfg)
	bunCmd := ShellEscape(cfg.BunBinary)
	for _, arg := range bunArgs[1:] {
		bunCmd += " " + ShellEscape(arg)
	}
	return bunCmd
}

// ProxyEnvVars returns environment variables for the sandbox bridge proxy.
func ProxyEnvVars() []string {
	port := strconv.Itoa(SandboxBridgePort)
	return []string{
		"HTTP_PROXY=http://127.0.0.1:" + port,
		"HTTPS_PROXY=http://127.0.0.1:" + port,
		"http_proxy=http://127.0.0.1:" + port,
		"https_proxy=http://127.0.0.1:" + port,
		"ALL_PROXY=http://127.0.0.1:" + port,
	}
}

// BuildSocatBridgeCommand creates a shell command that starts socat to bridge
// localhost traffic to a Unix socket, then runs the given command.
// Uses a retry loop instead of sleep to avoid race conditions.
func BuildSocatBridgeCommand(socketPath string, bunCmd string) string {
	return fmt.Sprintf(
		`socat TCP-LISTEN:%d,fork,reuseaddr UNIX-CONNECT:%s &
SOCAT_PID=$!
for i in 1 2 3 4 5 6 7 8 9 10; do
  if nc -z 127.0.0.1 %d 2>/dev/null; then break; fi
  sleep 0.05
done
%s
EXIT_CODE=$?
kill $SOCAT_PID 2>/dev/null
exit $EXIT_CODE`,
		SandboxBridgePort,
		ShellEscape(socketPath),
		SandboxBridgePort,
		bunCmd,
	)
}

// SetupCommand configures a command with standard I/O handling and environment.
// Returns buffers for stdout/stderr if streaming is not configured.
func SetupCommand(cmd *exec.Cmd, cfg *Config) (*bytes.Buffer, *bytes.Buffer) {
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

	// Build environment - use filtered safelist plus explicitly allowed vars
	env := FilterEnv(cfg.AllowedEnvVars)
	env = append(env, cfg.Env...)
	cmd.Env = env

	return &stdout, &stderr
}

// BuildResult creates a Result from command execution, extracting exit code and output.
func BuildResult(err error, cfg *Config, stdout, stderr *bytes.Buffer) (*Result, error) {
	result := &Result{}

	if cfg.Stdout == nil && stdout != nil {
		result.Stdout = stdout.String()
	}
	if cfg.Stderr == nil && stderr != nil {
		result.Stderr = stderr.String()
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		return result, err
	}

	return result, nil
}

// ResolvePath resolves symlinks and returns the real path
// This is important for Seatbelt profiles which require real paths
func ResolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet, return the absolute path
		if os.IsNotExist(err) {
			return absPath, nil
		}
		return "", err
	}

	return realPath, nil
}

// SeatbeltEscape escapes a string for use in Seatbelt profiles
func SeatbeltEscape(s string) string {
	// Escape backslashes and double quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// BuildEnvWithNodePath creates environment with NODE_PATH set
func BuildEnvWithNodePath(baseEnv []string, nodeModules string) []string {
	if nodeModules == "" {
		return baseEnv
	}
	return append(baseEnv, "NODE_PATH="+nodeModules)
}

// BuildEnvWithMemoryLimit adds BUN_JSC_forceRAMSize to hint memory limits to Bun.
// This is a soft limit - it makes Bun's GC more aggressive but is NOT enforced.
// Only nsjail on Linux provides hard memory limits via rlimit.
func BuildEnvWithMemoryLimit(baseEnv []string, memoryMB int) []string {
	if memoryMB <= 0 {
		return baseEnv
	}
	// BUN_JSC_forceRAMSize is in bytes
	bytes := memoryMB * 1024 * 1024
	return append(baseEnv, fmt.Sprintf("BUN_JSC_forceRAMSize=%d", bytes))
}
