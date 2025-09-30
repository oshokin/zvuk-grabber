package zvuk

import (
	"fmt"
	"strings"
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
	default:
		return fmt.Sprintf("unknown: %d", dc)
	}
}

// DownloadItem represents a full downloadable item, including its category, URL, and unique identifier.
type DownloadItem struct {
	// Type of content (track, album, playlist, etc.)
	Category DownloadCategory
	// Direct URL to the item
	URL string
	// Unique identifier of the item
	ItemID string
}

// ShortDownloadItem is a lightweight version of DownloadItem without the URL.
// It is useful when storing or processing items without needing the actual download link.
type ShortDownloadItem struct {
	// Type of content
	Category DownloadCategory
	// Unique identifier of the item
	ItemID string
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
	// trackIDs is the list of track IDs in the collection.
	trackIDs []int64
	// tracksCount is the total number of tracks in the collection.
	tracksCount int64
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
		return ".mp3"
	case TrackQualityFLAC:
		return ".flac"
	default:
		return ".bin"
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
