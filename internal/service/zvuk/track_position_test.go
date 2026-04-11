package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

func TestResolveTrackPosition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		task     *downloadTrackTask
		expected int64
	}{
		{
			name: "standalone track uses album position",
			task: &downloadTrackTask{
				trackIndex: 1,
				track: &zvuk.Track{
					Position: 8,
				},
				metadata: &downloadTracksMetadata{
					category: DownloadCategoryTrack,
				},
			},
			expected: 8,
		},
		{
			name: "album track uses track position",
			task: &downloadTrackTask{
				trackIndex: 2,
				track: &zvuk.Track{
					Position: 5,
				},
				metadata: &downloadTracksMetadata{
					category: DownloadCategoryAlbum,
				},
			},
			expected: 5,
		},
		{
			name: "standalone track falls back to index when position missing",
			task: &downloadTrackTask{
				trackIndex: 3,
				track: &zvuk.Track{
					Position: 0,
				},
				metadata: &downloadTracksMetadata{
					category: DownloadCategoryTrack,
				},
			},
			expected: 3,
		},
		{
			name: "playlist keeps sequential index",
			task: &downloadTrackTask{
				trackIndex: 4,
				track: &zvuk.Track{
					Position: 9,
				},
				metadata: &downloadTracksMetadata{
					category: DownloadCategoryPlaylist,
				},
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, resolveTrackPosition(tt.task))
		})
	}
}
