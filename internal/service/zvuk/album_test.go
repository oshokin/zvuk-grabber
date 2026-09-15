package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
)

// TestFetchAlbumData_FetchesTracksByIDs verifies album metadata is loaded from REST
// and tracks are fetched separately via GraphQL using release track_ids.
func TestFetchAlbumData_FetchesTracksByIDs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mock_zvuk_client.NewMockClient(ctrl)

	const (
		albumID = "52420521"
		trackID = "185766210"
		labelID = "3888855"
	)

	mockClient.EXPECT().
		GetAlbumsMetadata(gomock.Any(), []string{albumID}).
		Return(&zvuk.GetAlbumsMetadataResponse{
			Releases: map[string]*zvuk.Release{
				albumID: {
					ID:          52420521,
					Title:       "Лето",
					TrackIDs:    []int64{185766210},
					ArtistNames: []string{"Young P&H"},
					LabelID:     3888855,
					Date:        20260821,
				},
			},
		}, nil)

	mockClient.EXPECT().
		GetTracksMetadata(gomock.Any(), []string{trackID}).
		Return(map[string]*zvuk.Track{
			trackID: {
				ID:        185766210,
				Title:     "Лето",
				ReleaseID: 52420521,
			},
		}, nil)

	mockClient.EXPECT().
		GetLabelsMetadata(gomock.Any(), []string{labelID}).
		Return(map[string]*zvuk.Label{
			labelID: {Title: "KOALA MUSIC"},
		}, nil)

	service := &ServiceImpl{zvukClient: mockClient}

	result, err := service.fetchAlbumData(t.Context(), albumID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.tracks, trackID)
	assert.Equal(t, "Лето", result.tracks[trackID].Title)
	assert.Equal(t, int64(52420521), result.tracks[trackID].ReleaseID)
}
