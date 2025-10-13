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

// TestFinalizeAudiobookDescription_SingleChapter tests that description files
// are correctly renamed after download completes for single-chapter audiobooks.
func TestFinalizeAudiobookDescription_SingleChapter(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	service, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create a temp description file with UUID.
	tempDescPath := filepath.Join(setup.tempDir, "description_test-uuid-12345.txt")
	err := os.WriteFile(tempDescPath, []byte("Test description"), 0o644)
	require.NoError(t, err)

	// Create audio collection with temp path.
	audioCollection := &audioCollection{
		category:            DownloadCategoryAudiobook,
		title:               "Test Audiobook",
		tracksPath:          setup.tempDir,
		descriptionTempPath: tempDescPath,
		tracksCount:         1,
	}

	chapterFilename := "Author - Book Title.mp3"

	ctx := context.Background()

	// Finalize description (should rename to match chapter).
	service.finalizeAudiobookDescription(ctx, 1, audioCollection, chapterFilename)

	// Verify temp file was renamed.
	assert.NoFileExists(t, tempDescPath, "Temp description file should be renamed")

	// Verify final file exists with correct name.
	finalDescPath := filepath.Join(setup.tempDir, "Author - Book Title.txt")
	assert.FileExists(t, finalDescPath, "Final description file should exist")

	// Verify content is preserved.
	content, err := os.ReadFile(finalDescPath)
	require.NoError(t, err)
	assert.Equal(t, "Test description", string(content), "Description content should be preserved")
}

// TestFinalizeAudiobookDescription_MultiChapter tests that description files
// ARE renamed for multi-chapter audiobooks (to standard description.txt name).
func TestFinalizeAudiobookDescription_MultiChapter(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	service, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create a temp description file with UUID.
	tempDescPath := filepath.Join(setup.tempDir, "description_test-uuid-67890.txt")
	err := os.WriteFile(tempDescPath, []byte("Test description"), 0o644)
	require.NoError(t, err)

	// Create audio collection with temp path (3 chapters).
	audioCollection := &audioCollection{
		category:            DownloadCategoryAudiobook,
		title:               "Test Audiobook",
		tracksPath:          setup.tempDir,
		descriptionTempPath: tempDescPath,
		tracksCount:         3,
	}

	chapterFilename := "Chapter 03.mp3"

	ctx := context.Background()

	// Finalize description on last chapter (should rename to description.txt).
	service.finalizeAudiobookDescription(ctx, 3, audioCollection, chapterFilename)

	// Verify temp file was renamed.
	assert.NoFileExists(t, tempDescPath, "Temp description file should be renamed")

	// Verify final file exists with standard name.
	finalDescPath := filepath.Join(setup.tempDir, "description.txt")
	assert.FileExists(t, finalDescPath, "Final description file should exist as description.txt")

	// Verify content is preserved.
	content, err := os.ReadFile(finalDescPath)
	require.NoError(t, err)
	assert.Equal(t, "Test description", string(content), "Description content should be preserved")
}

// TestFinalizeAlbumCoverArt_SingleChapterAudiobook tests that cover files
// are correctly renamed after download completes for single-chapter audiobooks.
func TestFinalizeAlbumCoverArt_SingleChapterAudiobook(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	service, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create a temp cover file with UUID.
	tempCoverPath := filepath.Join(setup.tempDir, "cover_test-uuid-67890.jpg")
	err := os.WriteFile(tempCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	// Create audio collection with temp path.
	audioCollection := &audioCollection{
		category:      DownloadCategoryAudiobook,
		title:         "Test Audiobook",
		tracksPath:    setup.tempDir,
		coverPath:     tempCoverPath,
		coverTempPath: tempCoverPath,
		tracksCount:   1,
	}

	chapterFilename := "Author - Book Title.mp3"

	ctx := context.Background()

	// Finalize cover (should rename to match chapter).
	service.finalizeAlbumCoverArt(ctx, 1, audioCollection, chapterFilename)

	// Verify temp file was renamed.
	assert.NoFileExists(t, tempCoverPath, "Temp cover file should be renamed")

	// Verify final file exists with correct name.
	finalCoverPath := filepath.Join(setup.tempDir, "Author - Book Title.jpg")
	assert.FileExists(t, finalCoverPath, "Final cover file should exist")

	// Verify content is preserved.
	content, err := os.ReadFile(finalCoverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content), "Cover content should be preserved")
}

// TestFinalizeAlbumCoverArt_WithFolderForSingles tests that cover files
// are renamed to "cover.jpg" when CreateFolderForSingles is enabled.
func TestFinalizeAlbumCoverArt_WithFolderForSingles(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = true
	})
	defer setup.cleanup()

	service, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create a temp cover file with UUID.
	tempCoverPath := filepath.Join(setup.tempDir, "cover_test-uuid-11111.jpg")
	err := os.WriteFile(tempCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	// Create audio collection with temp path.
	audioCollection := &audioCollection{
		category:      DownloadCategoryAudiobook,
		title:         "Test Audiobook",
		tracksPath:    setup.tempDir,
		coverPath:     tempCoverPath,
		coverTempPath: tempCoverPath,
		tracksCount:   1,
	}

	chapterFilename := "01 - Chapter Title.mp3"

	ctx := context.Background()

	// Finalize cover (should rename to "cover.jpg").
	service.finalizeAlbumCoverArt(ctx, 1, audioCollection, chapterFilename)

	// Verify temp file was renamed.
	assert.NoFileExists(t, tempCoverPath, "Temp cover file should be renamed")

	// Verify final file exists with standard name.
	finalCoverPath := filepath.Join(setup.tempDir, "cover.jpg")
	assert.FileExists(t, finalCoverPath, "Final cover file should exist as cover.jpg")

	// Verify content is preserved.
	content, err := os.ReadFile(finalCoverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content), "Cover content should be preserved")
}

// TestUUIDBasedNaming_UniquePaths tests that UUID-based naming produces unique file paths.
func TestUUIDBasedNaming_UniquePaths(t *testing.T) {
	paths := make(map[string]bool)

	// Generate 100 UUID-based filenames and verify they're all unique.
	for range 100 {
		setup := newTestDownloadSetup(t)
		service, ok := setup.service.(*ServiceImpl)
		require.True(t, ok, "Service should be of type *ServiceImpl")

		// Simulate saving a description (uses UUID internally).
		tempDescPath := service.saveAudiobookDescription(
			context.Background(),
			"Test description",
			setup.tempDir,
		)

		require.NotEmpty(t, tempDescPath, "Description path should not be empty")
		require.False(t, paths[tempDescPath], "Path %s should be unique", tempDescPath)

		paths[tempDescPath] = true

		setup.cleanup()
	}

	assert.Len(t, paths, 100, "All 100 generated paths should be unique")
}
