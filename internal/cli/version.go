package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version string",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("buns %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
