package zvuk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/machinebox/graphql"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

const (
	// graphQLAuthHeader is the header name used to pass the Zvuk auth token in GraphQL requests.
	graphQLAuthHeader = "X-Auth-Token"

	// getAudiobookChaptersQuery loads audiobook metadata and chapter list from the GraphQL API.
	getAudiobookChaptersQuery = `
		query getBookChapters($ids: [ID!]!) {
			getBooks(ids: $ids) {
				title
				mark
				explicit
				publicationDate
				copyright
				description
				ageLimit
				fullDuration
				image {
					src
				}
				bookAuthors {
					id
					rname
				}
				publisher {
					id
					publisherName
					publisherBrand
				}
				performers {
					id
					rname
				}
				genres {
					id
					name
				}
				chapters {
					...PlayerChapterData
				}
			}
		}

		fragment PlayerChapterData on Chapter {
			id
			title
			availability
			duration
			position
		}
	`

	// getPlaylistTracksQuery loads playlist tracks in website order via GraphQL pagination.
	getPlaylistTracksQuery = `
		query getPlaylistTracks($id: ID!, $limit: Int = 30, $offset: Int = 0) {
			playlistTracks(id: $id, limit: $limit, offset: $offset) {
				id
			}
		}
	`

	// playlistTracksPageSize matches the Zvuk website playlistTracks page size.
	playlistTracksPageSize = 30
	// playlistTracksMaxPages caps pagination to avoid infinite loops on a broken offset.
	playlistTracksMaxPages = 1000
)

// runGraphQL authenticates and executes a GraphQL request.
func (c *ClientImpl) runGraphQL(ctx context.Context, request *graphql.Request) (map[string]any, error) {
	request.Header.Add(graphQLAuthHeader, c.cfg.ZvukAuthToken)

	var response map[string]any
	if err := c.graphQLClient.Run(ctx, request, &response); err != nil {
		return nil, err
	}

	return response, nil
}

// firstGraphQLMap extracts the first map item from a GraphQL response field.
func firstGraphQLMap(
	response map[string]any,
	field string,
	notFoundErr error,
	formatErr error,
) (map[string]any, error) {
	items, ok := response[field].([]any)
	if !ok || len(items) == 0 {
		return nil, notFoundErr
	}

	item, ok := items[0].(map[string]any)
	if !ok {
		return nil, formatErr
	}

	return item, nil
}

// parseGraphQLChildTracks parses child tracks from GraphQL collection data.
func parseGraphQLChildTracks[T any](
	ctx context.Context,
	data map[string]any,
	field string,
	kind string,
	parent *T,
	parse func(map[string]any, *T) (*Track, error),
) (map[string]*Track, []int64) {
	items, ok := data[field].([]any)
	if !ok {
		return map[string]*Track{}, []int64{}
	}

	tracks := make(map[string]*Track, len(items))
	trackIDs := make([]int64, 0, len(items))

	for _, item := range items {
		itemData := mapValueFromAny(item)
		if itemData == nil {
			continue
		}

		track, err := parse(itemData, parent)
		if err != nil {
			logger.Warnf(ctx, "Failed to parse %s: %v", kind, err)
			continue
		}

		tracks[strconv.FormatInt(track.ID, 10)] = track
		trackIDs = append(trackIDs, track.ID)
	}

	return tracks, trackIDs
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

	graphqlRequest.Var("id", artistID)
	graphqlRequest.Var("offset", offset)
	graphqlRequest.Var("limit", limit)

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	artist, err := firstGraphQLMap(
		graphQLResponse,
		"getArtists",
		ErrArtistNotFound,
		ErrUnexpectedArtistResponseFormat,
	)
	if err != nil {
		return nil, err
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

// getAudiobookViaGraphQL fetches a single audiobook with its tracks.
func (c *ClientImpl) getAudiobookViaGraphQL(
	ctx context.Context,
	audiobookID string,
) (*graphQLCollectionResult[Audiobook], error) {
	graphqlRequest := graphql.NewRequest(getAudiobookChaptersQuery)

	graphqlRequest.Var("ids", []string{audiobookID})

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	audiobookData, err := firstGraphQLMap(
		graphQLResponse,
		"getBooks",
		ErrAudiobookNotFound,
		ErrUnexpectedAudiobookFormat,
	)
	if err != nil {
		return nil, err
	}

	// Parse audiobook metadata.
	audiobook, err := parseAudiobookFromGraphQL(audiobookData, audiobookID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse audiobook: %w", err)
	}

	// Parse chapters as tracks.
	tracks, trackIDs := parseGraphQLChildTracks(
		ctx,
		audiobookData,
		"chapters",
		"chapter",
		audiobook,
		parseChapterAsTrack,
	)
	audiobook.TrackIDs = trackIDs

	return &graphQLCollectionResult[Audiobook]{item: audiobook, tracks: tracks}, nil
}

// GetStreamQualities retrieves streaming metadata for audiobook chapters and podcasts episodes.
func (c *ClientImpl) GetStreamQualities(
	ctx context.Context,
	chapterIDs []string,
) (map[string]*StreamQualities, error) {
	graphqlRequest := graphql.NewRequest(`
		query getStream($ids: [ID!]!, $quality: String, $encodeType: String, $includeFlacDrm: Boolean!) {
			mediaContents(ids: $ids, quality: $quality, encodeType: $encodeType) {
				... on Track {
					__typename
					stream {
						expire
						high
						mid
						flacdrm @include(if: $includeFlacDrm)
					}
				}
			... on Episode {
				__typename
				stream {
					expire
					high
					mid
					flacdrm @include(if: $includeFlacDrm)
				}
			}
			... on Chapter {
				__typename
				stream {
					expire
					high
					mid
					flacdrm @include(if: $includeFlacDrm)
				}
			}
			}
		}
	`)

	graphqlRequest.Var("ids", chapterIDs)
	graphqlRequest.Var("quality", defaultStreamQuality)
	graphqlRequest.Var("encodeType", defaultEncodeType)
	graphqlRequest.Var("includeFlacDrm", true)

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	// Navigate response - mediaContents returns array in same order as input IDs.
	data, ok := graphQLResponse["mediaContents"].([]any)
	if !ok {
		return nil, ErrUnexpectedMediaContentsFormat
	}

	result := make(map[string]*StreamQualities, len(chapterIDs))

	for i, contentData := range data {
		if i >= len(chapterIDs) {
			break
		}

		streamData := mapValue(mapValueFromAny(contentData), "stream")
		if streamData == nil {
			continue
		}

		result[chapterIDs[i]] = &StreamQualities{
			Mid:  stringValue(streamData, "mid"),
			High: stringValue(streamData, "high"),
			FLAC: stringValue(streamData, "flacdrm"),
		}
	}

	return result, nil
}

// getPlaylistTrackIDsViaGraphQL fetches playlist track IDs in the same order as the website.
func (c *ClientImpl) getPlaylistTrackIDsViaGraphQL(ctx context.Context, playlistID string) ([]int64, error) {
	return collectPaginatedPlaylistTrackIDs(
		playlistTracksPageSize,
		playlistTracksMaxPages,
		func(offset int) ([]any, error) {
			return c.fetchPlaylistTracksPage(ctx, playlistID, playlistTracksPageSize, offset)
		},
	)
}

// fetchPlaylistTracksPage loads one playlistTracks GraphQL page.
func (c *ClientImpl) fetchPlaylistTracksPage(
	ctx context.Context,
	playlistID string,
	limit int,
	offset int,
) ([]any, error) {
	graphqlRequest := graphql.NewRequest(getPlaylistTracksQuery)
	graphqlRequest.Var("id", playlistID)
	graphqlRequest.Var("limit", limit)
	graphqlRequest.Var("offset", offset)

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	items, ok := graphQLResponse["playlistTracks"].([]any)
	if !ok {
		if graphQLResponse["playlistTracks"] == nil {
			return []any{}, nil
		}

		return nil, ErrUnexpectedPlaylistTracksFormat
	}

	return items, nil
}

// collectPaginatedPlaylistTrackIDs walks playlistTracks pages until a short or empty page.
func collectPaginatedPlaylistTrackIDs(
	pageSize int,
	maxPages int,
	fetchPage func(offset int) ([]any, error),
) ([]int64, error) {
	if pageSize <= 0 || maxPages <= 0 {
		return []int64{}, nil
	}

	trackIDs := make([]int64, 0)

	for page := range maxPages {
		items, err := fetchPage(page * pageSize)
		if err != nil {
			return nil, err
		}

		trackIDs = append(trackIDs, parsePlaylistTrackIDs(items)...)

		if len(items) < pageSize {
			break
		}
	}

	return trackIDs, nil
}

// getTracksViaGraphQL fetches tracks metadata from GraphQL API.
func (c *ClientImpl) getTracksViaGraphQL(ctx context.Context, trackIDs []string) (map[string]*Track, error) {
	if len(trackIDs) == 0 {
		return map[string]*Track{}, nil
	}

	graphqlRequest := graphql.NewRequest(`
		query getTracks($ids: [ID!]!) {
			getTracks(ids: $ids) {
				id
				title
				lyrics
				credits
				duration
				availability
				position
				hasFlac
				artists {
					id
					title
				}
				image {
					src
				}
				genres {
					id
					name
				}
				release {
					id
					type
					title
					date
					credits
					artists {
						id
						title
					}
					image {
						src
					}
					label {
						id
						title
					}
				}
			}
		}
	`)
	graphqlRequest.Var("ids", trackIDs)

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	tracksData, ok := graphQLResponse["getTracks"].([]any)
	if !ok {
		return nil, ErrUnexpectedTracksResponseFormat
	}

	result := make(map[string]*Track, len(tracksData))
	for _, trackData := range tracksData {
		trackMap, hasExpectedFormat := trackData.(map[string]any)
		if !hasExpectedFormat {
			continue
		}

		track, parseErr := parseTrackFromGraphQL(trackMap)
		if parseErr != nil {
			logger.Warnf(ctx, "Failed to parse track from GraphQL: %v", parseErr)

			continue
		}

		result[strconv.FormatInt(track.ID, 10)] = track
	}

	return result, nil
}

// getPodcastViaGraphQL fetches a single podcast with its episodes.
func (c *ClientImpl) getPodcastViaGraphQL(
	ctx context.Context,
	podcastID string,
) (*graphQLCollectionResult[Podcast], error) {
	graphqlRequest := graphql.NewRequest(`
	query getPodcastEpisodes($ids: [ID!]!) {
		getPodcasts(ids: $ids) {
			title
			description
			category {
				id
				name
			}
			episodes {
				...PlayerEpisodeData
			}
		}
	}
	
	fragment PlayerEpisodeData on Episode {
		id
		title
		availability
		duration
		publicationDate
		explicit
		image {
			src
			palette
		}
		podcast {
			id
			title
			authors {
				id
				name
			}
			category {
				id
				name
			}
			image {
				src
				palette
			}
			explicit
			mark
		}
		mark
		__typename
	}
`)

	graphqlRequest.Var("ids", []string{podcastID})

	graphQLResponse, err := c.runGraphQL(ctx, graphqlRequest)
	if err != nil {
		return nil, err
	}

	podcastData, err := firstGraphQLMap(
		graphQLResponse,
		"getPodcasts",
		ErrPodcastNotFound,
		ErrUnexpectedPodcastFormat,
	)
	if err != nil {
		return nil, err
	}

	// Parse podcast metadata.
	podcast, err := parsePodcastFromGraphQL(podcastData, podcastID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse podcast: %w", err)
	}

	// Parse episodes as tracks.
	tracks, trackIDs := parseGraphQLChildTracks(ctx, podcastData, "episodes", "episode", podcast, parseEpisodeAsTrack)
	podcast.TrackIDs = trackIDs

	return &graphQLCollectionResult[Podcast]{item: podcast, tracks: tracks}, nil
}
