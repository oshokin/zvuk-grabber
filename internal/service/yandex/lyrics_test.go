package yandex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
	mock_yandex "github.com/oshokin/zvuk-grabber/internal/service/yandex/mocks"
)

func TestDownloadAndSaveLyrics_SkipsWhenLyricsUnavailable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)

	service := &ServiceImpl{
		cfg: &config.Config{
			DownloadLyrics:        true,
			APIRetryAttemptsCount: 5,
		},
		client:       client,
		sessionStats: stats.NewSession(time.Time{}, false),
	}

	text := service.downloadAndSaveLyrics(t.Context(), "/tmp/track.flac", &trackJob{
		track: &model.Track{
			ID:              model.NewFlexibleID("18930259"),
			Title:           "Fuck You Tonight",
			LyricsAvailable: false,
		},
	})

	assert.Empty(t, text)
}

func TestIsYandexLyricsMissing_NoLyricsFoundForTrack(t *testing.T) {
	t.Parallel()

	apiErr := new(model.ErrorResponse)
	apiErr.APIError.Name = "No lyrics found for track"

	assert.True(t, isYandexLyricsMissing(apiErr))
}
