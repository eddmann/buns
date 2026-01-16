package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/edwardsmale/buns/internal/cache"
	"github.com/spf13/cobra"
)

var (
	cleanBun   bool
	cleanDeps  bool
	cleanIndex bool
	cleanAll   bool
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the buns cache",
	Long:  `Manage cached Bun binaries, dependencies, and index data.`,
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show cached Bun builds and dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.Default()
		if err != nil {
			return err
		}

		// List Bun versions
		versions, err := c.ListBunVersions()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		fmt.Println("Bun binaries:")
		if len(versions) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, v := range versions {
				fmt.Printf("  %s\n", v)
			}
		}

		// List deps
		hashes, err := c.ListDepsHashes()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		fmt.Println("\nDependency caches:")
		if len(hashes) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, h := range hashes {
				// Truncate hash for display
				display := h
				if len(h) > 12 {
					display = h[:12] + "..."
				}
				fmt.Printf("  %s\n", display)
			}
		}

		// Index age
		fmt.Println("\nIndex cache:")
		indexTime := filepath.Join(c.IndexDir(), "fetched_at")
		if data, err := os.ReadFile(indexTime); err == nil {
			if t, err := time.Parse(time.RFC3339, string(data)); err == nil {
				age := time.Since(t).Round(time.Minute)
				fmt.Printf("  Last updated: %s ago\n", age)
			}
		} else {
			fmt.Println("  (not cached)")
		}

		// Total size
		size, err := c.Size()
		if err != nil {
			return err
		}
		fmt.Printf("\nTotal cache size: %s\n", formatSize(size))

		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove cached data",
	Long: `Remove cached data. By default, removes dependency caches.

Use flags to specify what to clean:
  --bun    Remove Bun binaries
  --deps   Remove dependencies (default)
  --index  Remove index cache
  --all    Remove everything`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.Default()
		if err != nil {
			return err
		}

		// Default to cleaning deps if no flags specified
		if !cleanBun && !cleanDeps && !cleanIndex && !cleanAll {
			cleanDeps = true
		}

		if cleanAll {
			fmt.Println("Removing all cache data...")
			if err := c.CleanAll(); err != nil {
				return err
			}
			fmt.Println("Done.")
			return nil
		}

		if cleanBun {
			fmt.Println("Removing Bun binaries...")
			if err := c.CleanBun(); err != nil {
				return err
			}
		}

		if cleanDeps {
			fmt.Println("Removing dependency caches...")
			if err := c.CleanDeps(); err != nil {
				return err
			}
		}

		if cleanIndex {
			fmt.Println("Removing index cache...")
			if err := c.CleanIndex(); err != nil {
				return err
			}
		}

		fmt.Println("Done.")
		return nil
	},
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print cache directory path",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.Default()
		if err != nil {
			return err
		}
		fmt.Println(c.BaseDir())
		return nil
	},
}

func init() {
	cacheCleanCmd.Flags().BoolVar(&cleanBun, "bun", false, "Remove Bun binaries")
	cacheCleanCmd.Flags().BoolVar(&cleanDeps, "deps", false, "Remove dependencies")
	cacheCleanCmd.Flags().BoolVar(&cleanIndex, "index", false, "Remove index cache")
	cacheCleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove everything")

	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheDirCmd)
	rootCmd.AddCommand(cacheCmd)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
