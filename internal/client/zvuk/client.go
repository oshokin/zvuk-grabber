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

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/machinebox/graphql"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	http_transport "github.com/oshokin/zvuk-grabber/internal/transport/http"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Client defines the interface for interacting with Zvuk's API.
type Client interface {
	// DownloadFromURL downloads content from the specified URL.
	DownloadFromURL(ctx context.Context, url string) (io.ReadCloser, error)
	// FetchTrack fetches track data from the specified URL.
	FetchTrack(ctx context.Context, trackURL string) (*FetchTrackResult, error)
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
	// labelsCache caches label metadata to reduce duplicate API calls for the same labels.
	labelsCache *lru.Cache[string, *Label]
	// albumsCache caches album metadata to reduce duplicate API calls for the same albums.
	albumsCache *lru.Cache[string, *Release]
	// tracksCache caches track metadata to reduce duplicate API calls for the same tracks.
	tracksCache *lru.Cache[string, *Track]
	// playlistsCache caches playlist metadata to reduce duplicate API calls for the same playlists.
	playlistsCache *lru.Cache[string, *Playlist]
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

const (
	// labelsCacheSize defines the maximum number of label entries to cache.
	// Approximately 500 unique labels exist globally across all music.
	labelsCacheSize = 500
	// albumsCacheSize defines the maximum number of album entries to cache.
	// Sized to hold recent albums accessed during typical usage.
	albumsCacheSize = 5000
	// tracksCacheSize defines the maximum number of track entries to cache.
	// Sized to hold recently accessed tracks.
	tracksCacheSize = 10000
	// playlistsCacheSize defines the maximum number of playlist entries to cache.
	// Playlists don't change frequently, so we cache them.
	playlistsCacheSize = 2000
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

	// Initialize LRU caches for metadata to reduce redundant API calls.
	labelsCache, err := lru.New[string, *Label](labelsCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create labels cache: %w", err)
	}

	albumsCache, err := lru.New[string, *Release](albumsCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create albums cache: %w", err)
	}

	tracksCache, err := lru.New[string, *Track](tracksCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracks cache: %w", err)
	}

	playlistsCache, err := lru.New[string, *Playlist](playlistsCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlists cache: %w", err)
	}

	// Create and return the ClientImpl instance.
	client := &ClientImpl{
		cfg:            cfg,
		baseURL:        baseURL.String(),
		httpClient:     httpClient,
		graphQLClient:  graphqlClient,
		labelsCache:    labelsCache,
		albumsCache:    albumsCache,
		tracksCache:    tracksCache,
		playlistsCache: playlistsCache,
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
func (c *ClientImpl) FetchTrack(ctx context.Context, trackURL string) (*FetchTrackResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, trackURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	// Add a Range header to request partial content.
	request.Header.Add("Range", "bytes=0-")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		response.Body.Close() //nolint:gosec // Error on close is not critical here.

		return nil, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	return &FetchTrackResult{
		Body:       response.Body,
		TotalBytes: response.ContentLength,
	}, nil
}

// GetAlbumsMetadata retrieves metadata for the specified album IDs.
// Uses an LRU cache to avoid redundant API calls for the same albums.
// Note: Only caches albums without tracks to avoid stale track data.
func (c *ClientImpl) GetAlbumsMetadata(
	ctx context.Context,
	releaseIDs []string,
	withTracks bool,
) (*GetAlbumsMetadataResponse, error) {
	if withTracks {
		return c.getAlbumsMetadataWithTracks(ctx, releaseIDs)
	}

	return c.getAlbumsMetadataFromCache(ctx, releaseIDs)
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
// Uses an LRU cache to avoid redundant API calls for the same labels.
func (c *ClientImpl) GetLabelsMetadata(ctx context.Context, labelIDs []string) (map[string]*Label, error) {
	result := make(map[string]*Label)
	uncachedIDs := make([]string, 0, len(labelIDs))

	// Check cache first for each label ID.
	for _, id := range labelIDs {
		if cached, ok := c.labelsCache.Get(id); ok {
			result[id] = cached
			logger.Debugf(ctx, "Label cache hit for ID: %s", id)
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	// If all labels were cached, return immediately.
	if len(uncachedIDs) == 0 {
		return result, nil
	}

	// Fetch uncached labels from API.
	logger.Debugf(ctx, "Fetching %d uncached labels from API", len(uncachedIDs))

	metadata, err := c.getEntitiesMetadata(ctx, zvukAPILabelURI, uncachedIDs, nil)
	if err != nil {
		return nil, err
	}

	// Store fetched labels in cache and add to result.
	for id, label := range metadata.Labels {
		c.labelsCache.Add(id, label)
		result[id] = label
	}

	return result, nil
}

// GetPlaylistsMetadata retrieves metadata for the specified playlist IDs.
// Uses an LRU cache to avoid redundant API calls for the same playlists.
func (c *ClientImpl) GetPlaylistsMetadata(
	ctx context.Context,
	playlistIDs []string,
) (*GetPlaylistsMetadataResponse, error) {
	playlists := make(map[string]*Playlist)
	tracks := make(map[string]*Track)
	uncachedIDs := make([]string, 0, len(playlistIDs))

	// Check cache first for each playlist ID.
	for _, id := range playlistIDs {
		if cached, ok := c.playlistsCache.Get(id); ok {
			playlists[id] = cached
			logger.Debugf(ctx, "Playlist cache hit for ID: %s", id)
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	// If all playlists were cached, return immediately.
	// Note: Tracks are not cached from playlist response to ensure fresh track data.
	if len(uncachedIDs) == 0 {
		return &GetPlaylistsMetadataResponse{
			Tracks:    tracks,
			Playlists: playlists,
		}, nil
	}

	// Fetch uncached playlists from API.
	query := url.Values{}
	query.Set("include", "track")

	logger.Debugf(ctx, "Fetching %d uncached playlists from API", len(uncachedIDs))

	result, err := c.getEntitiesMetadata(ctx, zvukAPIPlaylistURI, uncachedIDs, query)
	if err != nil {
		return nil, err
	}

	// Store fetched playlists in cache and add to result.
	for id, playlist := range result.Playlists {
		c.playlistsCache.Add(id, playlist)
		playlists[id] = playlist
	}

	// Add tracks from the API response.
	for id, track := range result.Tracks {
		tracks[id] = track
	}

	return &GetPlaylistsMetadataResponse{
		Tracks:    tracks,
		Playlists: playlists,
	}, nil
}

// GetStreamMetadata retrieves streaming metadata for a specific track and quality.
func (c *ClientImpl) GetStreamMetadata(ctx context.Context, trackID, quality string) (*StreamMetadata, error) {
	query := url.Values{}
	query.Set("id", trackID)
	query.Set("quality", quality)

	var result *StreamMetadata

	for i := range c.cfg.RetryAttemptsCount {
		fetchResult, err := fetchJSONWithQuery[GetStreamMetadataResponse](
			c,
			ctx,
			zvukAPIStreamMetadataURI,
			query,
		)
		if err == nil {
			result = fetchResult.Data.Result

			break
		}

		// Retry on specific HTTP status codes.
		if i < c.cfg.RetryAttemptsCount-1 && fetchResult != nil && fetchResult.StatusCode == http.StatusTeapot {
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

	result, err := fetchJSONWithQuery[GetLyricsResponse](c, ctx, zvukAPILyricsURI, query)
	if err != nil {
		return nil, err
	}

	return result.Data.Result, nil
}

// GetTracksMetadata retrieves metadata for the specified track IDs.
// Uses an LRU cache to avoid redundant API calls for the same tracks.
func (c *ClientImpl) GetTracksMetadata(ctx context.Context, trackIDs []string) (map[string]*Track, error) {
	result := make(map[string]*Track)
	uncachedIDs := make([]string, 0, len(trackIDs))

	// Check cache first for each track ID.
	for _, id := range trackIDs {
		if cached, ok := c.tracksCache.Get(id); ok {
			result[id] = cached
			logger.Debugf(ctx, "Track cache hit for ID: %s", id)
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	// If all tracks were cached, return immediately.
	if len(uncachedIDs) == 0 {
		return result, nil
	}

	// Fetch uncached tracks from API.
	logger.Debugf(ctx, "Fetching %d uncached tracks from API", len(uncachedIDs))

	metadata, err := c.getEntitiesMetadata(ctx, zvukAPITrackURI, uncachedIDs, nil)
	if err != nil {
		return nil, err
	}

	// Store fetched tracks in cache and add to result.
	for id, track := range metadata.Tracks {
		c.tracksCache.Add(id, track)
		result[id] = track
	}

	return result, nil
}

// GetUserProfile retrieves the user's profile information.
func (c *ClientImpl) GetUserProfile(ctx context.Context) (*UserProfile, error) {
	result, err := fetchJSON[GetUserProfileResponse](c, ctx, zvukAPIUserProfileURI)
	if err != nil {
		return nil, err
	}

	return result.Data.Result, nil
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

	result, err := fetchJSONWithQuery[GetMetadataResponse](c, ctx, entityURI, query)
	if err != nil {
		return nil, err
	}

	return result.Data.Result, nil
}

// getAlbumsMetadataWithTracks fetches album metadata including tracks without caching.
// This ensures track data is always fresh from the API.
func (c *ClientImpl) getAlbumsMetadataWithTracks(
	ctx context.Context,
	releaseIDs []string,
) (*GetAlbumsMetadataResponse, error) {
	query := url.Values{}
	query.Set("include", "track")

	result, err := c.getEntitiesMetadata(ctx, zvukAPIReleaseMetadataURI, releaseIDs, query)
	if err != nil {
		return nil, err
	}

	return &GetAlbumsMetadataResponse{
		Tracks:   result.Tracks,
		Releases: result.Releases,
	}, nil
}

// getAlbumsMetadataFromCache fetches album metadata using cache-first strategy.
// Returns cached albums when available and only fetches missing ones from the API.
func (c *ClientImpl) getAlbumsMetadataFromCache(
	ctx context.Context,
	releaseIDs []string,
) (*GetAlbumsMetadataResponse, error) {
	releases := make(map[string]*Release)
	uncachedIDs := make([]string, 0, len(releaseIDs))

	// Check cache first for each album ID.
	for _, id := range releaseIDs {
		if cached, ok := c.albumsCache.Get(id); ok {
			releases[id] = cached
			logger.Debugf(ctx, "Album cache hit for ID: %s", id)
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	// If all albums were cached, return immediately.
	if len(uncachedIDs) == 0 {
		return &GetAlbumsMetadataResponse{
			Tracks:   nil,
			Releases: releases,
		}, nil
	}

	// Fetch uncached albums from API.
	logger.Debugf(ctx, "Fetching %d uncached albums from API", len(uncachedIDs))

	result, err := c.getEntitiesMetadata(ctx, zvukAPIReleaseMetadataURI, uncachedIDs, nil)
	if err != nil {
		return nil, err
	}

	// Store fetched albums in cache and add to result.
	for id, release := range result.Releases {
		c.albumsCache.Add(id, release)
		releases[id] = release
	}

	return &GetAlbumsMetadataResponse{
		Tracks:   nil,
		Releases: releases,
	}, nil
}

//nolint:revive // Has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSON[T any](c *ClientImpl, ctx context.Context, uri string) (*FetchJSONResult[T], error) {
	return fetchJSONWithQuery[T](c, ctx, uri, nil)
}

//
//nolint:revive // Has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSONWithQuery[T any](
	c *ClientImpl,
	ctx context.Context,
	uri string,
	query url.Values,
) (*FetchJSONResult[T], error) {
	route, err := url.JoinPath(c.baseURL, uri)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, route, http.NoBody)
	if err != nil {
		return nil, err
	}

	if query != nil {
		request.URL.RawQuery = query.Encode()
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return &FetchJSONResult[T]{
			Data:       nil,
			StatusCode: response.StatusCode,
		}, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	var result T
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return &FetchJSONResult[T]{
			Data:       nil,
			StatusCode: response.StatusCode,
		}, err
	}

	return &FetchJSONResult[T]{
		Data:       &result,
		StatusCode: response.StatusCode,
	}, nil
}
