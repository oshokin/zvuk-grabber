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

// recordingTagProcessor records the last WriteTags request for assertions.
type recordingTagProcessor struct {
	// lastRequest is the most recent WriteTags request passed to WriteTags.
	lastRequest *WriteTagsRequest
}

// WriteTags stores the request and returns success without writing tags.
func (r *recordingTagProcessor) WriteTags(_ context.Context, req *WriteTagsRequest) error {
	r.lastRequest = req
	return nil
}

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

	require.NotNil(t, rec.lastRequest)
	assert.Equal(t, embeddableCoverPath, rec.lastRequest.CoverPath)
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

	require.NotNil(t, rec.lastRequest)
	assert.Equal(t, finalCoverPath, rec.lastRequest.CoverPath)
	assert.FileExists(t, finalTrackPath)
	assert.NoFileExists(t, tempTrackPath)
}
