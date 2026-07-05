package model

import "time"

// TrackShort represents a playlist track entry with ordering metadata.
type TrackShort struct {
	// ID is the playlist entry identifier.
	ID FlexibleID `json:"id"`
	// OriginalIndex is the track index before playlist reordering.
	OriginalIndex int `json:"originalIndex"`
	// Timestamp is when the track was added to the playlist.
	Timestamp time.Time `json:"timestamp"`
	// Track holds the embedded track metadata.
	Track Track `json:"track"`
	// Recent reports whether the track was recently played.
	Recent bool `json:"recent"`
	// OriginalShuffleIndex is the track index before shuffle mode.
	OriginalShuffleIndex int `json:"originalShuffleIndex"`
}
