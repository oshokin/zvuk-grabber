package yandex

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
)

// interruptTestClient stubs album lookups for interruption handling tests.
type interruptTestClient struct {
	// albumWithTracks is the callback used to simulate album resolution.
	albumWithTracks func(ctx context.Context, id string) (*model.Album, error)
	// albumWithTracksCalls counts how many album lookups were attempted.
	albumWithTracksCalls int
}

var (
	// errInterruptSignalReceived simulates an interrupted download flow in tests.
	errInterruptSignalReceived = errors.New("interrupt signal received")
	// errInterruptTestClientUnconfigured is returned when a stubbed client method is not configured.
	errInterruptTestClientUnconfigured = errors.New("interrupt test client is not configured")
)

// TrackInfo is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) TrackInfo(context.Context, string) (*model.Track, error) {
	return nil, errInterruptTestClientUnconfigured
}

// AlbumWithTracks delegates album resolution to the configured test callback.
func (c *interruptTestClient) AlbumWithTracks(ctx context.Context, id string) (*model.Album, error) {
	c.albumWithTracksCalls++
	if c.albumWithTracks == nil {
		return nil, errInterruptTestClientUnconfigured
	}

	return c.albumWithTracks(ctx, id)
}

// UsersPlaylist is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) UsersPlaylist(context.Context, string, string) (*model.Playlist, error) {
	return nil, errInterruptTestClientUnconfigured
}

// PlaylistByUUID is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) PlaylistByUUID(context.Context, string) (*model.Playlist, error) {
	return nil, errInterruptTestClientUnconfigured
}

// DownloadFLACBytes is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) DownloadFLACBytes(context.Context, string) ([]byte, error) {
	return nil, errInterruptTestClientUnconfigured
}

// ResolveMP3Link is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) ResolveMP3Link(context.Context, string, uint8) (string, int, error) {
	return "", 0, errInterruptTestClientUnconfigured
}

// OpenAudio is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) OpenAudio(context.Context, string) (*yandexclient.RemoteFile, error) {
	return nil, errInterruptTestClientUnconfigured
}

// DownloadCoverBytes is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) DownloadCoverBytes(context.Context, string) ([]byte, error) {
	return nil, errInterruptTestClientUnconfigured
}

// TrackLyrics is a stub implementation that reports the client as unconfigured.
func (*interruptTestClient) TrackLyrics(context.Context, string) (*model.TrackLyrics, error) {
	return nil, errInterruptTestClientUnconfigured
}

// TestDownloadURLs_StopsBeforeResolveWhenInterrupted verifies an already-canceled context skips resolution.
func TestDownloadURLs_StopsBeforeResolveWhenInterrupted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &interruptTestClient{
		albumWithTracks: func(context.Context, string) (*model.Album, error) {
			return nil, errInterruptSignalReceived
		},
	}
	service := &ServiceImpl{
		cfg:    &config.Config{DryRun: true},
		client: client,
	}

	service.DownloadURLs(ctx, []string{"https://music.yandex.ru/album/1"})

	snapshot := service.statsSnapshot()
	require.NotNil(t, snapshot)
	assert.Equal(t, 0, client.albumWithTracksCalls)
	assert.Zero(t, snapshot.Tracks.Failed)
	assert.Empty(t, snapshot.Errors)
}

// TestDownloadURLs_StopsAndSkipsResolveErrorAfterInterrupt verifies later resolve errors are ignored after interrupt.
func TestDownloadURLs_StopsAndSkipsResolveErrorAfterInterrupt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	client := &interruptTestClient{
		albumWithTracks: func(context.Context, string) (*model.Album, error) {
			cancel()
			return nil, errInterruptSignalReceived
		},
	}
	service := &ServiceImpl{
		cfg:    &config.Config{DryRun: true},
		client: client,
	}

	service.DownloadURLs(ctx, []string{
		"https://music.yandex.ru/album/1",
		"https://music.yandex.ru/album/2",
	})

	snapshot := service.statsSnapshot()
	require.NotNil(t, snapshot)
	assert.Equal(t, 1, client.albumWithTracksCalls)
	assert.Zero(t, snapshot.Tracks.Failed)
	assert.Empty(t, snapshot.Errors)
}
