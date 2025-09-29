package zvuk

// GetAlbumsMetadataResponse represents the response structure for fetching metadata about albums.
type GetAlbumsMetadataResponse struct {
	Tracks   map[string]*Track   `json:"tracks"`
	Releases map[string]*Release `json:"releases"`
}

// GetPlaylistsMetadataResponse represents the response structure for fetching metadata about playlists.
type GetPlaylistsMetadataResponse struct {
	Tracks    map[string]*Track    `json:"tracks"`
	Playlists map[string]*Playlist `json:"playlists"`
}

// GetUserProfileResponse represents the response structure for fetching a user's profile information.
type GetUserProfileResponse struct {
	Result *UserProfile `json:"result"`
}

// GetMetadataResponse represents the response structure for fetching general metadata.
type GetMetadataResponse struct {
	Result *Metadata `json:"result"`
}

// GetStreamMetadataResponse represents the response structure for fetching stream metadata.
type GetStreamMetadataResponse struct {
	Result *StreamMetadata `json:"result"`
}

// GetLyricsResponse represents the response structure for fetching lyrics.
type GetLyricsResponse struct {
	Result *Lyrics `json:"result"`
}

// GetLabelsMetadataResponse represents the response structure for fetching metadata about labels.
type GetLabelsMetadataResponse struct {
	Labels map[string]*Label `json:"labels"`
}

// UserProfile represents a user's profile information.
type UserProfile struct {
	Subscription *UserSubscription `json:"subscription"`
}

// UserSubscription represents a user's subscription details.
type UserSubscription struct {
	Title      string `json:"title"`
	Expiration int64  `json:"expiration"`
}

// Metadata represents a collection of metadata for tracks, playlists, releases, and labels.
type Metadata struct {
	Tracks    map[string]*Track    `json:"tracks"`
	Playlists map[string]*Playlist `json:"playlists"`
	Releases  map[string]*Release  `json:"releases"`
	Labels    map[string]*Label    `json:"labels"`
}

// StreamMetadata represents metadata for an audio stream.
type StreamMetadata struct {
	Stream string `json:"stream"`
}

// Playlist represents metadata for a playlist.
type Playlist struct {
	ID          int64   `json:"id"`
	BigImageURL string  `json:"image_url_big"`
	Title       string  `json:"title"`
	TrackIDs    []int64 `json:"track_ids"`
}

// Release represents metadata for a music release (e.g., album or single).
type Release struct {
	ID          int64    `json:"id"`
	Type        string   `json:"type"`
	ArtistIDs   []int64  `json:"artist_ids"`
	Title       string   `json:"title"`
	Image       *Image   `json:"image"`
	TrackIDs    []int64  `json:"track_ids"`
	ArtistNames []string `json:"artist_names"`
	Credits     string   `json:"credits"`
	LabelID     int64    `json:"label_id"`
	Date        int64    `json:"date"`
	GenreIDs    []int64  `json:"genre_ids"`
}

// Track represents metadata for a music track.
type Track struct {
	ID             int64    `json:"id"`
	HasFLAC        bool     `json:"has_flac"`
	ReleaseID      int64    `json:"release_id"`
	Lyrics         bool     `json:"lyrics"`
	Credits        string   `json:"credits"`
	Duration       int64    `json:"duration"`
	HighestQuality string   `json:"highest_quality"`
	Genres         []string `json:"genres"`
	Title          string   `json:"title"`
	ReleaseTitle   string   `json:"release_title"`
	Availability   int64    `json:"availability"`
	ArtistNames    []string `json:"artist_names"`
	Position       int64    `json:"position"`
	Image          *Image   `json:"image"`
}

// Image represents metadata for an image associated with a track, release, or playlist.
type Image struct {
	SourceURL string `json:"src"`
}

// Lyrics represents metadata for a track's lyrics.
type Lyrics struct {
	Type   string `json:"type"`
	Lyrics string `json:"lyrics"`
}

// Label represents metadata for a music label.
type Label struct {
	Title string `json:"title"`
}

// LyricsTypeSubtitle represents subtitle lyrics type.
const LyricsTypeSubtitle = "subtitle"

// LyricsTypeLRC represents LRC lyrics type.
const LyricsTypeLRC = "lrc"
