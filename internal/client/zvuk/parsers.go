package zvuk

import (
	"fmt"
	"strconv"
)

// parseAudiobookFromGraphQL converts GraphQL book response to Audiobook struct.
//
//nolint:funlen,gocognit // Complex parsing logic.
func parseAudiobookFromGraphQL(data map[string]any, audiobookID string) (*Audiobook, error) {
	audiobook := &Audiobook{}

	// Parse audiobook ID.
	parsedID, err := strconv.ParseInt(audiobookID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid audiobook ID: %w", err)
	}

	audiobook.ID = parsedID

	// Parse title.
	if title, ok := data["title"].(string); ok {
		audiobook.Title = title
	}

	// Parse publication date.
	if pubDate, ok := data["publicationDate"].(string); ok {
		audiobook.PublicationDate = pubDate
	}

	// Parse copyright.
	if copyright, ok := data["copyright"].(string); ok {
		audiobook.Copyright = copyright
	}

	// Parse description.
	if description, ok := data["description"].(string); ok {
		audiobook.Description = description
	}

	// Parse age limit.
	if ageLimit, ok := data["ageLimit"].(float64); ok {
		audiobook.AgeLimit = int64(ageLimit)
	}

	// Parse full duration.
	if fullDuration, ok := data["fullDuration"].(float64); ok {
		audiobook.FullDuration = int64(fullDuration)
	}

	// Parse image.
	if imageData, imageOk := data["image"].(map[string]any); imageOk {
		if src, srcOk := imageData["src"].(string); srcOk {
			audiobook.BigImageURL = src
		}
	}

	// Parse authors.
	if authorsData, authorsOk := data["bookAuthors"].([]any); authorsOk {
		for _, authorData := range authorsData {
			authorMap, authorOk := authorData.(map[string]any)
			if !authorOk {
				continue
			}

			if rname, rnameOk := authorMap["rname"].(string); rnameOk {
				audiobook.ArtistNames = append(audiobook.ArtistNames, rname)
			}
		}
	}

	// Parse publisher.
	if publisherData, publisherOk := data["publisher"].(map[string]any); publisherOk {
		if publisherName, nameOk := publisherData["publisherName"].(string); nameOk {
			audiobook.PublisherName = publisherName
		}

		if publisherBrand, brandOk := publisherData["publisherBrand"].(string); brandOk {
			audiobook.PublisherBrand = publisherBrand
		}
	}

	// Parse performers.
	if performersData, performersOk := data["performers"].([]any); performersOk {
		for _, performerData := range performersData {
			performerMap, performerOk := performerData.(map[string]any)
			if !performerOk {
				continue
			}

			if rname, rnameOk := performerMap["rname"].(string); rnameOk {
				audiobook.PerformerNames = append(audiobook.PerformerNames, rname)
			}
		}
	}

	// Parse genres.
	if genresData, genresOk := data["genres"].([]any); genresOk {
		for _, genreData := range genresData {
			genreMap, genreOk := genreData.(map[string]any)
			if !genreOk {
				continue
			}

			if name, nameOk := genreMap["name"].(string); nameOk {
				audiobook.Genres = append(audiobook.Genres, name)
			}
		}
	}

	// TrackIDs will be filled during chapter parsing.
	return audiobook, nil
}

// parseChapterAsTrack converts a GraphQL chapter response to Track struct.
// Uses audiobook data to fill in common fields (book info, authors, image).
func parseChapterAsTrack(data map[string]any, audiobook *Audiobook) (*Track, error) {
	track := &Track{}

	// Parse chapter ID as track ID.
	if id, ok := data["id"].(string); ok {
		parsedID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chapter ID: %w", err)
		}

		track.ID = parsedID
	}

	// Parse chapter title.
	if title, ok := data["title"].(string); ok {
		track.Title = title
	}

	// Parse duration.
	if duration, ok := data["duration"].(float64); ok {
		track.Duration = int64(duration)
	}

	// Parse position.
	if position, ok := data["position"].(float64); ok {
		track.Position = int64(position)
	}

	// Parse availability.
	if availability, ok := data["availability"].(float64); ok {
		track.Availability = int64(availability)
	}

	// Quality fields are intentionally left empty for audiobook chapters.
	// The service layer will determine actual quality from stream metadata.
	// This keeps the client as a simple data provider without business logic.

	// Use audiobook data for common fields.
	track.ReleaseID = audiobook.ID
	track.ReleaseTitle = audiobook.Title

	track.ArtistNames = audiobook.ArtistNames
	if audiobook.BigImageURL != "" {
		track.Image = &Image{SourceURL: audiobook.BigImageURL}
	}

	return track, nil
}

// parsePodcastFromGraphQL converts GraphQL podcast response to Podcast struct.
func parsePodcastFromGraphQL(data map[string]any, podcastID string) (*Podcast, error) {
	podcast := &Podcast{}

	// Parse podcast ID.
	parsedID, err := strconv.ParseInt(podcastID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid podcast ID: %w", err)
	}

	podcast.ID = parsedID

	// Parse title.
	if title, ok := data["title"].(string); ok {
		podcast.Title = title
	}

	// Parse description.
	if description, ok := data["description"].(string); ok {
		podcast.Description = description
	}

	// Parse category.
	if categoryData, categoryOk := data["category"].(map[string]any); categoryOk {
		if name, nameOk := categoryData["name"].(string); nameOk {
			podcast.Category = name
		}
	}

	// TrackIDs will be filled during episode parsing.
	return podcast, nil
}

// parseEpisodeAsTrack converts a GraphQL episode response to Track struct.
// Uses podcast data to fill in common fields (podcast info, authors, image).
func parseEpisodeAsTrack(data map[string]any, podcast *Podcast) (*Track, error) {
	track := &Track{}

	if err := parseEpisodeBasicFields(data, track, podcast); err != nil {
		return nil, err
	}

	parseEpisodeNestedPodcastData(data, track, podcast)

	// Use podcast data for common fields.
	track.ReleaseID = podcast.ID
	track.ReleaseTitle = podcast.Title

	return track, nil
}

// parseEpisodeBasicFields parses basic episode fields from GraphQL data.
func parseEpisodeBasicFields(data map[string]any, track *Track, podcast *Podcast) error {
	// Parse episode ID as track ID.
	if id, ok := data["id"].(string); ok {
		parsedID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid episode ID: %w", err)
		}

		track.ID = parsedID
	}

	// Parse episode title.
	if title, ok := data["title"].(string); ok {
		track.Title = title
	}

	// Parse duration.
	if duration, ok := data["duration"].(float64); ok {
		track.Duration = int64(duration)
	}

	// Parse availability.
	if availability, ok := data["availability"].(float64); ok {
		track.Availability = int64(availability)
	}

	// Parse publication date for episode metadata.
	if pubDate, ok := data["publicationDate"].(string); ok {
		track.Credits = pubDate // Store in Credits field for template usage
	}

	// Parse explicit flag.
	if explicit, ok := data["explicit"].(bool); ok && explicit {
		podcast.Explicit = true
	}

	return nil
}

// parseEpisodeNestedPodcastData parses nested podcast data from episode response.
func parseEpisodeNestedPodcastData(data map[string]any, track *Track, podcast *Podcast) {
	podcastData, ok := data["podcast"].(map[string]any)
	if !ok {
		return
	}

	parseEpisodePodcastImage(podcastData, track, podcast)
	parseEpisodePodcastAuthors(podcastData, track, podcast)
}

// parseEpisodePodcastImage extracts image URL from nested podcast data.
func parseEpisodePodcastImage(podcastData map[string]any, track *Track, podcast *Podcast) {
	imageData, ok := podcastData["image"].(map[string]any)
	if !ok {
		return
	}

	src, ok := imageData["src"].(string)
	if !ok {
		return
	}

	if podcast.BigImageURL == "" {
		podcast.BigImageURL = src
	}

	track.Image = &Image{SourceURL: src}
}

// parseEpisodePodcastAuthors extracts author names from nested podcast data.
func parseEpisodePodcastAuthors(podcastData map[string]any, track *Track, podcast *Podcast) {
	authorsData, ok := podcastData["authors"].([]any)
	if !ok {
		return
	}

	for _, authorData := range authorsData {
		authorMap, isAuthorMap := authorData.(map[string]any)
		if !isAuthorMap {
			continue
		}

		name, isNameString := authorMap["name"].(string)
		if !isNameString {
			continue
		}

		if len(podcast.ArtistNames) == 0 || !contains(podcast.ArtistNames, name) {
			podcast.ArtistNames = append(podcast.ArtistNames, name)
		}

		track.ArtistNames = append(track.ArtistNames, name)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}
