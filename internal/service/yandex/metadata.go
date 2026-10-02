package yandex

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// buildTargetPath renders the final output path for a track at the given quality.
func (s *ServiceImpl) buildTargetPath(
	ctx context.Context,
	job *trackJob,
	tags map[string]string,
	quality media.Quality,
) string {
	extension := quality.Extension()

	filename := s.buildTrackFilename(ctx, job, tags)
	filename = utils.SanitizeTemplateName(filename) + extension

	createFolderForSingles := s.cfg != nil && s.cfg.CreateFolderForSingles
	singleWithoutFolder := isSingleWithoutFolder(job, createFolderForSingles)

	folder := ""

	switch job.kind {
	case collectionAlbum:
		if !singleWithoutFolder {
			folder = s.templateManager.GetAlbumFolderName(ctx, tags)
		}
	case collectionAudiobook:
		if !singleWithoutFolder {
			folder = s.templateManager.GetAudiobookFolderName(ctx, tags)
		}
	case collectionPodcast:
		if !singleWithoutFolder {
			folder = s.templateManager.GetPodcastFolderName(ctx, tags)
		}
	case collectionPlaylist:
		folder = job.collectionTitle
	case collectionTrack:
		if !singleWithoutFolder {
			folder = s.templateManager.GetAlbumFolderName(ctx, tags)
		}
	}

	if strings.TrimSpace(folder) != "" {
		folder = utils.SanitizeTemplateName(folder)
	}

	if strings.TrimSpace(folder) == "" {
		return filepath.Join(s.outputPath(), filename)
	}

	if s.cfg.MaxFolderNameLength > 0 && int64(len([]rune(folder))) > s.cfg.MaxFolderNameLength {
		folder = string([]rune(folder)[:s.cfg.MaxFolderNameLength])
	}

	return filepath.Join(s.outputPath(), folder, filename)
}

// buildTrackFilename renders the output filename using collection-specific templates.
func (s *ServiceImpl) buildTrackFilename(ctx context.Context, job *trackJob, tags map[string]string) string {
	if job == nil {
		return s.templateManager.GetTrackFilename(ctx, false, tags, 0)
	}

	if job.kind == collectionAudiobook {
		return s.templateManager.GetAudiobookChapterFilename(ctx, tags, int64(job.trackCount))
	}

	if job.kind == collectionPodcast {
		return s.templateManager.GetPodcastEpisodeFilename(ctx, tags, int64(job.trackCount))
	}

	return s.templateManager.GetTrackFilename(ctx, job.kind == collectionPlaylist, tags, int64(job.trackCount))
}

// buildTags assembles template and tag-writer metadata for a track job.
//
//nolint:funlen // Centralized tag map keeps template inputs and tag writing consistent.
func (s *ServiceImpl) buildTags(job *trackJob) map[string]string {
	if job == nil || job.track == nil {
		return map[string]string{}
	}

	album := job.album

	trackNumber := job.trackNumber
	if trackNumber <= 0 {
		trackNumber = trackIndexFromAlbum(album, 1)
	}

	trackCount := job.trackCount
	if trackCount <= 0 {
		trackCount = trackCountFromAlbum(album, 1)
	}

	releaseDate := collectionReleaseDate(album, job.track)
	releaseYear := collectionReleaseYear(album, job.track)

	trackArtist := job.track.ArtistsString()

	albumArtist := albumArtists(album)
	if strings.TrimSpace(albumArtist) == "" {
		albumArtist = trackArtist
	}

	audiobookTitle := albumTitle(album, job.collectionTitle)

	audiobookAuthors := albumArtist
	if strings.TrimSpace(audiobookAuthors) == "" {
		audiobookAuthors = trackArtist
	}

	podcastTitle := albumTitle(album, job.collectionTitle)

	podcastAuthors := albumArtist
	if strings.TrimSpace(podcastAuthors) == "" {
		podcastAuthors = trackArtist
	}

	episodePublicationDate := trackPublicationDate(job.track)

	return map[string]string{
		media.TagType:                     job.kind,
		media.TagCollectionTitle:          job.collectionTitle,
		media.TagAlbumArtist:              albumArtist,
		media.TagAlbumID:                  albumID(album),
		media.TagAlbumTitle:               albumTitle(album, job.collectionTitle),
		media.TagAlbumTrackCount:          strconv.Itoa(trackCount),
		media.TagReleaseDate:              releaseDate,
		media.TagReleaseYear:              releaseYear,
		media.TagPlaylistID:               playlistID(job.playlist),
		media.TagPlaylistTitle:            playlistTitle(job.playlist),
		media.TagPlaylistTrackCount:       strconv.Itoa(trackCount),
		media.TagTrackArtist:              trackArtist,
		media.TagTrackCount:               strconv.Itoa(trackCount),
		media.TagTrackDuration:            (time.Duration(job.track.DurationMs) * time.Millisecond).String(),
		media.TagTrackGenre:               albumGenre(album),
		media.TagTrackID:                  yandexTrackID(job.track),
		media.TagTrackNumber:              strconv.Itoa(trackNumber),
		media.TagTrackNumberPad:           fmt.Sprintf("%02d", trackNumber),
		media.TagTrackTitle:               job.track.FullTitle(),
		media.TagAudiobookID:              albumID(album),
		media.TagAudiobookTitle:           audiobookTitle,
		media.TagAudiobookAuthors:         audiobookAuthors,
		media.TagAudiobookTrackCount:      strconv.Itoa(trackCount),
		media.TagAudiobookDescription:     albumDescription(album),
		media.TagAudiobookGenres:          albumGenre(album),
		media.TagAudiobookPublicationDate: releaseDate,
		media.TagPublishYear:              releaseYear,
		media.TagPodcastID:                albumID(album),
		media.TagPodcastTitle:             podcastTitle,
		media.TagPodcastAuthors:           podcastAuthors,
		media.TagPodcastTrackCount:        strconv.Itoa(trackCount),
		media.TagPodcastDescription:       albumDescription(album),
		media.TagPodcastCategory:          albumGenre(album),
		media.TagEpisodePublicationDate:   episodePublicationDate,
		media.TagEpisodeID:                yandexTrackID(job.track),
		media.TagEpisodeTitle:             job.track.FullTitle(),
		media.TagEpisodeNumber:            strconv.Itoa(trackNumber),
		media.TagEpisodeNumberPad:         fmt.Sprintf("%02d", trackNumber),
		media.TagEpisodeDuration:          (time.Duration(job.track.DurationMs) * time.Millisecond).String(),
	}
}

// preferredQuality maps config quality to a concrete media quality with MP3-high default.
func preferredQuality(quality uint8) media.Quality {
	value := media.FromConfig(quality)
	if value == media.QualityUnknown {
		return media.QualityMP3High
	}

	return value
}

// collectionKindFromAlbum maps Yandex album metadata to a collection kind string.
func collectionKindFromAlbum(album *model.Album) string {
	if album == nil {
		return collectionAlbum
	}

	albumType := strings.TrimSpace(album.Type)
	if strings.EqualFold(albumType, albumTypeAudiobook) {
		return collectionAudiobook
	}

	if strings.EqualFold(albumType, albumTypePodcast) ||
		strings.EqualFold(strings.TrimSpace(album.MetaType), albumMetaTypePodcast) {
		return collectionPodcast
	}

	return collectionAlbum
}

// firstAlbum returns the first embedded album from a track, if present.
func firstAlbum(track *model.Track) *model.Album {
	if track == nil || len(track.Albums) == 0 {
		return nil
	}

	return &track.Albums[0]
}

// yandexTrackID returns a trimmed string track ID or an empty string when unavailable.
func yandexTrackID(track *model.Track) string {
	if track == nil || track.ID == nil {
		return ""
	}

	return strings.TrimSpace(track.ID.String())
}

// yandexJobTrackID returns a trimmed string track ID from a job or an empty string when unavailable.
func yandexJobTrackID(job *trackJob) string {
	if job == nil {
		return ""
	}

	return yandexTrackID(job.track)
}

// albumID returns the string album ID or an empty string when album is nil.
func albumID(album *model.Album) string {
	if album == nil || album.ID == nil {
		return ""
	}

	return album.ID.String()
}

// albumTitle returns the album title or the provided fallback when missing.
func albumTitle(album *model.Album, fallback string) string {
	if album == nil || strings.TrimSpace(album.Title) == "" {
		return fallback
	}

	return album.Title
}

// albumGenre returns the album genre or an empty string when album is nil.
func albumGenre(album *model.Album) string {
	if album == nil {
		return ""
	}

	return album.Genre
}

// albumDescription returns the trimmed album description or an empty string.
func albumDescription(album *model.Album) string {
	if album == nil {
		return ""
	}

	return strings.TrimSpace(album.Description)
}

// albumArtists joins album artist names into a comma-separated string.
func albumArtists(album *model.Album) string {
	if album == nil || len(album.Artists) == 0 {
		return ""
	}

	artists := make([]string, 0, len(album.Artists))
	for _, artist := range album.Artists {
		name := strings.TrimSpace(artist.Name)
		if name == "" {
			continue
		}

		artists = append(artists, name)
	}

	return strings.Join(artists, ", ")
}

// collectionReleaseDate returns the best available collection or track publication date.
func collectionReleaseDate(album *model.Album, track *model.Track) string {
	if album != nil {
		if date := normalizeDateString(album.ReleaseDate); date != "" {
			return date
		}
	}

	if date := trackPublicationDate(track); date != "" {
		return date
	}

	return earliestAlbumPublicationDate(album)
}

// collectionReleaseYear extracts the release year from album, track, or publication date metadata.
func collectionReleaseYear(album *model.Album, track *model.Track) string {
	if album != nil && album.Year > 0 {
		return strconv.Itoa(album.Year)
	}

	if track != nil && track.MetaData.Year > 0 {
		return strconv.Itoa(track.MetaData.Year)
	}

	if date := collectionReleaseDate(album, track); len(date) >= 4 {
		return date[:4]
	}

	return ""
}

// trackPublicationDate returns a normalized publication date from track metadata.
func trackPublicationDate(track *model.Track) string {
	if track == nil {
		return ""
	}

	return normalizeDateString(track.PubDate)
}

// earliestAlbumPublicationDate returns the earliest track pubDate in an album.
func earliestAlbumPublicationDate(album *model.Album) string {
	if album == nil {
		return ""
	}

	earliest := ""

	for _, volume := range album.Volumes {
		for i := range volume {
			date := trackPublicationDate(&volume[i])
			if date == "" {
				continue
			}

			if earliest == "" || date < earliest {
				earliest = date
			}
		}
	}

	return earliest
}

// normalizeDateString converts a Yandex date string to YYYY-MM-DD when possible.
func normalizeDateString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Format("2006-01-02")
	}

	if len(raw) >= 10 && raw[4] == '-' && raw[7] == '-' {
		return raw[:10]
	}

	return raw
}

// trackIndexFromAlbum returns the album track index or the provided fallback.
func trackIndexFromAlbum(album *model.Album, fallback int) int {
	if album != nil && album.TrackPosition.Index > 0 {
		return album.TrackPosition.Index
	}

	return fallback
}

// trackCountFromAlbum returns the album track count or the provided fallback.
func trackCountFromAlbum(album *model.Album, fallback int) int {
	if album != nil && album.TrackCount > 0 {
		return album.TrackCount
	}

	return fallback
}

// countAlbumTracks returns the total number of tracks across album volumes.
func countAlbumTracks(album *model.Album) int {
	if album == nil {
		return 0
	}

	if album.TrackCount > 0 {
		return album.TrackCount
	}

	count := 0
	for _, volume := range album.Volumes {
		count += len(volume)
	}

	return count
}

// playlistID returns the playlist UUID or numeric kind as a string identifier.
func playlistID(playlist *model.Playlist) string {
	if playlist == nil {
		return ""
	}

	if strings.TrimSpace(playlist.PlaylistUUID) != "" {
		return playlist.PlaylistUUID
	}

	if playlist.Kind > 0 {
		return strconv.Itoa(playlist.Kind)
	}

	return ""
}

// playlistTitle returns the playlist title or an empty string when playlist is nil.
func playlistTitle(playlist *model.Playlist) string {
	if playlist == nil {
		return ""
	}

	return playlist.Title
}

// coverURL resolves the best available cover image URL for a track job.
func coverURL(job *trackJob) string {
	if job == nil || job.track == nil {
		return ""
	}

	uri := strings.TrimSpace(job.track.CoverURI)
	if uri == "" && job.album != nil {
		uri = strings.TrimSpace(job.album.CoverURI)
	}

	if uri == "" && job.playlist != nil {
		uri = strings.TrimSpace(job.playlist.Cover.URI)
	}

	if uri == "" && job.playlist != nil {
		uri = strings.TrimSpace(job.playlist.OGImage)
	}

	if uri == "" {
		return ""
	}

	uri = strings.ReplaceAll(uri, "%%", maxCoverSize)
	switch {
	case strings.HasPrefix(uri, "http://"), strings.HasPrefix(uri, "https://"):
		return uri
	case strings.HasPrefix(uri, "//"):
		return "https:" + uri
	default:
		return "https://" + uri
	}
}

// buildCoverPath chooses a sidecar or shared cover path next to the track output.
func (s *ServiceImpl) buildCoverPath(targetPath string, job *trackJob) string {
	coverDir := filepath.Dir(targetPath)
	if strings.TrimSpace(coverDir) == "" || coverDir == "." {
		return ""
	}

	createFolderForSingles := s.cfg != nil && s.cfg.CreateFolderForSingles

	// Keep singles without per-track folders close to Zvuk behavior:
	// one sidecar image next to the track, based on the track filename.
	// Podcasts can expose episode-specific cover images, so store sidecar covers
	// per episode to avoid races/overwrites in concurrent downloads.
	if job != nil && (isSingleWithoutFolder(job, createFolderForSingles) || job.kind == collectionPodcast) {
		baseName := strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath))
		if strings.TrimSpace(baseName) != "" {
			return filepath.Join(coverDir, baseName+coverExtension)
		}
	}

	// For album/playlist/audiobook folders (and singles in dedicated folders), keep a stable shared cover file.
	return filepath.Join(coverDir, defaultCoverName)
}

// isSingleWithoutFolder reports whether a job should be saved without a dedicated folder.
func isSingleWithoutFolder(job *trackJob, createFolderForSingles bool) bool {
	if job == nil || createFolderForSingles {
		return false
	}

	switch job.kind {
	case collectionTrack:
		return true
	case collectionAlbum, collectionAudiobook, collectionPodcast:
		return job.trackCount == 1
	default:
		return false
	}
}
