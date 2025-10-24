package rag

import (
	"github.com/liliang-cn/rago/v2/pkg/config"
)

// Variables exported from parent package
var (
	Cfg     *config.Config
	Verbose bool
	Quiet   bool
	Version string
)

// SetSharedVariables sets the shared variables from the parent package
func SetSharedVariables(cfg *config.Config, verbose, quiet bool, version string) {
	Cfg = cfg
	Verbose = verbose
	Quiet = quiet
	Version = version
}
