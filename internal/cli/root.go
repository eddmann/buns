package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

const logo = `
 ██████╗ ██╗   ██╗███╗   ██╗███████╗
 ██╔══██╗██║   ██║████╗  ██║██╔════╝
 ██████╔╝██║   ██║██╔██╗ ██║███████╗
 ██╔══██╗██║   ██║██║╚██╗██║╚════██║
 ██████╔╝╚██████╔╝██║ ╚████║███████║
 ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚══════╝
`

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:     "buns [script.ts]",
	Short:   "Run TypeScript/JavaScript scripts with inline dependencies",
	Version: Version,
	Long: `buns runs TypeScript/JavaScript scripts with inline npm dependencies
and automatic Bun version management.

Example:
  buns script.ts
  buns run script.ts --packages=zod@^3.0
  echo 'console.log("hi")' | buns run -`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		// Default behavior: buns script.ts → buns run script.ts
		return runScript(args[0], args[1:])
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress buns output")

	// Register script execution flags on root command too
	addRunFlags(rootCmd)

	rootCmd.SetVersionTemplate(fmt.Sprintf("buns %s (commit: %s, built: %s)\n", Version, GitCommit, BuildTime))
	rootCmd.SetHelpTemplate(logo + `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
}

func Execute() error {
	return rootCmd.Execute()
}
