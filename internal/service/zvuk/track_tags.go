package zvuk

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// trackTagContext contains data required to build track tags for:
// - filename/folder templates
// - audio tag writing (ID3/Vorbis)
//
// IMPORTANT: this is intended to be the single place that "fills" track-level metadata.
type trackTagContext struct {
	trackNumber     int64
	track           *zvuk.Track
	audioCollection *audioCollection
	albumTags       map[string]string
	category        DownloadCategory
}

func setIfNotBlank(tags map[string]string, key, value string) {
	if tags == nil {
		return
	}

	if strings.TrimSpace(value) == "" {
		return
	}

	tags[key] = value
}

func buildAudiobookTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := maps.Clone(collection.tags)

	result[TagCollectionTitle] = collection.title
	result[TagTrackArtist] = strings.Join(track.ArtistNames, ", ")
	result[TagTrackID] = strconv.FormatInt(track.ID, 10)
	result[TagTrackNumber] = strconv.FormatInt(ctx.trackNumber, 10)
	result[TagTrackNumberPad] = fmt.Sprintf("%0*d", trackNumberPaddingWidth, ctx.trackNumber)
	result[TagTrackTitle] = track.Title
	result[TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)

	return result
}

func buildPodcastTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := make(map[string]string, len(collection.tags)+16)
	maps.Copy(result, collection.tags)

	setIfNotBlank(result, TagCollectionTitle, collection.title)

	if collection.tracksCount > 0 {
		result[TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)
	}

	result[TagTrackArtist] = strings.Join(track.ArtistNames, ", ")
	setIfNotBlank(result, TagTrackGenre, strings.Join(track.Genres, ", "))

	publicationDate := parseEpisodePublicationDate(track.Credits)
	trackNumber := strconv.FormatInt(ctx.trackNumber, 10)
	trackNumberPad := fmt.Sprintf("%0*d", trackNumberPaddingWidth, ctx.trackNumber)

	result[TagEpisodeID] = strconv.FormatInt(track.ID, 10)
	result[TagEpisodeTitle] = track.Title
	result[TagEpisodeDuration] = strconv.FormatInt(track.Duration, 10)
	result[TagEpisodeNumber] = trackNumber
	result[TagEpisodeNumberPad] = trackNumberPad
	setIfNotBlank(result, TagEpisodePublicationDate, publicationDate)

	result[TagTrackID] = strconv.FormatInt(track.ID, 10)
	result[TagTrackTitle] = track.Title
	result[TagTrackNumber] = trackNumber
	result[TagTrackNumberPad] = trackNumberPad
	result[TagTrackDuration] = strconv.FormatInt(track.Duration, 10)

	return result
}

func buildDefaultTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := make(map[string]string, len(ctx.albumTags)+len(collection.tags)+16)
	maps.Copy(result, ctx.albumTags)
	maps.Copy(result, collection.tags)

	result[TagCollectionTitle] = collection.title
	result[TagTrackArtist] = strings.Join(track.ArtistNames, ", ")
	setIfNotBlank(result, TagTrackGenre, strings.Join(track.Genres, ", "))

	result[TagTrackID] = strconv.FormatInt(track.ID, 10)
	result[TagTrackNumber] = strconv.FormatInt(ctx.trackNumber, 10)
	result[TagTrackNumberPad] = fmt.Sprintf("%0*d", trackNumberPaddingWidth, ctx.trackNumber)
	result[TagTrackTitle] = track.Title
	result[TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)

	return result
}

// buildTrackTags builds a single tag map for the given track in its download context.
//
// This intentionally centralizes all key names and precedence rules:
// - albumTags are the base (when applicable).
// - collection tags override album tags (e.g. playlist overrides "type").
// - track-specific tags override everything.
func buildTrackTags(ctx *trackTagContext) map[string]string {
	if ctx == nil || ctx.track == nil || ctx.audioCollection == nil {
		return nil
	}

	switch ctx.category {
	case DownloadCategoryAudiobook:
		return buildAudiobookTrackTags(ctx)
	case DownloadCategoryPodcast:
		return buildPodcastTrackTags(ctx)
	default:
		return buildDefaultTrackTags(ctx)
	}
}
