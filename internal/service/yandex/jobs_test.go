package yandex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	mock_yandex "github.com/oshokin/zvuk-grabber/internal/service/yandex/mocks"
)

// resolvedAlbumJob is the album-context fields checked after URL job resolution.
type resolvedAlbumJob struct {
	// albumID is the album identifier chosen for the track.
	albumID string
	// collectionID is the parent collection identifier on the job.
	collectionID string
	// collectionTitle is the parent collection title on the job.
	collectionTitle string
	// trackNumber is the one-based position in the collection.
	trackNumber int
	// trackCount is the collection size.
	trackCount int
}

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
	require.NotNil(t, jobs[0])
	require.NotNil(t, jobs[0].album)
	require.NotNil(t, jobs[0].album.ID)

	job := jobs[0]
	want := &resolvedAlbumJob{
		albumID:         "20",
		collectionID:    "20",
		collectionTitle: "Second album",
		trackNumber:     7,
		trackCount:      15,
	}
	got := &resolvedAlbumJob{
		albumID:         job.album.ID.String(),
		collectionID:    job.collectionID,
		collectionTitle: job.collectionTitle,
		trackNumber:     job.trackNumber,
		trackCount:      job.trackCount,
	}
	assert.Equal(t, want, got)
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
	require.NotNil(t, jobs[0])
	require.NotNil(t, jobs[0].album)
	require.NotNil(t, jobs[0].album.ID)

	job := jobs[0]
	want := &resolvedAlbumJob{
		albumID:         "10",
		collectionID:    "10",
		collectionTitle: "First album",
		trackNumber:     1,
		trackCount:      9,
	}
	got := &resolvedAlbumJob{
		albumID:         job.album.ID.String(),
		collectionID:    job.collectionID,
		collectionTitle: job.collectionTitle,
		trackNumber:     job.trackNumber,
		trackCount:      job.trackCount,
	}
	assert.Equal(t, want, got)
}
