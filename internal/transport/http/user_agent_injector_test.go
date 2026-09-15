package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// TestNewUserAgentInjector tests the NewUserAgentInjector function.
func TestNewUserAgentInjector(t *testing.T) {
	t.Parallel()

	next := http.DefaultTransport
	injector := NewUserAgentInjector(next, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	assert.NotNil(t, injector)
	assert.Implements(t, (*http.RoundTripper)(nil), injector)
}

// TestUserAgentInjector_RoundTrip_WithExistingUserAgent tests RoundTrip when User-Agent header already exists.
func TestUserAgentInjector_RoundTrip_WithExistingUserAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ExistingAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))

	injector := NewUserAgentInjector(server.Client().Transport, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "ExistingAgent/1.0")

	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_WithoutUserAgent tests RoundTrip when User-Agent header is missing.
func TestUserAgentInjector_RoundTrip_WithoutUserAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))

	injector := NewUserAgentInjector(server.Client().Transport, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_WithEmptyUserAgent tests RoundTrip when User-Agent header is empty.
func TestUserAgentInjector_RoundTrip_WithEmptyUserAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))

	injector := NewUserAgentInjector(server.Client().Transport, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "")

	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_ErrorHandling tests error handling in RoundTrip.
func TestUserAgentInjector_RoundTrip_ErrorHandling(t *testing.T) {
	t.Parallel()

	injector := NewUserAgentInjector(http.DefaultTransport, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://[::1]:0", nil)
	require.NoError(t, err)

	resp, err := injector.RoundTrip(req) //nolint:bodyclose // Body is empty on error.
	require.Error(t, err)
	assert.Nil(t, resp)
}

// TestUserAgentInjector_IntegrationWithSimpleUserAgentProvider tests integration with SimpleUserAgentProvider.
func TestUserAgentInjector_IntegrationWithSimpleUserAgentProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "IntegrationTest/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))

	provider := utils.NewSimpleUserAgentProvider("IntegrationTest/1.0")
	injector := NewUserAgentInjector(server.Client().Transport, provider)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_MultipleRequests tests that the injector works correctly with multiple requests.
func TestUserAgentInjector_MultipleRequests(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))

	injector := NewUserAgentInjector(server.Client().Transport, utils.NewSimpleUserAgentProvider("TestAgent/1.0"))

	for range 5 {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := injector.RoundTrip(req)
		require.NoError(t, err)

		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
