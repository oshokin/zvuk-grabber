package model

import "time"

// Account represents the authenticated Yandex Music account profile.
type Account struct {
	// Birthday is the account birthday string.
	Birthday string `json:"birthday,omitempty"`
	// DisplayName is the public display name.
	DisplayName string `json:"displayName,omitempty"`
	// FirstName is the account first name.
	FirstName string `json:"firstName,omitempty"`
	// FullName is the account full name.
	FullName string `json:"fullName,omitempty"`
	// HostedUser reports whether the account is a hosted child profile.
	HostedUser bool `json:"hostedUser,omitempty"`
	// Login is the account login name.
	Login string `json:"login,omitempty"`
	// Now is the server timestamp from the account status response.
	Now time.Time `json:"now"`
	// Region is the account region code.
	Region int `json:"region,omitempty"`
	// RegisteredAt is the account registration timestamp.
	RegisteredAt time.Time `json:"registeredAt"`
	// SecondName is the account last name.
	SecondName string `json:"secondName,omitempty"`
	// ServiceAvailable reports whether Yandex Music is available for the account.
	ServiceAvailable bool `json:"serviceAvailable"`
	// Uid is the numeric account user identifier.
	Uid int `json:"uid,omitempty"`
}
