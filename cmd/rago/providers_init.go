package rago

import (
	"context"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// initializeProviders is a wrapper for the shared provider initialization
func initializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	return utils.InitializeProviders(ctx, cfg)
}
