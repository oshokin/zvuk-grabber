package yandex

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

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
	assert.Equal(t, "***", sanitized["Authorization"])
	assert.Equal(t, "***", sanitized["Cookie"])
	assert.Equal(t, "***", sanitized["Set-Cookie"])
	assert.Equal(t, "ok", sanitized["X-Test"])
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

	firstChunkWritten := make(chan struct{})
	requestCancelledEarly := make(chan struct{}, 1)
	handlerErr := make(chan error, 1)

	reportHandlerErr := func(err error) {
		if err == nil {
			return
		}

		select {
		case handlerErr <- err:
		default:
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		flusher, hasFlusher := writer.(http.Flusher)

		if _, err := writer.Write([]byte("a")); err != nil {
			reportHandlerErr(fmt.Errorf("failed to write first chunk: %w", err))
			return
		}

		if hasFlusher {
			flusher.Flush()
		}

		close(firstChunkWritten)

		select {
		case <-request.Context().Done():
			requestCancelledEarly <- struct{}{}
			return
		case <-time.After(120 * time.Millisecond):
		}

		if _, err := writer.Write([]byte("b")); err != nil {
			reportHandlerErr(fmt.Errorf("failed to write second chunk: %w", err))
			return
		}

		if hasFlusher {
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewHttpClient()
	client.SetDownloadTimeout(2 * time.Second)

	stream, err := client.OpenStreamWithContext(&httptransport.RequestLogContext{}, server.URL)
	require.NoError(t, err)

	defer stream.Body.Close()

	<-firstChunkWritten

	select {
	case <-requestCancelledEarly:
		t.Fatal("stream request context canceled before caller closed response body")
	case <-time.After(60 * time.Millisecond):
	}

	body, err := io.ReadAll(stream.Body)
	require.NoError(t, err)
	assert.Equal(t, "ab", string(body))

	select {
	case err = <-handlerErr:
		require.NoError(t, err)
	default:
	}
}
