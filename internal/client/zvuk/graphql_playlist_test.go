package zvuk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestGetPlaylistsMetadata_UsesGraphQLTrackOrder verifies playlist numbering follows
// website GraphQL order instead of stale REST track_ids.
func TestGetPlaylistsMetadata_UsesGraphQLTrackOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.Contains(request.URL.Path, "graphql"):
			writePlaylistTracksGraphQLPage(t, writer, request)
		case strings.Contains(request.URL.Path, "playlists"):
			writeJSONResponse(writer, GetMetadataResponse{
				Result: &Metadata{
					Playlists: map[string]*Playlist{
						"6457461": {
							ID:       6457461,
							Title:    "Правильный мужской плейлист",
							TrackIDs: []int64{1, 81848793, 2, 55106485, 3},
						},
					},
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newPlaylistTestClient(t, server.URL)
	response, err := client.GetPlaylistsMetadata(t.Context(), []string{"6457461"})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Contains(t, response.Playlists, "6457461")

	assert.Equal(
		t,
		[]int64{64870395, 143304072, 149362305},
		response.Playlists["6457461"].TrackIDs,
	)
}

// TestGetPlaylistsMetadata_FallsBackToRESTTrackIDsWhenGraphQLFails verifies REST order
// is kept when playlistTracks cannot be loaded.
func TestGetPlaylistsMetadata_FallsBackToRESTTrackIDsWhenGraphQLFails(t *testing.T) {
	t.Parallel()

	restTrackIDs := []int64{1, 81848793, 2}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.Contains(request.URL.Path, "graphql"):
			writer.WriteHeader(http.StatusInternalServerError)
		case strings.Contains(request.URL.Path, "playlists"):
			writeJSONResponse(writer, GetMetadataResponse{
				Result: &Metadata{
					Playlists: map[string]*Playlist{
						"6457461": {
							ID:       6457461,
							Title:    "Правильный мужской плейлист",
							TrackIDs: restTrackIDs,
						},
					},
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newPlaylistTestClient(t, server.URL)
	response, err := client.GetPlaylistsMetadata(t.Context(), []string{"6457461"})
	require.NoError(t, err)
	require.Contains(t, response.Playlists, "6457461")
	assert.Equal(t, restTrackIDs, response.Playlists["6457461"].TrackIDs)
}

func newPlaylistTestClient(t *testing.T, baseURL string) *ClientImpl {
	t.Helper()

	client, err := NewClient(&config.Config{
		ZvukAuthToken:       "test_token",
		ZvukBaseURL:         baseURL,
		RetryAttemptsCount:  1,
		ParsedMinRetryPause: 0,
		ParsedMaxRetryPause: 0,
	})
	require.NoError(t, err)

	typedClient, ok := client.(*ClientImpl)
	require.True(t, ok)

	return typedClient
}

func writePlaylistTracksGraphQLPage(t *testing.T, writer http.ResponseWriter, request *http.Request) {
	t.Helper()

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var payload struct {
		Variables struct {
			Offset int `json:"offset"`
		} `json:"variables"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))

	tracks := []any{
		map[string]any{"id": "64870395"},
		nil,
		map[string]any{"id": "143304072"},
		map[string]any{"id": "149362305"},
	}
	if payload.Variables.Offset > 0 {
		tracks = []any{}
	}

	writeJSONResponse(writer, map[string]any{
		"data": map[string]any{
			"playlistTracks": tracks,
		},
	})
}
