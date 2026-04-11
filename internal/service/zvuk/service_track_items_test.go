package zvuk

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errTrackMetadataFetch = errors.New("unexpected HTTP status: 404")

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
