package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValueOrZero verifies type assertion and zero-value fallback behavior.
func TestValueOrZero(t *testing.T) {
	t.Parallel()

	assert.True(t, valueOrZero[bool](true))
	assert.False(t, valueOrZero[bool]("not a bool"))
	assert.InDelta(t, float64(42), valueOrZero[float64](float64(42)), 0)
	assert.Zero(t, valueOrZero[float64]("not a number"))
	assert.Nil(t, valueOrZero[map[string]any]("not a map"))
}

// TestParseGraphQLChildTracksReturnsEmptyResultForUnexpectedFieldType verifies empty results for invalid child lists.
func TestParseGraphQLChildTracksReturnsEmptyResultForUnexpectedFieldType(t *testing.T) {
	t.Parallel()

	tracks, trackIDs := parseGraphQLChildTracks(
		t.Context(),
		map[string]any{"chapters": "unexpected value"},
		"chapters",
		"chapter",
		&Audiobook{},
		func(map[string]any, *Audiobook) (*Track, error) {
			t.Fatal("parser must not be called for an unexpected child-list type")

			return nil, ErrUnexpectedTracksResponseFormat
		},
	)

	assert.Empty(t, tracks)
	assert.Empty(t, trackIDs)
}

// TestParsePlaylistTrackIDsSkipsNilAndInvalidEntries verifies website-style playlist numbering.
func TestParsePlaylistTrackIDsSkipsNilAndInvalidEntries(t *testing.T) {
	t.Parallel()

	trackIDs := parsePlaylistTrackIDs([]any{
		map[string]any{"id": "64870395"},
		nil,
		map[string]any{"id": ""},
		"not a track",
		map[string]any{"id": "143304072"},
	})

	assert.Equal(t, []int64{64870395, 143304072}, trackIDs)
}

// TestCollectPaginatedPlaylistTrackIDsUsesRawPageLength verifies pagination keeps going
// when a full page contains skipped (nil) entries.
func TestCollectPaginatedPlaylistTrackIDsUsesRawPageLength(t *testing.T) {
	t.Parallel()

	pages := map[int][]any{
		0: {
			map[string]any{"id": "1"},
			nil,
		},
		2: {
			map[string]any{"id": "2"},
		},
	}

	trackIDs, err := collectPaginatedPlaylistTrackIDs(2, 10, func(offset int) ([]any, error) {
		return pages[offset], nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, trackIDs)
}
