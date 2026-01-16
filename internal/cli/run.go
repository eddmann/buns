package cli

import (
	"os"
	"strings"

	"github.com/edwardsmale/buns/internal/cache"
	"github.com/edwardsmale/buns/internal/exec"
	"github.com/spf13/cobra"
)

var (
	bunVersion  string
	packagesArg string
)

var runCmd = &cobra.Command{
	Use:   "run <script.ts> [-- args...]",
	Short: "Run a TypeScript/JavaScript script with inline dependencies",
	Long: `Run a TypeScript/JavaScript script, installing any declared dependencies first.

Example:
  buns run script.ts
  buns run script.ts -- --flag value
  buns run --packages=zod@^3.0 script.ts
  echo 'console.log("hi")' | buns run -`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScript(args[0], args[1:])
	},
}

func init() {
	runCmd.Flags().StringVar(&bunVersion, "bun", "", "Bun version constraint (overrides script)")
	runCmd.Flags().StringVar(&packagesArg, "packages", "", "Comma-separated packages to add")
	rootCmd.AddCommand(runCmd)
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

	// Create runner
	runner := exec.NewRunner(c, verbose, quiet)

	// Run the script
	exitCode, err := runner.Run(exec.RunOptions{
		Script:        script,
		Args:          args,
		BunConstraint: bunVersion,
		ExtraPackages: extraPackages,
	})

	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
