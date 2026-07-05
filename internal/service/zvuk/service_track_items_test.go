package zvuk

import (
	"context"
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

	impl.downloadTrackItems(context.Background(), []*DownloadItem{
		{
			Category: DownloadCategoryTrack,
			URL:      "https://zvuk.com/track/" + trackID,
			ItemID:   trackID,
		},
	})

	require.Len(t, impl.stats.Errors, 1)
	assert.Equal(t, DownloadCategoryTrack, impl.stats.Errors[0].Category)
	assert.Equal(t, trackID, impl.stats.Errors[0].ItemID)
	assert.Equal(t, "fetching track metadata", impl.stats.Errors[0].Phase)
	assert.ErrorIs(t, impl.stats.Errors[0].Error, errTrackMetadataFetch)
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

	impl.downloadTrackItems(context.Background(), []*DownloadItem{
		{
			Category: DownloadCategoryTrack,
			URL:      "https://zvuk.com/track/" + trackIDString,
			ItemID:   trackIDString,
		},
	})

	assert.Equal(t, int64(1), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(1), impl.stats.TracksSkipped)
	assert.Equal(t, int64(1), impl.stats.TracksSkippedExists)
	assert.Empty(t, impl.stats.Errors)
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

	impl.finalizeCollectionAssets(context.Background(), &downloadTracksMetadata{
		category: DownloadCategoryTrack,
	})

	assert.NoFileExists(t, embeddableCoverPath, "Embeddable cover should be renamed in track mode")
	assert.FileExists(t, finalCoverPath, "Final cover should be created in track mode")

	content, err := os.ReadFile(finalCoverPath)
	require.NoError(t, err)
	assert.Equal(t, "fake image data", string(content))
}
