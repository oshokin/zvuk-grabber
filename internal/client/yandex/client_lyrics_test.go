package yandex

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignLyricsRequest verifies the lyrics request signature for a known fixture.
func TestSignLyricsRequest(t *testing.T) {
	t.Parallel()

	signature := signLyricsRequest("565378", 1668329814)
	assert.Equal(t, "tNv3p9w458vwJtXEDPbEV9JtBlm7Mo3N8/Y1iKTU5Mw=", signature)
}

// TestSignLyricsRequest_NormalizesTrackIDWithAlbumSuffix verifies album suffixes are stripped before signing.
func TestSignLyricsRequest_NormalizesTrackIDWithAlbumSuffix(t *testing.T) {
	t.Parallel()

	signatureWithSuffix := signLyricsRequest("565378:42886", 1668329814)
	signatureWithoutSuffix := signLyricsRequest("565378", 1668329814)
	assert.Equal(t, signatureWithoutSuffix, signatureWithSuffix)
}

// TestBuildTrackLyricsURL_ContainsSignedQueryParams verifies signed lyrics URL query parameters.
func TestBuildTrackLyricsURL_ContainsSignedQueryParams(t *testing.T) {
	t.Parallel()

	timestamp := int64(1783159639)
	rawURL := buildTrackLyricsURL("19326613", lyricsFormatLRC, timestamp)

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	assert.Equal(t, "/tracks/19326613/lyrics", parsed.Path)

	want := url.Values{
		"format":    []string{lyricsFormatLRC},
		"timeStamp": []string{"1783159639"},
		"sign":      []string{signLyricsRequest("19326613", timestamp)},
	}
	assert.Equal(t, want, parsed.Query())
}

// TestBuildTrackLyricsURL_NormalizesTrackIDInPath verifies album suffixes are stripped from the URL path.
func TestBuildTrackLyricsURL_NormalizesTrackIDInPath(t *testing.T) {
	t.Parallel()

	rawURL := buildTrackLyricsURL("19326613:2176030", lyricsFormatLRC, 1783159639)
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	assert.Equal(t, "/tracks/19326613/lyrics", parsed.Path)
}
