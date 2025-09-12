package rag

import (
	"context"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// Variables exported from parent package
var (
	Cfg     *config.Config
	Verbose bool
	Quiet   bool
	Version string
)

// InitializeProviders is a wrapper for the shared provider initialization
func InitializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	return utils.InitializeProviders(ctx, cfg)
}

// SetSharedVariables sets the shared variables from the parent package
func SetSharedVariables(cfg *config.Config, verbose, quiet bool, version string) {
	Cfg = cfg
	Verbose = verbose
	Quiet = quiet
	Version = version
}
