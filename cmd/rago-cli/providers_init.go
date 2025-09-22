package main

import (
	"context"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// initializeProviders is a wrapper for the shared provider initialization
func initializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	return providers.InitializeProviders(ctx, cfg)
}
