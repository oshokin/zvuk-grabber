package zvuk

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/media"
)

// DownloadCategory represents the type of content being downloaded.
type DownloadCategory uint8

// SkipReason represents why a track was skipped.
type SkipReason uint8

// DownloadItem represents a full downloadable item, including its category, URL, and unique identifier.
type DownloadItem struct {
	// Category is the type of content. (track, album, playlist, etc.).
	Category DownloadCategory
	// URL is the direct URL to the item.
	URL string
	// ItemID is the unique identifier of the item.
	ItemID string
}

// ShortDownloadItem is a lightweight version of DownloadItem without the URL.
// It is useful when storing or processing items without needing the actual download link.
type ShortDownloadItem struct {
	// Category is the type of content.
	Category DownloadCategory
	// ItemID is the unique identifier of the item.
	ItemID string
}

// DownloadStatistics tracks metrics for a download session.
type DownloadStatistics struct {
	// StartTime is when the download session began.
	StartTime time.Time
	// EndTime is when the download session completed.
	EndTime time.Time
	// IsDryRun indicates if this was a dry-run preview.
	IsDryRun bool
	// TotalTracksProcessed is the total number of tracks attempted.
	TotalTracksProcessed int64
	// TracksDownloaded is the number of tracks successfully downloaded.
	TracksDownloaded int64
	// TracksSkipped is the total number of tracks skipped for any reason.
	TracksSkipped int64
	// TracksSkippedExists is the number of tracks skipped because they already exist.
	TracksSkippedExists int64
	// TracksSkippedQuality is the number of tracks skipped due to quality threshold.
	TracksSkippedQuality int64
	// TracksSkippedDuration is the number of tracks skipped due to duration threshold.
	TracksSkippedDuration int64
	// TracksFailed is the number of tracks that failed to download.
	TracksFailed int64
	// TotalBytesDownloaded is the total size of downloaded content in bytes.
	TotalBytesDownloaded int64
	// LyricsDownloaded is the number of lyrics files downloaded.
	LyricsDownloaded int64
	// LyricsSkipped is the number of lyrics files skipped (already exist).
	LyricsSkipped int64
	// CoversDownloaded is the number of cover art files downloaded.
	CoversDownloaded int64
	// CoversSkipped is the number of cover art files skipped (already exist).
	CoversSkipped int64
	// DescriptionsSaved is the number of description files downloaded.
	DescriptionsSaved int64
	// DescriptionsSkipped is the number of description files skipped (already exist).
	DescriptionsSkipped int64
	// Errors is a list of all errors encountered during the download process.
	Errors []*DownloadError
}

// DownloadError represents a single error that occurred during download.
type DownloadError struct {
	// Category is the type of item that failed (track, album, playlist, etc.).
	Category DownloadCategory
	// ItemID is the unique identifier of the item that failed.
	ItemID string
	// ItemTitle is the human-readable title of the item.
	ItemTitle string
	// ItemURL is the URL of the failed item (for albums/playlists/artists).
	ItemURL string
	// ParentCategory is the type of parent collection (album/playlist) for tracks.
	ParentCategory DownloadCategory
	// ParentID is the ID of the parent collection.
	ParentID string
	// ParentTitle is the title of the parent collection.
	ParentTitle string
	// Phase indicates when the error occurred (e.g., "fetching metadata", "downloading track").
	Phase string
	// Error is the error.
	Error error
}

// DownloadTrackResult contains the result of downloadAndSaveTrack operation.
type DownloadTrackResult struct {
	// IsExist indicates whether the track file already existed (download was skipped).
	IsExist bool
	// TempPath is the path to the temporary .part file (empty if download was skipped or failed).
	TempPath string
	// BytesDownloaded is the number of bytes successfully downloaded.
	BytesDownloaded int64
}

// TrackQuality represents the audio quality level.
type TrackQuality = media.Quality

// audioCollection represents a collection of audio tracks with associated metadata.
type audioCollection struct {
	// category indicates the type of collection (album, playlist, etc.).
	category DownloadCategory
	// id is the collection ID.
	id string
	// title is the collection name.
	title string
	// tags contains metadata key-value pairs for the collection.
	tags map[string]string
	// tracksPath is the directory path where tracks will be saved.
	tracksPath string
	// embeddableCoverPath is the path for cover that will be embedded to the track tags.
	embeddableCoverPath string
	// coverPath is the file path for the collection's cover art.
	coverPath string
	// embeddableDescriptionPath is the path for description that will be embedded to the track tags.
	embeddableDescriptionPath string
	// descriptionPath is the file path for the collection's description.
	descriptionPath string
	// trackIDs is the list of track IDs in the collection.
	trackIDs []int64
	// tracksCount is the total number of tracks in the collection.
	tracksCount int64
}

const (
	// defaultFolderPermissions sets the default permissions for folders: (rwxr-xr-x).
	defaultFolderPermissions os.FileMode = 0o755

	// File extensions.
	extensionMP3 = ".mp3"
	// extensionJPG is the file extension for JPEG cover images.
	extensionJPG = ".jpg"
	// extensionPNG is the file extension for PNG cover images.
	extensionPNG = ".png"
	// extensionTXT is the file extension for plain-text description files.
	extensionTXT = ".txt"
	// extensionLRC is the file extension for lyrics files.
	extensionLRC = ".lrc"

	// Default filenames and values.
	defaultCoverFilename = "cover"
	// defaultDescriptionFilename is the default base filename for collection descriptions.
	defaultDescriptionFilename = "description"
	// defaultUnknownYear is the placeholder year when release year is unknown.
	defaultUnknownYear = "0000"
	// trackNumberPaddingWidth is the zero-padding width for track number tags.
	trackNumberPaddingWidth = 2

	// downloadCategoryUnknownName is the lowercase name for unknown download category.
	downloadCategoryUnknownName = "unknown"
	// downloadCategoryTrackName is the lowercase name for track download category.
	downloadCategoryTrackName = "track"
	// downloadCategoryAlbumName is the lowercase name for album download category.
	downloadCategoryAlbumName = "album"
	// downloadCategoryPlaylistName is the lowercase name for playlist download category.
	downloadCategoryPlaylistName = "playlist"
	// downloadCategoryArtistName is the lowercase name for artist download category.
	downloadCategoryArtistName = "artist"
)

const (
	// DownloadCategoryUnknown - unknown category.
	DownloadCategoryUnknown DownloadCategory = iota
	// DownloadCategoryTrack - single track.
	DownloadCategoryTrack
	// DownloadCategoryAlbum - full album.
	DownloadCategoryAlbum
	// DownloadCategoryPlaylist - playlist.
	DownloadCategoryPlaylist
	// DownloadCategoryArtist - complete artist's discography.
	DownloadCategoryArtist
	// DownloadCategoryAudiobook - audiobook.
	DownloadCategoryAudiobook
	// DownloadCategoryPodcast - podcast.
	DownloadCategoryPodcast
)

const (
	// SkipReasonExists - track file already exists.
	SkipReasonExists SkipReason = iota
	// SkipReasonQuality - track quality below minimum threshold.
	SkipReasonQuality
	// SkipReasonDuration - track duration outside acceptable range.
	SkipReasonDuration
)

// Enum values for TrackQuality.
const (
	// TrackQualityUnknown represents an unknown or unspecified audio quality.
	TrackQualityUnknown = media.QualityUnknown
	// TrackQualityMP3Mid represents MP3 format at 128 Kbps.
	TrackQualityMP3Mid = media.QualityMP3Mid
	// TrackQualityMP3High represents MP3 format at 320 Kbps.
	TrackQualityMP3High = media.QualityMP3High
	// TrackQualityFLAC represents FLAC lossless format.
	TrackQualityFLAC = media.QualityFLAC
)

// Constants for repeated string literals.
const (
	// TrackQualityMP3MidString is the string representation for mid quality.
	TrackQualityMP3MidString = media.QualityMP3MidString
	// TrackQualityMP3HighString is the string representation for high quality.
	TrackQualityMP3HighString = media.QualityMP3HighString
	// TrackQualityFLACString is the string representation for FLAC quality.
	TrackQualityFLACString = media.QualityFLACString
)

// downloadCategoryNames maps DownloadCategory values to display strings.
var downloadCategoryNames = [...]struct {
	// lower is the lowercase category name used in paths and logs.
	lower string
	// title is the human-readable title-case category name.
	title string
}{
	{lower: downloadCategoryUnknownName, title: "Unknown"},
	{lower: downloadCategoryTrackName, title: "Track"},
	{lower: downloadCategoryAlbumName, title: "Album"},
	{lower: downloadCategoryPlaylistName, title: "Playlist"},
	{lower: downloadCategoryArtistName, title: "Artist"},
	{lower: "audiobook", title: "Audiobook"},
	{lower: "podcast", title: "Podcast"},
}

// String returns a human-readable representation of the DownloadCategory.
func (dc DownloadCategory) String() string {
	if int(dc) < len(downloadCategoryNames) {
		return downloadCategoryNames[dc].lower
	}

	return fmt.Sprintf("unknown: %d", dc)
}

// ToTitleCase returns the title case representation of the DownloadCategory.
func (dc DownloadCategory) ToTitleCase() string {
	if int(dc) < len(downloadCategoryNames) {
		return downloadCategoryNames[dc].title
	}

	return fmt.Sprintf("Unknown: %d", dc)
}

// IsChapterCollection returns true for collections whose tracks are chapters or episodes.
func (dc DownloadCategory) IsChapterCollection() bool {
	return dc == DownloadCategoryAudiobook || dc == DownloadCategoryPodcast
}

// ToSubcategory returns the subcategory representation of the DownloadCategory.
func (dc DownloadCategory) ToSubcategory() string {
	switch dc {
	case DownloadCategoryArtist:
		return downloadCategoryAlbumName
	case DownloadCategoryAudiobook:
		return "chapter"
	case DownloadCategoryPodcast:
		return "episode"
	default:
		return downloadCategoryTrackName
	}
}

// IsSupported returns true if the category is supported for downloading.
func (dc DownloadCategory) IsSupported() bool {
	return dc == DownloadCategoryAlbum || dc == DownloadCategoryPlaylist || dc.IsChapterCollection()
}

// String returns a human-readable representation of the SkipReason.
func (sr SkipReason) String() string {
	switch sr {
	case SkipReasonExists:
		return "already exists"
	case SkipReasonQuality:
		return "quality filter"
	case SkipReasonDuration:
		return "duration filter"
	default:
		return fmt.Sprintf("unknown reason: %d", sr)
	}
}

// String returns a human-readable representation of the DownloadItem.
func (di *DownloadItem) String() string {
	return fmt.Sprintf("category: %v, ID: %s", di.Category, di.ItemID)
}

// GetShortVersion converts a full DownloadItem into a ShortDownloadItem by stripping the URL.
func (di *DownloadItem) GetShortVersion() ShortDownloadItem {
	return ShortDownloadItem{
		Category: di.Category,
		ItemID:   di.ItemID,
	}
}

// ParseQuality converts a string to a Quality enum.
func ParseQuality(s string) TrackQuality {
	return media.ParseQuality(strings.ToLower(strings.TrimSpace(s)))
}
