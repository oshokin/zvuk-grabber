package http

import (
	"errors"
	"net/http"
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
