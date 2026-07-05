package yandex

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
)

// jobsTestClient stubs TrackInfo lookups for job resolution tests.
type jobsTestClient struct {
	// trackInfo is the callback used to resolve track metadata.
	trackInfo func(ctx context.Context, id string) (*model.Track, error)
}

// errJobsTestClientUnconfigured is returned when a stubbed client method is not configured.
var errJobsTestClientUnconfigured = errors.New("jobs test client is not configured")

// TrackInfo resolves track metadata through the configured test callback.
func (c *jobsTestClient) TrackInfo(ctx context.Context, id string) (*model.Track, error) {
	if c.trackInfo == nil {
		return nil, errJobsTestClientUnconfigured
	}

	return c.trackInfo(ctx, id)
}

// AlbumWithTracks is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) AlbumWithTracks(context.Context, string) (*model.Album, error) {
	return nil, errJobsTestClientUnconfigured
}

// UsersPlaylist is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) UsersPlaylist(context.Context, string, string) (*model.Playlist, error) {
	return nil, errJobsTestClientUnconfigured
}

// PlaylistByUUID is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) PlaylistByUUID(context.Context, string) (*model.Playlist, error) {
	return nil, errJobsTestClientUnconfigured
}

// DownloadFLACBytes is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) DownloadFLACBytes(context.Context, string) ([]byte, error) {
	return nil, errJobsTestClientUnconfigured
}

// ResolveMP3Link is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) ResolveMP3Link(context.Context, string, uint8) (string, int, error) {
	return "", 0, errJobsTestClientUnconfigured
}

// OpenAudio is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) OpenAudio(context.Context, string) (*yandexclient.RemoteFile, error) {
	return nil, errJobsTestClientUnconfigured
}

// DownloadCoverBytes is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) DownloadCoverBytes(context.Context, string) ([]byte, error) {
	return nil, errJobsTestClientUnconfigured
}

// TrackLyrics is a stub implementation that reports the client as unconfigured.
func (*jobsTestClient) TrackLyrics(context.Context, string) (*model.TrackLyrics, error) {
	return nil, errJobsTestClientUnconfigured
}

// TestResolveJobs_TrackURLUsesAlbumIDContext verifies track URLs resolve against the album in the URL.
func TestResolveJobs_TrackURLUsesAlbumIDContext(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		client: &jobsTestClient{
			trackInfo: func(context.Context, string) (*model.Track, error) {
				return &model.Track{
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
				}, nil
			},
		},
	}

	jobs, err := service.resolveJobs(context.Background(), "https://music.yandex.ru/album/20/track/11")
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

	service := &ServiceImpl{
		client: &jobsTestClient{
			trackInfo: func(context.Context, string) (*model.Track, error) {
				return &model.Track{
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
				}, nil
			},
		},
	}

	jobs, err := service.resolveJobs(context.Background(), "https://music.yandex.ru/album/99/track/11")
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
