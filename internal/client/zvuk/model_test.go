package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetAlbumsMetadataResponse tests the GetAlbumsMetadataResponse structure.
func TestGetAlbumsMetadataResponse(t *testing.T) {
	t.Parallel()

	track := &Track{ID: 1, Title: "Test Track"}
	release := &Release{ID: 1, Title: "Test Album"}

	response := GetAlbumsMetadataResponse{
		Tracks:   map[string]*Track{"track1": track},
		Releases: map[string]*Release{"release1": release},
	}

	assert.NotNil(t, response.Tracks)
	assert.NotNil(t, response.Releases)
	assert.Contains(t, response.Tracks, "track1")
	assert.Contains(t, response.Releases, "release1")
	assert.Equal(t, "Test Track", response.Tracks["track1"].Title)
	assert.Equal(t, "Test Album", response.Releases["release1"].Title)
}

// TestGetPlaylistsMetadataResponse tests the GetPlaylistsMetadataResponse structure.
func TestGetPlaylistsMetadataResponse(t *testing.T) {
	t.Parallel()

	track := &Track{ID: 1, Title: "Test Track"}
	playlist := &Playlist{ID: 1, Title: "Test Playlist"}

	response := GetPlaylistsMetadataResponse{
		Tracks:    map[string]*Track{"track1": track},
		Playlists: map[string]*Playlist{"playlist1": playlist},
	}

	assert.NotNil(t, response.Tracks)
	assert.NotNil(t, response.Playlists)
	assert.Contains(t, response.Tracks, "track1")
	assert.Contains(t, response.Playlists, "playlist1")
	assert.Equal(t, "Test Track", response.Tracks["track1"].Title)
	assert.Equal(t, "Test Playlist", response.Playlists["playlist1"].Title)
}

// TestGetLyricsResponse tests the GetLyricsResponse structure.
func TestGetLyricsResponse(t *testing.T) {
	t.Parallel()

	lyrics := &Lyrics{
		Type:   LyricsTypeSubtitle,
		Lyrics: "Test lyrics content",
	}

	response := GetLyricsResponse{
		Result: lyrics,
	}

	assert.NotNil(t, response.Result)
	assert.Equal(t, LyricsTypeSubtitle, response.Result.Type)
	assert.Equal(t, "Test lyrics content", response.Result.Lyrics)
}

// TestGetMetadataResponse tests the GetMetadataResponse structure.
func TestGetMetadataResponse(t *testing.T) {
	t.Parallel()

	track := &Track{ID: 1, Title: "Test Track"}

	response := GetMetadataResponse{
		Result: &Metadata{
			Tracks: map[string]*Track{"track1": track},
		},
	}

	assert.NotNil(t, response.Result)
	assert.NotNil(t, response.Result.Tracks)
	assert.Contains(t, response.Result.Tracks, "track1")
	assert.Equal(t, "Test Track", response.Result.Tracks["track1"].Title)
}

// TestGetStreamMetadataResponse tests the GetStreamMetadataResponse structure.
func TestGetStreamMetadataResponse(t *testing.T) {
	t.Parallel()

	stream := &StreamMetadata{
		Stream: "https://example.com/stream.mp3",
	}

	response := GetStreamMetadataResponse{
		Result: stream,
	}

	assert.NotNil(t, response.Result)
	assert.Equal(t, "https://example.com/stream.mp3", response.Result.Stream)
}

// TestGetUserProfileResponse tests the GetUserProfileResponse structure.
func TestGetUserProfileResponse(t *testing.T) {
	t.Parallel()

	subscription := &UserSubscription{
		Title:      "Premium",
		Expiration: 1234567890,
	}

	profile := &UserProfile{
		Subscription: subscription,
	}

	response := GetUserProfileResponse{
		Result: profile,
	}

	assert.NotNil(t, response.Result)
	assert.NotNil(t, response.Result.Subscription)
	assert.Equal(t, "Premium", response.Result.Subscription.Title)
	assert.Equal(t, int64(1234567890), response.Result.Subscription.Expiration)
}

// TestImage tests the Image structure.
func TestImage(t *testing.T) {
	t.Parallel()

	image := &Image{
		SourceURL: "https://example.com/image.jpg",
	}

	assert.Equal(t, "https://example.com/image.jpg", image.SourceURL)
}

// TestLabel tests the Label structure.
func TestLabel(t *testing.T) {
	t.Parallel()

	label := &Label{
		Title: "Test Label",
	}

	assert.Equal(t, "Test Label", label.Title)
}

// TestLyrics tests the Lyrics structure.
func TestLyrics(t *testing.T) {
	t.Parallel()

	lyrics := &Lyrics{
		Type:   LyricsTypeSubtitle,
		Lyrics: "Test lyrics content",
	}

	assert.Equal(t, LyricsTypeSubtitle, lyrics.Type)
	assert.Equal(t, "Test lyrics content", lyrics.Lyrics)
}

// TestMetadata tests the Metadata structure.
func TestMetadata(t *testing.T) {
	t.Parallel()

	track := &Track{ID: 1, Title: "Test Track"}

	metadata := &Metadata{
		Tracks: map[string]*Track{"track1": track},
	}

	assert.NotNil(t, metadata.Tracks)
	assert.Contains(t, metadata.Tracks, "track1")
	assert.Equal(t, "Test Track", metadata.Tracks["track1"].Title)
}

// TestPlaylist tests the Playlist structure.
func TestPlaylist(t *testing.T) {
	t.Parallel()

	playlist := &Playlist{
		ID:    1,
		Title: "Test Playlist",
	}

	assert.Equal(t, int64(1), playlist.ID)
	assert.Equal(t, "Test Playlist", playlist.Title)
}

// TestRelease tests the Release structure.
func TestRelease(t *testing.T) {
	t.Parallel()

	release := &Release{
		ID:    1,
		Title: "Test Release",
	}

	assert.Equal(t, int64(1), release.ID)
	assert.Equal(t, "Test Release", release.Title)
}

// TestStreamMetadata tests the StreamMetadata structure.
func TestStreamMetadata(t *testing.T) {
	t.Parallel()

	stream := &StreamMetadata{
		Stream: "https://example.com/stream.mp3",
	}

	assert.Equal(t, "https://example.com/stream.mp3", stream.Stream)
}

// TestTrack tests the Track structure.
func TestTrack(t *testing.T) {
	t.Parallel()

	track := &Track{
		ID:    1,
		Title: "Test Track",
	}

	assert.Equal(t, int64(1), track.ID)
	assert.Equal(t, "Test Track", track.Title)
}

// TestUserProfile tests the UserProfile structure.
func TestUserProfile(t *testing.T) {
	t.Parallel()

	subscription := &UserSubscription{
		Title:      "Premium",
		Expiration: 1234567890,
	}

	profile := &UserProfile{
		Subscription: subscription,
	}

	assert.NotNil(t, profile.Subscription)
	assert.Equal(t, "Premium", profile.Subscription.Title)
	assert.Equal(t, int64(1234567890), profile.Subscription.Expiration)
}

// TestUserSubscription tests the UserSubscription structure.
func TestUserSubscription(t *testing.T) {
	t.Parallel()

	subscription := &UserSubscription{
		Title:      "Premium",
		Expiration: 1234567890,
	}

	assert.Equal(t, "Premium", subscription.Title)
	assert.Equal(t, int64(1234567890), subscription.Expiration)
}

// TestConstants tests the constants.
func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "subtitle", LyricsTypeSubtitle)
	assert.Equal(t, "lrc", LyricsTypeLRC)
}

// TestEmptyStructures tests that empty structures can be created.
func TestEmptyStructures(t *testing.T) {
	t.Parallel()

	// Test empty GetAlbumsMetadataResponse.
	response := new(GetAlbumsMetadataResponse)
	assert.NotNil(t, response)
	assert.Nil(t, response.Tracks)
	assert.Nil(t, response.Releases)

	// Test empty GetLyricsResponse.
	lyricsResponse := new(GetLyricsResponse)
	assert.NotNil(t, lyricsResponse)
	assert.Nil(t, lyricsResponse.Result)

	// Test empty GetMetadataResponse.
	metadataResponse := new(GetMetadataResponse)
	assert.NotNil(t, metadataResponse)
	assert.Nil(t, metadataResponse.Result)

	// Test empty GetPlaylistsMetadataResponse.
	playlistsResponse := new(GetPlaylistsMetadataResponse)
	assert.NotNil(t, playlistsResponse)
	assert.Nil(t, playlistsResponse.Tracks)
	assert.Nil(t, playlistsResponse.Playlists)

	// Test empty GetStreamMetadataResponse.
	streamResponse := new(GetStreamMetadataResponse)
	assert.NotNil(t, streamResponse)
	assert.Nil(t, streamResponse.Result)

	// Test empty GetUserProfileResponse.
	profileResponse := new(GetUserProfileResponse)
	assert.NotNil(t, profileResponse)
	assert.Nil(t, profileResponse.Result)
}
