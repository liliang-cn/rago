package main

import (
	"log"
	"os"

	"github.com/liliang-cn/rago/cmd/rago"
)

// version is set during build time
var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Set version for CLI
	rago.SetVersion(version)

	return rago.Execute()
}
