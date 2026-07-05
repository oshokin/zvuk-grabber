package model

// ID wraps a flexible identifier value from API payloads.
type ID struct {
	// Value is the parsed identifier.
	Value FlexibleID `json:"id"`
}
