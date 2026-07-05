package yandex

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
)

// TestPickBitrate_Quality1PrefersNearestAtOrAbove128 verifies quality 1 picks the nearest MP3 at or above 128 kbps.
func TestPickBitrate_Quality1PrefersNearestAtOrAbove128(t *testing.T) {
	t.Parallel()

	infos := []model.DownloadInfo{
		{Codec: CodecMP3, BitrateInKbps: Bitrate64, DownloadInfoURL: "u64"},
		{Codec: CodecMP3, BitrateInKbps: Bitrate192, DownloadInfoURL: "u192"},
		{Codec: CodecMP3, BitrateInKbps: Bitrate320, DownloadInfoURL: "u320"},
	}

	selected := pickBitrate(infos, 1)
	assert.Equal(t, Bitrate192, selected.BitrateInKbps)
}

// TestPickBitrate_Quality2PrefersHighestMP3 verifies quality 2 selects the highest available MP3 bitrate.
func TestPickBitrate_Quality2PrefersHighestMP3(t *testing.T) {
	t.Parallel()

	infos := []model.DownloadInfo{
		{Codec: CodecMP3, BitrateInKbps: Bitrate128, DownloadInfoURL: "u128"},
		{Codec: CodecMP3, BitrateInKbps: Bitrate192, DownloadInfoURL: "u192"},
		{Codec: CodecMP3, BitrateInKbps: Bitrate320, DownloadInfoURL: "u320"},
	}

	selected := pickBitrate(infos, 2)
	assert.Equal(t, Bitrate320, selected.BitrateInKbps)
}

// TestPickBitrate_Quality1FallsBackToBestBelow128 verifies quality 1 falls back to the best MP3 below 128 kbps.
func TestPickBitrate_Quality1FallsBackToBestBelow128(t *testing.T) {
	t.Parallel()

	infos := []model.DownloadInfo{
		{Codec: CodecMP3, BitrateInKbps: 96, DownloadInfoURL: "u96"},
		{Codec: CodecMP3, BitrateInKbps: Bitrate64, DownloadInfoURL: "u64"},
	}

	selected := pickBitrate(infos, 1)
	assert.Equal(t, 96, selected.BitrateInKbps)
}
