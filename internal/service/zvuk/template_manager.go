package zvuk

import (
	"context"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// Compatibility aliases for the shared media template manager.

// TemplateManager is a compatibility alias to the shared media template manager.
type TemplateManager = media.TemplateManager

// NewTemplateManager creates a shared template manager.
func NewTemplateManager(ctx context.Context, cfg *config.Config) TemplateManager {
	return media.NewTemplateManager(ctx, cfg)
}
