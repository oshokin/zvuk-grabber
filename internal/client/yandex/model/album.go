package model

// Album represents Yandex Music album metadata including track volumes.
type Album struct {
	// ID is the album identifier.
	ID *FlexibleID `json:"id"`
	// Title is the album display title.
	Title string `json:"title"`
	// Type is the album type label, such as audiobook or podcast.
	Type string `json:"type,omitempty"`
	// MetaType is the secondary album classification label.
	MetaType string `json:"metaType,omitempty"`
	// TrackCount is the total number of tracks in the album.
	TrackCount int `json:"trackCount,omitempty"`
	// Available reports whether the album can be streamed or downloaded.
	Available bool `json:"available"`
	// CoverURI is the cover image URI template for the album.
	CoverURI string `json:"coverUri,omitempty"`
	// Artists lists the album performing artists.
	Artists []Artist `json:"artists,omitempty"`
	// Description is the album description text.
	Description string `json:"description,omitempty"`
	// Genre is the album genre label.
	Genre string `json:"genre,omitempty"`
	// Year is the album release year.
	Year int `json:"year,omitempty"`
	// ReleaseDate is the album release date string.
	ReleaseDate string `json:"releaseDate,omitempty"`
	// TrackPosition is the track position when the album is embedded in a track.
	TrackPosition TrackPosition `json:"trackPosition,omitzero"`
	// Volumes lists album discs or volumes, each containing tracks.
	Volumes [][]Track `json:"volumes,omitempty"`
}

// TrackPosition describes a track index within an album volume.
type TrackPosition struct {
	// Volume is the one-based disc or volume number.
	Volume int `json:"volume,omitempty"`
	// Index is the one-based track index within the volume.
	Index int `json:"index,omitempty"`
}
