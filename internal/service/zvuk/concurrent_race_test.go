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

// TestFinalizeCover_SingleChapterAudiobook tests that cover files
// are correctly renamed after download completes for single-chapter audiobooks.
func TestFinalizeCover_SingleChapterAudiobook(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create an embeddable cover file with UUID.
	embeddableCoverPath := filepath.Join(setup.tempDir, "cover_test-uuid-67890.jpg")
	err := os.WriteFile(embeddableCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	// Create audio collection with cover paths.
	audioCollection := &audioCollection{
		category:            DownloadCategoryAudiobook,
		title:               "Test Audiobook",
		tracksPath:          setup.tempDir,
		embeddableCoverPath: embeddableCoverPath,
		coverPath:           filepath.Join(setup.tempDir, "Author - Book Title.jpg"),
		tracksCount:         1,
	}

	// Finalize cover (should rename embeddable cover to cover path).
	impl.finalizeCover(context.Background(), 1, audioCollection)

	// Verify temp file was renamed.
	assert.NoFileExists(t, embeddableCoverPath, "Embeddable cover file should be renamed")

	// Verify final file exists with correct name.
	finalCoverPath := filepath.Join(setup.tempDir, "Author - Book Title.jpg")
	assert.FileExists(t, finalCoverPath, "Final cover file should exist")

	// Verify content is preserved.
	content, err := os.ReadFile(finalCoverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content), "Cover content should be preserved")
}

// TestFinalizeCover_WithFolderForSingles tests that cover files
// are renamed to "cover.jpg" when CreateFolderForSingles is enabled.
func TestFinalizeCover_WithFolderForSingles(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = true
	})
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create an embeddable cover file with UUID.
	embeddableCoverPath := filepath.Join(setup.tempDir, "cover_test-uuid-11111.jpg")
	err := os.WriteFile(embeddableCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	// Create audio collection with cover paths.
	audioCollection := &audioCollection{
		category:            DownloadCategoryAudiobook,
		title:               "Test Audiobook",
		tracksPath:          setup.tempDir,
		embeddableCoverPath: embeddableCoverPath,
		coverPath:           filepath.Join(setup.tempDir, "cover.jpg"),
		tracksCount:         1,
	}

	// Finalize cover (should rename to "cover.jpg").
	impl.finalizeCover(context.Background(), 1, audioCollection)

	// Verify temp file was renamed.
	assert.NoFileExists(t, embeddableCoverPath, "Embeddable cover file should be renamed")

	// Verify final file exists with standard name.
	coverPath := filepath.Join(setup.tempDir, "cover.jpg")
	assert.FileExists(t, coverPath, "Cover file should exist as cover.jpg")

	// Verify content is preserved.
	content, err := os.ReadFile(coverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content), "Cover content should be preserved")
}

// TestFinalizeDescription_SingleChapter tests that description files
// are correctly renamed after download completes for single-chapter audiobooks.
func TestFinalizeDescription_SingleChapter(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create an embeddable description file with UUID.
	embeddableDescriptionPath := filepath.Join(setup.tempDir, "description_test-uuid-12345.txt")
	err := os.WriteFile(embeddableDescriptionPath, []byte("Test description"), 0o644)
	require.NoError(t, err)

	// Create audio collection with description paths.
	audioCollection := &audioCollection{
		category:                  DownloadCategoryAudiobook,
		title:                     "Test Audiobook",
		tracksPath:                setup.tempDir,
		embeddableDescriptionPath: embeddableDescriptionPath,
		descriptionPath:           filepath.Join(setup.tempDir, "Author - Book Title.txt"),
		tracksCount:               1,
	}

	// Finalize description (should rename to match chapter).
	impl.finalizeDescription(context.Background(), audioCollection, 1)

	// Verify temp file was renamed.
	assert.NoFileExists(t, embeddableDescriptionPath, "Embeddable description file should be renamed")

	// Verify final file exists with correct name.
	descriptionPath := filepath.Join(setup.tempDir, "Author - Book Title.txt")
	assert.FileExists(t, descriptionPath, "Description file should exist")

	// Verify content is preserved.
	content, err := os.ReadFile(descriptionPath)
	require.NoError(t, err)
	assert.Equal(t, "Test description", string(content), "Description content should be preserved")
}

// TestFinalizeDescription_MultiChapter tests that description files
// are renamed for multi-chapter audiobooks (to standard description.txt name).
func TestFinalizeDescription_MultiChapter(t *testing.T) {
	setup := newTestDownloadSetup(t, func(cfg *config.Config) {
		cfg.CreateFolderForSingles = false
	})
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	// Create an embeddable description file with UUID.
	embeddableDescriptionPath := filepath.Join(setup.tempDir, "description_test-uuid-67890.txt")
	err := os.WriteFile(embeddableDescriptionPath, []byte("Test description"), 0o644)
	require.NoError(t, err)

	// Create audio collection with description paths (3 chapters).
	audioCollection := &audioCollection{
		category:                  DownloadCategoryAudiobook,
		title:                     "Test Audiobook",
		tracksPath:                setup.tempDir,
		embeddableDescriptionPath: embeddableDescriptionPath,
		descriptionPath:           filepath.Join(setup.tempDir, "description.txt"),
		tracksCount:               3,
	}

	// Finalize description on last chapter (should rename to description.txt).
	impl.finalizeDescription(context.Background(), audioCollection, 3)

	// Verify temp file was renamed.
	assert.NoFileExists(t, embeddableDescriptionPath, "Embeddable description file should be renamed")

	// Verify final file exists with standard name.
	descriptionPath := filepath.Join(setup.tempDir, "description.txt")
	assert.FileExists(t, descriptionPath, "Description file should exist as description.txt")

	// Verify content is preserved.
	content, err := os.ReadFile(descriptionPath)
	require.NoError(t, err)
	assert.Equal(t, "Test description", string(content), "Description content should be preserved")
}

// TestUUIDBasedNaming_UniquePaths tests that UUID-based naming produces unique file paths.
func TestUUIDBasedNaming_UniquePaths(t *testing.T) {
	paths := make(map[string]bool)

	// Generate 100 UUID-based filenames and verify they're all unique.
	for range 100 {
		setup := newTestDownloadSetup(t)
		impl, ok := setup.service.(*ServiceImpl)
		require.True(t, ok, "Service should be of type *ServiceImpl")

		// Simulate saving a description (uses UUID internally).
		tempDescPath, _ := impl.saveDescription(
			context.Background(),
			DownloadCategoryAudiobook,
			setup.tempDir,
			"Test description",
			"",
		)

		require.NotEmpty(t, tempDescPath, "Description path should not be empty")
		require.False(t, paths[tempDescPath], "Path %s should be unique", tempDescPath)

		paths[tempDescPath] = true

		setup.cleanup()
	}

	assert.Len(t, paths, 100, "All 100 generated paths should be unique")
}
