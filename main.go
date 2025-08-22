package main

import (
	"log"
	"os"
	"runtime/debug"

	"github.com/liliang-cn/rago/cmd/rago"
)

// version is set during build time
var version = "dev"

func getVersion() string {
	if version != "dev" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}

		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) > 7 {
					return "dev-" + setting.Value[:7]
				}
				return "dev-" + setting.Value
			}
		}
	}

	return version
}

func main() {
	if err := run(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Set version for CLI
	rago.SetVersion(getVersion())

	return rago.Execute()
}
