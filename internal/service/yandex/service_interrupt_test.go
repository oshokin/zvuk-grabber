package yandex

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	mock_yandex "github.com/oshokin/zvuk-grabber/internal/service/yandex/mocks"
)

// errInterruptSignalReceived simulates an interrupted download flow in tests.
var errInterruptSignalReceived = errors.New("interrupt signal received")

// TestDownloadURLs_StopsBeforeResolveWhenInterrupted verifies an already-canceled context skips resolution.
func TestDownloadURLs_StopsBeforeResolveWhenInterrupted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)

	service := &ServiceImpl{
		cfg:    &config.Config{DryRun: true},
		client: client,
	}

	require.ErrorIs(t, service.DownloadURLs(ctx, []string{"https://music.yandex.ru/album/1"}), context.Canceled)

	snapshot := service.statsSnapshot()
	require.NotNil(t, snapshot)
	assert.Zero(t, snapshot.Tracks.Failed)
	assert.Empty(t, snapshot.Errors)
}

// TestDownloadURLs_StopsAndSkipsResolveErrorAfterInterrupt verifies later resolve errors are ignored after interrupt.
func TestDownloadURLs_StopsAndSkipsResolveErrorAfterInterrupt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().
		AlbumWithTracks(gomock.Any(), "1").
		DoAndReturn(func(context.Context, string) (*model.Album, error) {
			cancel()
			return nil, errInterruptSignalReceived
		})

	service := &ServiceImpl{
		cfg:    &config.Config{DryRun: true},
		client: client,
	}

	require.ErrorIs(t, service.DownloadURLs(ctx, []string{
		"https://music.yandex.ru/album/1",
		"https://music.yandex.ru/album/2",
	}), context.Canceled)

	snapshot := service.statsSnapshot()
	require.NotNil(t, snapshot)
	assert.Zero(t, snapshot.Tracks.Failed)
	assert.Empty(t, snapshot.Errors)
}
