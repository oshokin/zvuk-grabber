package download

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
)

// roundTripFunc replaces network timing for deterministic retry and cancellation tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

// TestResumeAndRestart exercises real truncated HTTP bodies and validates the final file byte for byte.
//
//nolint:gocognit,nestif // The scenario matrix keeps each simulated CDN response next to its assertions.
func TestResumeAndRestart(t *testing.T) {
	cases := []*struct {
		// name identifies the response variation.
		name string
		// etag is the initial resource validator.
		etag string
		// status is the response to the second request.
		status int
		// resumed reports whether a Range request is safe.
		resumed bool
		// attempts is the final number of requests.
		attempts int32
	}{
		{"resume", `"version1"`, http.StatusPartialContent, true, 2},
		{"ignored range", `"version1"`, http.StatusOK, true, 2},
		{"no validator", "", http.StatusOK, false, 2},
		{"weak validator", `W/"version1"`, http.StatusOK, false, 2},
		{"range rejected", `"version1"`, http.StatusRequestedRangeNotSatisfiable, true, 3},
		{"wrong offset", `"version1"`, http.StatusPartialContent, true, 3},
		{"changed validator", `"version1"`, http.StatusPartialContent, true, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const complete = "0123456789abcdefghij"

			var calls atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := calls.Add(1)

				assert.Equal(t, "identity", r.Header.Get("Accept-Encoding"))
				w.Header().Set("ETag", tc.etag)

				if attempt == 1 {
					assert.Empty(t, r.Header.Get("Range"))
					w.Header().Set("Content-Length", "20")
					_, err := io.WriteString(w, complete[:7])
					assert.NoError(t, err)

					return
				}

				if attempt == 2 {
					if tc.resumed {
						assert.Equal(t, "bytes=7-", r.Header.Get("Range"))
						assert.Equal(t, tc.etag, r.Header.Get("If-Range"))
					} else {
						assert.Empty(t, r.Header.Get("Range"))
					}

					if tc.status == http.StatusRequestedRangeNotSatisfiable {
						w.WriteHeader(tc.status)
						return
					}

					if tc.status == http.StatusPartialContent {
						w.Header().Set("Content-Range", "bytes 7-19/20")

						if tc.name == "wrong offset" {
							w.Header().Set("Content-Range", "bytes 6-19/20")
						}

						if tc.name == "changed validator" {
							w.Header().Set("ETag", `"version2"`)
						}

						w.WriteHeader(tc.status)
						_, err := io.WriteString(w, complete[7:])
						assert.NoError(t, err)

						return
					}
				}

				if attempt > 2 {
					assert.Empty(t, r.Header.Get("Range"))
				}

				_, err := io.WriteString(w, complete)
				assert.NoError(t, err)
			}))
			defer server.Close()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			stream, err := Open(server.Client(), req, testConfig())
			require.NoError(t, err)

			defer stream.Close()

			destination, err := os.Create(filepath.Join(t.TempDir(), "audio.part"))
			require.NoError(t, err)

			defer destination.Close()

			written, err := Copy(t.Context(), destination, stream, new(files.CopyStreamOptions))
			require.NoError(t, err)
			require.Equal(t, int64(len(complete)), written)

			data, err := os.ReadFile(destination.Name())
			require.NoError(t, err)
			require.Equal(t, complete, string(data))
			require.Equal(t, tc.attempts, calls.Load())
		})
	}
}

// RoundTrip invokes the fake transport.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestRetryBudget covers headers and bodies sharing a single finite attempt budget.
func TestRetryBudget(t *testing.T) {
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusForbidden, http.StatusOK} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Length", "20")
				w.Header().Set("ETag", `"stable"`)
				w.WriteHeader(status)
				_, err := io.WriteString(w, "short")
				assert.NoError(t, err)
			}))
			defer server.Close()

			cfg := testConfig()
			cfg.MaxRetries = 2
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			stream, err := Open(server.Client(), req, cfg)
			if err == nil {
				defer stream.Close()

				file, fileErr := os.Create(filepath.Join(t.TempDir(), "part"))
				require.NoError(t, fileErr)

				defer file.Close()

				_, err = Copy(t.Context(), file, stream, nil)
			}

			var transferErr *Error
			require.ErrorAs(t, err, &transferErr)

			expected := int32(3)
			if status == http.StatusForbidden {
				expected = 1
			}

			require.Equal(t, expected, calls.Load())
		})
	}
}

// TestRetryAfterAndCancellation uses virtual time to check server-directed waits and cancellation.
func TestRetryAfterAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := config.DefaultDownloadHTTPConfig()

		var calls int

		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++

			status := http.StatusTooManyRequests
			if calls > 1 {
				status = http.StatusOK
			}

			return &http.Response{
				StatusCode:    status,
				Header:        http.Header{"Retry-After": {"5"}},
				Body:          io.NopCloser(strings.NewReader("ok")),
				ContentLength: 2,
				Request:       r,
			}, nil
		})}
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"https://example.test/audio?secret=hidden",
			nil,
		)
		require.NoError(t, err)

		start := time.Now()
		stream, err := Open(client, req, cfg)
		require.NoError(t, err)
		require.NoError(t, stream.Close())
		require.Equal(t, 5*time.Second, time.Since(start))
		require.Equal(t, 2, calls)
		ctx, cancel := context.WithCancel(t.Context())
		calls = 0
		client.Transport = roundTripFunc(
			func(*http.Request) (*http.Response, error) { calls++; return nil, io.ErrUnexpectedEOF },
		)

		go func() { time.Sleep(time.Millisecond); cancel() }()

		_, err = Open(client, req.Clone(ctx), cfg)
		require.ErrorIs(t, err, context.Canceled)
		require.NotContains(t, err.Error(), "secret")
		require.Equal(t, 1, calls)
	})
}

// TestLocalWriteFailureDoesNotRetry prevents disk failures from triggering more network requests.
func TestLocalWriteFailureDoesNotRetry(t *testing.T) {
	var calls int

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++

		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("content")),
			ContentLength: 7,
			Request:       r,
		}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.test/audio", nil)
	require.NoError(t, err)
	stream, err := Open(client, req, testConfig())
	require.NoError(t, err)

	defer stream.Close()

	file, err := os.Create(filepath.Join(t.TempDir(), "part"))
	require.NoError(t, err)
	require.NoError(t, file.Close())
	_, err = Copy(t.Context(), file, stream, nil)
	require.ErrorIs(t, err, os.ErrClosed)
	require.Equal(t, 1, calls)
}

// TestContentRangeValidation rejects malformed, overflowing and unknown representation sizes.
func TestContentRangeValidation(t *testing.T) {
	for _, value := range []string{"", "bytes 0-9/*", "bytes 1-0/10", "bytes 0-10/10", "bytes -1-9/10", "bytes 0-999999999999999999999/10", "items 0-9/10"} {
		_, _, _, err := parseContentRange(value)
		require.Error(t, err, value)
	}

	start, end, total, err := parseContentRange("bytes 7-19/20")
	require.NoError(t, err)
	require.Equal(t, int64(7), start)
	require.Equal(t, int64(19), end)
	require.Equal(t, int64(20), total)
}

// TestDisabledRetriesAndResume verifies the user's explicit opt-outs.
func TestDisabledRetriesAndResume(t *testing.T) {
	for _, maxRetries := range []int{0, 1} {
		var calls atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("Range"))
			w.Header().Set("ETag", `"stable"`)
			w.Header().Set("Content-Length", "6")

			content := "abcdef"
			if calls.Add(1) == 1 {
				content = "abc"
			}

			_, err := io.WriteString(w, content)
			assert.NoError(t, err)
		}))
		cfg := testConfig()
		cfg.MaxRetries = maxRetries
		cfg.Resume = false
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		stream, err := Open(server.Client(), req, cfg)
		require.NoError(t, err)
		data, err := ReadAll(t.Context(), stream, nil)
		require.NoError(t, stream.Close())
		server.Close()

		if maxRetries == 0 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, "abcdef", string(data))
		}

		require.Equal(t, int32(maxRetries+1), calls.Load())
	}
}

// TestValidatorAndRetryAfterRules checks date strength and both standard Retry-After formats.
func TestValidatorAndRetryAfterRules(t *testing.T) {
	now := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Date", now.Format(http.TimeFormat))
	resp.Header.Set("Last-Modified", now.Add(-2*time.Minute).Format(http.TimeFormat))
	require.NotEmpty(t, responseValidator(resp))
	resp.Header.Set("Last-Modified", now.Format(http.TimeFormat))
	require.Empty(t, responseValidator(resp))
	resp.Header.Set("ETag", `W/"weak"`)
	require.Empty(t, responseValidator(resp))
	require.Equal(t, 2*time.Second, retryAfterDelay("2", now))
	require.Equal(t, 2*time.Second, retryAfterDelay(now.Add(2*time.Second).Format(http.TimeFormat), now))
	require.Zero(t, retryAfterDelay("invalid", now))
	require.Greater(t, retryAfterDelay("999999999999999999999999999", now), time.Hour)
}

// testConfig removes real backoff waits without changing the retry budget.
func testConfig() *config.DownloadHTTPConfig {
	cfg := config.DefaultDownloadHTTPConfig()
	cfg.RetryInitialDelay = time.Nanosecond
	cfg.RetryMaxDelay = time.Nanosecond

	return cfg
}
