package zvuk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/machinebox/graphql"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

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

// getAudiobookViaGraphQL fetches a single audiobook with its tracks using GraphQL.
//
//nolint:funlen // GraphQL query requires length.
func (c *ClientImpl) getAudiobookViaGraphQL(
	ctx context.Context,
	audiobookID string,
) (*GetAudiobookResult, error) {
	graphqlRequest := graphql.NewRequest(`
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
`)

	graphqlRequest.Header.Add("X-Auth-Token", c.cfg.AuthToken)
	graphqlRequest.Var("ids", []string{audiobookID})

	var graphQLResponse map[string]any
	if err := c.graphQLClient.Run(ctx, graphqlRequest, &graphQLResponse); err != nil {
		return nil, err
	}

	// Navigate the response map.
	data, ok := graphQLResponse["getBooks"].([]any)
	if !ok || len(data) == 0 {
		return nil, ErrAudiobookNotFound
	}

	audiobookData, dataOk := data[0].(map[string]any)
	if !dataOk {
		return nil, ErrUnexpectedAudiobookFormat
	}

	// Parse audiobook metadata.
	audiobook, err := parseAudiobookFromGraphQL(audiobookData, audiobookID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse audiobook: %w", err)
	}

	// Parse chapters as tracks.
	tracks := make(map[string]*Track)

	if chaptersData, chaptersOk := audiobookData["chapters"].([]any); chaptersOk {
		for _, chapterData := range chaptersData {
			chapterMap, chapterOk := chapterData.(map[string]any)
			if !chapterOk {
				continue
			}

			track, parseErr := parseChapterAsTrack(chapterMap, audiobook)
			if parseErr != nil {
				logger.Warnf(ctx, "Failed to parse chapter: %v", parseErr)
				continue
			}

			tracks[strconv.FormatInt(track.ID, 10)] = track
			audiobook.TrackIDs = append(audiobook.TrackIDs, track.ID)
		}
	}

	return &GetAudiobookResult{
		Audiobook: audiobook,
		Tracks:    tracks,
	}, nil
}

// GetChapterStreamMetadata retrieves streaming metadata for audiobook chapters via GraphQL.
//
//nolint:funlen // GraphQL query construction and response parsing require comprehensive implementation.
func (c *ClientImpl) GetChapterStreamMetadata(
	ctx context.Context,
	chapterIDs []string,
) (map[string]*ChapterStreamMetadata, error) {
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

	graphqlRequest.Header.Add("X-Auth-Token", c.cfg.AuthToken)
	graphqlRequest.Var("ids", chapterIDs)
	graphqlRequest.Var("quality", defaultStreamQuality)
	graphqlRequest.Var("encodeType", defaultEncodeType)
	graphqlRequest.Var("includeFlacDrm", true)

	var graphQLResponse map[string]any
	if err := c.graphQLClient.Run(ctx, graphqlRequest, &graphQLResponse); err != nil {
		return nil, err
	}

	// Navigate response - mediaContents returns array in same order as input IDs.
	data, ok := graphQLResponse["mediaContents"].([]any)
	if !ok {
		return nil, ErrUnexpectedMediaContentsFormat
	}

	result := make(map[string]*ChapterStreamMetadata, len(chapterIDs))
	for i, contentData := range data {
		contentMap, contentOk := contentData.(map[string]any)
		if !contentOk {
			continue
		}

		streamData, streamOk := contentMap["stream"].(map[string]any)
		if !streamOk {
			continue
		}

		if i >= len(chapterIDs) {
			continue
		}

		chapterID := chapterIDs[i]

		// Extract all available stream URLs.
		metadata := &ChapterStreamMetadata{}
		if midURL, midOk := streamData["mid"].(string); midOk {
			metadata.Mid = midURL
		}

		if highURL, highOk := streamData["high"].(string); highOk {
			metadata.High = highURL
		}

		if flacURL, flacOk := streamData["flacdrm"].(string); flacOk {
			metadata.FLAC = flacURL
		}

		result[chapterID] = metadata
	}

	return result, nil
}

// getPodcastViaGraphQL fetches a single podcast with its episodes using GraphQL.
//
//nolint:funlen // GraphQL query requires length.
func (c *ClientImpl) getPodcastViaGraphQL(
	ctx context.Context,
	podcastID string,
) (*GetPodcastResult, error) {
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

	graphqlRequest.Header.Add("X-Auth-Token", c.cfg.AuthToken)
	graphqlRequest.Var("ids", []string{podcastID})

	var graphQLResponse map[string]any
	if err := c.graphQLClient.Run(ctx, graphqlRequest, &graphQLResponse); err != nil {
		return nil, err
	}

	// Navigate the response map.
	data, ok := graphQLResponse["getPodcasts"].([]any)
	if !ok || len(data) == 0 {
		return nil, ErrPodcastNotFound
	}

	podcastData, dataOk := data[0].(map[string]any)
	if !dataOk {
		return nil, ErrUnexpectedPodcastFormat
	}

	// Parse podcast metadata.
	podcast, err := parsePodcastFromGraphQL(podcastData, podcastID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse podcast: %w", err)
	}

	// Parse episodes as tracks.
	tracks := make(map[string]*Track)

	if episodesData, episodesOk := podcastData["episodes"].([]any); episodesOk {
		for _, episodeData := range episodesData {
			episodeMap, episodeOk := episodeData.(map[string]any)
			if !episodeOk {
				continue
			}

			track, parseErr := parseEpisodeAsTrack(episodeMap, podcast)
			if parseErr != nil {
				logger.Warnf(ctx, "Failed to parse episode: %v", parseErr)
				continue
			}

			tracks[strconv.FormatInt(track.ID, 10)] = track
			podcast.TrackIDs = append(podcast.TrackIDs, track.ID)
		}
	}

	return &GetPodcastResult{
		Podcast: podcast,
		Tracks:  tracks,
	}, nil
}
