package model

import (
	"bytes"
	"encoding/json"
	"errors"
)

// FlexibleID is a string-like identifier that accepts JSON numbers or strings.
type FlexibleID struct {
	// value holds the normalized identifier text.
	value string
}

// errEmptyID is returned when a FlexibleID JSON payload is empty.
var errEmptyID = errors.New("empty id")

// NewFlexibleID creates a FlexibleID from a plain string value.
func NewFlexibleID(value string) *FlexibleID {
	return &FlexibleID{value: value}
}

// String returns the identifier as a plain string.
func (id *FlexibleID) String() string {
	if id == nil {
		return ""
	}

	return id.value
}

// MarshalJSON encodes the identifier as a JSON string.
func (id *FlexibleID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.value)
}

// UnmarshalJSON parses a FlexibleID from either a JSON string or number.
func (id *FlexibleID) UnmarshalJSON(data []byte) error {
	value, err := parseFlexibleIDJSON(data)
	if err != nil {
		return err
	}

	id.value = value

	return nil
}

// parseFlexibleIDJSON parses null, string, and numeric JSON identifier payloads.
func parseFlexibleIDJSON(data []byte) (string, error) {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return "", nil
	}

	if len(data) == 0 {
		return "", errEmptyID
	}

	if data[0] == '"' {
		var value string

		if err := json.Unmarshal(data, &value); err != nil {
			return "", err
		}

		return value, nil
	}

	var value json.Number

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&value); err != nil {
		return "", err
	}

	return value.String(), nil
}
