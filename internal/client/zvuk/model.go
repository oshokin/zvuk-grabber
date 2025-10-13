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

// ChapterStreamMetadata represents all available stream URLs for a chapter.
// This is a simple data container - quality selection logic belongs in the service layer.
type ChapterStreamMetadata struct {
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

// UserProfile represents a user's profile information.
type UserProfile struct {
	// Subscription contains the user's subscription details.
	Subscription *UserSubscription `json:"subscription"`
}

// UserSubscription represents a user's subscription details.
type UserSubscription struct {
	// Title is the subscription plan name.
	Title string `json:"title"`
	// Expiration is the subscription expiration timestamp.
	Expiration int64 `json:"expiration"`
}

// Metadata represents a collection of metadata for tracks, playlists, releases, audiobooks, podcasts, and labels.
type Metadata struct {
	// Tracks is a map of track ID to track metadata.
	Tracks map[string]*Track `json:"tracks"`
	// Playlists is a map of playlist ID to playlist metadata.
	Playlists map[string]*Playlist `json:"playlists"`
	// Releases is a map of release ID to release metadata.
	Releases map[string]*Release `json:"releases"`
	// Audiobooks is a map of audiobook ID to audiobook metadata.
	Audiobooks map[string]*Audiobook `json:"abooks"`
	// Podcasts is a map of podcast ID to podcast metadata.
	Podcasts map[string]*Podcast `json:"podcasts"`
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

// Playlist represents metadata for a playlist.
type Playlist struct {
	// ID is the unique playlist identifier.
	ID int64 `json:"id"`
	// BigImageURL is the URL for the playlist's large cover image.
	BigImageURL string `json:"image_url_big"`
	// Title is the playlist name.
	Title string `json:"title"`
	// TrackIDs is the list of track IDs in the playlist.
	TrackIDs []int64 `json:"track_ids"`
}

// GetAudiobookResult represents the result of fetching audiobook data.
type GetAudiobookResult struct {
	// Audiobook is the audiobook metadata.
	Audiobook *Audiobook
	// Tracks is a map of chapter IDs to their track metadata.
	Tracks map[string]*Track
}

// GetPodcastResult represents the result of fetching podcast data.
type GetPodcastResult struct {
	// Podcast is the podcast metadata.
	Podcast *Podcast
	// Tracks is a map of episode IDs to their track metadata.
	Tracks map[string]*Track
}

// Audiobook represents metadata for an audiobook.
type Audiobook struct {
	// ID is the unique audiobook identifier.
	ID int64 `json:"id"`
	// BigImageURL is the URL for the audiobook's large cover image.
	BigImageURL string `json:"image_url_big"`
	// Title is the audiobook name.
	Title string `json:"title"`
	// ArtistNames is the list of author/narrator names for the audiobook.
	ArtistNames []string `json:"artist_names"`
	// TrackIDs is the list of track (chapter) IDs in the audiobook.
	TrackIDs []int64 `json:"track_ids"`
	// Date is the audiobook release date timestamp.
	Date int64 `json:"date"`
	// PublicationDate is the audiobook publication date.
	PublicationDate string `json:"publication_date"`
	// Copyright is the copyright holder.
	Copyright string `json:"copyright"`
	// Description is the audiobook description.
	Description string `json:"description"`
	// AgeLimit is the age rating.
	AgeLimit int64 `json:"age_limit"`
	// FullDuration is the total duration in seconds.
	FullDuration int64 `json:"full_duration"`
	// PublisherName is the publisher name.
	PublisherName string `json:"publisher_name"`
	// PublisherBrand is the publisher brand.
	PublisherBrand string `json:"publisher_brand"`
	// PerformerNames is the list of performer/narrator names.
	PerformerNames []string `json:"performer_names"`
	// Genres is the list of genre names.
	Genres []string `json:"genres"`
}

// Podcast represents metadata for a podcast.
type Podcast struct {
	// ID is the unique podcast identifier.
	ID int64 `json:"id"`
	// BigImageURL is the URL for the podcast's large cover image.
	BigImageURL string `json:"image_url_big"`
	// Title is the podcast name.
	Title string `json:"title"`
	// ArtistNames is the list of author/host names for the podcast.
	ArtistNames []string `json:"artist_names"`
	// TrackIDs is the list of track (episode) IDs in the podcast.
	TrackIDs []int64 `json:"track_ids"`
	// Description is the podcast description.
	Description string `json:"description"`
	// Category is the podcast category/genre.
	Category string `json:"category"`
	// Explicit indicates if the podcast contains explicit content.
	Explicit bool `json:"explicit"`
}

// Release represents metadata for a music release (e.g., album or single).
type Release struct {
	// ID is the unique release identifier.
	ID int64 `json:"id"`
	// Type indicates the release type (album, single, etc.).
	Type string `json:"type"`
	// ArtistIDs is the list of artist IDs associated with the release.
	ArtistIDs []int64 `json:"artist_ids"`
	// Title is the release name.
	Title string `json:"title"`
	// Image contains the release cover art metadata.
	Image *Image `json:"image"`
	// TrackIDs is the list of track IDs in the release.
	TrackIDs []int64 `json:"track_ids"`
	// ArtistNames is the list of artist names associated with the release.
	ArtistNames []string `json:"artist_names"`
	// Credits contains production and other credits information.
	Credits string `json:"credits"`
	// LabelID is the ID of the music label.
	LabelID int64 `json:"label_id"`
	// Date is the release date timestamp.
	Date int64 `json:"date"`
	// GenreIDs is the list of genre IDs for the release.
	GenreIDs []int64 `json:"genre_ids"`
}

// Track represents metadata for a music track.
type Track struct {
	// ID is the unique track identifier.
	ID int64 `json:"id"`
	// HasFLAC indicates whether FLAC quality is available.
	HasFLAC bool `json:"has_flac"`
	// ReleaseID is the ID of the release containing this track.
	ReleaseID int64 `json:"release_id"`
	// Lyrics indicates whether lyrics are available for this track.
	Lyrics bool `json:"lyrics"`
	// Credits contains production and other credits information.
	Credits string `json:"credits"`
	// Duration is the track length in seconds.
	Duration int64 `json:"duration"`
	// HighestQuality indicates the highest available audio quality.
	HighestQuality string `json:"highest_quality"`
	// Genres is the list of genre names for the track.
	Genres []string `json:"genres"`
	// Title is the track name.
	Title string `json:"title"`
	// ReleaseTitle is the name of the release containing this track.
	ReleaseTitle string `json:"release_title"`
	// Availability indicates the track's availability status.
	Availability int64 `json:"availability"`
	// ArtistNames is the list of artist names for the track.
	ArtistNames []string `json:"artist_names"`
	// Position is the track's position in the release.
	Position int64 `json:"position"`
	// Image contains the track cover art metadata.
	Image *Image `json:"image"`
}

// Image represents metadata for an image associated with a track, release, or playlist.
type Image struct {
	// SourceURL is the URL of the image.
	SourceURL string `json:"src"`
}

// Lyrics represents metadata for a track's lyrics.
type Lyrics struct {
	// Type indicates the lyrics format (subtitle, lrc, etc.).
	Type string `json:"type"`
	// Lyrics contains the actual lyrics content.
	Lyrics string `json:"lyrics"`
}

// Label represents metadata for a music label.
type Label struct {
	// Title is the label name.
	Title string `json:"title"`
}

// LyricsTypeSubtitle represents subtitle lyrics type.
const LyricsTypeSubtitle = "subtitle"

// LyricsTypeLRC represents LRC lyrics type.
const LyricsTypeLRC = "lrc"
