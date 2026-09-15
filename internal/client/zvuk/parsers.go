package zvuk

import (
	"fmt"
	"slices"
	"strconv"
)

// stringField extracts a string value from a map by key.
func stringField(data map[string]any, key string) (string, bool) {
	value, ok := data[key].(string)
	return value, ok
}

// stringValue extracts a string value from a map, returning empty string when missing.
func stringValue(data map[string]any, key string) string {
	value, _ := stringField(data, key)
	return value
}

// valueOrZero type-asserts a value to T, returning the zero value on failure.
func valueOrZero[T any](value any) T {
	result, ok := value.(T)
	if !ok {
		var zero T

		return zero
	}

	return result
}

// int64Value extracts an int64 value from a map, coercing from JSON float64.
func int64Value(data map[string]any, key string) int64 {
	return int64(valueOrZero[float64](data[key]))
}

// boolValue extracts a boolean value from a map by key.
func boolValue(data map[string]any, key string) bool {
	return valueOrZero[bool](data[key])
}

// mapValue extracts a nested map from a map by key.
func mapValue(data map[string]any, key string) map[string]any {
	return valueOrZero[map[string]any](data[key])
}

// mapValueFromAny type-asserts a value to map[string]any.
func mapValueFromAny(value any) map[string]any {
	return valueOrZero[map[string]any](value)
}

// imageSource extracts the image source URL from nested image metadata.
func imageSource(data map[string]any) (string, bool) {
	return stringField(mapValue(data, "image"), "src")
}

// imageSourceValue extracts the image source URL, returning empty string when missing.
func imageSourceValue(data map[string]any) string {
	value, _ := imageSource(data)
	return value
}

// parseOptionalID parses an optional string ID field, returning zero when absent.
func parseOptionalID(data map[string]any, kind string) (int64, error) {
	id, ok := stringField(data, "id")
	if !ok {
		return 0, nil
	}

	parsedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID: %w", kind, err)
	}

	return parsedID, nil
}

// parseAudiobookFromGraphQL converts GraphQL book response to Audiobook struct.
func parseAudiobookFromGraphQL(data map[string]any, audiobookID string) (*Audiobook, error) {
	parsedID, err := strconv.ParseInt(audiobookID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid audiobook ID: %w", err)
	}

	publisher := mapValue(data, "publisher")
	audiobook := &Audiobook{
		ID:              parsedID,
		Title:           stringValue(data, "title"),
		PublicationDate: stringValue(data, "publicationDate"),
		Copyright:       stringValue(data, "copyright"),
		Description:     stringValue(data, "description"),
		AgeLimit:        int64Value(data, "ageLimit"),
		FullDuration:    int64Value(data, "fullDuration"),
		BigImageURL:     imageSourceValue(data),
		PublisherName:   stringValue(publisher, "publisherName"),
		PublisherBrand:  stringValue(publisher, "publisherBrand"),
	}

	audiobook.ArtistNames = parseMapStrings(data["bookAuthors"], "rname", false)
	audiobook.PerformerNames = parseMapStrings(data["performers"], "rname", false)
	audiobook.Genres = parseMapStrings(data["genres"], "name", false)

	return audiobook, nil
}

// parseChapterAsTrack converts a GraphQL chapter response to Track struct.
func parseChapterAsTrack(data map[string]any, audiobook *Audiobook) (*Track, error) {
	parsedID, err := parseOptionalID(data, "chapter")
	if err != nil {
		return nil, err
	}

	track := &Track{
		ID:          parsedID,
		Title:       stringValue(data, "title"),
		Duration:    int64Value(data, "duration"),
		Position:    int64Value(data, "position"),
		ReleaseID:   audiobook.ID,
		ArtistNames: audiobook.ArtistNames,
	}

	if audiobook.BigImageURL != "" {
		track.Image = &Image{SourceURL: audiobook.BigImageURL}
	}

	return track, nil
}

// parsePodcastFromGraphQL converts GraphQL podcast response to Podcast struct.
func parsePodcastFromGraphQL(data map[string]any, podcastID string) (*Podcast, error) {
	parsedID, err := strconv.ParseInt(podcastID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid podcast ID: %w", err)
	}

	return &Podcast{
		ID:          parsedID,
		Title:       stringValue(data, "title"),
		Description: stringValue(data, "description"),
		Category:    stringValue(mapValue(data, "category"), "name"),
	}, nil
}

// parseEpisodeAsTrack converts a GraphQL episode response to Track struct.
func parseEpisodeAsTrack(data map[string]any, podcast *Podcast) (*Track, error) {
	parsedID, err := parseOptionalID(data, "episode")
	if err != nil {
		return nil, err
	}

	track := &Track{
		ID:        parsedID,
		Title:     stringValue(data, "title"),
		Duration:  int64Value(data, "duration"),
		Credits:   stringValue(data, "publicationDate"),
		ReleaseID: podcast.ID,
	}

	if boolValue(data, "explicit") {
		podcast.Explicit = true
	}

	podcastData := mapValue(data, "podcast")
	if src, ok := imageSource(podcastData); ok {
		if podcast.BigImageURL == "" {
			podcast.BigImageURL = src
		}

		track.Image = &Image{SourceURL: src}
	}

	for _, name := range parseMapStrings(podcastData["authors"], "name", false) {
		if len(podcast.ArtistNames) == 0 || !slices.Contains(podcast.ArtistNames, name) {
			podcast.ArtistNames = append(podcast.ArtistNames, name)
		}

		track.ArtistNames = append(track.ArtistNames, name)
	}

	return track, nil
}

// parseTrackFromGraphQL converts a GraphQL track response to Track struct.
func parseTrackFromGraphQL(data map[string]any) (*Track, error) {
	trackIDRaw, parsedTrackID, err := parseTrackID(data)
	if err != nil {
		return nil, err
	}

	track := &Track{ID: parsedTrackID}
	parseTrackBaseFields(data, track)

	if src, ok := imageSource(data); ok {
		track.Image = &Image{SourceURL: src}
	}

	track.Genres = parseMapStrings(data["genres"], "name", false)

	releaseData, parsedReleaseID, err := parseTrackRelease(data, trackIDRaw)
	if err != nil {
		return nil, err
	}

	track.ReleaseID = parsedReleaseID
	if len(track.ArtistNames) == 0 {
		track.ArtistNames = parseArtistTitles(releaseData["artists"])
	}

	return track, nil
}

// parseArtistTitles extracts artist title strings from GraphQL artist data.
func parseArtistTitles(data any) []string {
	return parseMapStrings(data, "title", true)
}

// parseMapStrings extracts string values from a slice of GraphQL map items.
func parseMapStrings(data any, key string, skipEmpty bool) []string {
	items, ok := data.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		value, exists := stringField(mapValueFromAny(item), key)
		if !exists || (skipEmpty && value == "") {
			continue
		}

		result = append(result, value)
	}

	return result
}

// parsePlaylistTrackIDs extracts track IDs from a playlistTracks GraphQL page.
// Nil or incomplete entries are skipped so numbering matches the website list.
func parsePlaylistTrackIDs(items []any) []int64 {
	trackIDs := make([]int64, 0, len(items))

	for _, item := range items {
		if item == nil {
			continue
		}

		_, parsedID, err := parseTrackID(mapValueFromAny(item))
		if err != nil || parsedID == 0 {
			continue
		}

		trackIDs = append(trackIDs, parsedID)
	}

	return trackIDs
}

// parseTrackID parses and validates the track ID from GraphQL data.
func parseTrackID(data map[string]any) (string, int64, error) {
	trackIDRaw, ok := stringField(data, "id")
	if !ok || trackIDRaw == "" {
		return "", 0, ErrTrackIDMissing
	}

	parsedTrackID, err := strconv.ParseInt(trackIDRaw, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid track ID: %w", err)
	}

	return trackIDRaw, parsedTrackID, nil
}

// parseTrackBaseFields populates basic track fields from GraphQL data.
func parseTrackBaseFields(data map[string]any, track *Track) {
	track.Title = stringValue(data, "title")
	track.Lyrics = boolValue(data, "lyrics")
	track.Credits = stringValue(data, "credits")
	track.Duration = int64Value(data, "duration")
	track.Position = int64Value(data, "position")
	track.HasFLAC = boolValue(data, "hasFlac")

	track.HighestQuality = "high"
	if track.HasFLAC {
		track.HighestQuality = "flac"
	}

	track.ArtistNames = parseArtistTitles(data["artists"])
}

// parseTrackRelease extracts release metadata and ID from GraphQL track data.
func parseTrackRelease(data map[string]any, trackIDRaw string) (map[string]any, int64, error) {
	releaseData, ok := data["release"].(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("%w: track '%s'", ErrTrackReleaseDataMissing, trackIDRaw)
	}

	releaseIDRaw, ok := stringField(releaseData, "id")
	if !ok || releaseIDRaw == "" {
		return nil, 0, fmt.Errorf("%w: track '%s'", ErrTrackReleaseIDMissing, trackIDRaw)
	}

	parsedReleaseID, err := strconv.ParseInt(releaseIDRaw, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid release ID for track '%s': %w", trackIDRaw, err)
	}

	return releaseData, parsedReleaseID, nil
}
