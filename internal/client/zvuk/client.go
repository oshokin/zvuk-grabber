package zvuk

//go:generate $MOCKGEN -source=client.go -destination=mocks/client_mock.go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/machinebox/graphql"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	http_transport "github.com/oshokin/zvuk-grabber/internal/transport/http"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Client defines the interface for interacting with Zvuk's API.
//

type Client interface {
	// DownloadFromURL downloads content from the specified URL.
	DownloadFromURL(ctx context.Context, url string) (io.ReadCloser, error)
	// FetchTrack fetches track data from the specified URL.
	FetchTrack(ctx context.Context, trackURL string) (io.ReadCloser, int64, error)
	// GetAlbumsMetadata retrieves metadata for the specified album IDs.
	GetAlbumsMetadata(ctx context.Context, releaseIDs []string, withTracks bool) (*GetAlbumsMetadataResponse, error)
	// GetAlbumURL constructs the URL for a specific album.
	GetAlbumURL(releaseID string) (string, error)
	// GetArtistReleaseIDs retrieves release IDs for a specific artist.
	GetArtistReleaseIDs(ctx context.Context, artistID string, offset int, limit int) ([]string, error)
	// GetBaseURL returns the base URL of the Zvuk API.
	GetBaseURL() string
	// GetLabelsMetadata retrieves metadata for the specified label IDs.
	GetLabelsMetadata(ctx context.Context, labelIDs []string) (map[string]*Label, error)
	// GetPlaylistsMetadata retrieves metadata for the specified playlist IDs.
	GetPlaylistsMetadata(ctx context.Context, playlistIDs []string) (*GetPlaylistsMetadataResponse, error)
	// GetStreamMetadata retrieves streaming metadata for a specific track and quality.
	GetStreamMetadata(ctx context.Context, trackID, quality string) (*StreamMetadata, error)
	// GetTrackLyrics retrieves lyrics for a specific track.
	GetTrackLyrics(ctx context.Context, trackID string) (*Lyrics, error)
	// GetTracksMetadata retrieves metadata for the specified track IDs.
	GetTracksMetadata(ctx context.Context, trackIDs []string) (map[string]*Track, error)
	// GetUserProfile retrieves the user's profile information.
	GetUserProfile(ctx context.Context) (*UserProfile, error)
}

// ClientImpl implements the Client interface for interacting with Zvuk's API.
type ClientImpl struct {
	// cfg contains the application configuration.
	cfg *config.Config
	// baseURL is the base URL for API requests.
	baseURL string
	// httpClient is the HTTP client for making requests.
	httpClient *http.Client
	// graphQLClient is the GraphQL client for making queries.
	graphQLClient *graphql.Client
}

const (
	// zvukAPIGraphQLURI is the URI path for GraphQL API endpoint.
	zvukAPIGraphQLURI = "api/v1/graphql"
	// zvukAPILabelURI is the URI path for label metadata endpoint.
	zvukAPILabelURI = "api/tiny/labels"
	// zvukAPILyricsURI is the URI path for lyrics endpoint.
	zvukAPILyricsURI = "api/tiny/lyrics"
	// zvukAPIPlaylistURI is the URI path for playlist metadata endpoint.
	zvukAPIPlaylistURI = "api/tiny/playlists"
	// zvukAPIReleaseMetadataURI is the URI path for release metadata endpoint.
	zvukAPIReleaseMetadataURI = "api/tiny/releases"
	// zvukAPIReleaseURIPath is the URI path component for releases.
	zvukAPIReleaseURIPath = "releases"
	// zvukAPIStreamMetadataURI is the URI path for stream metadata endpoint.
	zvukAPIStreamMetadataURI = "api/tiny/track/stream"
	// zvukAPITrackURI is the URI path for track metadata endpoint.
	zvukAPITrackURI = "api/tiny/tracks"
	// zvukAPIUserProfileURI is the URI path for user profile endpoint.
	zvukAPIUserProfileURI = "api/v2/tiny/profile"
)

// Static error definitions for better error handling.
var (
	// ErrUnexpectedHTTPStatus indicates an unexpected HTTP status code was received.
	ErrUnexpectedHTTPStatus = errors.New("unexpected HTTP status")
	// ErrArtistNotFound indicates that the requested artist was not found.
	ErrArtistNotFound = errors.New("artist not found")
	// ErrUnexpectedArtistResponseFormat indicates an unexpected artist API response format.
	ErrUnexpectedArtistResponseFormat = errors.New("unexpected artist response format")
	// ErrUnexpectedReleasesResponseFormat indicates an unexpected releases API response format.
	ErrUnexpectedReleasesResponseFormat = errors.New("unexpected releases response format")
	// ErrFailedToFetchStreamMetadata indicates failure to fetch stream metadata after all retry attempts.
	ErrFailedToFetchStreamMetadata = errors.New("failed to fetch stream metadata after retries")
)

// NewClient creates and returns a new instance of ClientImpl.
// It initializes the HTTP and GraphQL clients with the provided configuration.
func NewClient(cfg *config.Config) (Client, error) {
	// Create a cookie jar to manage cookies for the HTTP client.
	cookies, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Parse the base URL for Zvuk's API.
	baseURL, err := url.Parse(cfg.ZvukBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	}

	// Set the authentication cookie.
	cookie := &http.Cookie{
		Name:  "auth",
		Value: cfg.AuthToken,
	}
	cookies.SetCookies(baseURL, []*http.Cookie{cookie})

	// Initialize the HTTP client with custom transport and timeout.
	httpClient := &http.Client{
		Transport: http_transport.NewUserAgentInjector(
			http_transport.NewLogTransport(http.DefaultTransport, 0),
			utils.NewSimpleUserAgentProvider(http_transport.DefaultUserAgent)),
		Jar:     cookies,
		Timeout: http_transport.DefaultTimeout,
	}

	// Initialize the GraphQL client.
	graphQLURL := baseURL.JoinPath(zvukAPIGraphQLURI)
	graphqlClient := graphql.NewClient(graphQLURL.String(), graphql.WithHTTPClient(httpClient))

	// Create and return the ClientImpl instance.
	client := &ClientImpl{
		cfg:           cfg,
		baseURL:       baseURL.String(),
		httpClient:    httpClient,
		graphQLClient: graphqlClient,
	}

	return client, nil
}

// DownloadFromURL downloads content from the specified URL.
func (c *ClientImpl) DownloadFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		response.Body.Close() //nolint:gosec // Error on close is not critical here.

		return nil, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	return response.Body, nil
}

// FetchTrack fetches track data from the specified URL.
func (c *ClientImpl) FetchTrack(ctx context.Context, trackURL string) (io.ReadCloser, int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, trackURL, http.NoBody)
	if err != nil {
		return nil, 0, err
	}

	// Add a Range header to request partial content.
	request.Header.Add("Range", "bytes=0-")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		response.Body.Close() //nolint:gosec // Error on close is not critical here.

		return nil, 0, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	return response.Body, response.ContentLength, nil
}

// GetAlbumsMetadata retrieves metadata for the specified album IDs.
func (c *ClientImpl) GetAlbumsMetadata(
	ctx context.Context,
	releaseIDs []string,
	withTracks bool,
) (*GetAlbumsMetadataResponse, error) {
	query := url.Values{}
	if withTracks {
		query.Set("include", "track")
	}

	result, err := c.getEntitiesMetadata(ctx, zvukAPIReleaseMetadataURI, releaseIDs, query)
	if err != nil {
		return nil, err
	}

	return &GetAlbumsMetadataResponse{
		Tracks:   result.Tracks,
		Releases: result.Releases,
	}, nil
}

// GetAlbumURL constructs the URL for a specific album.
func (c *ClientImpl) GetAlbumURL(releaseID string) (string, error) {
	return url.JoinPath(c.baseURL, zvukAPIReleaseURIPath, releaseID)
}

// GetArtistReleaseIDs retrieves release IDs for a specific artist.
func (c *ClientImpl) GetArtistReleaseIDs(ctx context.Context, artistID string, offset, limit int) ([]string, error) {
	graphqlRequest := graphql.NewRequest(`
		query getArtistReleases($id: ID!, $limit: Int!, $offset: Int!) { 
			getArtists(ids: [$id]) { 
				__typename 
				releases(limit: $limit, offset: $offset) { 
					__typename 
					...ReleaseGqlFragment 
				} 
			} 
		} 
		fragment ReleaseGqlFragment on Release { 
			id 
		}
	`)

	graphqlRequest.Header.Add("X-Auth-Token", c.cfg.AuthToken)
	graphqlRequest.Var("id", artistID)
	graphqlRequest.Var("offset", offset)
	graphqlRequest.Var("limit", limit)

	var graphQLResponse map[string]any
	if err := c.graphQLClient.Run(ctx, graphqlRequest, &graphQLResponse); err != nil {
		return nil, err
	}

	// Navigate the response map manually.
	data, ok := graphQLResponse["getArtists"].([]any)
	if !ok || len(data) == 0 {
		return nil, ErrArtistNotFound
	}

	artist, ok := data[0].(map[string]any)
	if !ok {
		return nil, ErrUnexpectedArtistResponseFormat
	}

	releases, ok := artist["releases"].([]any)
	if !ok {
		return nil, ErrUnexpectedReleasesResponseFormat
	}

	releaseIDs := make([]string, 0, len(releases))

	for _, r := range releases {
		release, hasExpectedFormat := r.(map[string]any)
		if !hasExpectedFormat {
			continue
		}

		if id, exists := release["id"].(string); exists && id != "" {
			releaseIDs = append(releaseIDs, id)
		}
	}

	return releaseIDs, nil
}

// GetBaseURL returns the base URL of the Zvuk API.
func (c *ClientImpl) GetBaseURL() string {
	return c.baseURL
}

// GetLabelsMetadata retrieves metadata for the specified label IDs.
func (c *ClientImpl) GetLabelsMetadata(ctx context.Context, labelIDs []string) (map[string]*Label, error) {
	result, err := c.getEntitiesMetadata(ctx, zvukAPILabelURI, labelIDs, nil)
	if err != nil {
		return nil, err
	}

	return result.Labels, nil
}

// GetPlaylistsMetadata retrieves metadata for the specified playlist IDs.
func (c *ClientImpl) GetPlaylistsMetadata(
	ctx context.Context,
	playlistIDs []string,
) (*GetPlaylistsMetadataResponse, error) {
	query := url.Values{}
	query.Set("include", "track")

	result, err := c.getEntitiesMetadata(ctx, zvukAPIPlaylistURI, playlistIDs, query)
	if err != nil {
		return nil, err
	}

	return &GetPlaylistsMetadataResponse{
		Tracks:    result.Tracks,
		Playlists: result.Playlists,
	}, nil
}

// GetStreamMetadata retrieves streaming metadata for a specific track and quality.
func (c *ClientImpl) GetStreamMetadata(ctx context.Context, trackID, quality string) (*StreamMetadata, error) {
	query := url.Values{}
	query.Set("id", trackID)
	query.Set("quality", quality)

	var result *StreamMetadata

	for i := range c.cfg.RetryAttemptsCount {
		response, statusCode, err := fetchJSONWithQuery[GetStreamMetadataResponse](
			c,
			ctx,
			zvukAPIStreamMetadataURI,
			query,
		)
		if err == nil {
			result = response.Result

			break
		}

		// Retry on specific HTTP status codes.
		if i < c.cfg.RetryAttemptsCount-1 && statusCode == http.StatusTeapot {
			logger.Infof(ctx, "Retrying due to error (%d attempts left): %v", c.cfg.RetryAttemptsCount-i-1, err)
			utils.RandomPause(c.cfg.ParsedMaxRetryPause, c.cfg.ParsedMaxRetryPause)

			continue
		}

		return nil, err
	}

	if result == nil {
		return nil, ErrFailedToFetchStreamMetadata
	}

	return result, nil
}

// GetTrackLyrics retrieves lyrics for a specific track.
func (c *ClientImpl) GetTrackLyrics(ctx context.Context, trackID string) (*Lyrics, error) {
	query := url.Values{}
	query.Set("track_id", trackID)

	response, _, err := fetchJSONWithQuery[GetLyricsResponse](c, ctx, zvukAPILyricsURI, query)
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

// GetTracksMetadata retrieves metadata for the specified track IDs.
func (c *ClientImpl) GetTracksMetadata(ctx context.Context, trackIDs []string) (map[string]*Track, error) {
	result, err := c.getEntitiesMetadata(ctx, zvukAPITrackURI, trackIDs, nil)
	if err != nil {
		return nil, err
	}

	return result.Tracks, nil
}

// GetUserProfile retrieves the user's profile information.
func (c *ClientImpl) GetUserProfile(ctx context.Context) (*UserProfile, error) {
	response, _, err := fetchJSON[GetUserProfileResponse](c, ctx, zvukAPIUserProfileURI)
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

func (c *ClientImpl) getEntitiesMetadata(
	ctx context.Context,
	entityURI string,
	entityIDs []string,
	query url.Values,
) (*Metadata, error) {
	if len(query) == 0 {
		query = url.Values{}
	}

	query.Set("ids", strings.Join(entityIDs, ","))

	response, _, err := fetchJSONWithQuery[GetMetadataResponse](c, ctx, entityURI, query)
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

//nolint:revive // has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSON[T any](c *ClientImpl, ctx context.Context, uri string) (*T, int, error) {
	return fetchJSONWithQuery[T](c, ctx, uri, nil)
}

//nolint:revive // has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSONWithQuery[T any](c *ClientImpl, ctx context.Context, uri string, query url.Values) (*T, int, error) {
	route, err := url.JoinPath(c.baseURL, uri)
	if err != nil {
		return nil, 0, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, route, http.NoBody)
	if err != nil {
		return nil, 0, err
	}

	if query != nil {
		request.URL.RawQuery = query.Encode()
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	var result T
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, response.StatusCode, err
	}

	return &result, response.StatusCode, nil
}
