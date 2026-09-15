//nolint:nestif,err113 // URL parser intentionally keeps pattern declarations and validation branches in one place.
package yandex

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"uuid"
)

// sourceKind identifies the Yandex Music URL shape parsed from user input.
type sourceKind uint8

// sourceRef is a normalized reference extracted from a Yandex Music URL.
type sourceRef struct {
	// kind is the parsed URL category.
	kind sourceKind
	// URL is the original trimmed user input.
	URL string
	// TrackID is the track identifier for single-track URLs.
	TrackID string
	// AlbumID is the album identifier for album or track URLs.
	AlbumID string
	// PlaylistID is the numeric playlist identifier for legacy playlist URLs.
	PlaylistID string
	// PlaylistUUID is the UUID playlist identifier for modern playlist URLs.
	PlaylistUUID string
	// Username is the playlist owner login for legacy playlist URLs.
	Username string
}

const (
	// sourceKindTrack means the URL points to a single track.
	sourceKindTrack sourceKind = iota + 1
	// sourceKindAlbum means the URL points to a full album.
	sourceKindAlbum
	// sourceKindLegacyPlaylist means the URL uses the /users/.../playlists/... shape.
	sourceKindLegacyPlaylist
	// sourceKindPlaylistUUID means the URL uses the /playlists/{uuid} shape.
	sourceKindPlaylistUUID
)

var (
	// yandexMusicHostPattern matches Yandex Music hostnames across regional domains.
	yandexMusicHostPattern = `music\.yandex\.[^/]+`
	// trackPattern matches album track URLs with album and track IDs.
	trackPattern = regexp.MustCompile(
		`^(?:https?://)?` + yandexMusicHostPattern + `/album/(?P<albumId>\d+)/track/(?P<trackId>\d+)/?(?:\?.*)?$`,
	)
	// albumPattern matches album URLs with an album ID.
	albumPattern = regexp.MustCompile(
		`^(?:https?://)?` + yandexMusicHostPattern + `/album/(?P<albumId>\d+)/?(?:\?.*)?$`,
	)
	// playlistPattern matches legacy user playlist URLs.
	playlistPattern = regexp.MustCompile(
		`^(?:https?://)?` + yandexMusicHostPattern +
			`/users/(?P<username>[^/]+)/playlists/(?P<playlistId>\d+)/?(?:\?.*)?$`,
	)
	// playlistUUIDPattern matches modern playlist URLs identified by UUID.
	playlistUUIDPattern = regexp.MustCompile(
		`^(?:https?://)?` + yandexMusicHostPattern +
			`/playlists/(?P<playlistUuid>(?:[a-z]{2}\.)?[0-9a-fA-F-]{36})/?(?:\?.*)?$`,
	)
)

// parseURL parses a Yandex Music URL into a normalized source reference.
func parseURL(raw string) (*sourceRef, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return nil, errors.New("empty Yandex Music URL")
	}

	if matches := trackPattern.FindStringSubmatch(input); matches != nil {
		return &sourceRef{kind: sourceKindTrack, URL: input, AlbumID: matches[1], TrackID: matches[2]}, nil
	}

	if matches := albumPattern.FindStringSubmatch(input); matches != nil {
		return &sourceRef{kind: sourceKindAlbum, URL: input, AlbumID: matches[1]}, nil
	}

	if matches := playlistPattern.FindStringSubmatch(input); matches != nil {
		return &sourceRef{kind: sourceKindLegacyPlaylist, URL: input, Username: matches[1], PlaylistID: matches[2]}, nil
	}

	if matches := playlistUUIDPattern.FindStringSubmatch(input); matches != nil {
		playlistID := matches[1]

		uuidPart := playlistID
		if prefix, rest, found := strings.Cut(playlistID, "."); found {
			if len(prefix) != 2 {
				return nil, fmt.Errorf("invalid Yandex playlist UUID prefix: %s", prefix)
			}

			uuidPart = rest
		}

		if _, err := uuid.Parse(uuidPart); err != nil {
			return nil, fmt.Errorf("invalid Yandex playlist UUID: %w", err)
		}

		return &sourceRef{kind: sourceKindPlaylistUUID, URL: input, PlaylistUUID: playlistID}, nil
	}

	return nil, fmt.Errorf("unsupported Yandex Music URL: %s", input)
}
