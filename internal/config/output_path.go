package config

import (
	"path/filepath"
	"strings"
)

// ProviderID identifies a source provider used for output path grouping.
type ProviderID string

const (
	// ProviderZvuk identifies Zvuk downloads.
	ProviderZvuk ProviderID = "zvuk"
	// ProviderYandex identifies Yandex Music downloads.
	ProviderYandex ProviderID = "yandex"
)

// ResolveProviderOutputPath returns an output path grouped by provider when enabled.
func ResolveProviderOutputPath(outputPath string, groupByProvider bool, provider ProviderID) string {
	basePath := strings.TrimSpace(outputPath)
	if !groupByProvider || strings.TrimSpace(string(provider)) == "" {
		return basePath
	}

	return filepath.Join(basePath, string(provider))
}

// ResolveOutputPath resolves the effective output path for the given provider.
func (cfg *Config) ResolveOutputPath(provider ProviderID) string {
	if cfg == nil {
		return ""
	}

	return ResolveProviderOutputPath(cfg.OutputPath, cfg.GroupByProvider, provider)
}
