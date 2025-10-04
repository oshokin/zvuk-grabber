package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/utils"
	mock_utils "github.com/oshokin/zvuk-grabber/internal/utils/mocks"
)

// TestNewUserAgentInjector tests the NewUserAgentInjector function.
func TestNewUserAgentInjector(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)
	mockProvider.EXPECT().GetUserAgent().Return("TestAgent/1.0").AnyTimes()

	next := http.DefaultTransport
	injector := NewUserAgentInjector(next, mockProvider)

	assert.NotNil(t, injector)
	assert.Implements(t, (*http.RoundTripper)(nil), injector)
}

// TestUserAgentInjector_RoundTrip_WithExistingUserAgent tests RoundTrip when User-Agent header already exists.
func TestUserAgentInjector_RoundTrip_WithExistingUserAgent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ExistingAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create injector with mock provider.
	injector := NewUserAgentInjector(http.DefaultTransport, mockProvider)

	// Create request with existing User-Agent header.
	req, err := http.NewRequest(http.MethodGet, server.URL, nil) //nolint:noctx // Test code, context not needed.
	require.NoError(t, err)
	req.Header.Set("User-Agent", "ExistingAgent/1.0")

	// Execute request.
	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck // Test cleanup, error is not critical.

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_WithoutUserAgent tests RoundTrip when User-Agent header is missing.
func TestUserAgentInjector_RoundTrip_WithoutUserAgent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)
	mockProvider.EXPECT().GetUserAgent().Return("TestAgent/1.0").Times(1)

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create injector with mock provider.
	injector := NewUserAgentInjector(http.DefaultTransport, mockProvider)

	// Create request without User-Agent header.
	req, err := http.NewRequest(http.MethodGet, server.URL, nil) //nolint:noctx // Test code, context not needed.
	require.NoError(t, err)

	// Execute request.
	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck // Test cleanup, error is not critical.

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_WithEmptyUserAgent tests RoundTrip when User-Agent header is empty.
func TestUserAgentInjector_RoundTrip_WithEmptyUserAgent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)
	mockProvider.EXPECT().GetUserAgent().Return("TestAgent/1.0").Times(1)

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create injector with mock provider.
	injector := NewUserAgentInjector(http.DefaultTransport, mockProvider)

	// Create request with empty User-Agent header.
	req, err := http.NewRequest(http.MethodGet, server.URL, nil) //nolint:noctx // Test code, context not needed
	require.NoError(t, err)
	req.Header.Set("User-Agent", "")

	// Execute request.
	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck // Test cleanup, error is not critical.

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_RoundTrip_ErrorHandling tests error handling in RoundTrip.
func TestUserAgentInjector_RoundTrip_ErrorHandling(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)
	mockProvider.EXPECT().GetUserAgent().Return("TestAgent/1.0").AnyTimes()

	// Create injector with mock provider.
	injector := NewUserAgentInjector(http.DefaultTransport, mockProvider)

	// Create request with invalid URL that will definitely fail
	req, err := http.NewRequest(http.MethodGet, "http://[::1]:0", nil) //nolint:noctx // Test code, context not needed
	require.NoError(t, err)

	// Execute request. - should return an error.
	resp, err := injector.RoundTrip(req) //nolint:bodyclose // Body is empty on error.
	require.Error(t, err)
	assert.Nil(t, resp)
}

// TestUserAgentInjector_IntegrationWithSimpleUserAgentProvider tests integration with SimpleUserAgentProvider.
func TestUserAgentInjector_IntegrationWithSimpleUserAgentProvider(t *testing.T) {
	t.Parallel()

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "IntegrationTest/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create injector with SimpleUserAgentProvider.
	provider := utils.NewSimpleUserAgentProvider("IntegrationTest/1.0")
	injector := NewUserAgentInjector(http.DefaultTransport, provider)

	// Create request without User-Agent header.
	req, err := http.NewRequest(http.MethodGet, server.URL, nil) //nolint:noctx // Test code, context not needed.
	require.NoError(t, err)

	// Execute request.
	resp, err := injector.RoundTrip(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck // Test cleanup, error is not critical.

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestUserAgentInjector_MultipleRequests tests that the injector works correctly with multiple requests.
func TestUserAgentInjector_MultipleRequests(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mock_utils.NewMockUserAgentProvider(ctrl)
	mockProvider.EXPECT().GetUserAgent().Return("TestAgent/1.0").Times(5)

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create injector with mock provider.
	injector := NewUserAgentInjector(http.DefaultTransport, mockProvider)

	// Make multiple requests.
	for range 5 {
		req, err := http.NewRequest(http.MethodGet, server.URL, nil) //nolint:noctx // Test code, context not needed.
		require.NoError(t, err)

		resp, err := injector.RoundTrip(req)
		require.NoError(t, err)
		resp.Body.Close() //nolint:errcheck,gosec // Test cleanup, error is not critical.

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
