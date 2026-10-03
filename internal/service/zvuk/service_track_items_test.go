package zvuk

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// errTrackMetadataFetch simulates a track metadata HTTP 404 error.
var errTrackMetadataFetch = errors.New("unexpected HTTP status: 404")

// TestDownloadTrackItems_RecordsMetadataFetchError verifies metadata fetch failures are recorded.
func TestDownloadTrackItems_RecordsMetadataFetchError(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t)
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	trackID := "114003338"

	setup.mockClient.EXPECT().
		GetTracksMetadata(gomock.Any(), []string{trackID}).
		Return(nil, errTrackMetadataFetch).
		Times(1)

	impl.downloadTrackItems(t.Context(), []*DownloadItem{
		{
			Category: DownloadCategoryTrack,
			URL:      "https://zvuk.com/track/" + trackID,
			ItemID:   trackID,
		},
	})

	require.Len(t, impl.stats.Errors, 1)

	want := &DownloadError{
		Category:       DownloadCategoryTrack,
		ItemID:         trackID,
		ItemTitle:      "Standalone track",
		ParentCategory: DownloadCategoryTrack,
		ParentID:       "standalone-tracks",
		ParentTitle:    "standalone track URLs",
		Phase:          "fetching track metadata",
		Error:          errTrackMetadataFetch,
	}
	assert.Equal(t, want, impl.stats.Errors[0])
}

// TestDownloadTrackItems_SkipsTracksCoveredByRegisteredCollections verifies duplicate tracks are skipped.
func TestDownloadTrackItems_SkipsTracksCoveredByRegisteredCollections(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t)
	defer setup.cleanup()

	impl, ok := setup.service.(*ServiceImpl)
	require.True(t, ok, "Service should be of type *ServiceImpl")

	const (
		trackIDInt    int64 = 114003338
		trackIDString       = "114003338"
	)

	impl.audioCollectionsMutex.Lock()
	impl.audioCollections[ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   "21788107",
	}] = &audioCollection{
		category:    DownloadCategoryAlbum,
		id:          "21788107",
		title:       "Covered Album",
		trackIDs:    []int64{trackIDInt},
		tracksCount: 1,
	}
	impl.audioCollectionsMutex.Unlock()

	impl.downloadTrackItems(t.Context(), []*DownloadItem{
		{
			Category: DownloadCategoryTrack,
			URL:      "https://zvuk.com/track/" + trackIDString,
			ItemID:   trackIDString,
		},
	})

	want := &DownloadStatistics{
		TotalTracksProcessed: 1,
		TracksSkipped:        1,
		TracksSkippedExists:  1,
	}
	assert.Equal(t, want, impl.stats)
}

// TestFinalizeCollectionAssets_TrackModeFinalizesRegisteredCollectionCover verifies
// standalone-track flow finalizes cover assets from registered collections.
func TestFinalizeCollectionAssets_TrackModeFinalizesRegisteredCollectionCover(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t)
	defer setup.cleanup()

	impl := setup.impl(t)

	embeddableCoverPath := filepath.Join(setup.tempDir, "cover_test-uuid.jpg")
	finalCoverPath := filepath.Join(setup.tempDir, "cover.jpg")

	err := os.WriteFile(embeddableCoverPath, []byte("fake image data"), 0o644)
	require.NoError(t, err)

	impl.audioCollectionsMutex.Lock()
	impl.audioCollections[ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   "39588100",
	}] = &audioCollection{
		category:            DownloadCategoryAlbum,
		id:                  "39588100",
		title:               "Woke",
		embeddableCoverPath: embeddableCoverPath,
		coverPath:           finalCoverPath,
		tracksCount:         1,
	}
	impl.audioCollectionsMutex.Unlock()

	impl.finalizeCollectionAssets(t.Context(), &downloadTracksMetadata{
		category: DownloadCategoryTrack,
	})

	assert.NoFileExists(t, embeddableCoverPath, "Embeddable cover should be renamed in track mode")
	assert.FileExists(t, finalCoverPath, "Final cover should be created in track mode")

	content, err := os.ReadFile(finalCoverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content))
}
