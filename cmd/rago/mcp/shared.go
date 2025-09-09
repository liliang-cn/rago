package mcp

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
)

// SetSharedVariables sets the shared variables from the parent package
func SetSharedVariables(config *config.Config, verbose, quiet bool) {
	Cfg = config
	Verbose = verbose
	Quiet = quiet
}

// InitializeProviders is a wrapper for the shared provider initialization
func InitializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	return utils.InitializeProviders(ctx, cfg)
}