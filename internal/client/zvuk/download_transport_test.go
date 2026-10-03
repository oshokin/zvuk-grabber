package zvuk

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// transportFunc is an http.RoundTripper backed by a function.
type transportFunc func(*http.Request) (*http.Response, error)

// TestDownloadClientSeparateFromAPI uses downloadClient for tracks and httpClient for other assets.
func TestDownloadClientSeparateFromAPI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ZvukAuthToken = "test"
	cfg.ZvukBaseURL = "https://zvuk.com"
	cfg.ParsedDownloadSpeedLimit = 115 * 1024
	client, err := NewClient(cfg)
	require.NoError(t, err)

	c, ok := client.(*ClientImpl)
	require.True(t, ok)
	require.Equal(t, 60*time.Second, c.httpClient.Timeout)
	require.Zero(t, c.downloadClient.Timeout)
	require.NotSame(t, c.httpClient, c.downloadClient)

	apiCalled, downloadCalled := false, false

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, writeErr := io.WriteString(w, "body")
			assert.NoError(t, writeErr)
		}),
	)
	defer server.Close()

	c.httpClient.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		apiCalled = true
		return http.DefaultTransport.RoundTrip(r)
	})
	c.downloadClient.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		downloadCalled = true
		return http.DefaultTransport.RoundTrip(r)
	})
	result, err := c.FetchTrack(t.Context(), server.URL)
	require.NoError(t, err)

	_ = result.Body.Close()

	require.True(t, downloadCalled)
	require.False(t, apiCalled)
	cover, err := c.DownloadFromURL(t.Context(), server.URL)
	require.NoError(t, err)

	_ = cover.Close()

	require.True(t, apiCalled)
}

// RoundTrip calls the wrapped function.
func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
