package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDownloadHTTPConfigYAML loads defaults, overrides, and rejects invalid values.
func TestDownloadHTTPConfigYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("quality: 3\n"), 0o600))
	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, DefaultDownloadHTTPConfig(), cfg.ZvukDownloadHTTP)
	require.NoError(t, os.WriteFile(path, []byte(`zvuk_download_http:
  timeout: 15m
  read_idle_timeout: 0s
  receive_buffer_bytes: 0
  http1_only: false
`), 0o600))
	cfg, err = LoadConfig(path)
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, 15*time.Minute, cfg.ZvukDownloadHTTP.Timeout)
	require.Zero(t, cfg.ZvukDownloadHTTP.ReadIdleTimeout)
	require.Zero(t, cfg.ZvukDownloadHTTP.ReceiveBufferBytes)
	require.False(t, cfg.ZvukDownloadHTTP.HTTP1Only)
	require.Equal(t, DefaultDownloadHTTPDialTimeout, cfg.ZvukDownloadHTTP.DialTimeout)
	require.NoError(t, os.WriteFile(path, renderConfigContent(cfg), 0o600))
	saved, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, cfg.ZvukDownloadHTTP, saved.ZvukDownloadHTTP)

	for _, yaml := range []string{
		"timeout: -1s", "dial_timeout: -1s", "tls_handshake_timeout: -1s",
		"response_header_timeout: -1s", "read_idle_timeout: -1s",
		"receive_buffer_bytes: -1", "receive_buffer_bytes: 2147483648", "timeout: nonsense",
	} {
		require.NoError(t, os.WriteFile(path, []byte("zvuk_download_http:\n  "+yaml+"\n"), 0o600))

		cfg, err = LoadConfig(path)
		if err == nil {
			err = ValidateConfig(cfg)
		}

		require.Error(t, err, yaml)
	}
}
