package zvuk

//go:generate $MOCKGEN -source=template_manager.go -destination=mocks/template_manager_mock.go

import (
	"context"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// TODO: Remove this compatibility shim after migrating remaining mocks/tests to internal/media.

// TemplateManager is a compatibility alias to the shared media template manager.
type TemplateManager = media.TemplateManager

// NewTemplateManager creates a shared template manager.
func NewTemplateManager(ctx context.Context, cfg *config.Config) TemplateManager {
	return media.NewTemplateManager(ctx, cfg)
}
