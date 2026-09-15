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

// TestParseURL_PlaylistUUID verifies modern playlist UUID URLs keep the original identifier text.
func TestParseURL_PlaylistUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawURL   string
		wantUUID string
		wantErr  bool
	}{
		{
			name: "canonical uuid",
			rawURL: "https://music.yandex.ru/playlists/" +
				"123e4567-e89b-12d3-a456-426614174000",
			wantUUID: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name: "prefixed uuid",
			rawURL: "https://music.yandex.ru/playlists/" +
				"xx.123e4567-e89b-12d3-a456-426614174000",
			wantUUID: "xx.123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name: "invalid uuid",
			rawURL: "https://music.yandex.ru/playlists/" +
				"not-a-uuid",
			wantErr: true,
		},
		{
			name: "non-hex symbols",
			rawURL: "https://music.yandex.ru/playlists/" +
				"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
			wantErr: true,
		},
		{
			name: "wrong length",
			rawURL: "https://music.yandex.ru/playlists/" +
				"123e4567-e89b-12d3-a456-42661417400",
			wantErr: true,
		},
		{
			name: "invalid prefix",
			rawURL: "https://music.yandex.ru/playlists/" +
				"x.123e4567-e89b-12d3-a456-426614174000",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ref, err := parseURL(test.rawURL)

			if test.wantErr {
				require.Error(t, err)
				assert.Nil(t, ref)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, ref)
			assert.Equal(t, sourceKindPlaylistUUID, ref.kind)
			assert.Equal(t, test.wantUUID, ref.PlaylistUUID)
		})
	}
}
