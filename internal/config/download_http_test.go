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
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, DefaultDownloadHTTPConfig(), cfg.ZvukDownloadHTTP)

	require.NoError(t, os.WriteFile(path, []byte(`zvuk_download_http:
  timeout: 15m
  read_idle_timeout: 0s
  receive_buffer: "0"
  http1_only: false
`), 0o600))

	want := DefaultDownloadHTTPConfig()
	want.Timeout = 15 * time.Minute
	want.ReadIdleTimeout = 0
	want.ReceiveBuffer = "0"
	want.ReceiveBufferBytes = 0
	want.HTTP1Only = false

	cfg, err = LoadConfig(path)
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, want, cfg.ZvukDownloadHTTP)

	require.NoError(t, os.WriteFile(path, renderConfigContent(cfg), 0o600))

	saved, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, cfg.ZvukDownloadHTTP, saved.ZvukDownloadHTTP)

	for _, yaml := range []string{
		"timeout: -1s", "dial_timeout: -1s", "tls_handshake_timeout: -1s",
		"response_header_timeout: -1s", "read_idle_timeout: -1s",
		`receive_buffer: "-1"`, `receive_buffer: "3GiB"`, `receive_buffer: nope`, "timeout: nonsense",
	} {
		require.NoError(t, os.WriteFile(path, []byte("zvuk_download_http:\n  "+yaml+"\n"), 0o600))

		cfg, err = LoadConfig(path)
		if err == nil {
			err = ValidateConfig(cfg)
		}

		require.Error(t, err, yaml)
	}
}

// TestDownloadHTTPReceiveBufferParsesHumanSizes accepts the same size syntax as download_speed_limit.
func TestDownloadHTTPReceiveBufferParsesHumanSizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`zvuk_download_http:
  receive_buffer: "256KiB"
yandex_music_download_http:
  receive_buffer: "115 KiB"
`), 0o600))

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, 256*1024, cfg.ZvukDownloadHTTP.ReceiveBufferBytes)
	require.Equal(t, 115*1024, cfg.YandexMusicDownloadHTTP.ReceiveBufferBytes)
}

// TestProviderDownloadSettingsRemainIndependent checks defaults, explicit disabling and YAML round trips.
func TestProviderDownloadSettingsRemainIndependent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`zvuk_download_http:
  max_retries: 0
  resume: false
yandex_music_download_http:
  max_retries: 7
  resume: true
  read_idle_timeout: 45s
  retry_initial_delay: 2s
  retry_max_delay: 20s
`), 0o600))

	wantZvuk := DefaultDownloadHTTPConfig()
	wantZvuk.MaxRetries = 0
	wantZvuk.Resume = false

	wantYandex := DefaultDownloadHTTPConfig()
	wantYandex.MaxRetries = 7
	wantYandex.Resume = true
	wantYandex.ReadIdleTimeout = 45 * time.Second
	wantYandex.RetryInitialDelay = 2 * time.Second
	wantYandex.RetryMaxDelay = 20 * time.Second

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, wantZvuk, cfg.ZvukDownloadHTTP)
	require.Equal(t, wantYandex, cfg.YandexMusicDownloadHTTP)

	require.NoError(t, os.WriteFile(path, renderConfigContent(cfg), 0o600))

	saved, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, cfg.ZvukDownloadHTTP, saved.ZvukDownloadHTTP)
	require.Equal(t, cfg.YandexMusicDownloadHTTP, saved.YandexMusicDownloadHTTP)

	for _, block := range []string{"zvuk_download_http", "yandex_music_download_http"} {
		for _, setting := range []string{"max_retries: -1", "max_retries: 101", "retry_initial_delay: 0s", "retry_max_delay: 500ms", "retry_initial_delay: nonsense"} {
			require.NoError(t, os.WriteFile(path, []byte(block+":\n  "+setting+"\n"), 0o600))

			loaded, loadErr := LoadConfig(path)
			if loadErr == nil {
				loadErr = ValidateConfig(loaded)
			}

			require.Error(t, loadErr)
			require.Contains(t, loadErr.Error(), block)
		}
	}
}

// TestDownloadHTTPConfigClone detaches nested copies from the original settings.
func TestDownloadHTTPConfigClone(t *testing.T) {
	original := DefaultDownloadHTTPConfig()
	cloned := original.Clone()
	require.Equal(t, original, cloned)
	require.NotSame(t, original, cloned)

	cloned.Timeout = time.Minute
	cloned.Resume = false
	cloned.ReceiveBuffer = "0"

	require.Zero(t, original.Timeout)
	require.True(t, original.Resume)
	require.Equal(t, DefaultDownloadHTTPReceiveBuffer, original.ReceiveBuffer)

	require.Nil(t, (*DownloadHTTPConfig)(nil).Clone())
}
