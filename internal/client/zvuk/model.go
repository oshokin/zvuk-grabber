package zvuk

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

// Metadata represents a collection of metadata for tracks, playlists, releases, and labels.
type Metadata struct {
	// Tracks is a map of track ID to track metadata.
	Tracks map[string]*Track `json:"tracks"`
	// Playlists is a map of playlist ID to playlist metadata.
	Playlists map[string]*Playlist `json:"playlists"`
	// Releases is a map of release ID to release metadata.
	Releases map[string]*Release `json:"releases"`
	// Labels is a map of label ID to label metadata.
	Labels map[string]*Label `json:"labels"`
}

// StreamMetadata represents metadata for an audio stream.
type StreamMetadata struct {
	// Stream is the URL for streaming the audio content.
	Stream string `json:"stream"`
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
