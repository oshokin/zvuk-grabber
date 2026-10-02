//nolint:gocritic,err113 // Download-plan API intentionally returns concise tuple results and wraps provider-specific resolution errors inline.
package yandex

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/media/lossless"
)

// NewAuthorizedClient creates a Yandex Music client with an OAuth token.
func NewAuthorizedClient(token string) *Client {
	client := NewClient(nil)
	client.SetToken(strings.TrimSpace(token))

	return client
}

// ResolveMP3Link resolves a direct MP3 URL for a track.
// preferredQuality: 1 prefers 128-ish MP3; 2/3 prefer best available MP3.
func (c *Client) ResolveMP3Link(ctx context.Context, trackID string, preferredQuality uint8) (string, int, error) {
	trackID = normalizeTrackID(trackID)

	infos, err := c.TracksDownloadInfo(ctx, trackID)
	if err != nil {
		return "", 0, err
	}

	selected := pickBitrate(infos, preferredQuality)
	if selected == nil || selected.BitrateInKbps == 0 || strings.TrimSpace(selected.DownloadInfoURL) == "" {
		return "", 0, fmt.Errorf("no MP3 download options available for track %s", trackID)
	}

	link, err := c.TrackDownloadLink(ctx, selected.DownloadInfoURL)
	if err != nil {
		return "", 0, err
	}

	return link, selected.BitrateInKbps, nil
}

// OpenAudio opens a remote audio stream for direct file writing.
func (c *Client) OpenAudio(ctx context.Context, rawURL string) (*RemoteFile, error) {
	stream, err := c.httpClient.OpenStreamWithContext(
		c.requestContext(ctx, "audio_download", "open_audio_stream"),
		rawURL,
	)
	if err != nil {
		return nil, err
	}

	return &RemoteFile{
		Body:          stream.Body,
		ContentLength: stream.ContentLength,
		ContentType:   stream.ContentType,
	}, nil
}

// DownloadCoverBytes downloads cover bytes using the configured Yandex HTTP client.
func (c *Client) DownloadCoverBytes(ctx context.Context, rawURL string) ([]byte, error) {
	return c.httpClient.DownloadBytesWithContext(c.requestContext(ctx, "cover_download", "download_cover"), rawURL)
}

// DownloadFLACBytes tries to download a real FLAC stream for a track.
func (c *Client) DownloadFLACBytes(ctx context.Context, trackID string) ([]byte, error) {
	trackID = normalizeTrackID(trackID)

	userUID, err := c.ensureUserUID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify account status: %w", err)
	}

	info, err := c.losslessDownloader.GetDownloadInfo(
		c.requestContext(ctx, "lossless_file_info", "fetch_lossless_download_info"),
		trackID,
		userUID,
	)
	if err != nil {
		return nil, err
	}

	codec := media.ParseCodec(info.Codec)
	if !codec.IsFLACFamily() {
		return nil, fmt.Errorf("expected FLAC lossless, got %s", codec.Description())
	}

	data, err := c.losslessDownloader.DownloadAudio(
		c.requestContext(ctx, "lossless_audio", "download_lossless_audio"),
		info,
	)
	if err != nil {
		return nil, err
	}

	return lossless.Normalize(data)
}

// pickBitrate selects the best MP3 download option for the preferred quality tier.
func pickBitrate(infos []model.DownloadInfo, preferredQuality uint8) *model.DownloadInfo {
	filtered := make([]*model.DownloadInfo, 0, len(infos))
	for i := range infos {
		info := &infos[i]
		if !strings.EqualFold(strings.TrimSpace(info.Codec), CodecMP3) {
			continue
		}

		if strings.TrimSpace(info.DownloadInfoURL) == "" || info.BitrateInKbps <= 0 {
			continue
		}

		filtered = append(filtered, info)
	}

	if len(filtered) == 0 {
		return nil
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].BitrateInKbps < filtered[j].BitrateInKbps
	})

	// quality=1: nearest 128 Kbps from above; fallback to best available below 128.
	if preferredQuality <= 1 {
		for _, candidate := range filtered {
			if candidate.BitrateInKbps >= Bitrate128 {
				return candidate
			}
		}

		return filtered[len(filtered)-1]
	}

	// quality=2/3: best MP3 available (ideally 320 Kbps).
	return filtered[len(filtered)-1]
}
