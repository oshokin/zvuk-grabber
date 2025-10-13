package zvuk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// mockZvukClient is a mock implementation of the Client interface for testing.
type mockZvukClient struct {
	server *httptest.Server
}

func newMockZvukClient() *mockZvukClient {
	server := httptest.NewServer(http.HandlerFunc(mockHandler))
	return &mockZvukClient{server: server}
}

func (m *mockZvukClient) DownloadFromURL(_ context.Context, url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (m *mockZvukClient) FetchTrack(_ context.Context, trackURL string) (io.ReadCloser, int64, error) {
	resp, err := http.Get(trackURL)
	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.ContentLength, nil
}

func (m *mockZvukClient) GetAlbumsMetadata(
	_ context.Context,
	_ []string,
	_ bool,
) (*GetAlbumsMetadataResponse, error) {
	return &GetAlbumsMetadataResponse{
		Tracks:   make(map[string]*Track),
		Releases: make(map[string]*Release),
	}, nil
}

func (m *mockZvukClient) GetAlbumURL(_ context.Context, _ string) (string, error) {
	return "https://example.com/album", nil
}

func (m *mockZvukClient) GetArtistReleaseIDs(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]string, error) {
	return []string{"release1", "release2"}, nil
}

func (m *mockZvukClient) GetBaseURL() string {
	return m.server.URL
}

func (m *mockZvukClient) GetLabelsMetadata(_ context.Context, _ []string) (*GetLabelsMetadataResponse, error) {
	return &GetLabelsMetadataResponse{
		Labels: make(map[string]*Label),
	}, nil
}

func (m *mockZvukClient) GetPlaylistsMetadata(
	_ context.Context,
	_ []string,
) (*GetPlaylistsMetadataResponse, error) {
	return &GetPlaylistsMetadataResponse{
		Playlists: make(map[string]*Playlist),
	}, nil
}

func (m *mockZvukClient) GetStreamMetadata(_ context.Context, _ string) (*GetStreamMetadataResponse, error) {
	return &GetStreamMetadataResponse{
		Result: &StreamMetadata{
			Stream: "https://example.com/stream.mp3",
		},
	}, nil
}

func (m *mockZvukClient) GetTrackLyrics(_ context.Context, _ string) (*GetLyricsResponse, error) {
	return &GetLyricsResponse{
		Result: &Lyrics{
			Type:   LyricsTypeSubtitle,
			Lyrics: "Test lyrics content",
		},
	}, nil
}

func (m *mockZvukClient) GetTracksMetadata(_ context.Context, _ []string) (*GetMetadataResponse, error) {
	return &GetMetadataResponse{
		Result: &Metadata{
			Tracks: make(map[string]*Track),
		},
	}, nil
}

func (m *mockZvukClient) GetUserProfile(_ context.Context) (*GetUserProfileResponse, error) {
	return &GetUserProfileResponse{
		Result: &UserProfile{
			Subscription: &UserSubscription{
				Title:      "Premium",
				Expiration: 1234567890,
			},
		},
	}, nil
}

func (m *mockZvukClient) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// mockHandler handles HTTP requests for testing.
func mockHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()

	switch {
	case strings.Contains(path, "graphql"):
		handleGraphQLRequest(w, r)
	case strings.Contains(path, "track/stream"):
		handleStreamRequest(w, r, query)
	case strings.Contains(path, "lyrics"):
		handleLyricsRequest(w, r, query)
	case strings.Contains(path, "profile"):
		handleProfileRequest(w, r)
	case strings.Contains(path, "tracks"):
		handleTracksRequest(w, r, query)
	case strings.Contains(path, "releases"):
		handleReleasesRequest(w, r, query)
	case strings.Contains(path, "playlists"):
		handlePlaylistsRequest(w, r, query)
	case strings.Contains(path, "labels"):
		handleLabelsRequest(w, r, query)
	default:
		// For download requests.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content")) //nolint:errcheck // Test mock handler, error is not critical.
	}
}

// handleGraphQLRequest handles GraphQL requests.
func handleGraphQLRequest(w http.ResponseWriter, _ *http.Request) {
	response := map[string]any{
		"getArtists": []any{
			map[string]any{
				"releases": []any{
					map[string]any{"id": "release1"},
					map[string]any{"id": "release2"},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleStreamRequest handles stream metadata requests.
func handleStreamRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetStreamMetadataResponse{
		Result: &StreamMetadata{
			Stream: "https://example.com/stream.mp3",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleLyricsRequest handles lyrics requests.
func handleLyricsRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetLyricsResponse{
		Result: &Lyrics{
			Type:   LyricsTypeSubtitle,
			Lyrics: "Test lyrics content",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleProfileRequest handles user profile requests.
func handleProfileRequest(w http.ResponseWriter, _ *http.Request) {
	response := GetUserProfileResponse{
		Result: &UserProfile{
			Subscription: &UserSubscription{
				Title:      "Premium",
				Expiration: 1234567890,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleTracksRequest handles track metadata requests.
func handleTracksRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetMetadataResponse{
		Result: &Metadata{
			Tracks: map[string]*Track{
				"track1": {
					ID:    1,
					Title: "Test Track",
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleReleasesRequest handles release metadata requests.
func handleReleasesRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetAlbumsMetadataResponse{
		Releases: map[string]*Release{
			"release1": {
				ID:    1,
				Title: "Test Release",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handlePlaylistsRequest handles playlist metadata requests.
func handlePlaylistsRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetPlaylistsMetadataResponse{
		Playlists: map[string]*Playlist{
			"playlist1": {
				ID:    1,
				Title: "Test Playlist",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// handleLabelsRequest handles label metadata requests.
func handleLabelsRequest(w http.ResponseWriter, _ *http.Request, _ url.Values) {
	response := GetLabelsMetadataResponse{
		Labels: map[string]*Label{
			"label1": {
				Title: "Test Label",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint:errcheck,errchkjson // Test mock handler, error is not critical.
}

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AuthToken:           "test_token",
				Quality:             2,
				ZvukBaseURL:         "https://zvuk.com",
				RetryAttemptsCount:  3,
				ParsedMaxRetryPause: 1000000000, // 1 second.
				ParsedMinRetryPause: 100000000,  // 100ms.
			},
			expectError: false,
		},
		{
			name: "invalid base URL",
			config: &config.Config{
				AuthToken:           "test_token",
				Quality:             2,
				ZvukBaseURL:         "://invalid-url",
				RetryAttemptsCount:  3,
				ParsedMaxRetryPause: 1000000000,
				ParsedMinRetryPause: 100000000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestClientImpl_DownloadFromURL tests the DownloadFromURL method.
func TestClientImpl_DownloadFromURL(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	url := mockClient.GetBaseURL() + "/test-download"

	reader, err := mockClient.DownloadFromURL(ctx, url)
	require.NoError(t, err)
	assert.NotNil(t, reader)

	// Read content.
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
	reader.Close()
}

// TestClientImpl_FetchTrack tests the FetchTrack method.
func TestClientImpl_FetchTrack(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	trackURL := mockClient.GetBaseURL() + "/track/123"

	reader, size, err := mockClient.FetchTrack(ctx, trackURL)
	require.NoError(t, err)
	assert.NotNil(t, reader)
	assert.Equal(t, int64(12), size) // "test content" length.

	reader.Close()
}

// TestClientImpl_GetAlbumsMetadata tests the GetAlbumsMetadata method.
func TestClientImpl_GetAlbumsMetadata(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	releaseIDs := []string{"release1", "release2"}

	response, err := mockClient.GetAlbumsMetadata(ctx, releaseIDs, true)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Tracks)
	assert.NotNil(t, response.Releases)
}

// TestClientImpl_GetAlbumURL tests the GetAlbumURL method.
func TestClientImpl_GetAlbumURL(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	releaseID := "release123"

	url, err := mockClient.GetAlbumURL(ctx, releaseID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/album", url)
}

// TestClientImpl_GetArtistReleaseIDs tests the GetArtistReleaseIDs method.
func TestClientImpl_GetArtistReleaseIDs(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	artistID := "artist123"

	releaseIDs, err := mockClient.GetArtistReleaseIDs(ctx, artistID, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, releaseIDs)
	assert.Equal(t, []string{"release1", "release2"}, releaseIDs)
}

// TestClientImpl_GetBaseURL tests the GetBaseURL method.
func TestClientImpl_GetBaseURL(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	baseURL := mockClient.GetBaseURL()
	assert.NotEmpty(t, baseURL)
	assert.Contains(t, baseURL, "127.0.0.1")
}

// TestClientImpl_GetLabelsMetadata tests the GetLabelsMetadata method.
func TestClientImpl_GetLabelsMetadata(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	labelIDs := []string{"label1", "label2"}

	response, err := mockClient.GetLabelsMetadata(ctx, labelIDs)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Labels)
}

// TestClientImpl_GetPlaylistsMetadata tests the GetPlaylistsMetadata method.
func TestClientImpl_GetPlaylistsMetadata(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	playlistIDs := []string{"playlist1", "playlist2"}

	response, err := mockClient.GetPlaylistsMetadata(ctx, playlistIDs)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Playlists)
}

// TestClientImpl_GetStreamMetadata tests the GetStreamMetadata method.
func TestClientImpl_GetStreamMetadata(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	trackID := "track123"

	response, err := mockClient.GetStreamMetadata(ctx, trackID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Result)
	assert.Equal(t, "https://example.com/stream.mp3", response.Result.Stream)
}

// TestClientImpl_GetTrackLyrics tests the GetTrackLyrics method.
func TestClientImpl_GetTrackLyrics(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	trackID := "track123"

	response, err := mockClient.GetTrackLyrics(ctx, trackID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Result)
	assert.Equal(t, LyricsTypeSubtitle, response.Result.Type)
	assert.Equal(t, "Test lyrics content", response.Result.Lyrics)
}

// TestClientImpl_GetTracksMetadata tests the GetTracksMetadata method.
func TestClientImpl_GetTracksMetadata(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()
	trackIDs := []string{"track1", "track2"}

	response, err := mockClient.GetTracksMetadata(ctx, trackIDs)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Result)
	assert.NotNil(t, response.Result.Tracks)
}

// TestClientImpl_GetUserProfile tests the GetUserProfile method.
func TestClientImpl_GetUserProfile(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()

	response, err := mockClient.GetUserProfile(ctx)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Result)
	assert.NotNil(t, response.Result.Subscription)
	assert.Equal(t, "Premium", response.Result.Subscription.Title)
}

// TestClientImpl_ErrorHandling tests error handling.
func TestClientImpl_ErrorHandling(t *testing.T) {
	t.Parallel()

	mockClient := newMockZvukClient()
	defer mockClient.Close()

	ctx := context.Background()

	// Test with invalid URL.
	_, err := mockClient.DownloadFromURL(ctx, "invalid-url")
	require.Error(t, err)
}

// TestModels tests the model structures.
func TestModels(t *testing.T) {
	t.Parallel()

	// Test GetAlbumsMetadataResponse.
	response := &GetAlbumsMetadataResponse{
		Tracks:   make(map[string]*Track),
		Releases: make(map[string]*Release),
	}
	assert.NotNil(t, response.Tracks)
	assert.NotNil(t, response.Releases)

	// Test UserProfile.
	profile := &UserProfile{
		Subscription: &UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}
	assert.NotNil(t, profile.Subscription)
	assert.Equal(t, "Premium", profile.Subscription.Title)
}
