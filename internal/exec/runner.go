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
	Script        string
	Args          []string
	BunConstraint string   // Override bun version from CLI
	ExtraPackages []string // Additional packages from CLI
	TypeCheck     bool     // Run TypeScript type checking before execution

	// Sandbox options
	Sandbox     sandbox.Sandbox // Sandbox instance (set by CLI)
	Network     bool            // Whether network is enabled
	AllowHosts  []string        // Allowed hosts for network access
	AllowRead   []string        // Additional readable paths
	AllowWrite  []string        // Additional writable paths
	AllowEnv    []string        // Environment variables to pass through
	MemoryMB    int             // Memory limit in MB
	TimeoutSecs int             // Execution timeout in seconds
	CPUSeconds  int             // CPU time limit in seconds
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

	if opts.TypeCheck {
		exitCode, err := r.typeCheckScript(bunPath, scriptPath, packages, version.Original())
		if err != nil {
			return 1, err
		}
		if exitCode != 0 {
			return exitCode, nil
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

const (
	typeScriptPackage = "typescript@^5.9"
	bunTypesPackage   = "@types/bun"
)

type typeCheckConfig struct {
	CompilerOptions typeCheckCompilerOptions `json:"compilerOptions"`
	Files           []string                 `json:"files"`
}

type typeCheckCompilerOptions struct {
	Lib                        []string            `json:"lib"`
	Target                     string              `json:"target"`
	Module                     string              `json:"module"`
	ModuleDetection            string              `json:"moduleDetection"`
	ModuleResolution           string              `json:"moduleResolution"`
	AllowJS                    bool                `json:"allowJs"`
	AllowImportingTSExtensions bool                `json:"allowImportingTsExtensions"`
	VerbatimModuleSyntax       bool                `json:"verbatimModuleSyntax"`
	NoEmit                     bool                `json:"noEmit"`
	Strict                     bool                `json:"strict"`
	SkipLibCheck               bool                `json:"skipLibCheck"`
	Types                      []string            `json:"types"`
	TypeRoots                  []string            `json:"typeRoots"`
	BaseURL                    string              `json:"baseUrl"`
	Paths                      map[string][]string `json:"paths"`
}

func (r *Runner) typeCheckScript(bunPath, scriptPath string, packages []string, bunVersion string) (int, error) {
	r.log("Typechecking script...")

	typeCheckPackages := buildTypeCheckPackages(packages, bunVersion, true)
	typeCheckDir, err := r.ensureTypeCheckDeps(bunPath, typeCheckPackages)
	if err != nil {
		r.log("Warning: failed to install %s@%s, falling back to latest %s: %v", bunTypesPackage, bunVersion, bunTypesPackage, err)

		fallbackPackages := buildTypeCheckPackages(packages, bunVersion, false)
		var fallbackErr error
		typeCheckDir, fallbackErr = r.ensureTypeCheckDeps(bunPath, fallbackPackages)
		if fallbackErr != nil {
			return 1, fmt.Errorf("failed to install typecheck dependencies: %v; fallback failed: %w", err, fallbackErr)
		}
	}

	configPath, err := writeTypeCheckConfig(scriptPath, typeCheckDir)
	if err != nil {
		return 1, err
	}
	defer func() { _ = os.Remove(configPath) }()

	exitCode, err := r.runTypeCheck(bunPath, typeCheckDir, configPath)
	if err != nil {
		return 1, err
	}

	if exitCode == 0 {
		r.log("Typecheck passed")
	} else if !r.quiet {
		fmt.Fprintln(os.Stderr, "[buns] Typecheck failed; script was not executed.")
	}

	return exitCode, nil
}

func buildTypeCheckPackages(packages []string, bunVersion string, pinBunTypes bool) []string {
	result := make([]string, 0, len(packages)+2)
	result = append(result, packages...)
	result = append(result, typeScriptPackage)

	if pinBunTypes && bunVersion != "" {
		result = append(result, fmt.Sprintf("%s@%s", bunTypesPackage, bunVersion))
	} else {
		result = append(result, bunTypesPackage)
	}

	return result
}

func (r *Runner) ensureTypeCheckDeps(bunPath string, packages []string) (string, error) {
	hash := cache.HashPackages(packages)
	typeCheckDir := r.cache.TypecheckDirForHash(hash)

	r.log("Typecheck dependencies hash: %s", hash[:12]+"...")

	if cache.IsPackageInstallHit(typeCheckDir) {
		r.log("Typecheck cache hit: %s", typeCheckDir)
		return typeCheckDir, nil
	}

	r.log("Typecheck cache miss: %s", typeCheckDir)
	if err := os.RemoveAll(typeCheckDir); err != nil {
		return "", err
	}
	if err := r.installDeps(bunPath, typeCheckDir, packages); err != nil {
		_ = os.RemoveAll(typeCheckDir)
		return "", err
	}

	r.log("Typecheck dependencies installed")
	return typeCheckDir, nil
}

func buildTypeCheckConfig(scriptPath, typeCheckDir string) typeCheckConfig {
	nodeModules := filepath.Join(typeCheckDir, "node_modules")

	return typeCheckConfig{
		CompilerOptions: typeCheckCompilerOptions{
			Lib:                        []string{"ESNext"},
			Target:                     "ESNext",
			Module:                     "Preserve",
			ModuleDetection:            "force",
			ModuleResolution:           "bundler",
			AllowJS:                    true,
			AllowImportingTSExtensions: true,
			VerbatimModuleSyntax:       true,
			NoEmit:                     true,
			Strict:                     true,
			SkipLibCheck:               true,
			Types:                      []string{"bun"},
			TypeRoots:                  []string{filepath.Join(nodeModules, "@types")},
			BaseURL:                    typeCheckDir,
			Paths: map[string][]string{
				"*": []string{"node_modules/*"},
			},
		},
		Files: []string{scriptPath},
	}
}

func writeTypeCheckConfig(scriptPath, typeCheckDir string) (string, error) {
	config := buildTypeCheckConfig(scriptPath, typeCheckDir)

	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	configFile, err := os.CreateTemp("", "buns-tsconfig-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create typecheck config: %w", err)
	}
	defer func() { _ = configFile.Close() }()

	if _, err := configFile.Write(configBytes); err != nil {
		_ = os.Remove(configFile.Name())
		return "", fmt.Errorf("failed to write typecheck config: %w", err)
	}

	return configFile.Name(), nil
}

func (r *Runner) runTypeCheck(bunPath, typeCheckDir, configPath string) (int, error) {
	tscPath := filepath.Join(typeCheckDir, "node_modules", "typescript", "lib", "tsc.js")
	cmd := exec.Command(bunPath, tscPath, "--project", configPath, "--pretty", "--noErrorTruncation")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	return 0, nil
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
