package model

// Artist represents a Yandex Music artist reference.
type Artist struct {
	// ID is the artist identifier.
	ID FlexibleID `json:"id"`
	// Name is the artist display name.
	Name string `json:"name"`
}
