package main

import (
	"log"
	"os"

	"github.com/liliang-cn/rago/cmd/rago"
)

// version is set during build time
var version = "dev"

func main() {
	// Set version for CLI
	rago.SetVersion(version)

	if err := rago.Execute(); err != nil {
		log.Printf("Error executing command: %v", err)
		os.Exit(1)
	}
}
