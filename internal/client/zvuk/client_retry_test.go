package zvuk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestClientImpl_GetStreamMetadata_RetriesTeapot verifies transient HTTP 418 responses are retried.
func TestClientImpl_GetStreamMetadata_RetriesTeapot(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	client := newRetryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+zvukAPIStreamMetadataURI {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		call := calls.Add(1)
		if call < 3 {
			w.WriteHeader(http.StatusTeapot)

			return
		}

		response := &GetStreamMetadataResponse{
			Result: &StreamMetadata{
				Stream: "https://example.com/stream.mp3",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}, 3, 0, 0)

	result, err := client.GetStreamMetadata(t.Context(), "track-id", defaultStreamQuality)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://example.com/stream.mp3", result.Stream)
	assert.EqualValues(t, 3, calls.Load())
}

// TestClientImpl_GetStreamMetadata_DoesNotRetryNonTeapot verifies non-retryable HTTP errors fail fast.
func TestClientImpl_GetStreamMetadata_DoesNotRetryNonTeapot(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	client := newRetryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+zvukAPIStreamMetadataURI {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}, 5, 0, 0)

	_, err := client.GetStreamMetadata(t.Context(), "track-id", defaultStreamQuality)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnexpectedHTTPStatus)
	assert.EqualValues(t, 1, calls.Load())
}

// TestClientImpl_GetStreamMetadata_CancelledDuringRetryPause verifies cancellation aborts retry backoff.
func TestClientImpl_GetStreamMetadata_CancelledDuringRetryPause(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int64

		cfg := new(config.Config)
		cfg.ZvukAuthToken = "test_token"
		cfg.ZvukBaseURL = "https://zvuk.example"
		cfg.APIRetryAttemptsCount = 5
		cfg.ParsedAPIMinRetryPause = 500 * time.Millisecond
		cfg.ParsedAPIMaxRetryPause = 500 * time.Millisecond

		client, err := NewClient(cfg)
		require.NoError(t, err)

		typedClient, ok := client.(*ClientImpl)
		require.True(t, ok)

		typedClient.httpClient.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)

			resp := new(http.Response)
			resp.StatusCode = http.StatusTeapot
			resp.Body = http.NoBody
			resp.Header = make(http.Header)
			resp.Request = r

			return resp, nil
		})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()

		startedAt := time.Now()
		_, err = typedClient.GetStreamMetadata(ctx, "track-id", defaultStreamQuality)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		assert.EqualValues(t, 1, calls.Load())
		assert.Equal(t, 25*time.Millisecond, time.Since(startedAt))
	})
}

// newRetryTestClient builds a client backed by an httptest server for retry behavior tests.
func newRetryTestClient(
	t *testing.T,
	handler http.HandlerFunc,
	attempts int64,
	minPause, maxPause time.Duration,
) *ClientImpl {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(&config.Config{
		ZvukAuthToken:          "test_token",
		ZvukBaseURL:            server.URL,
		APIRetryAttemptsCount:  attempts,
		ParsedAPIMinRetryPause: minPause,
		ParsedAPIMaxRetryPause: maxPause,
	})
	require.NoError(t, err)

	typedClient, ok := client.(*ClientImpl)
	require.True(t, ok)

	return typedClient
}
