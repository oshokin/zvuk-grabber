package zvuk

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	zvukclient "github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mocks "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media/lossless"
)

// TestDownloadAndSaveRemuxesBeforeTagging remuxes FLAC-in-MP4 before tagging, and leaves no file on remux failure.
func TestDownloadAndSaveRemuxesBeforeTagging(t *testing.T) {
	data, readErr := os.ReadFile("../../media/lossless/testdata/tone.mp4")
	require.NoError(t, readErr)

	for _, broken := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "truncated"}[broken], func(t *testing.T) {
			payload := data
			if broken {
				payload = data[:len(data)-1]
			}

			ctrl := gomock.NewController(t)
			client := mocks.NewMockClient(ctrl)
			client.EXPECT().
				FetchTrack(gomock.Any(), "https://example.test/track").
				Return(&zvukclient.FetchTrackResult{Body: io.NopCloser(bytes.NewReader(payload)), TotalBytes: int64(len(payload))}, nil)

			cfg := config.DefaultConfig()
			s := &ServiceImpl{cfg: cfg, zvukClient: client}
			target := filepath.Join(t.TempDir(), "track.flac")
			result, err := s.downloadAndSaveTrack(t.Context(), "https://example.test/track", target)
			entries, listErr := os.ReadDir(filepath.Dir(target))
			require.NoError(t, listErr)

			if broken {
				require.Error(t, err)
				require.Empty(t, entries)

				return
			}

			require.NoError(t, err)
			require.Len(t, entries, 1)

			native, err := os.ReadFile(result.TempPath)
			require.NoError(t, err)
			require.True(t, lossless.HasFLACMarker(native))
			require.Equal(t, int64(len(data)), result.BytesDownloaded)
			require.NoError(t, os.Rename(result.TempPath, target)) // Closed handles, including on Windows.
		})
	}
}

// TestNormalizeLeavesSourceUntouched keeps the download file intact when remux is canceled.
func TestNormalizeLeavesSourceUntouched(t *testing.T) {
	data, readErr := os.ReadFile("../../media/lossless/testdata/tone.mp4")
	require.NoError(t, readErr)
	f, err := os.CreateTemp(t.TempDir(), "source")
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write(data)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = normalizeDownloadedFLAC(ctx, f, "track.flac")
	require.ErrorIs(t, err, context.Canceled)
	original, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	require.Equal(t, data, original)

	entries, err := os.ReadDir(filepath.Dir(f.Name()))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	result, err := normalizeDownloadedFLAC(t.Context(), f, "track.mp3")
	require.NoError(t, err)
	require.Empty(t, result)
}
