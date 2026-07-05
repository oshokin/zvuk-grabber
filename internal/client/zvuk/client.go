package zvuk

//go:generate $MOCKGEN -source=client.go -destination=mocks/client_mock.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/machinebox/graphql"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
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
	// GetAudiobooksMetadata retrieves metadata for the specified audiobook IDs.
	GetAudiobooksMetadata(ctx context.Context, audiobookIDs []string) (*GetAudiobooksMetadataResponse, error)
	// GetPodcastsMetadata retrieves metadata for the specified podcast IDs.
	GetPodcastsMetadata(ctx context.Context, podcastIDs []string) (*GetPodcastsMetadataResponse, error)
	// GetBaseURL returns the base URL of the Zvuk API.
	GetBaseURL() string
	// GetLabelsMetadata retrieves metadata for the specified label IDs.
	GetLabelsMetadata(ctx context.Context, labelIDs []string) (map[string]*Label, error)
	// GetPlaylistsMetadata retrieves metadata for the specified playlist IDs.
	GetPlaylistsMetadata(ctx context.Context, playlistIDs []string) (*GetPlaylistsMetadataResponse, error)
	// GetStreamMetadata retrieves streaming metadata for a specific track and quality.
	GetStreamMetadata(ctx context.Context, trackID, quality string) (*StreamMetadata, error)
	// GetStreamQualities retrieves streaming metadata for audiobook chapters and podcasts episodes.
	GetStreamQualities(ctx context.Context, streamIDs []string) (map[string]*StreamQualities, error)
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
	// audiobooksCache caches audiobook metadata to reduce duplicate API calls for the same audiobooks.
	audiobooksCache *lru.Cache[string, *Audiobook]
	// podcastsCache caches podcast metadata to reduce duplicate API calls for the same podcasts.
	podcastsCache *lru.Cache[string, *Podcast]
	// streamMetadataRetryEngine retries transient stream metadata failures.
	streamMetadataRetryEngine *retry.Engine
}

// graphQLCollectionResult holds a GraphQL collection item and its child tracks.
type graphQLCollectionResult[T any] struct {
	// item is the parsed collection metadata.
	item *T
	// tracks is a map of child track ID to track metadata.
	tracks map[string]*Track
}

// metadataCaches groups per-entity LRU caches used by ClientImpl.
type metadataCaches struct {
	// labels stores label metadata cache.
	labels *lru.Cache[string, *Label]
	// albums stores album metadata cache.
	albums *lru.Cache[string, *Release]
	// tracks stores track metadata cache.
	tracks *lru.Cache[string, *Track]
	// playlists stores playlist metadata cache.
	playlists *lru.Cache[string, *Playlist]
	// audiobooks stores audiobook metadata cache.
	audiobooks *lru.Cache[string, *Audiobook]
	// podcasts stores podcast metadata cache.
	podcasts *lru.Cache[string, *Podcast]
}

// retryableStreamMetadataError marks stream metadata failures that should be retried.
type retryableStreamMetadataError struct {
	// err is the underlying transport/API error.
	err error
}

// Error returns the underlying error string.
func (e *retryableStreamMetadataError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}

	return e.err.Error()
}

// Unwrap exposes the underlying error.
func (e *retryableStreamMetadataError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

// newStreamMetadataRetryEngine builds a reusable retry engine for GetStreamMetadata.
func newStreamMetadataRetryEngine(cfg *config.Config) (*retry.Engine, error) {
	maxRetries := uint64(1)
	retryClassifier := func(err error) bool {
		var retryableErr *retryableStreamMetadataError
		return errors.As(err, &retryableErr)
	}

	if cfg.RetryAttemptsCount <= 1 {
		retryClassifier = func(error) bool {
			return false
		}
	} else {
		maxRetries = uint64(cfg.RetryAttemptsCount - 1)
	}

	return retry.NewEngine(&retry.EngineConfig{
		MaxRetries:  maxRetries,
		DelayPolicy: retry.NewRandomRangePolicy(cfg.ParsedMinRetryPause, cfg.ParsedMaxRetryPause),
		IsRetryable: retryClassifier,
	})
}

// newClientCookieJar creates a cookie jar preloaded with the Zvuk auth token.
func newClientCookieJar(baseURL *url.URL, authToken string) (*cookiejar.Jar, error) {
	cookies, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	cookies.SetCookies(baseURL, []*http.Cookie{
		{
			Name:     "auth",
			Value:    authToken,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
	})

	return cookies, nil
}

// newClientMetadataCaches initializes the in-memory metadata caches used by the client.
func newClientMetadataCaches() (*metadataCaches, error) {
	labels, err := newMetadataCache[*Label](labelsCacheSize, "labels")
	if err != nil {
		return nil, err
	}

	albums, err := newMetadataCache[*Release](albumsCacheSize, "albums")
	if err != nil {
		return nil, err
	}

	tracks, err := newMetadataCache[*Track](tracksCacheSize, "tracks")
	if err != nil {
		return nil, err
	}

	playlists, err := newMetadataCache[*Playlist](playlistsCacheSize, "playlists")
	if err != nil {
		return nil, err
	}

	audiobooks, err := newMetadataCache[*Audiobook](audiobooksCacheSize, "audiobooks")
	if err != nil {
		return nil, err
	}

	podcasts, err := newMetadataCache[*Podcast](podcastsCacheSize, "podcasts")
	if err != nil {
		return nil, err
	}

	return &metadataCaches{
		labels:     labels,
		albums:     albums,
		tracks:     tracks,
		playlists:  playlists,
		audiobooks: audiobooks,
		podcasts:   podcasts,
	}, nil
}

// newMetadataCache creates an LRU cache for entity metadata.
func newMetadataCache[T any](size int, entityName string) (*lru.Cache[string, T], error) {
	cache, err := lru.New[string, T](size)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s cache: %w", entityName, err)
	}

	return cache, nil
}

// splitCachedMetadata separates cached and uncached entity IDs.
func splitCachedMetadata[T any](
	ctx context.Context,
	ids []string,
	cache *lru.Cache[string, T],
	entityName string,
) (map[string]T, []string) {
	cachedEntities := make(map[string]T)
	uncachedIDs := make([]string, 0, len(ids))

	for _, id := range ids {
		if cached, ok := cache.Get(id); ok {
			cachedEntities[id] = cached
			logger.Debugf(ctx, "%s cache hit for ID: %s", entityName, id)
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	return cachedEntities, uncachedIDs
}

// storeCachedMetadata adds fetched entities to the cache and destination map.
func storeCachedMetadata[T any](cache *lru.Cache[string, T], destination, source map[string]T) {
	for id, entity := range source {
		cache.Add(id, entity)
		destination[id] = entity
	}
}

// fetchCachedMetadata fetches uncached entities and merges them with cached ones.
func fetchCachedMetadata[T any](
	ctx context.Context,
	ids []string,
	cache *lru.Cache[string, T],
	entityName string,
	source string,
	fetch func(context.Context, []string) (map[string]T, error),
) (map[string]T, error) {
	entities, uncachedIDs := splitCachedMetadata(ctx, ids, cache, entityName)
	if len(uncachedIDs) == 0 {
		return entities, nil
	}

	logger.Debugf(ctx, "Fetching %d uncached %ss from %s", len(uncachedIDs), strings.ToLower(entityName), source)

	fetched, err := fetch(ctx, uncachedIDs)
	if err != nil {
		return nil, err
	}

	storeCachedMetadata(cache, entities, fetched)

	return entities, nil
}

// fetchCachedCollectionMetadata fetches uncached collection entities and their tracks.
func fetchCachedCollectionMetadata[T any](
	ctx context.Context,
	ids []string,
	cache *lru.Cache[string, T],
	entityName string,
	source string,
	fetch func(context.Context, []string) (map[string]T, map[string]*Track, error),
) (map[string]T, map[string]*Track, error) {
	entities, uncachedIDs := splitCachedMetadata(ctx, ids, cache, entityName)

	tracks := make(map[string]*Track)
	if len(uncachedIDs) == 0 {
		return entities, tracks, nil
	}

	logger.Debugf(ctx, "Fetching %d uncached %ss from %s", len(uncachedIDs), strings.ToLower(entityName), source)

	fetchedEntities, fetchedTracks, err := fetch(ctx, uncachedIDs)
	if err != nil {
		return nil, nil, err
	}

	storeCachedMetadata(cache, entities, fetchedEntities)
	maps.Copy(tracks, fetchedTracks)

	return entities, tracks, nil
}

// fetchGraphQLCollections fetches multiple GraphQL collections by ID.
func fetchGraphQLCollections[T any](
	ctx context.Context,
	ids []string,
	entityName string,
	fetch func(context.Context, string) (*graphQLCollectionResult[T], error),
) (map[string]*T, map[string]*Track, error) {
	entities := make(map[string]*T, len(ids))
	tracks := make(map[string]*Track)

	for _, id := range ids {
		result, err := fetch(ctx, id)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch %s %s: %w", strings.ToLower(entityName), id, err)
		}

		entities[id] = result.item
		maps.Copy(tracks, result.tracks)
	}

	return entities, tracks, nil
}

// NewClient creates and returns a new instance of ClientImpl.
// It initializes the HTTP and GraphQL clients with the provided configuration.
func NewClient(cfg *config.Config) (Client, error) {
	if strings.TrimSpace(cfg.ZvukAuthToken) == "" {
		return nil, config.ErrEmptyZvukAuthToken
	}

	// Parse the base URL for Zvuk's API.
	baseURL, err := url.Parse(cfg.ZvukBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	}

	cookies, err := newClientCookieJar(baseURL, cfg.ZvukAuthToken)
	if err != nil {
		return nil, err
	}

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

	caches, err := newClientMetadataCaches()
	if err != nil {
		return nil, err
	}

	streamMetadataRetryEngine, err := newStreamMetadataRetryEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream metadata retry engine: %w", err)
	}

	// Create and return the ClientImpl instance.
	client := &ClientImpl{
		cfg:                       cfg,
		baseURL:                   baseURL.String(),
		httpClient:                httpClient,
		graphQLClient:             graphqlClient,
		labelsCache:               caches.labels,
		albumsCache:               caches.albums,
		tracksCache:               caches.tracks,
		playlistsCache:            caches.playlists,
		audiobooksCache:           caches.audiobooks,
		podcastsCache:             caches.podcasts,
		streamMetadataRetryEngine: streamMetadataRetryEngine,
	}

	return client, nil
}

// DownloadFromURL downloads content from the specified URL.
func (c *ClientImpl) DownloadFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	body, _, err := c.openDownloadResponse(ctx, url, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// FetchTrack fetches track data from the specified URL.
func (c *ClientImpl) FetchTrack(ctx context.Context, trackURL string) (*FetchTrackResult, error) {
	body, totalBytes, err := c.openDownloadResponse(ctx, trackURL, func(request *http.Request) {
		request.Header.Add("Range", "bytes=0-")
	}, http.StatusOK, http.StatusPartialContent)
	if err != nil {
		return nil, err
	}

	return &FetchTrackResult{
		Body:       body,
		TotalBytes: totalBytes,
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

// GetBaseURL returns the base URL of the Zvuk API.
func (c *ClientImpl) GetBaseURL() string {
	return c.baseURL
}

// GetLabelsMetadata retrieves metadata for the specified label IDs.
// Uses an LRU cache to avoid redundant API calls for the same labels.
func (c *ClientImpl) GetLabelsMetadata(ctx context.Context, labelIDs []string) (map[string]*Label, error) {
	return fetchCachedMetadata(
		ctx,
		labelIDs,
		c.labelsCache,
		"Label",
		"API",
		func(ctx context.Context, ids []string) (map[string]*Label, error) {
			metadata, err := c.getEntitiesMetadata(ctx, zvukAPILabelURI, ids, nil)
			if err != nil {
				return nil, err
			}

			return metadata.Labels, nil
		},
	)
}

// GetPlaylistsMetadata retrieves metadata for the specified playlist IDs.
// Uses an LRU cache to avoid redundant API calls for the same playlists.
func (c *ClientImpl) GetPlaylistsMetadata(
	ctx context.Context,
	playlistIDs []string,
) (*GetPlaylistsMetadataResponse, error) {
	playlists, tracks, err := fetchCachedCollectionMetadata(
		ctx,
		playlistIDs,
		c.playlistsCache,
		"Playlist",
		"API",
		func(ctx context.Context, ids []string) (map[string]*Playlist, map[string]*Track, error) {
			metadata, fetchErr := c.getEntitiesMetadata(ctx, zvukAPIPlaylistURI, ids, url.Values{"include": {"track"}})
			if fetchErr != nil {
				return nil, nil, fetchErr
			}

			return metadata.Playlists, metadata.Tracks, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &GetPlaylistsMetadataResponse{Tracks: tracks, Playlists: playlists}, nil
}

// GetAudiobooksMetadata retrieves metadata for the specified audiobook IDs.
// Uses an LRU cache to avoid redundant API calls for the same audiobooks.
func (c *ClientImpl) GetAudiobooksMetadata(
	ctx context.Context,
	audiobookIDs []string,
) (*GetAudiobooksMetadataResponse, error) {
	audiobooks, tracks, err := fetchCachedCollectionMetadata(
		ctx,
		audiobookIDs,
		c.audiobooksCache,
		"Audiobook",
		"GraphQL API",
		func(ctx context.Context, ids []string) (map[string]*Audiobook, map[string]*Track, error) {
			return fetchGraphQLCollections(ctx, ids, "Audiobook", c.getAudiobookViaGraphQL)
		},
	)
	if err != nil {
		return nil, err
	}

	return &GetAudiobooksMetadataResponse{Tracks: tracks, Audiobooks: audiobooks}, nil
}

// GetPodcastsMetadata retrieves metadata for the specified podcast IDs.
// Uses an LRU cache to avoid redundant API calls for the same podcasts.
func (c *ClientImpl) GetPodcastsMetadata(
	ctx context.Context,
	podcastIDs []string,
) (*GetPodcastsMetadataResponse, error) {
	podcasts, tracks, err := fetchCachedCollectionMetadata(
		ctx,
		podcastIDs,
		c.podcastsCache,
		"Podcast",
		"GraphQL API",
		func(ctx context.Context, ids []string) (map[string]*Podcast, map[string]*Track, error) {
			return fetchGraphQLCollections(ctx, ids, "Podcast", c.getPodcastViaGraphQL)
		},
	)
	if err != nil {
		return nil, err
	}

	return &GetPodcastsMetadataResponse{Tracks: tracks, Podcasts: podcasts}, nil
}

// GetStreamMetadata retrieves streaming metadata for a specific track and quality.
func (c *ClientImpl) GetStreamMetadata(ctx context.Context, trackID, quality string) (*StreamMetadata, error) {
	query := url.Values{}
	query.Set("id", trackID)
	query.Set("quality", quality)

	engine := c.streamMetadataRetryEngine
	if engine == nil {
		var engineErr error

		engine, engineErr = newStreamMetadataRetryEngine(c.cfg)
		if engineErr != nil {
			return nil, fmt.Errorf("failed to initialize stream metadata retry engine: %w", engineErr)
		}
	}

	var result *StreamMetadata

	err := engine.Run(ctx, &retry.Request{
		Operation: func(ctx context.Context) error {
			fetchResult, fetchErr := fetchJSONWithQuery[GetStreamMetadataResponse](
				c,
				ctx,
				zvukAPIStreamMetadataURI,
				query,
			)
			if fetchErr != nil {
				if fetchResult != nil && fetchResult.StatusCode == http.StatusTeapot {
					return &retryableStreamMetadataError{err: fetchErr}
				}

				return fetchErr
			}

			result = fetchResult.Data.Result

			return nil
		},
		OnRetry: func(ctx context.Context, info *retry.AttemptInfo) {
			attemptsLeft := max(c.cfg.RetryAttemptsCount-utils.SafeUint64ToInt64(info.Retry), 0)

			logger.Infof(ctx, "Retrying due to error (%d attempts left): %v", attemptsLeft, info.Err)
		},
	})
	if err != nil {
		var retryableErr *retryableStreamMetadataError
		if errors.As(err, &retryableErr) {
			err = retryableErr.Unwrap()
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
	return fetchCachedMetadata(ctx, trackIDs, c.tracksCache, "Track", "GraphQL API", c.getTracksViaGraphQL)
}

// GetUserProfile retrieves the user's profile information.
func (c *ClientImpl) GetUserProfile(ctx context.Context) (*UserProfile, error) {
	result, err := fetchJSON[GetUserProfileResponse](c, ctx, zvukAPIUserProfileURI)
	if err != nil {
		return nil, err
	}

	return result.Data.Result, nil
}

// openDownloadResponse creates a GET request and validates the response status.
func (c *ClientImpl) openDownloadResponse(
	ctx context.Context,
	url string,
	configure func(*http.Request),
	allowedStatuses ...int,
) (io.ReadCloser, int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, 0, err
	}

	if configure != nil {
		configure(request)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}

	if !slices.Contains(allowedStatuses, response.StatusCode) {
		response.Body.Close() //nolint:gosec // Error on close is not critical here.

		return nil, 0, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	return response.Body, response.ContentLength, nil
}

// getEntitiesMetadata fetches metadata for the given entity IDs.
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
	releases, err := fetchCachedMetadata(
		ctx,
		releaseIDs,
		c.albumsCache,
		"Album",
		"API",
		func(ctx context.Context, ids []string) (map[string]*Release, error) {
			metadata, fetchErr := c.getEntitiesMetadata(ctx, zvukAPIReleaseMetadataURI, ids, nil)
			if fetchErr != nil {
				return nil, fetchErr
			}

			return metadata.Releases, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &GetAlbumsMetadataResponse{Releases: releases}, nil
}
