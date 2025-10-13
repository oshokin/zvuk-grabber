package zvuk

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// defaultFolderPermissions sets the default permissions for folders: (rwxr-xr-x).
	defaultFolderPermissions os.FileMode = 0o755

	// File extensions.
	extensionMP3  = ".mp3"
	extensionFLAC = ".flac"
	extensionBin  = ".bin"
	extensionJPG  = ".jpg"
	extensionPNG  = ".png"
	extensionTXT  = ".txt"
	extensionLRC  = ".lrc"

	// Default filenames and values.
	defaultCoverBasename       = "cover"
	defaultDescriptionBasename = "description"
	defaultUnknownYear         = "0000"
	trackNumberPaddingWidth    = 2
)

// DownloadCategory represents the type of content being downloaded.
type DownloadCategory uint8

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

// String returns a human-readable representation of the DownloadCategory.
func (dc DownloadCategory) String() string {
	switch dc {
	case DownloadCategoryUnknown:
		return "unknown"
	case DownloadCategoryTrack:
		return "track"
	case DownloadCategoryAlbum:
		return "album"
	case DownloadCategoryPlaylist:
		return "playlist"
	case DownloadCategoryArtist:
		return "artist"
	case DownloadCategoryAudiobook:
		return "audiobook"
	case DownloadCategoryPodcast:
		return "podcast"
	default:
		return fmt.Sprintf("unknown: %d", dc)
	}
}

// SkipReason represents why a track was skipped.
type SkipReason uint8

const (
	// SkipReasonExists - track file already exists.
	SkipReasonExists SkipReason = iota
	// SkipReasonQuality - track quality below minimum threshold.
	SkipReasonQuality
	// SkipReasonDuration - track duration outside acceptable range.
	SkipReasonDuration
)

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
	// Errors is a list of all errors encountered during the download process.
	Errors []DownloadError
}

// DownloadError represents a single error that occurred during download.
type DownloadError struct {
	// Category is the type of item that failed (track, album, playlist, artist).
	Category DownloadCategory
	// ItemID is the unique identifier of the item that failed.
	ItemID string
	// ItemTitle is the human-readable title of the item.
	ItemTitle string
	// ItemURL is the URL of the failed item (for albums/playlists/artists).
	ItemURL string
	// ErrorMessage is the error message.
	ErrorMessage string
	// Phase indicates when the error occurred (e.g., "fetching metadata", "downloading track").
	Phase string
	// ParentCategory is the type of parent collection (album/playlist) for tracks.
	ParentCategory DownloadCategory
	// ParentID is the ID of the parent collection.
	ParentID string
	// ParentTitle is the title of the parent collection.
	ParentTitle string
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

// String returns a human-readable representation of the DownloadItem.
func (di DownloadItem) String() string {
	return fmt.Sprintf("category: %v, ID: %s", di.Category, di.ItemID)
}

// GetShortVersion converts a full DownloadItem into a ShortDownloadItem by stripping the URL.
func (di DownloadItem) GetShortVersion() ShortDownloadItem {
	return ShortDownloadItem{
		Category: di.Category,
		ItemID:   di.ItemID,
	}
}

// TrackQuality represents the audio quality level.
type TrackQuality uint8

// Enum values for TrackQuality.
const (
	// TrackQualityUnknown represents an unknown or unspecified audio quality.
	TrackQualityUnknown TrackQuality = iota
	// TrackQualityMP3Mid represents MP3 format at 128 Kbps.
	TrackQualityMP3Mid
	// TrackQualityMP3High represents MP3 format at 320 Kbps.
	TrackQualityMP3High
	// TrackQualityFLAC represents FLAC lossless format.
	TrackQualityFLAC
)

// Constants for repeated string literals.
const (
	// TrackQualityMP3MidString is the string representation for mid quality.
	TrackQualityMP3MidString = "mid"
	// TrackQualityMP3HighString is the string representation for high quality.
	TrackQualityMP3HighString = "high"
	// TrackQualityFLACString is the string representation for FLAC quality.
	TrackQualityFLACString = "flac"
)

// String returns the display value of the Quality enum.
func (tq TrackQuality) String() string {
	//nolint:exhaustive // All meaningful cases are explicitly handled; default covers unknown values.
	switch tq {
	case TrackQualityMP3Mid:
		return "MP3, 128 Kbps (standard quality)"
	case TrackQualityMP3High:
		return "MP3, 320 Kbps (high quality)"
	case TrackQualityFLAC:
		return "FLAC, 16/24-bit (lossless quality)"
	default:
		return "unknown format"
	}
}

// Extension returns the file extension for the Quality enum.
func (tq TrackQuality) Extension() string {
	//nolint:exhaustive // All meaningful cases are explicitly handled; default covers unknown values.
	switch tq {
	case TrackQualityMP3High, TrackQualityMP3Mid:
		return extensionMP3
	case TrackQualityFLAC:
		return extensionFLAC
	default:
		return extensionBin
	}
}

// AsStreamURLParameterValue returns the API parameter value for the TrackQuality.
func (tq TrackQuality) AsStreamURLParameterValue() string {
	//nolint:exhaustive // All meaningful cases are explicitly handled; default covers unknown values.
	switch tq {
	case TrackQualityMP3Mid:
		return TrackQualityMP3MidString
	case TrackQualityMP3High:
		return TrackQualityMP3HighString
	case TrackQualityFLAC:
		return TrackQualityFLACString
	default:
		return ""
	}
}

// ParseQuality converts a string to a Quality enum.
func ParseQuality(s string) TrackQuality {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case TrackQualityMP3MidString, "med":
		return TrackQualityMP3Mid
	case TrackQualityMP3HighString:
		return TrackQualityMP3High
	case TrackQualityFLACString:
		return TrackQualityFLAC
	default:
		return TrackQualityUnknown
	}
}

// audioCollection represents a collection of audio tracks with associated metadata.
type audioCollection struct {
	// category indicates the type of collection (album, playlist, etc.).
	category DownloadCategory
	// title is the collection name.
	title string
	// tags contains metadata key-value pairs for the collection.
	tags map[string]string
	// tracksPath is the directory path where tracks will be saved.
	tracksPath string
	// coverPath is the file path for the collection's cover art.
	coverPath string
	// coverTempPath is the temporary UUID-based path for cover (used during concurrent downloads).
	coverTempPath string
	// descriptionTempPath is the temporary UUID-based path for description (audiobooks only).
	descriptionTempPath string
	// trackIDs is the list of track IDs in the collection.
	trackIDs []int64
	// tracksCount is the total number of tracks in the collection.
	tracksCount int64
}
