package yandex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	mock_yandex "github.com/oshokin/zvuk-grabber/internal/service/yandex/mocks"
)

// TestResolveJobs_TrackURLUsesAlbumIDContext verifies track URLs resolve against the album in the URL.
func TestResolveJobs_TrackURLUsesAlbumIDContext(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().TrackInfo(gomock.Any(), "11").Return(&model.Track{
		ID:    model.NewFlexibleID("11"),
		Title: "Track title",
		Albums: []model.Album{
			{
				ID:            model.NewFlexibleID("10"),
				Title:         "First album",
				TrackCount:    9,
				TrackPosition: model.TrackPosition{Index: 1},
			},
			{
				ID:            model.NewFlexibleID("20"),
				Title:         "Second album",
				TrackCount:    15,
				TrackPosition: model.TrackPosition{Index: 7},
			},
		},
	}, nil)

	service := &ServiceImpl{client: client}

	jobs, err := service.resolveJobs(t.Context(), "https://music.yandex.ru/album/20/track/11")
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	job := jobs[0]
	require.NotNil(t, job)
	require.NotNil(t, job.album)
	require.NotNil(t, job.album.ID)
	assert.Equal(t, "20", job.album.ID.String())
	assert.Equal(t, "20", job.collectionID)
	assert.Equal(t, "Second album", job.collectionTitle)
	assert.Equal(t, 7, job.trackNumber)
	assert.Equal(t, 15, job.trackCount)
}

// TestResolveJobs_TrackURLFallsBackToFirstAlbumWhenRequestedMissing verifies fallback when album ID is absent.
func TestResolveJobs_TrackURLFallsBackToFirstAlbumWhenRequestedMissing(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().TrackInfo(gomock.Any(), "11").Return(&model.Track{
		ID:    model.NewFlexibleID("11"),
		Title: "Track title",
		Albums: []model.Album{
			{
				ID:            model.NewFlexibleID("10"),
				Title:         "First album",
				TrackCount:    9,
				TrackPosition: model.TrackPosition{Index: 1},
			},
			{
				ID:            model.NewFlexibleID("20"),
				Title:         "Second album",
				TrackCount:    15,
				TrackPosition: model.TrackPosition{Index: 7},
			},
		},
	}, nil)

	service := &ServiceImpl{client: client}

	jobs, err := service.resolveJobs(t.Context(), "https://music.yandex.ru/album/99/track/11")
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	job := jobs[0]
	require.NotNil(t, job)
	require.NotNil(t, job.album)
	require.NotNil(t, job.album.ID)
	assert.Equal(t, "10", job.album.ID.String())
	assert.Equal(t, "10", job.collectionID)
	assert.Equal(t, "First album", job.collectionTitle)
	assert.Equal(t, 1, job.trackNumber)
	assert.Equal(t, 9, job.trackCount)
}
