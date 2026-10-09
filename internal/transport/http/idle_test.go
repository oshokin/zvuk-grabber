package http

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// closeIdleRecorder records CloseIdleConnections on a fake RoundTripper.
type closeIdleRecorder struct {
	// closed is set when CloseIdleConnections runs.
	closed bool
}

// closeIdleClientsWant is the expected close state of API and audio transports.
type closeIdleClientsWant struct {
	// API is whether the API transport was closed.
	API bool
	// Audio is whether the audio transport was closed.
	Audio bool
}

// keepAliveConnState is the close flag for a tracked keep-alive connection.
type keepAliveConnState struct {
	// closed is set when the dialed connection is closed.
	closed atomic.Bool
}

// keepAliveCloseWant is the socket state after the request and after idle close.
type keepAliveCloseWant struct {
	// ClosedAfterRequest is whether the socket closed after the body was fully read.
	ClosedAfterRequest bool
	// ClosedAfterIdleClose is whether the socket closed after CloseIdleHTTPClient.
	ClosedAfterIdleClose bool
}

// trackingKeepAliveConn records Close on a pooled keep-alive connection.
type trackingKeepAliveConn struct {
	net.Conn

	// state is the shared close flag observed by the test.
	state *keepAliveConnState
}

// opaqueRoundTripper wraps a transport without CloseIdleConnections, reproducing the hang.
type opaqueRoundTripper struct {
	// next is the inner HTTP round tripper.
	next http.RoundTripper
}

// errCloseIdleRoundTripUnused is returned if a close-idle test transport is used for a request.
var errCloseIdleRoundTripUnused = errors.New("round trip is unused")

// RoundTrip is unused; this transport only exists to observe CloseIdleConnections.
func (r *closeIdleRecorder) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errCloseIdleRoundTripUnused
}

// CloseIdleConnections records that idle connections were closed.
func (r *closeIdleRecorder) CloseIdleConnections() {
	r.closed = true
}

// Close marks the keep-alive socket closed, then closes the inner connection.
func (c *trackingKeepAliveConn) Close() error {
	c.state.closed.Store(true)

	return c.Conn.Close()
}

// RoundTrip forwards the request without exposing CloseIdleConnections.
func (o *opaqueRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return o.next.RoundTrip(request)
}

// TestKeepAliveStaysOpenUntilIdleClose reproduces the CLI hang: the socket stays
// pooled after the download, then CloseIdleHTTPClient closes it so the process can exit.
func TestKeepAliveStaysOpenUntilIdleClose(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)

		if _, err := writer.Write([]byte("ok")); err != nil {
			t.Errorf("failed to write body: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	state := new(keepAliveConnState)
	client := &http.Client{Transport: newKeepAliveTrackingTransport(state)}
	want := &keepAliveCloseWant{ClosedAfterRequest: false, ClosedAfterIdleClose: true}
	got := snapshotKeepAliveCloseWant(t, client, server.URL, state)

	require.Equal(t, want, got)
}

// TestOpaqueRoundTripperSwallowsIdleClose reproduces the Zvuk wrapper bug:
// http.Client.CloseIdleConnections is a no-op when the outer RoundTripper
// does not implement CloseIdleConnections, so the keep-alive socket stays open.
func TestOpaqueRoundTripperSwallowsIdleClose(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)

		if _, err := writer.Write([]byte("ok")); err != nil {
			t.Errorf("failed to write body: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	state := new(keepAliveConnState)
	transport := newKeepAliveTrackingTransport(state)
	t.Cleanup(transport.CloseIdleConnections)

	client := &http.Client{Transport: &opaqueRoundTripper{next: transport}}
	want := &keepAliveCloseWant{ClosedAfterRequest: false, ClosedAfterIdleClose: false}
	got := snapshotKeepAliveCloseWant(t, client, server.URL, state)

	require.Equal(t, want, got)
}

// TestWrappedDownloadTransportClosesKeepAlive verifies the Zvuk download stack
// (User-Agent injector over log transport) closes the pooled socket.
func TestWrappedDownloadTransportClosesKeepAlive(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)

		if _, err := writer.Write([]byte("ok")); err != nil {
			t.Errorf("failed to write body: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	state := new(keepAliveConnState)
	transport := NewUserAgentInjector(
		NewLogTransport(newKeepAliveTrackingTransport(state), 1024),
		utils.NewSimpleUserAgentProvider("TestAgent/1.0"),
	)
	client := &http.Client{Transport: transport}
	want := &keepAliveCloseWant{ClosedAfterRequest: false, ClosedAfterIdleClose: true}
	got := snapshotKeepAliveCloseWant(t, client, server.URL, state)

	require.Equal(t, want, got)
}

// TestUserAgentInjector_CloseIdleConnections verifies stdlib Client.CloseIdleConnections
// reaches the wrapped transport through the User-Agent injector.
func TestUserAgentInjector_CloseIdleConnections(t *testing.T) {
	t.Parallel()

	got := new(closeIdleRecorder)
	client := &http.Client{
		Transport: NewUserAgentInjector(got, utils.NewSimpleUserAgentProvider("TestAgent/1.0")),
	}

	client.CloseIdleConnections()
	require.True(t, got.closed)
}

// TestLogTransport_CloseIdleConnections verifies debug logging still forwards idle closes.
func TestLogTransport_CloseIdleConnections(t *testing.T) {
	t.Parallel()

	got := new(closeIdleRecorder)
	client := &http.Client{
		Transport: NewLogTransport(got, 1024),
	}

	client.CloseIdleConnections()
	require.True(t, got.closed)
}

// TestWrappedDownloadTransport_CloseIdleConnections verifies the Zvuk download stack
// (User-Agent injector over log transport) still closes the inner HTTP transport.
func TestWrappedDownloadTransport_CloseIdleConnections(t *testing.T) {
	t.Parallel()

	got := new(closeIdleRecorder)
	client := &http.Client{
		Transport: NewUserAgentInjector(
			NewLogTransport(got, 1024),
			utils.NewSimpleUserAgentProvider("TestAgent/1.0"),
		),
	}

	client.CloseIdleConnections()
	require.True(t, got.closed)
}

// TestClient_CloseIdleConnections closes both API and audio HTTP clients.
func TestClient_CloseIdleConnections(t *testing.T) {
	t.Parallel()

	apiTransport := new(closeIdleRecorder)
	audioTransport := new(closeIdleRecorder)
	client := NewClient(&ClientOptions{
		HTTPClient:      &http.Client{Transport: apiTransport},
		AudioHTTPClient: &http.Client{Transport: audioTransport},
	})

	want := &closeIdleClientsWant{API: true, Audio: true}

	client.CloseIdleConnections()

	got := &closeIdleClientsWant{
		API:   apiTransport.closed,
		Audio: audioTransport.closed,
	}
	require.Equal(t, want, got)
}

// newKeepAliveTrackingTransport dials through a wrapper that records socket close.
func newKeepAliveTrackingTransport(state *keepAliveConnState) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck // net/http initializes DefaultTransport to *http.Transport.
	transport.DisableKeepAlives = false

	dialer := new(net.Dialer)
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return &trackingKeepAliveConn{Conn: conn, state: state}, nil
	}

	return transport
}

// doKeepAliveGET reads a keep-alive response to completion so the socket can be pooled.
func doKeepAliveGET(t *testing.T, client *http.Client, rawURL string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

// snapshotKeepAliveCloseWant records socket close flags around CloseIdleHTTPClient.
func snapshotKeepAliveCloseWant(
	t *testing.T,
	client *http.Client,
	rawURL string,
	state *keepAliveConnState,
) *keepAliveCloseWant {
	t.Helper()

	doKeepAliveGET(t, client, rawURL)

	got := &keepAliveCloseWant{ClosedAfterRequest: state.closed.Load()}

	CloseIdleHTTPClient(client)

	got.ClosedAfterIdleClose = state.closed.Load()

	return got
}
