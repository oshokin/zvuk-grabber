package zvuk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

type recordingTagProcessor struct {
	lastReq *WriteTagsRequest
}

func (r *recordingTagProcessor) WriteTags(_ context.Context, req *WriteTagsRequest) error {
	r.lastReq = req
	return nil
}

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

	rec := &recordingTagProcessor{}
	impl := &ServiceImpl{
		cfg: &config.Config{
			OutputPath: tmpDir,
			DryRun:     false,
		},
		tagProcessor: rec,
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

	impl.writeTrackMetadata(context.Background(), task, map[string]string{}, nil, tempTrackPath)

	require.NotNil(t, rec.lastReq)
	assert.Equal(t, embeddableCoverPath, rec.lastReq.CoverPath)
	assert.FileExists(t, finalTrackPath)
	assert.NoFileExists(t, tempTrackPath)
}

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

	rec := &recordingTagProcessor{}
	impl := &ServiceImpl{
		cfg: &config.Config{
			OutputPath: tmpDir,
			DryRun:     false,
		},
		tagProcessor: rec,
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

	impl.writeTrackMetadata(context.Background(), task, map[string]string{}, nil, tempTrackPath)

	require.NotNil(t, rec.lastReq)
	assert.Equal(t, finalCoverPath, rec.lastReq.CoverPath)
	assert.FileExists(t, finalTrackPath)
	assert.NoFileExists(t, tempTrackPath)
}
