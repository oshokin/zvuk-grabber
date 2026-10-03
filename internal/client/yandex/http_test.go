package yandex

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

// roundTripFunc is an http.RoundTripper backed by a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the wrapped function.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestSanitizeHeaders_RedactsSensitiveHeaders verifies sensitive HTTP headers are redacted in logs.
func TestSanitizeHeaders_RedactsSensitiveHeaders(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": {"OAuth y0__secret"},
		"Cookie":        {"Session_id=abc"},
		"Set-Cookie":    {"sessionid2=def"},
		"X-Test":        {"ok"},
	}

	sanitized := httptransport.SanitizeHeaders(headers)
	want := map[string]string{
		"Authorization": "***",
		"Cookie":        "***",
		"Set-Cookie":    "***",
		"X-Test":        "ok",
	}
	assert.Equal(t, want, sanitized)
}

// TestSanitizeURL_RedactsSensitiveQueryFields verifies sensitive query parameters are redacted in URLs.
func TestSanitizeURL_RedactsSensitiveQueryFields(t *testing.T) {
	t.Parallel()

	rawURL := "https://api.music.yandex.net/tracks/1/download-info?client_secret=abc123&token=qwerty&limit=10"
	sanitized := httptransport.SanitizeURLWithKeys(rawURL, requestSensitiveQueryKeys, requestSensitiveFieldKeys)

	assert.Contains(t, sanitized, "client_secret=%2A%2A%2A")
	assert.Contains(t, sanitized, "token=%2A%2A%2A")
	assert.Contains(t, sanitized, "limit=10")
	assert.NotContains(t, sanitized, "abc123")
	assert.NotContains(t, sanitized, "qwerty")
}

// TestResponsePreview_OnlyTextContentTypes verifies response previews are generated only for text content types.
func TestResponsePreview_OnlyTextContentTypes(t *testing.T) {
	t.Parallel()

	jsonBody := []byte(`{"access_token":"secret-token","name":"ok"}`)
	preview := httptransport.ResponsePreview("application/json", jsonBody)
	assert.Contains(t, preview, "access_token")
	assert.Contains(t, preview, "***")
	assert.NotContains(t, preview, "secret-token")

	binaryPreview := httptransport.ResponsePreview("application/octet-stream", []byte{0x01, 0x02, 0x03})
	assert.Empty(t, binaryPreview)
}

// TestResponsePreview_TruncatesTo512Bytes verifies long response previews are truncated.
func TestResponsePreview_TruncatesTo512Bytes(t *testing.T) {
	t.Parallel()

	longValue := strings.Repeat("a", 600)
	preview := httptransport.ResponsePreview("text/plain", []byte(longValue))
	assert.LessOrEqual(t, len(preview), 530)
	assert.Contains(t, preview, "...(truncated)")
}

// TestOpenStreamWithContext_KeepStreamAliveUntilCallerClose verifies the stream context stays alive until the body is closed.
func TestOpenStreamWithContext_KeepStreamAliveUntilCallerClose(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		requestCancelledEarly := make(chan struct{}, 1)
		writeSecond := make(chan struct{})

		opts := new(httptransport.ClientOptions)
		opts.DownloadTimeout = 2 * time.Second
		opts.ProviderName = providerNameYandex
		opts.AudioHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			reader, writer := io.Pipe()

			go func() {
				defer writer.Close()

				_, writeErr := writer.Write([]byte("a"))
				assert.NoError(t, writeErr)

				select {
				case <-r.Context().Done():
					requestCancelledEarly <- struct{}{}
					return
				case <-writeSecond:
				}

				_, writeErr = writer.Write([]byte("b"))
				assert.NoError(t, writeErr)
			}()

			resp := new(http.Response)
			resp.StatusCode = http.StatusOK
			resp.Body = reader
			resp.ContentLength = 2
			resp.Header = make(http.Header)
			resp.Request = r

			return resp, nil
		})}

		client := new(HttpClient)
		client.transport = httptransport.NewClient(opts)

		reqCtx := new(httptransport.RequestLogContext)
		reqCtx.Ctx = t.Context()
		stream, err := client.OpenStreamWithContext(reqCtx, "https://example.test/audio")
		require.NoError(t, err)

		defer stream.Body.Close()

		first := make([]byte, 1)
		n, err := stream.Body.Read(first)
		require.NoError(t, err)
		require.Equal(t, "a", string(first[:n]))
		synctest.Wait()

		select {
		case <-requestCancelledEarly:
			t.Fatal("stream request context canceled before caller closed response body")
		default:
		}

		close(writeSecond)

		rest, err := io.ReadAll(stream.Body)
		require.NoError(t, err)
		assert.Equal(t, "ab", string(first)+string(rest))
	})
}
