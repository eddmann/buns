package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eddmann/buns/internal/bun"
	"github.com/eddmann/buns/internal/cache"
	"github.com/eddmann/buns/internal/index"
	"github.com/eddmann/buns/internal/metadata"
	"github.com/eddmann/buns/internal/proxy"
	"github.com/eddmann/buns/internal/sandbox"
)

// Runner executes scripts with their dependencies
type Runner struct {
	cache    *cache.Cache
	index    *index.Index
	resolver *bun.Resolver
	verbose  bool
	quiet    bool
}

// NewRunner creates a new script runner
func NewRunner(c *cache.Cache, verbose, quiet bool) *Runner {
	idx := index.New(c.IndexDir())
	return &Runner{
		cache:    c,
		index:    idx,
		resolver: bun.NewResolver(idx),
		verbose:  verbose,
		quiet:    quiet,
	}
}

// RunOptions contains options for running a script
type RunOptions struct {
	Script         string
	Args           []string
	BunConstraint  string   // Override bun version from CLI
	ExtraPackages  []string // Additional packages from CLI

	// Sandbox options
	Sandbox        sandbox.Sandbox // Sandbox instance (set by CLI)
	Network        bool            // Whether network is enabled
	AllowHosts     []string        // Allowed hosts for network access
	AllowRead      []string        // Additional readable paths
	AllowWrite     []string        // Additional writable paths
	AllowEnv       []string        // Environment variables to pass through
	MemoryMB       int             // Memory limit in MB
	TimeoutSecs    int             // Execution timeout in seconds
	CPUSeconds     int             // CPU time limit in seconds
}

// Run executes a script with its dependencies
func (r *Runner) Run(opts RunOptions) (int, error) {
	// Read script content
	var content []byte
	var err error
	var scriptPath string

	if opts.Script == "-" {
		// Read from stdin
		content, err = os.ReadFile("/dev/stdin")
		if err != nil {
			return 1, fmt.Errorf("failed to read stdin: %w", err)
		}
		// Write to temp file
		tmpFile, err := os.CreateTemp("", "buns-*.ts")
		if err != nil {
			return 1, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		if _, err := tmpFile.Write(content); err != nil {
			return 1, fmt.Errorf("failed to write temp file: %w", err)
		}
		_ = tmpFile.Close()
		scriptPath = tmpFile.Name()
	} else {
		scriptPath, err = filepath.Abs(opts.Script)
		if err != nil {
			return 1, fmt.Errorf("failed to resolve script path: %w", err)
		}
		content, err = os.ReadFile(scriptPath)
		if err != nil {
			return 1, fmt.Errorf("script not found: %s", opts.Script)
		}
	}

	r.log("Parsing script metadata...")

	// Parse metadata
	meta, err := metadata.Parse(content)
	if err != nil {
		return 1, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Merge packages
	packages := meta.Packages
	if len(opts.ExtraPackages) > 0 {
		packages = append(packages, opts.ExtraPackages...)
	}

	// Resolve bun version
	bunConstraint := opts.BunConstraint
	if bunConstraint == "" {
		bunConstraint = meta.Bun
	}

	if r.verbose && (bunConstraint != "" || len(packages) > 0) {
		r.log("Found: bun=%q, packages=%v", bunConstraint, packages)
	}

	r.log("Resolving Bun version for constraint '%s'", bunConstraint)

	version, err := r.resolver.Resolve(bunConstraint)
	if err != nil {
		return 1, fmt.Errorf("no Bun version satisfies '%s'", bunConstraint)
	}

	r.log("Matched: %s", version.Original())

	// Get bun binary
	downloader := bun.NewDownloader(r.cache.BunDir(), r.verbose, r.quiet)
	bunPath, err := downloader.GetBinary(version)
	if err != nil {
		return 1, fmt.Errorf("failed to download Bun: %w", err)
	}

	r.log("Bun binary: %s", bunPath)

	// Handle dependencies
	var depsDir string
	if len(packages) > 0 {
		hash := cache.HashPackages(packages)
		depsDir = r.cache.DepsDirForHash(hash)

		r.log("Dependencies hash: %s", hash[:12]+"...")

		if r.cache.IsDepsHit(hash) {
			r.log("Cache hit: %s", depsDir)
		} else {
			r.log("Cache miss: %s", depsDir)
			if err := r.installDeps(bunPath, depsDir, packages); err != nil {
				return 1, fmt.Errorf("failed to install dependencies: %w", err)
			}
			r.log("Dependencies installed")
		}
	}

	// If sandbox is set and provides isolation, use sandboxed execution
	if opts.Sandbox != nil && opts.Sandbox.IsSandboxed() {
		return r.execScriptSandboxed(bunPath, scriptPath, opts, depsDir)
	}

	// Execute script normally
	r.log("Executing: %s run %s", bunPath, scriptPath)
	return r.execScript(bunPath, scriptPath, opts.Args, depsDir)
}

// execScriptSandboxed runs the script in a sandbox
func (r *Runner) execScriptSandboxed(bunPath, scriptPath string, opts RunOptions, depsDir string) (int, error) {
	sb := opts.Sandbox

	// Start proxy if network is needed and we're sandboxing
	var proxyMgr *proxy.Manager
	var proxyEnv []string
	var proxySocketPath string
	var proxyPort int
	var proxySOCKS5Port int

	needsProxy := sb.IsSandboxed() && opts.Network

	if needsProxy {
		r.log("Starting proxy server...")
		var err error
		proxyMgr, err = proxy.NewManager(proxy.ManagerConfig{
			AllowedHosts: opts.AllowHosts,
			Verbose:      r.verbose,
		})
		if err != nil {
			return 1, fmt.Errorf("failed to start proxy: %w", err)
		}
		defer proxyMgr.Stop()

		proxyEnv = proxyMgr.EnvVars()
		proxySocketPath = proxyMgr.SocketPath()
		proxyPort = proxyMgr.Port()
		proxySOCKS5Port = proxyMgr.SOCKS5Port()

		r.log("Proxy started on port %d", proxyPort)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = filepath.Dir(scriptPath)
	}

	// Build node_modules path
	var nodeModules string
	if depsDir != "" {
		nodeModules = filepath.Join(depsDir, "node_modules")
	}

	// Build sandbox config
	cfg := &sandbox.Config{
		Network:         opts.Network,
		AllowedHosts:    opts.AllowHosts,
		ProxySocketPath: proxySocketPath,
		ProxyPort:       proxyPort,
		ProxySOCKS5Port: proxySOCKS5Port,

		ReadablePaths: opts.AllowRead,
		WritablePaths: opts.AllowWrite,
		WorkDir:       workDir,

		MemoryMB:   opts.MemoryMB,
		Timeout:    time.Duration(opts.TimeoutSecs) * time.Second,
		CPUSeconds: opts.CPUSeconds,

		BunBinary:   bunPath,
		ScriptPath:  scriptPath,
		ScriptArgs:  opts.Args,
		NodeModules: nodeModules,

		Env:            proxyEnv,
		AllowedEnvVars: opts.AllowEnv,

		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Verbose: r.verbose,
	}

	r.log("Using sandbox: %s", sb.Name())

	// Create context with timeout
	ctx := context.Background()
	if opts.TimeoutSecs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.TimeoutSecs)*time.Second)
		defer cancel()
	}

	r.log("Executing sandboxed: %s run %s", bunPath, scriptPath)

	result, err := sb.Execute(ctx, cfg)
	if err != nil {
		return 1, fmt.Errorf("execution failed: %w", err)
	}

	r.log("Exit code: %d", result.ExitCode)
	return result.ExitCode, nil
}

// installDeps installs packages to the deps directory
func (r *Runner) installDeps(bunPath, depsDir string, packages []string) error {
	if err := os.MkdirAll(depsDir, 0755); err != nil {
		return err
	}

	// Generate package.json
	deps := make(map[string]string)
	for _, pkg := range packages {
		name, version := parsePackageSpec(pkg)
		if version == "" {
			version = "*"
		}
		deps[name] = version
	}

	pkgJSON := map[string]interface{}{
		"name":         "buns-deps",
		"private":      true,
		"dependencies": deps,
	}

	pkgJSONBytes, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		return err
	}

	pkgJSONPath := filepath.Join(depsDir, "package.json")
	if err := os.WriteFile(pkgJSONPath, pkgJSONBytes, 0644); err != nil {
		return err
	}

	// Run bun install
	cmd := exec.Command(bunPath, "install")
	cmd.Dir = depsDir
	if !r.quiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// execScript runs the script with the bun binary (non-sandboxed)
func (r *Runner) execScript(bunPath, scriptPath string, args []string, depsDir string) (int, error) {
	cmdArgs := []string{"run", scriptPath}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(bunPath, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set NODE_PATH if we have dependencies
	if depsDir != "" {
		nodeModules := filepath.Join(depsDir, "node_modules")
		env := os.Environ()
		env = append(env, "NODE_PATH="+nodeModules)
		cmd.Env = env
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	r.log("Exit code: 0")
	return 0, nil
}

func (r *Runner) log(format string, args ...interface{}) {
	if r.verbose {
		fmt.Printf("[buns] "+format+"\n", args...)
	}
}

// parsePackageSpec splits "name@version" into name and version
func parsePackageSpec(spec string) (name, version string) {
	// Handle scoped packages (@org/name@version)
	if strings.HasPrefix(spec, "@") {
		idx := strings.LastIndex(spec, "@")
		if idx > 0 && idx != strings.Index(spec, "@") {
			return spec[:idx], spec[idx+1:]
		}
		return spec, ""
	}

	// Regular package (name@version)
	if idx := strings.Index(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}

	return spec, ""
}
