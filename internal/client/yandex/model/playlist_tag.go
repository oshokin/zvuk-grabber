package model

import (
	"bytes"
	"encoding/json"
	"errors"
)

// PlaylistTag represents a playlist classification tag.
type PlaylistTag struct {
	// ID is the tag identifier.
	ID string `json:"id"`
	// Value is the tag value.
	Value string `json:"value"`
	// Name is the human-readable tag name.
	Name string `json:"name"`
}

// playlistTagObject is a PlaylistTag alias for decoding object-form JSON without UnmarshalJSON recursion.
type playlistTagObject PlaylistTag

// errEmptyPlaylistTag is returned when a PlaylistTag JSON payload is empty.
var errEmptyPlaylistTag = errors.New("empty playlist tag")

// UnmarshalJSON accepts playlist tags as either plain strings or full objects.
func (tag *PlaylistTag) UnmarshalJSON(data []byte) error {
	object, err := unmarshalPlaylistTagObject(data)
	if err != nil {
		return err
	}

	tag.ID = object.ID
	tag.Value = object.Value
	tag.Name = object.Name

	return nil
}

// unmarshalPlaylistTagObject parses null, string, and object playlist tag JSON payloads.
func unmarshalPlaylistTagObject(data []byte) (playlistTagObject, error) {
	data = bytes.TrimSpace(data)

	switch {
	case bytes.Equal(data, []byte("null")):
		return playlistTagObject{}, nil
	case len(data) == 0:
		return playlistTagObject{}, errEmptyPlaylistTag
	case data[0] == '"':
		return unmarshalPlaylistTagString(data)
	default:
		return unmarshalPlaylistTagObjectForm(data)
	}
}

// unmarshalPlaylistTagString parses a playlist tag encoded as a plain JSON string.
func unmarshalPlaylistTagString(data []byte) (playlistTagObject, error) {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return playlistTagObject{}, err
	}

	return playlistTagObject{Value: value, Name: value}, nil
}

// unmarshalPlaylistTagObjectForm parses a playlist tag encoded as a JSON object.
func unmarshalPlaylistTagObjectForm(data []byte) (playlistTagObject, error) {
	var object playlistTagObject

	if err := json.Unmarshal(data, &object); err != nil {
		return playlistTagObject{}, err
	}

	normalizePlaylistTagObject(&object)

	return object, nil
}

// normalizePlaylistTagObject fills missing Value and Name from each other.
func normalizePlaylistTagObject(object *playlistTagObject) {
	if object.Value == "" {
		object.Value = object.Name
	}

	if object.Name == "" {
		object.Name = object.Value
	}
}
