package yandex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/transport/download"
	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

// TestYandexEncryptedAudioResume verifies unaligned encrypted offsets and complete decryption.
func TestYandexEncryptedAudioResume(t *testing.T) {
	const key = "00112233445566778899aabbccddeeff"

	plain := bytes.Repeat([]byte("audio-frame"), 40)
	encrypted, err := DecryptData(plain, key)
	require.NoError(t, err)

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"encrypted-audio"`)

		if calls.Add(1) == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(len(encrypted)))
			_, writeErr := w.Write(encrypted[:17])
			assert.NoError(t, writeErr)

			return
		}

		assert.Equal(t, "bytes=17-", r.Header.Get("Range"))
		assert.Equal(t, `"encrypted-audio"`, r.Header.Get("If-Range"))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 17-%d/%d", len(encrypted)-1, len(encrypted)))
		w.WriteHeader(http.StatusPartialContent)
		_, writeErr := w.Write(encrypted[17:])
		assert.NoError(t, writeErr)
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.MaxConcurrentDownloads = 2
	cfg.YandexMusicDownloadHTTP.RetryInitialDelay = time.Nanosecond
	cfg.YandexMusicDownloadHTTP.RetryMaxDelay = time.Nanosecond
	spoolDir := t.TempDir()
	t.Setenv("TMPDIR", spoolDir)
	t.Setenv("TMP", spoolDir)
	t.Setenv("TEMP", spoolDir)

	client := NewHttpClient(cfg)
	downloader := newLosslessDownloader(client)
	reqCtx := &httptransport.RequestLogContext{Ctx: t.Context()}
	info := &DownloadInfo{URLs: []string{server.URL}, Key: key}
	actual, err := downloader.DownloadAudio(reqCtx, info)
	require.NoError(t, err)
	require.Equal(t, plain, actual)
	require.Equal(t, int32(2), calls.Load())

	entries, err := os.ReadDir(spoolDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

// TestYandexAudioLimitIsAppliedBeforeDecryption checks that audio network transfer consumes the limit.
func TestYandexAudioLimitIsAppliedBeforeDecryption(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const length = 3 * 1024

		body := bytes.Repeat([]byte("x"), length)
		opts := new(httptransport.ClientOptions)
		opts.ProviderName = providerNameYandex
		opts.AudioHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			resp := new(http.Response)
			resp.StatusCode = http.StatusOK
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(length)
			resp.Header = make(http.Header)
			resp.Request = r

			return resp, nil
		})}

		client := new(HttpClient)
		client.speedLimit = 1024
		client.transport = httptransport.NewClient(opts)

		start := time.Now()
		reqCtx := new(httptransport.RequestLogContext)
		reqCtx.Ctx = t.Context()
		data, err := client.DownloadAudioBytesWithContext(reqCtx, "https://example.test/audio")
		require.NoError(t, err)
		require.Len(t, data, length)
		require.GreaterOrEqual(t, time.Since(start), 1900*time.Millisecond)
	})
}

// TestYandexMP3AndCoverIsolation checks resumable audio and bounded ordinary cover requests.
func TestYandexMP3AndCoverIsolation(t *testing.T) {
	var audioCalls, coverCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cover" {
			coverCalls.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		w.Header().Set("ETag", `"mp3"`)

		if audioCalls.Add(1) == 1 {
			w.Header().Set("Content-Length", "6")
			_, err := io.WriteString(w, "abc")
			assert.NoError(t, err)

			return
		}

		assert.Equal(t, "bytes=3-", r.Header.Get("Range"))
		w.Header().Set("Content-Range", "bytes 3-5/6")
		w.WriteHeader(http.StatusPartialContent)
		_, err := io.WriteString(w, "def")
		assert.NoError(t, err)
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.YandexMusicDownloadHTTP.RetryInitialDelay = time.Nanosecond
	cfg.YandexMusicDownloadHTTP.RetryMaxDelay = time.Nanosecond
	client := NewAuthorizedClient(cfg)
	remote, err := client.OpenAudio(t.Context(), server.URL+"/mp3")
	require.NoError(t, err)

	defer remote.Body.Close()

	file, err := os.Create(filepath.Join(t.TempDir(), "mp3.part"))
	require.NoError(t, err)

	defer file.Close()

	_, err = download.Copy(t.Context(), file, remote.Body, nil)
	require.NoError(t, err)
	data, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	require.Equal(t, "abcdef", string(data))
	_, err = client.DownloadCoverBytes(t.Context(), server.URL+"/cover")
	require.Error(t, err)
	require.Equal(t, int32(1), coverCalls.Load())
}
