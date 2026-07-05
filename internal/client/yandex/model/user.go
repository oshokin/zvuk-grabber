package model

// User represents a Yandex Music user profile.
type User struct {
	// UID is the numeric user identifier.
	UID int `json:"uid"`
	// Login is the user login name.
	Login string `json:"login"`
	// DisplayName is the public display name.
	DisplayName string `json:"displayName"`
	// FullName is the user full name.
	FullName string `json:"fullName"`
	// Sex is the user gender label.
	Sex string `json:"sex"`
	// Verified reports whether the user profile is verified.
	Verified bool `json:"verified"`
}
