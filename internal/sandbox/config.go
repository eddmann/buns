package sandbox

import (
	"io"
	"time"
)

// Config holds sandbox configuration.
type Config struct {
	// Network settings
	Network         bool     // Allow network access (via proxy)
	AllowedHosts    []string // Allowed hosts for proxy
	ProxySocketPath string   // Unix socket path for proxy (Linux)
	ProxyPort       int      // TCP port for HTTP proxy (macOS/fallback)
	ProxySOCKS5Port int      // TCP port for SOCKS5 proxy

	// Filesystem settings
	ReadablePaths []string // Additional paths to allow reading
	WritablePaths []string // Additional paths to allow writing
	WorkDir       string   // Working directory

	// Resource limits
	MemoryMB   int           // Memory limit in MB
	Timeout    time.Duration // Execution timeout
	CPUSeconds int           // CPU time limit

	// Bun settings
	BunBinary   string   // Path to Bun binary
	ScriptPath  string   // Path to script to execute
	ScriptArgs  []string // Arguments to pass to script
	NodeModules string   // Path to node_modules (for NODE_PATH)

	// Environment
	Env            []string // Environment variables to pass (proxy vars, etc.)
	AllowedEnvVars []string // Additional env vars to pass from host (--allow-env flag)

	// I/O streams - if set, streams directly instead of buffering
	Stdin  io.Reader // Standard input (nil = no input)
	Stdout io.Writer // Standard output (nil = buffer to Result.Stdout)
	Stderr io.Writer // Standard error (nil = buffer to Result.Stderr)

	// Output
	Verbose bool
}

// SandboxBridgePort is the fixed port for socat bridge in isolated namespaces
const SandboxBridgePort = 19850
