package download

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestPacedTransportNegotiatesHTTP1OverTLS forces HTTP/1.1 when a speed limit is active.
func TestPacedTransportNegotiatesHTTP1OverTLS(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := io.WriteString(w, r.Proto)
		assert.NoError(t, writeErr)
	}))
	server.EnableHTTP2 = true
	server.TLS = &tls.Config{NextProtos: []string{"h2", "http/1.1"}}

	server.StartTLS()
	defer server.Close()

	original, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	originalHTTP2 := original.ForceAttemptHTTP2

	for _, paced := range []bool{true, false} {
		tr := NewTransport(config.DefaultDownloadHTTPConfig(), paced)
		// Trust only the local test server certificate, preserving TLS verification.
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = new(tls.Config)
		}

		serverTransport, isTransport := server.Client().Transport.(*http.Transport)
		require.True(t, isTransport)

		tr.TLSClientConfig.RootCAs = serverTransport.TLSClientConfig.RootCAs
		defer tr.CloseIdleConnections()

		resp, err := (&http.Client{Transport: tr}).Get(server.URL)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		require.NoError(t, err)

		if paced {
			require.Equal(t, "HTTP/1.1", string(body))
			require.Equal(t, "http/1.1", resp.TLS.NegotiatedProtocol)
		} else {
			require.Equal(t, "HTTP/2.0", string(body))
		}
	}

	require.Equal(t, originalHTTP2, original.ForceAttemptHTTP2)
}

// TestDownloadSurvivesBeyondOldTotalTimeout keeps the body open after the API client timeout.
func TestDownloadSurvivesBeyondOldTotalTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		go func() {
			// Consume the GET headers before sending a streaming response.
			b := make([]byte, 4096)
			_, readErr := server.Read(b)
			assert.NoError(t, readErr)

			_, writeErr := io.WriteString(server, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\na")
			assert.NoError(t, writeErr)

			time.Sleep(70 * time.Second)

			_, writeErr = io.WriteString(server, "b")
			assert.NoError(t, writeErr)
		}()

		cfg := config.DefaultDownloadHTTPConfig()
		cfg.ReadIdleTimeout = 2 * time.Minute
		tr := NewTransport(cfg, true)
		tr.Proxy = nil

		tr.DialContext = func(context.Context, string, string) (net.Conn, error) {
			return &idleReadConn{Conn: client, timeout: cfg.ReadIdleTimeout}, nil
		}
		defer tr.CloseIdleConnections()

		resp, err := (&http.Client{Transport: tr}).Get("http://example.test/track")
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, "ab", string(body))
	})
}

// TestIdleReadTimeoutAndLimiterPause times out blocked reads, not limiter pauses.
func TestIdleReadTimeoutAndLimiterPause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader, writer := net.Pipe()
		defer reader.Close()
		defer writer.Close()

		conn := &idleReadConn{Conn: reader, timeout: time.Second}

		go func() { _, writeErr := writer.Write([]byte("a")); assert.NoError(t, writeErr) }()

		b := make([]byte, 1)
		_, err := conn.Read(b)
		require.NoError(t, err)
		time.Sleep(3 * time.Second) // Simulates a slow consumer/rate limiter.

		go func() { _, writeErr := writer.Write([]byte("b")); assert.NoError(t, writeErr) }()

		_, err = conn.Read(b)
		require.NoError(t, err)
		require.Equal(t, byte('b'), b[0])

		_, err = conn.Read(b) // No writer: actual stalled I/O must time out.

		var timeout net.Error

		require.ErrorAs(t, err, &timeout)
		require.True(t, timeout.Timeout())
	})
}

// TestDownloadHeaderTimeoutAndCancellation bounds headers and honors request cancellation.
func TestDownloadHeaderTimeoutAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/body" {
			w.Header().Set("Content-Length", "2")

			_, writeErr := w.Write([]byte("a"))
			assert.NoError(t, writeErr)
			assert.NoError(t, http.NewResponseController(w).Flush())
		}

		<-r.Context().Done()
	}))
	defer server.Close()

	cfg := config.DefaultDownloadHTTPConfig()
	cfg.ResponseHeaderTimeout = 30 * time.Millisecond

	tr := NewTransport(cfg, true)
	defer tr.CloseIdleConnections()

	c := &http.Client{Transport: tr}

	headerResponse, err := c.Get(server.URL)
	if headerResponse != nil {
		_ = headerResponse.Body.Close()
	}

	require.Error(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/body", nil)
	require.NoError(t, err)

	resp, err := c.Do(req)
	require.NoError(t, err)
	cancel()

	_, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	require.ErrorIs(t, err, context.Canceled, "%v", err)
}
