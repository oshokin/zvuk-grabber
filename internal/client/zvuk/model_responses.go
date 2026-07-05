package zvuk

import "io"

// GetAlbumsMetadataResponse represents the response structure for fetching metadata about albums.
type GetAlbumsMetadataResponse struct {
	// Tracks is a map of track ID to track metadata.
	Tracks map[string]*Track `json:"tracks"`
	// Releases is a map of release ID to release metadata.
	Releases map[string]*Release `json:"releases"`
}

// GetPlaylistsMetadataResponse represents the response structure for fetching metadata about playlists.
type GetPlaylistsMetadataResponse struct {
	// Tracks is a map of track ID to track metadata.
	Tracks map[string]*Track `json:"tracks"`
	// Playlists is a map of playlist ID to playlist metadata.
	Playlists map[string]*Playlist `json:"playlists"`
}

// GetAudiobooksMetadataResponse represents the response structure for fetching metadata about audiobooks.
type GetAudiobooksMetadataResponse struct {
	// Tracks is a map of track ID to track metadata.
	Tracks map[string]*Track `json:"tracks"`
	// Audiobooks is a map of audiobook ID to audiobook metadata.
	Audiobooks map[string]*Audiobook `json:"audiobooks"`
}

// GetPodcastsMetadataResponse represents the response structure for fetching metadata about podcasts.
type GetPodcastsMetadataResponse struct {
	// Tracks is a map of episode ID to episode metadata (represented as Track).
	Tracks map[string]*Track `json:"tracks"`
	// Podcasts is a map of podcast ID to podcast metadata.
	Podcasts map[string]*Podcast `json:"podcasts"`
}

// GetUserProfileResponse represents the response structure for fetching a user's profile information.
type GetUserProfileResponse struct {
	// Result contains the user's profile data.
	Result *UserProfile `json:"result"`
}

// GetMetadataResponse represents the response structure for fetching general metadata.
type GetMetadataResponse struct {
	// Result contains the requested metadata.
	Result *Metadata `json:"result"`
}

// GetStreamMetadataResponse represents the response structure for fetching stream metadata.
type GetStreamMetadataResponse struct {
	// Result contains the stream metadata including the URL.
	Result *StreamMetadata `json:"result"`
}

// StreamQualities represents all available stream URLs
// for a track, audiobook chapter, or podcast episode.
type StreamQualities struct {
	// Mid is the mid-quality (MP3 128kbps) stream URL.
	Mid string
	// High is the high-quality (MP3 320kbps) stream URL.
	High string
	// FLAC is the FLAC quality stream URL.
	FLAC string
}

// GetLyricsResponse represents the response structure for fetching lyrics.
type GetLyricsResponse struct {
	// Result contains the lyrics data.
	Result *Lyrics `json:"result"`
}

// GetLabelsMetadataResponse represents the response structure for fetching metadata about labels.
type GetLabelsMetadataResponse struct {
	// Labels is a map of label ID to label metadata.
	Labels map[string]*Label `json:"labels"`
}

// StreamMetadata represents metadata for an audio stream.
type StreamMetadata struct {
	// Stream is the URL for streaming the audio content.
	Stream string `json:"stream"`
}

// FetchTrackResult contains the result of FetchTrack operation.
type FetchTrackResult struct {
	// Body is the track audio data stream.
	Body io.ReadCloser
	// TotalBytes is the expected total size of the track in bytes.
	TotalBytes int64
}

// FetchJSONResult contains the result of fetching JSON data from the API.
type FetchJSONResult[T any] struct {
	// Data is the parsed JSON response.
	Data *T
	// StatusCode is the HTTP status code.
	StatusCode int
}
