package yandex

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
	mock_media "github.com/oshokin/zvuk-grabber/internal/media/mocks"
)

// TestAlbumJobs_UsesAudiobookKind verifies audiobook albums produce audiobook collection jobs.
func TestAlbumJobs_UsesAudiobookKind(t *testing.T) {
	t.Parallel()

	album := &model.Album{
		ID:         model.NewFlexibleID("29276970"),
		Type:       "audiobook",
		Title:      "Тютчев Фёдор Иванович Стихотворения",
		TrackCount: 2,
		Volumes: [][]model.Track{
			{
				{
					ID:    model.NewFlexibleID("121642017"),
					Title: "Умом Россию не понять",
				},
			},
			{
				{
					ID:    model.NewFlexibleID("121642018"),
					Title: "Чародейкою Зимою",
				},
			},
		},
	}

	jobs := albumJobs("https://music.yandex.ru/album/29276970", album)
	require.Len(t, jobs, 2)
	assert.Equal(t, collectionAudiobook, jobs[0].kind)
	assert.Equal(t, collectionAudiobook, jobs[1].kind)
}

// TestBuildTargetPath_AudiobookUsesAudiobookTemplates verifies audiobook jobs use audiobook path templates.
func TestBuildTargetPath_AudiobookUsesAudiobookTemplates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	templateManager := mock_media.NewMockTemplateManager(ctrl)
	templateManager.EXPECT().GetAudiobookFolderName(gomock.Any(), gomock.Any()).Return("audiobook-folder")
	templateManager.EXPECT().
		GetAudiobookChapterFilename(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("chapter-name")

	service := &ServiceImpl{
		cfg: &config.Config{
			OutputPath:      "downloads",
			GroupByProvider: true,
		},
		templateManager: templateManager,
	}

	path := service.buildTargetPath(t.Context(), &trackJob{
		kind:       collectionAudiobook,
		trackCount: 18,
	}, map[string]string{}, media.QualityMP3Mid)

	assert.Equal(
		t,
		filepath.Join("downloads", "yandex", "audiobook-folder", "chapter-name.mp3"),
		path,
	)
}

// TestBuildTargetPath_AlbumSingleWithoutFolder verifies one-track albums stay flat when single folders are disabled.
func TestBuildTargetPath_AlbumSingleWithoutFolder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	templateManager := mock_media.NewMockTemplateManager(ctrl)
	templateManager.EXPECT().
		GetTrackFilename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("single-track-name")

	service := &ServiceImpl{
		cfg: &config.Config{
			OutputPath:             "downloads",
			GroupByProvider:        true,
			CreateFolderForSingles: false,
		},
		templateManager: templateManager,
	}

	path := service.buildTargetPath(t.Context(), &trackJob{
		kind:       collectionAlbum,
		trackCount: 1,
	}, map[string]string{}, media.QualityMP3Mid)

	assert.Equal(
		t,
		filepath.Join("downloads", "yandex", "single-track-name.mp3"),
		path,
	)
}

// TestBuildTargetPath_AudiobookSingleWithoutFolder verifies one-track audiobooks stay flat when single folders are disabled.
func TestBuildTargetPath_AudiobookSingleWithoutFolder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	templateManager := mock_media.NewMockTemplateManager(ctrl)
	templateManager.EXPECT().
		GetAudiobookChapterFilename(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("single-chapter-name")

	service := &ServiceImpl{
		cfg: &config.Config{
			OutputPath:             "downloads",
			GroupByProvider:        true,
			CreateFolderForSingles: false,
		},
		templateManager: templateManager,
	}

	path := service.buildTargetPath(t.Context(), &trackJob{
		kind:       collectionAudiobook,
		trackCount: 1,
	}, map[string]string{}, media.QualityMP3Mid)

	assert.Equal(
		t,
		filepath.Join("downloads", "yandex", "single-chapter-name.mp3"),
		path,
	)
}

// TestAlbumJobs_UsesPodcastKind verifies podcast albums produce podcast collection jobs.
func TestAlbumJobs_UsesPodcastKind(t *testing.T) {
	t.Parallel()

	album := &model.Album{
		ID:         model.NewFlexibleID("24956069"),
		Type:       "podcast",
		Title:      "MINAEV LIVE",
		TrackCount: 2,
		Volumes: [][]model.Track{
			{
				{
					ID:    model.NewFlexibleID("152515635"),
					Title: "Episode 1",
				},
			},
			{
				{
					ID:    model.NewFlexibleID("152396762"),
					Title: "Episode 2",
				},
			},
		},
	}

	jobs := albumJobs("https://music.yandex.ru/album/24956069", album)
	require.Len(t, jobs, 2)
	assert.Equal(t, collectionPodcast, jobs[0].kind)
	assert.Equal(t, collectionPodcast, jobs[1].kind)
}

// TestCollectionKindFromAlbum_MetaTypePodcast verifies podcast meta type maps to podcast collection kind.
func TestCollectionKindFromAlbum_MetaTypePodcast(t *testing.T) {
	t.Parallel()

	assert.Equal(t, collectionPodcast, collectionKindFromAlbum(&model.Album{MetaType: "podcast"}))
}

// TestBuildTargetPath_PodcastUsesPodcastTemplates verifies podcast jobs use podcast path templates.
func TestBuildTargetPath_PodcastUsesPodcastTemplates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	templateManager := mock_media.NewMockTemplateManager(ctrl)
	templateManager.EXPECT().GetPodcastFolderName(gomock.Any(), gomock.Any()).Return("podcast-folder")
	templateManager.EXPECT().
		GetPodcastEpisodeFilename(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("episode-name")

	service := &ServiceImpl{
		cfg: &config.Config{
			OutputPath:      "downloads",
			GroupByProvider: true,
		},
		templateManager: templateManager,
	}

	path := service.buildTargetPath(t.Context(), &trackJob{
		kind:       collectionPodcast,
		trackCount: 323,
	}, map[string]string{}, media.QualityMP3Mid)

	assert.Equal(
		t,
		filepath.Join("downloads", "yandex", "podcast-folder", "episode-name.mp3"),
		path,
	)
}

// TestBuildTargetPath_PodcastSingleWithoutFolder verifies one-track podcasts stay flat when single folders are disabled.
func TestBuildTargetPath_PodcastSingleWithoutFolder(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	templateManager := mock_media.NewMockTemplateManager(ctrl)
	templateManager.EXPECT().
		GetPodcastEpisodeFilename(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("single-episode-name")

	service := &ServiceImpl{
		cfg: &config.Config{
			OutputPath:             "downloads",
			GroupByProvider:        true,
			CreateFolderForSingles: false,
		},
		templateManager: templateManager,
	}

	path := service.buildTargetPath(t.Context(), &trackJob{
		kind:       collectionPodcast,
		trackCount: 1,
	}, map[string]string{}, media.QualityMP3Mid)

	assert.Equal(
		t,
		filepath.Join("downloads", "yandex", "single-episode-name.mp3"),
		path,
	)
}

// TestQualityLogFormatting_UsesActualMP3Bitrate verifies quality logs report the resolved MP3 bitrate.
func TestQualityLogFormatting_UsesActualMP3Bitrate(t *testing.T) {
	t.Parallel()

	payload := &audioPayload{
		quality: media.QualityMP3Mid,
		bitrate: 192,
		codec:   media.CodecMP3.String(),
	}

	assert.Equal(t, "MP3, 192 Kbps", qualityDescriptionForLog(payload))
	assert.Equal(t, "mp3_192", qualityTokenForLog(payload))
}

// TestBuildTags_PodcastFallbacksForMissingMetadata verifies podcast tag fallbacks when metadata is incomplete.
func TestBuildTags_PodcastFallbacksForMissingMetadata(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{}
	job := &trackJob{
		kind:            collectionPodcast,
		collectionTitle: "MINAEV LIVE",
		trackNumber:     1,
		trackCount:      323,
		track: &model.Track{
			ID:         model.NewFlexibleID("152515635"),
			Title:      "Episode title",
			DurationMs: 120000,
		},
		album: &model.Album{
			ID:    model.NewFlexibleID("24956069"),
			Title: "MINAEV LIVE",
		},
	}

	tags := service.buildTags(job)
	assert.Equal(t, "MINAEV LIVE", tags[media.TagPodcastAuthors])
	assert.Equal(t, unknownReleaseYear, tags[media.TagEpisodePublicationDate])
}
