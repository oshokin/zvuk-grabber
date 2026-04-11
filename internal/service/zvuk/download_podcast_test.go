package zvuk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

func TestDownloadCollection_PodcastPassesEpisodeIDsToGetStreamQualities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	service := NewService(
		&config.Config{
			OutputPath:             t.TempDir(),
			MaxConcurrentDownloads: 1,
			ParsedLogLevel:         logger.Level(),
			ParsedMaxDownloadPause: 100 * time.Millisecond,
		},
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	podcastID := "29997388"
	episodeOneID := int64(11)
	episodeTwoID := int64(22)

	mockClient.EXPECT().
		GetPodcastsMetadata(gomock.Any(), []string{podcastID}).
		Return(&zvuk.GetPodcastsMetadataResponse{
			Podcasts: map[string]*zvuk.Podcast{
				podcastID: {
					ID:          29997388,
					Title:       "Test Podcast",
					ArtistNames: []string{"Test Author"},
					TrackIDs:    []int64{episodeOneID, episodeTwoID},
				},
			},
			Tracks: map[string]*zvuk.Track{
				"11": {ID: episodeOneID, Title: "Episode 11", Position: 2},
				"22": {ID: episodeTwoID, Title: "Episode 22", Position: 1},
			},
		}, nil).
		Times(1)

	mockClient.EXPECT().
		GetStreamQualities(gomock.Any(), []string{"22", "11"}).
		Return(map[string]*zvuk.StreamQualities{
			"22": {High: "https://example.com/22"},
			"11": {High: "https://example.com/11"},
		}, nil).
		Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stop before per-track downloads; this test only verifies stream IDs wiring

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.downloadCollection(ctx, &DownloadItem{
		Category: DownloadCategoryPodcast,
		ItemID:   podcastID,
		URL:      "https://zvuk.com/podcast/29997388",
	})
}
