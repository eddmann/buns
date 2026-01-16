package main

import (
	"os"

	"github.com/eddmann/buns/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
