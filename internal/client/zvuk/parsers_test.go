package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
