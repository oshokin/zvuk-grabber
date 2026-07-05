package model

import "time"

// Playlist represents Yandex Music playlist metadata.
type Playlist struct {
	// Available reports whether the playlist can be accessed.
	Available bool `json:"available"`
	// Collective reports whether the playlist is collaborative.
	Collective bool `json:"collective"`
	// Cover holds playlist cover image metadata.
	Cover Cover `json:"cover"`
	// Created is the playlist creation timestamp.
	Created time.Time `json:"created"`
	// DurationMs is the total playlist duration in milliseconds.
	DurationMs int `json:"durationMs"`
	// HasTrailer reports whether the playlist has a trailer preview.
	HasTrailer bool `json:"hasTrailer"`
	// IsBanner reports whether the playlist is shown as a banner.
	IsBanner bool `json:"isBanner"`
	// IsPremiere reports whether the playlist is marked as a premiere.
	IsPremiere bool `json:"isPremiere"`
	// Kind is the numeric playlist identifier for legacy URLs.
	Kind int `json:"kind"`
	// LastOwnerPlaylists lists recent playlists from the same owner.
	LastOwnerPlaylists []Playlist `json:"lastOwnerPlaylists"`
	// LikesCount is the number of playlist likes.
	LikesCount int `json:"likesCount"`
	// Modified is the last modification timestamp.
	Modified time.Time `json:"modified"`
	// OGImage is the Open Graph image URL for the playlist.
	OGImage string `json:"ogImage"`
	// Owner is the playlist owner profile.
	Owner User `json:"owner"`
	// Pager contains pagination metadata for playlist tracks.
	Pager Pager `json:"pager"`
	// PlaylistUUID is the UUID playlist identifier for modern URLs.
	PlaylistUUID string `json:"playlistUuid"`
	// Revision is the playlist revision counter.
	Revision int `json:"revision"`
	// Snapshot is the playlist snapshot version.
	Snapshot int `json:"snapshot"`
	// Tags lists playlist classification tags.
	Tags []PlaylistTag `json:"tags"`
	// Title is the playlist display title.
	Title string `json:"title"`
	// TrackCount is the number of tracks in the playlist.
	TrackCount int `json:"trackCount"`
	// Tracks lists playlist track entries.
	Tracks []TrackShort `json:"tracks"`
	// UID is the owner user identifier.
	UID int `json:"uid"`
	// Visibility is the playlist visibility setting.
	Visibility string `json:"visibility"`
}
