package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/eddmann/buns/internal/cache"
	"github.com/eddmann/buns/internal/exec"
	"github.com/eddmann/buns/internal/sandbox"
	"github.com/spf13/cobra"
)

var (
	bunVersion  string
	packagesArg string

	// Sandbox flags
	sandboxEnabled bool
	offline        bool
	allowHostsArg  string
	allowReadArg   string
	allowWriteArg  string
	allowEnvArg    string
	memoryLimit    int
	timeoutSecs    int
	cpuLimit       int
)

var runCmd = &cobra.Command{
	Use:   "run <script.ts> [-- args...]",
	Short: "Run a TypeScript/JavaScript script with inline dependencies",
	Long: `Run a TypeScript/JavaScript script, installing any declared dependencies first.

Example:
  buns run script.ts
  buns run script.ts -- --flag value
  buns run --packages=zod@^3.0 script.ts
  echo 'console.log("hi")' | buns run -

Sandbox examples:
  buns run --sandbox script.ts                    # Full isolation
  buns run --offline script.ts                    # Block all network
  buns run --allow-host api.github.com script.ts  # Allow specific hosts
  buns run --sandbox --allow-read /data script.ts # With filesystem access`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScript(args[0], args[1:])
	},
}

func init() {
	addRunFlags(runCmd)
	rootCmd.AddCommand(runCmd)
}

// addRunFlags registers script execution flags on a command.
// Called on both rootCmd and runCmd so flags work with both
// `buns script.ts --sandbox` and `buns run script.ts --sandbox`.
func addRunFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&bunVersion, "bun", "", "Bun version constraint (overrides script)")
	cmd.Flags().StringVar(&packagesArg, "packages", "", "Comma-separated packages to add")

	// Sandbox flags
	cmd.Flags().BoolVar(&sandboxEnabled, "sandbox", false, "Enable full sandboxing (filesystem + process isolation)")
	cmd.Flags().BoolVar(&offline, "offline", false, "Block all network access")
	cmd.Flags().StringVar(&allowHostsArg, "allow-host", "", "Allow network to specific hosts (comma-separated, supports wildcards like *.github.com)")
	cmd.Flags().StringVar(&allowReadArg, "allow-read", "", "Allow reading additional paths (comma-separated)")
	cmd.Flags().StringVar(&allowWriteArg, "allow-write", "", "Allow writing to additional paths (comma-separated)")
	cmd.Flags().StringVar(&allowEnvArg, "allow-env", "", "Pass through environment variables (comma-separated)")
	cmd.Flags().IntVar(&memoryLimit, "memory", 128, "Memory limit in MB")
	cmd.Flags().IntVar(&timeoutSecs, "timeout", 30, "Execution timeout in seconds")

	// CPU limit only available on Linux (requires nsjail for enforcement)
	if runtime.GOOS == "linux" {
		cmd.Flags().IntVar(&cpuLimit, "cpu", 30, "CPU time limit in seconds (Linux only)")
	}
}

// runScript executes a script with its dependencies
func runScript(script string, args []string) error {
	// Get cache
	c, err := cache.Default()
	if err != nil {
		return err
	}

	// Ensure cache directories exist
	if err := c.EnsureDirs(); err != nil {
		return err
	}

	// Parse extra packages from CLI
	var extraPackages []string
	if packagesArg != "" {
		extraPackages = strings.Split(packagesArg, ",")
		for i, p := range extraPackages {
			extraPackages[i] = strings.TrimSpace(p)
		}
	}

	// Parse sandbox options
	var allowHosts, allowRead, allowWrite, allowEnv []string

	if allowHostsArg != "" {
		allowHosts = splitAndTrim(allowHostsArg)
	}
	if allowReadArg != "" {
		allowRead = splitAndTrim(allowReadArg)
	}
	if allowWriteArg != "" {
		allowWrite = splitAndTrim(allowWriteArg)
	}
	if allowEnvArg != "" {
		allowEnv = splitAndTrim(allowEnvArg)
	}

	// Determine sandbox
	var sb sandbox.Sandbox = &sandbox.None{}
	if sandboxEnabled {
		sb = sandbox.Detect(true)
		if !sb.IsSandboxed() {
			return fmt.Errorf("--sandbox requested but no sandbox is available on this system")
		}
	} else if offline || len(allowHosts) > 0 {
		sb = sandbox.Detect(false)
		if !sb.IsSandboxed() {
			return fmt.Errorf("--offline/--allow-host requires network sandboxing, but no sandbox is available on this system")
		}
	}

	// Determine network access
	network := !offline

	// Create runner
	runner := exec.NewRunner(c, verbose, quiet)

	// Run the script
	exitCode, err := runner.Run(exec.RunOptions{
		Script:        script,
		Args:          args,
		BunConstraint: bunVersion,
		ExtraPackages: extraPackages,

		// Sandbox options
		Sandbox:     sb,
		Network:     network,
		AllowHosts:  allowHosts,
		AllowRead:   allowRead,
		AllowWrite:  allowWrite,
		AllowEnv:    allowEnv,
		MemoryMB:    memoryLimit,
		TimeoutSecs: timeoutSecs,
		CPUSeconds:  cpuLimit,
	})

	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

// splitAndTrim splits a comma-separated string and trims whitespace
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
