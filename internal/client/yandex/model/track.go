package model

import (
	"fmt"
	"strings"
)

// Track represents Yandex Music track metadata.
type Track struct {
	// Available reports whether the track can be streamed or downloaded.
	Available bool `json:"available"`
	// Artists lists the track performing artists.
	Artists []Artist `json:"artists"`
	// Albums lists album releases that contain this track.
	Albums []Album `json:"albums"`
	// CanPublish reports whether the track can be published by the user.
	CanPublish bool `json:"canPublish"`
	// CoverURI is the cover image URI template for the track.
	CoverURI string `json:"coverUri,omitempty"`
	// DesiredVisibility is the user-requested visibility setting.
	DesiredVisibility string `json:"desiredVisibility"`
	// DurationMs is the track duration in milliseconds.
	DurationMs int `json:"durationMs"`
	// Filename is the provider-side filename hint.
	Filename string `json:"filename"`
	// ID is the track identifier.
	ID *FlexibleID `json:"id"`
	// LyricsAvailable reports whether the API advertises lyrics for this track.
	LyricsAvailable bool `json:"lyricsAvailable"`
	// MetaData holds extended track metadata.
	MetaData MetaData `json:"metaData,omitzero"`
	// PubDate is the episode or chapter publication date, typically YYYY-MM-DD.
	PubDate string `json:"pubDate,omitempty"`
	// Title is the track title without version suffix.
	Title string `json:"title"`
	// Version is the track version or remix label.
	Version string `json:"version"`
}

// FullTitle returns the track title combined with its version suffix.
func (t *Track) FullTitle() string {
	if t == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s", t.Title, t.Version))
}

// ArtistsString joins track artist names into a comma-separated string.
func (t *Track) ArtistsString() string {
	if t == nil {
		return ""
	}

	artists := make([]string, len(t.Artists))
	for i, artist := range t.Artists {
		artists[i] = artist.Name
	}

	return strings.Join(artists, ", ")
}
