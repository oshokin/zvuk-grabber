package zvuk

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// fetchJSONRefererGot is the Referer observed by the test server.
type fetchJSONRefererGot struct {
	// Referer is the Referer header received by the test server.
	Referer string
}

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

// TestFetchJSONWithQuery_SetsReferer verifies REST API requests carry a Referer,
// without which the zvuk.com anti-bot answers 418.
func TestFetchJSONWithQuery_SetsReferer(t *testing.T) {
	t.Parallel()

	got := new(fetchJSONRefererGot)

	server := httptest.NewTestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got.Referer = request.Header.Get(refererHeader)

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)

		if _, err := writer.Write([]byte("{}")); err != nil {
			t.Errorf("failed to write JSON: %v", err)
		}
	}))

	client := &ClientImpl{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	query := url.Values{"ids": []string{"33343728"}}

	route, err := url.JoinPath(server.URL, zvukAPIReleaseMetadataURI)
	require.NoError(t, err)

	parsed, err := url.Parse(route)
	require.NoError(t, err)

	parsed.RawQuery = query.Encode()

	want := &fetchJSONRefererGot{Referer: parsed.String()}

	_, err = fetchJSONWithQuery[map[string]any](client, t.Context(), zvukAPIReleaseMetadataURI, query)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
