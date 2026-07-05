package model

// Cover represents Yandex Music cover image metadata.
type Cover struct {
	// Custom reports whether the cover is user-uploaded.
	Custom bool `json:"custom"`
	// Dir is the cover storage directory prefix.
	Dir string `json:"dir"`
	// Type is the cover image type label.
	Type string `json:"type"`
	// URI is the cover image URI template.
	URI string `json:"uri"`
	// Version is the cover image version token.
	Version string `json:"version"`
}
