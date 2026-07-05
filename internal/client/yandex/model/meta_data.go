package model

// MetaData represents extended track metadata from Yandex Music API payloads.
type MetaData struct {
	// RealID is the canonical track identifier.
	RealID string `json:"realId"`
	// State is the track processing or availability state.
	State string `json:"state"`
	// StorageDir is the provider storage directory prefix.
	StorageDir string `json:"storageDir"`
	// Title is the metadata title override.
	Title string `json:"title"`
	// TrackSource is the track source label.
	TrackSource string `json:"trackSource"`
	// UGCAtristName is the user-generated content artist name.
	UGCAtristName string `json:"ugcArtistName,omitempty"`
	// UserInfo holds the uploading user profile.
	UserInfo User `json:"userInfo"`
	// Volume is the album volume number for the track.
	Volume int `json:"volume"`
	// Year is the track release year.
	Year int `json:"year"`
}
