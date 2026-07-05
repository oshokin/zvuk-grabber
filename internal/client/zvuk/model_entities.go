package zvuk

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
	// Title is the release name.
	Title string `json:"title"`
	// Image contains the release cover art metadata.
	Image *Image `json:"image"`
	// TrackIDs is the list of track IDs in the release.
	TrackIDs []int64 `json:"track_ids"`
	// ArtistNames is the list of artist names associated with the release.
	ArtistNames []string `json:"artist_names"`
	// LabelID is the ID of the music label.
	LabelID int64 `json:"label_id"`
	// Date is the release date timestamp.
	Date int64 `json:"date"`
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
