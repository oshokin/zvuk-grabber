package zvuk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
	mock_media "github.com/oshokin/zvuk-grabber/internal/media/mocks"
)

// TestWriteTrackMetadata_UsesEmbeddableCoverPath verifies embeddable cover path is preferred over final cover.
func TestWriteTrackMetadata_UsesEmbeddableCoverPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	embeddableCoverPath := filepath.Join(tmpDir, "cover_test-uuid.jpg")
	err := os.WriteFile(embeddableCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	finalCoverPath := filepath.Join(tmpDir, "cover.jpg")

	tempTrackPath := filepath.Join(tmpDir, "track.mp3.part")
	err = os.WriteFile(tempTrackPath, []byte("fake audio data"), 0o644)
	require.NoError(t, err)

	finalTrackPath := filepath.Join(tmpDir, "track.mp3")

	ctrl := gomock.NewController(t)
	tagProcessor := mock_media.NewMockTagProcessor(ctrl)

	var lastRequest *media.WriteTagsRequest

	tagProcessor.EXPECT().
		WriteTags(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *media.WriteTagsRequest) error {
			lastRequest = req
			return nil
		})

	impl := &ServiceImpl{
		cfg: &config.Config{
			OutputPath: tmpDir,
			DryRun:     false,
		},
		tagProcessor: tagProcessor,
	}

	task := &downloadTrackTask{
		trackIDString: "1",
		quality:       TrackQualityMP3Mid,
		trackPath:     finalTrackPath,
		audioCollection: &audioCollection{
			embeddableCoverPath: embeddableCoverPath,
			coverPath:           finalCoverPath,
		},
		metadata: &downloadTracksMetadata{
			category: DownloadCategoryAlbum,
		},
	}

	impl.writeTrackMetadata(t.Context(), task, map[string]string{}, nil, tempTrackPath)

	require.NotNil(t, lastRequest)
	assert.Equal(t, embeddableCoverPath, lastRequest.CoverPath)
	assert.FileExists(t, finalTrackPath)
	assert.NoFileExists(t, tempTrackPath)
}

// TestWriteTrackMetadata_FallsBackToFinalCoverPath verifies fallback to final cover when embeddable is missing.
func TestWriteTrackMetadata_FallsBackToFinalCoverPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	embeddableCoverPath := filepath.Join(tmpDir, "cover_test-uuid.jpg") // doesn't exist

	finalCoverPath := filepath.Join(tmpDir, "cover.jpg")
	err := os.WriteFile(finalCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	tempTrackPath := filepath.Join(tmpDir, "track.mp3.part")
	err = os.WriteFile(tempTrackPath, []byte("fake audio data"), 0o644)
	require.NoError(t, err)

	finalTrackPath := filepath.Join(tmpDir, "track.mp3")

	ctrl := gomock.NewController(t)
	tagProcessor := mock_media.NewMockTagProcessor(ctrl)

	var lastRequest *media.WriteTagsRequest

	tagProcessor.EXPECT().
		WriteTags(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *media.WriteTagsRequest) error {
			lastRequest = req
			return nil
		})

	impl := &ServiceImpl{
		cfg: &config.Config{
			OutputPath: tmpDir,
			DryRun:     false,
		},
		tagProcessor: tagProcessor,
	}

	task := &downloadTrackTask{
		trackIDString: "1",
		quality:       TrackQualityMP3Mid,
		trackPath:     finalTrackPath,
		audioCollection: &audioCollection{
			embeddableCoverPath: embeddableCoverPath,
			coverPath:           finalCoverPath,
		},
		metadata: &downloadTracksMetadata{
			category: DownloadCategoryAlbum,
		},
	}

	impl.writeTrackMetadata(t.Context(), task, map[string]string{}, nil, tempTrackPath)

	require.NotNil(t, lastRequest)
	assert.Equal(t, finalCoverPath, lastRequest.CoverPath)
	assert.FileExists(t, finalTrackPath)
	assert.NoFileExists(t, tempTrackPath)
}
