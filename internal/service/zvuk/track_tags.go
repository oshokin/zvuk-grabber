package zvuk

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// trackTagContext contains data required to build track tags for:
// - filename/folder templates
// - audio tag writing (ID3/Vorbis)
//
// IMPORTANT: this is intended to be the single place that "fills" track-level metadata.
type trackTagContext struct {
	// trackNumber is the 1-based position of the track in the collection.
	trackNumber int64
	// track contains the track metadata.
	track *zvuk.Track
	// audioCollection is the parent collection context for the track.
	audioCollection *audioCollection
	// albumTags contains album-level metadata tags.
	albumTags map[string]string
	// category is the download category of the parent collection.
	category DownloadCategory
}

// setIfNotBlank sets a tag value only when the value is non-empty.
func setIfNotBlank(tags map[string]string, key, value string) {
	if tags == nil {
		return
	}

	if strings.TrimSpace(value) == "" {
		return
	}

	tags[key] = value
}

// fillCommonTrackTags populates shared track tag fields and returns number strings.
func fillCommonTrackTags(tags map[string]string, trackNumber int64, track *zvuk.Track) (string, string) {
	trackNumberValue := strconv.FormatInt(trackNumber, 10)
	trackNumberPad := fmt.Sprintf("%0*d", trackNumberPaddingWidth, trackNumber)

	tags[media.TagTrackArtist] = strings.Join(track.ArtistNames, ", ")
	tags[media.TagTrackID] = strconv.FormatInt(track.ID, 10)
	tags[media.TagTrackNumber] = trackNumberValue
	tags[media.TagTrackNumberPad] = trackNumberPad
	tags[media.TagTrackTitle] = track.Title

	return trackNumberValue, trackNumberPad
}

// buildAudiobookTrackTags builds metadata tags for an audiobook chapter track.
func buildAudiobookTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := maps.Clone(collection.tags)

	result[media.TagCollectionTitle] = collection.title
	fillCommonTrackTags(result, ctx.trackNumber, track)
	result[media.TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)

	return result
}

// buildPodcastTrackTags builds metadata tags for a podcast episode track.
func buildPodcastTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := make(map[string]string, len(collection.tags)+16)
	maps.Copy(result, collection.tags)

	setIfNotBlank(result, media.TagCollectionTitle, collection.title)

	if collection.tracksCount > 0 {
		result[media.TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)
	}

	setIfNotBlank(result, media.TagTrackGenre, strings.Join(track.Genres, ", "))

	publicationDate := parseEpisodePublicationDate(track.Credits)
	trackNumber, trackNumberPad := fillCommonTrackTags(result, ctx.trackNumber, track)

	result[media.TagEpisodeID] = result[media.TagTrackID]
	result[media.TagEpisodeTitle] = result[media.TagTrackTitle]
	result[media.TagEpisodeDuration] = strconv.FormatInt(track.Duration, 10)
	result[media.TagEpisodeNumber] = trackNumber
	result[media.TagEpisodeNumberPad] = trackNumberPad
	setIfNotBlank(result, media.TagEpisodePublicationDate, publicationDate)

	result[media.TagTrackDuration] = strconv.FormatInt(track.Duration, 10)

	return result
}

// buildDefaultTrackTags builds metadata tags for album, playlist, and standalone tracks.
func buildDefaultTrackTags(ctx *trackTagContext) map[string]string {
	track := ctx.track
	collection := ctx.audioCollection
	result := make(map[string]string, len(ctx.albumTags)+len(collection.tags)+16)
	maps.Copy(result, ctx.albumTags)
	maps.Copy(result, collection.tags)

	result[media.TagCollectionTitle] = collection.title
	setIfNotBlank(result, media.TagTrackGenre, strings.Join(track.Genres, ", "))
	fillCommonTrackTags(result, ctx.trackNumber, track)
	result[media.TagTrackCount] = strconv.FormatInt(collection.tracksCount, 10)

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
