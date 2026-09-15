package yandex

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

func TestCollapseNameSeparators(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Author - Title", collapseNameSeparators("0000 - Author - Title"))
	assert.Equal(t, "Author - Title", collapseNameSeparators(" - Author - Title"))
	assert.Equal(t, "Совсем другое дело", collapseNameSeparators("Совсем другое дело - Совсем другое дело"))
	assert.Equal(t, "01 - Title", collapseNameSeparators("01 - Title"))
	assert.Equal(t, "Title", collapseNameSeparators("0000 - Title"))
}

func TestCollectionReleaseYear_UsesAlbumYear(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "2024", collectionReleaseYear(&model.Album{Year: 2024}, nil))
}

func TestCollectionReleaseYear_UsesTrackPubDateWhenAlbumYearMissing(t *testing.T) {
	t.Parallel()

	year := collectionReleaseYear(&model.Album{
		Volumes: [][]model.Track{
			{
				{PubDate: "2026-09-15"},
				{PubDate: "2026-08-25"},
			},
		},
	}, &model.Track{PubDate: "2026-09-15"})

	assert.Equal(t, "2026", year)
}

func TestBuildTags_UsesTrackPubDateForPodcastEpisode(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{}
	tags := service.buildTags(&trackJob{
		kind:            collectionPodcast,
		collectionTitle: "Совсем другое дело",
		trackNumber:     1,
		trackCount:      4,
		track: &model.Track{
			ID:      model.NewFlexibleID("154868318"),
			Title:   "Вишневая девятка. Как я сдаю в аренду машину из девяностых",
			PubDate: "2026-09-15",
		},
		album: &model.Album{
			ID:    model.NewFlexibleID("43686140"),
			Title: "Совсем другое дело",
			Type:  "podcast",
		},
	})

	assert.Equal(t, "2026-09-15", tags[media.TagEpisodePublicationDate])
	assert.Equal(t, "2026", tags[media.TagReleaseYear])
	assert.Empty(t, tags[media.TagPodcastAuthors])
}

func TestBuildTags_OmitsMissingAudiobookYear(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{}
	tags := service.buildTags(&trackJob{
		kind:            collectionAudiobook,
		collectionTitle: "Татьяна Столяр. «Я есть жир»",
		trackNumber:     1,
		trackCount:      18,
		track: &model.Track{
			ID:    model.NewFlexibleID("155603183"),
			Title: "Татьяна Столяр. «Я есть жир». Часть 1",
			Artists: []model.Artist{
				{Name: "Татьяна Столяр"},
			},
		},
		album: &model.Album{
			ID:    model.NewFlexibleID("43888536"),
			Title: "Татьяна Столяр. «Я есть жир»",
			Type:  "audiobook",
			Artists: []model.Artist{
				{Name: "Татьяна Столяр"},
			},
		},
	})

	assert.Empty(t, tags[media.TagPublishYear])
	assert.Empty(t, tags[media.TagReleaseYear])
}

func TestBuildTargetPath_OmitsMissingAudiobookYear(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		OutputPath:                       "downloads",
		GroupByProvider:                  true,
		AudiobookFolderTemplate:          config.DefaultAudiobookFolderTemplate,
		AudiobookChapterFilenameTemplate: config.DefaultAudiobookChapterFilenameTemplate,
		MaxFolderNameLength:              100,
		CreateFolderForSingles:           true,
	}
	service := &ServiceImpl{
		cfg:             cfg,
		templateManager: media.NewTemplateManager(t.Context(), cfg),
	}

	job := &trackJob{
		kind:            collectionAudiobook,
		collectionTitle: "Татьяна Столяр. «Я есть жир»",
		trackNumber:     1,
		trackCount:      18,
		track: &model.Track{
			ID:    model.NewFlexibleID("155603183"),
			Title: "Татьяна Столяр. «Я есть жир». Часть 1",
			Artists: []model.Artist{
				{Name: "Татьяна Столяр"},
			},
		},
		album: &model.Album{
			ID:    model.NewFlexibleID("43888536"),
			Title: "Татьяна Столяр. «Я есть жир»",
			Type:  "audiobook",
			Artists: []model.Artist{
				{Name: "Татьяна Столяр"},
			},
		},
	}

	path := service.buildTargetPath(t.Context(), job, service.buildTags(job), media.QualityMP3Mid)
	assert.Equal(
		t,
		filepath.Join(
			"downloads",
			"yandex",
			"Татьяна Столяр - Татьяна Столяр. «Я есть жир»",
			"01 - Татьяна Столяр. «Я есть жир». Часть 1.mp3",
		),
		path,
	)
	assert.NotContains(t, path, "0000")
}

func TestBuildTargetPath_UsesPodcastPubDateInFilename(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		OutputPath:                     "downloads",
		GroupByProvider:                true,
		PodcastFolderTemplate:          config.DefaultPodcastFolderTemplate,
		PodcastEpisodeFilenameTemplate: config.DefaultPodcastEpisodeFilenameTemplate,
		MaxFolderNameLength:            100,
		CreateFolderForSingles:         true,
	}
	service := &ServiceImpl{
		cfg:             cfg,
		templateManager: media.NewTemplateManager(t.Context(), cfg),
	}

	job := &trackJob{
		kind:            collectionPodcast,
		collectionTitle: "Совсем другое дело",
		trackNumber:     1,
		trackCount:      4,
		track: &model.Track{
			ID:      model.NewFlexibleID("154868318"),
			Title:   "Вишневая девятка. Как я сдаю в аренду машину из девяностых",
			PubDate: "2026-09-15",
		},
		album: &model.Album{
			ID:    model.NewFlexibleID("43686140"),
			Title: "Совсем другое дело",
			Type:  "podcast",
		},
	}

	path := service.buildTargetPath(t.Context(), job, service.buildTags(job), media.QualityMP3Mid)
	assert.Equal(
		t,
		filepath.Join(
			"downloads",
			"yandex",
			"Совсем другое дело",
			"2026-09-15 - Вишневая девятка. Как я сдаю в аренду машину из девяностых.mp3",
		),
		path,
	)
}
