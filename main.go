package main

import (
	"os"

	"github.com/kartoza/kartoza-pg-ai/cmd"
)

// version is set during build via ldflags
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
