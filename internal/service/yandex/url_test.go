package yandex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseURL_SupportedShapes verifies supported Yandex Music URL shapes are parsed correctly.
func TestParseURL_SupportedShapes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawURL   string
		wantKind sourceKind
	}{
		{
			name:     "track url",
			rawURL:   "https://music.yandex.ru/album/10/track/11",
			wantKind: sourceKindTrack,
		},
		{
			name:     "track url with trailing slash",
			rawURL:   "https://music.yandex.ru/album/10/track/11/",
			wantKind: sourceKindTrack,
		},
		{
			name:     "album url",
			rawURL:   "https://music.yandex.ru/album/10",
			wantKind: sourceKindAlbum,
		},
		{
			name:     "album url with trailing slash",
			rawURL:   "https://music.yandex.ru/album/10/",
			wantKind: sourceKindAlbum,
		},
		{
			name:     "legacy playlist url",
			rawURL:   "https://music.yandex.ru/users/test-user/playlists/42",
			wantKind: sourceKindLegacyPlaylist,
		},
		{
			name:     "legacy playlist url with trailing slash",
			rawURL:   "https://music.yandex.ru/users/test-user/playlists/42/",
			wantKind: sourceKindLegacyPlaylist,
		},
		{
			name:     "uuid playlist url",
			rawURL:   "https://music.yandex.ru/playlists/123e4567-e89b-12d3-a456-426614174000",
			wantKind: sourceKindPlaylistUUID,
		},
		{
			name:     "uuid playlist url with trailing slash",
			rawURL:   "https://music.yandex.ru/playlists/123e4567-e89b-12d3-a456-426614174000/",
			wantKind: sourceKindPlaylistUUID,
		},
		{
			name:     "url with query",
			rawURL:   "https://music.yandex.ru/album/10/track/11?utm_source=test",
			wantKind: sourceKindTrack,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ref, err := parseURL(testCase.rawURL)
			require.NoError(t, err)
			require.NotNil(t, ref)
			assert.Equal(t, testCase.wantKind, ref.kind)
		})
	}
}

// TestParseURL_InvalidURL verifies unsupported URLs return an error.
func TestParseURL_InvalidURL(t *testing.T) {
	t.Parallel()

	ref, err := parseURL("https://music.yandex.ru/radio")
	require.Error(t, err)
	assert.Nil(t, ref)
}
