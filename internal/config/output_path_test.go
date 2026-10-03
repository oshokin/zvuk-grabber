package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveProviderOutputPath verifies provider-specific output path resolution.
func TestResolveProviderOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		outputPath      string
		groupByProvider bool
		provider        ProviderID
		expected        string
	}{
		{
			name:            "grouping disabled",
			outputPath:      "downloads",
			groupByProvider: false,
			provider:        ProviderZvuk,
			expected:        "downloads",
		},
		{
			name:            "grouping enabled for zvuk",
			outputPath:      "downloads",
			groupByProvider: true,
			provider:        ProviderZvuk,
			expected:        filepath.Join("downloads", "zvuk"),
		},
		{
			name:            "grouping enabled for yandex",
			outputPath:      "downloads",
			groupByProvider: true,
			provider:        ProviderYandex,
			expected:        filepath.Join("downloads", "yandex"),
		},
		{
			name:            "empty provider keeps base path",
			outputPath:      "downloads",
			groupByProvider: true,
			provider:        "",
			expected:        "downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := ResolveProviderOutputPath(tt.outputPath, tt.groupByProvider, tt.provider)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// TestConfigResolveOutputPath verifies Config.ResolveOutputPath delegates to provider resolution.
func TestConfigResolveOutputPath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		OutputPath:      "downloads",
		GroupByProvider: true,
	}

	assert.Equal(t, filepath.Join("downloads", "zvuk"), cfg.ResolveOutputPath(ProviderZvuk))

	cfg.GroupByProvider = false
	assert.Equal(t, "downloads", cfg.ResolveOutputPath(ProviderZvuk))

	var nilCfg *Config

	assert.Empty(t, nilCfg.ResolveOutputPath(ProviderZvuk))
}
