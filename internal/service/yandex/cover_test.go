package yandex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

// noopTagProcessor is a TagProcessor test double that skips tag writes.
type noopTagProcessor struct{}

// WriteTags is a no-op tag write implementation for tests.
func (*noopTagProcessor) WriteTags(_ context.Context, _ *media.WriteTagsRequest) error {
	return nil
}

// TestWriteAudioFile_KeepExistingCoverWhenReplaceDisabled verifies existing covers are kept when replacement is disabled.
func TestWriteAudioFile_KeepExistingCoverWhenReplaceDisabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "track.mp3")
	existingCoverPath := filepath.Join(tempDir, "track"+coverExtension)

	err := os.WriteFile(existingCoverPath, []byte("cover-data"), 0o600)
	require.NoError(t, err)

	service := &ServiceImpl{
		cfg: &config.Config{
			ReplaceTracks:          false,
			ReplaceCovers:          false,
			CreateFolderForSingles: false,
		},
		tagProcessor: new(noopTagProcessor),
		sessionStats: stats.NewSession(time.Time{}, false),
	}

	payload := &audioPayload{
		quality: media.QualityMP3High,
		data:    []byte("audio"),
	}

	job := &trackJob{
		kind: collectionTrack,
		track: &model.Track{
			CoverURI: "avatars.yandex.net/get-music-content/cover%%",
		},
	}

	written, err := service.writeAudioFile(context.Background(), targetPath, payload, job, map[string]string{}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload.data)), written)

	_, err = os.Stat(existingCoverPath)
	require.NoError(t, err)
}

// TestBuildCoverPath_UsesCollectionCoverForAlbumAndTrackCoverForFlatSingles verifies cover path selection by collection kind.
func TestBuildCoverPath_UsesCollectionCoverForAlbumAndTrackCoverForFlatSingles(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			CreateFolderForSingles: false,
		},
	}

	albumCoverPath := service.buildCoverPath(
		filepath.Join("/tmp", "album", "01 - Track.mp3"),
		&trackJob{kind: collectionAlbum},
	)
	assert.Equal(t, filepath.Join("/tmp", "album", defaultCoverName), albumCoverPath)

	singleCoverPath := service.buildCoverPath(
		filepath.Join("/tmp", "Singles", "Artist - Song.mp3"),
		&trackJob{kind: collectionTrack},
	)
	assert.Equal(t, filepath.Join("/tmp", "Singles", "Artist - Song"+coverExtension), singleCoverPath)

	podcastCoverPath := service.buildCoverPath(
		filepath.Join("/tmp", "Podcast", "Episode 01.mp3"),
		&trackJob{kind: collectionPodcast},
	)
	assert.Equal(t, filepath.Join("/tmp", "Podcast", "Episode 01"+coverExtension), podcastCoverPath)
}

// TestBuildCoverPath_UsesTrackCoverForSingleAlbumWithoutFolder verifies one-track albums use sidecar cover files.
func TestBuildCoverPath_UsesTrackCoverForSingleAlbumWithoutFolder(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			CreateFolderForSingles: false,
		},
	}

	coverPath := service.buildCoverPath(
		filepath.Join("/tmp", "Singles", "Artist - Single.mp3"),
		&trackJob{
			kind:       collectionAlbum,
			trackCount: 1,
		},
	)
	assert.Equal(t, filepath.Join("/tmp", "Singles", "Artist - Single"+coverExtension), coverPath)
}

// TestCoverURL_UsesMaximumSizePlaceholder verifies cover URLs use the maximum size placeholder.
func TestCoverURL_UsesMaximumSizePlaceholder(t *testing.T) {
	t.Parallel()

	job := &trackJob{
		kind: collectionTrack,
		track: &model.Track{
			CoverURI: "avatars.yandex.net/get-music-content/13449652/8b0fbc15.a.33676016-1/%%",
		},
	}

	assert.Equal(
		t,
		"https://avatars.yandex.net/get-music-content/13449652/8b0fbc15.a.33676016-1/m1000x1000",
		coverURL(job),
	)
}
