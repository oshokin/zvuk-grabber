package zvuk

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// testDownloadSetup encapsulates common test dependencies and configuration.
type testDownloadSetup struct {
	ctrl       *gomock.Controller
	mockClient *mock_zvuk_client.MockClient
	service    Service
	config     *config.Config
	tempDir    string
}

// newTestDownloadSetup creates a standard test setup with optional config overrides.
func newTestDownloadSetup(t *testing.T, configOverrides ...func(*config.Config)) *testDownloadSetup {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	tempDir := t.TempDir()

	cfg := &config.Config{
		OutputPath:             tempDir,
		MaxConcurrentDownloads: 1,
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(),
		ParsedMaxDownloadPause: 100 * time.Millisecond,
	}

	// Apply overrides.
	for _, override := range configOverrides {
		override(cfg)
	}

	service := NewService(
		cfg,
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	return &testDownloadSetup{
		ctrl:       ctrl,
		mockClient: mockClient,
		service:    service,
		config:     cfg,
		tempDir:    tempDir,
	}
}

// cleanup releases test resources.
func (s *testDownloadSetup) cleanup() {
	s.ctrl.Finish()
}

// testMetadataBuilder helps build test metadata with sensible defaults.
type testMetadataBuilder struct {
	trackIDs       []int64
	albumID        int64
	tracksMetadata map[string]*zvuk.Track
	albumsMetadata map[string]*zvuk.Release
	albumsTags     map[string]map[string]string
	labelsMetadata map[string]*zvuk.Label
}

// newTestMetadata creates a metadata builder with auto-generated tracks and album.
func newTestMetadata(trackIDs []int64, albumID int64) *testMetadataBuilder {
	builder := &testMetadataBuilder{
		trackIDs:       trackIDs,
		albumID:        albumID,
		tracksMetadata: make(map[string]*zvuk.Track),
		albumsMetadata: make(map[string]*zvuk.Release),
		albumsTags:     make(map[string]map[string]string),
		labelsMetadata: map[string]*zvuk.Label{"999": {Title: "Test Label"}},
	}

	// Auto-generate tracks.
	for i, tid := range trackIDs {
		tidStr := strconv.FormatInt(tid, 10)
		builder.tracksMetadata[tidStr] = &zvuk.Track{
			ID:             tid,
			Title:          fmt.Sprintf("Track %d", i+1),
			ReleaseID:      albumID,
			Position:       int64(i + 1),
			HighestQuality: "flac",
			HasFLAC:        true,
		}
	}

	// Auto-generate album.
	albumIDStr := strconv.FormatInt(albumID, 10)
	builder.albumsMetadata[albumIDStr] = &zvuk.Release{
		ID:          albumID,
		Title:       "Test Album",
		ArtistNames: []string{"Test Artist"},
		Date:        20240101,
		LabelID:     999,
		TrackIDs:    trackIDs,
	}
	builder.albumsTags[albumIDStr] = map[string]string{"albumTitle": "Test Album"}

	return builder
}

// withTrackDuration sets the duration for a specific track.
func (b *testMetadataBuilder) withTrackDuration(trackID int64, duration int64) *testMetadataBuilder {
	tidStr := strconv.FormatInt(trackID, 10)
	if track, ok := b.tracksMetadata[tidStr]; ok {
		track.Duration = duration
	}

	return b
}

// withTrackQuality sets the quality for a specific track.
func (b *testMetadataBuilder) withTrackQuality(
	trackID int64,
	highestQuality string,
	hasFLAC bool,
) *testMetadataBuilder {
	tidStr := strconv.FormatInt(trackID, 10)
	if track, ok := b.tracksMetadata[tidStr]; ok {
		track.HighestQuality = highestQuality
		track.HasFLAC = hasFLAC
	}

	return b
}

// withAlbumTitle sets a custom album title.
func (b *testMetadataBuilder) withAlbumTitle(title string) *testMetadataBuilder {
	albumIDStr := strconv.FormatInt(b.albumID, 10)
	if album, ok := b.albumsMetadata[albumIDStr]; ok {
		album.Title = title
	}

	b.albumsTags[albumIDStr]["albumTitle"] = title

	return b
}

// build creates the final downloadTracksMetadata structure.
func (b *testMetadataBuilder) build() *downloadTracksMetadata {
	return &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        b.trackIDs,
		tracksMetadata:  b.tracksMetadata,
		albumsMetadata:  b.albumsMetadata,
		albumsTags:      b.albumsTags,
		labelsMetadata:  b.labelsMetadata,
		audioCollection: nil,
	}
}

// setupMockStreamMetadata configures mock expectations for GetStreamMetadata.
func setupMockStreamMetadata(
	mockClient *mock_zvuk_client.MockClient,
	trackIDString string,
	quality string,
	streamURL string,
) {
	streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}
	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), trackIDString, quality).
		Return(streamMetadata, nil)
}

// setupMockFetchTrack configures mock expectations for FetchTrack.
func setupMockFetchTrack(
	mockClient *mock_zvuk_client.MockClient,
	streamURL string,
	audioData []byte,
) {
	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(audioData)),
		TotalBytes: int64(len(audioData)),
	}
	mockClient.EXPECT().
		FetchTrack(gomock.Any(), streamURL).
		Return(fetchTrackResult, nil)
}

// makeFakeAudioData creates deterministic fake audio data for testing.
func makeFakeAudioData(sizeKB int) []byte {
	fakeData := make([]byte, sizeKB*1024)
	for i := range fakeData {
		fakeData[i] = byte(i % 256)
	}

	return fakeData
}

// findPartFiles finds all .part files in the given directory.
func findPartFiles(t *testing.T, dir string) []string {
	t.Helper()

	var partFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".part" {
			partFiles = append(partFiles, path)
		}

		return nil
	})

	require.NoError(t, err, "Failed to walk directory for .part files")

	return partFiles
}

// findAudioFiles finds all audio files (.mp3, .flac, .bin) in the given directory.
func findAudioFiles(t *testing.T, dir string) []string {
	t.Helper()

	var audioFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)
		if !info.IsDir() &&
			(ext == constants.ExtensionMP3 || ext == constants.ExtensionFLAC || ext == constants.ExtensionBin) {
			audioFiles = append(audioFiles, path)
		}

		return nil
	})

	require.NoError(t, err, "Failed to walk directory for audio files")

	return audioFiles
}

// findFileWithExtension finds the first file with the specified extension and returns its path.
// Also verifies the file content matches expectedContent if provided.
func findFileWithExtension(t *testing.T, dir, ext string, expectedContent []byte) (string, bool) {
	t.Helper()

	var (
		foundPath string
		found     bool
	)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ext {
			found = true
			foundPath = path

			// Verify content if provided.
			if expectedContent != nil {
				content, readErr := os.ReadFile(path)
				require.NoError(t, readErr, "Failed to read file: %s", path)
				assert.Len(t, content, len(expectedContent),
					"File size should match expected size (no truncation)")
				assert.Equal(t, expectedContent, content,
					"File content should match source data exactly")
			}
		}

		return nil
	})

	require.NoError(t, err, "Failed to walk directory")

	return foundPath, found
}
