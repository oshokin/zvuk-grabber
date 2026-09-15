package zvuk

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFetchJSON_InvalidPayloadReturnsError verifies malformed API JSON fails as a client error.
func TestFetchJSON_InvalidPayloadReturnsError(t *testing.T) {
	t.Parallel()

	server := httptest.NewTestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)

		if _, err := writer.Write([]byte("{")); err != nil {
			t.Errorf("failed to write malformed JSON: %v", err)
		}
	}))

	client := &ClientImpl{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := fetchJSON[map[string]any](client, t.Context(), "/bad")
	require.Error(t, err)
}
