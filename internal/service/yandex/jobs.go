package yandex

import (
	"context"
	"errors"
	"fmt"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
)

// errUnsupportedYandexURLKind is returned when a parsed URL kind has no job resolver.
var errUnsupportedYandexURLKind = errors.New("unsupported Yandex Music URL kind")

// resolveJobs parses a URL and expands it into one or more track download jobs.
func (s *ServiceImpl) resolveJobs(ctx context.Context, rawURL string) ([]*trackJob, error) {
	ref, err := parseURL(rawURL)
	if err != nil {
		return nil, err
	}

	switch ref.kind {
	case sourceKindTrack:
		track, fetchErr := retryValue(ctx, s, "fetching Yandex track metadata", func() (*model.Track, error) {
			return s.client.TrackInfo(ctx, ref.TrackID)
		})
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch track metadata: %w", fetchErr)
		}

		album := trackAlbumByID(track, ref.AlbumID)
		if album == nil {
			album = firstAlbum(track)
		}

		collectionKind := collectionTrack

		albumKind := collectionKindFromAlbum(album)
		if albumKind == collectionAudiobook || albumKind == collectionPodcast {
			collectionKind = albumKind
		}

		return []*trackJob{{
			kind:            collectionKind,
			sourceURL:       ref.URL,
			collectionID:    albumID(album),
			collectionTitle: albumTitle(album, track.FullTitle()),
			track:           track,
			album:           album,
			trackNumber:     trackIndexFromAlbum(album, 1),
			trackCount:      trackCountFromAlbum(album, 1),
		}}, nil

	case sourceKindAlbum:
		album, fetchErr := retryValue(ctx, s, "fetching Yandex album metadata", func() (*model.Album, error) {
			return s.client.AlbumWithTracks(ctx, ref.AlbumID)
		})
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch album metadata: %w", fetchErr)
		}

		return albumJobs(ref.URL, album), nil

	case sourceKindLegacyPlaylist:
		playlist, fetchErr := retryValue(ctx, s, "fetching Yandex playlist metadata", func() (*model.Playlist, error) {
			return s.client.UsersPlaylist(ctx, ref.PlaylistID, ref.Username)
		})
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch playlist metadata: %w", fetchErr)
		}

		return playlistJobs(ref.URL, playlist), nil

	case sourceKindPlaylistUUID:
		playlist, fetchErr := retryValue(
			ctx,
			s,
			"fetching Yandex UUID playlist metadata",
			func() (*model.Playlist, error) {
				return s.client.PlaylistByUUID(ctx, ref.PlaylistUUID)
			},
		)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch playlist metadata: %w", fetchErr)
		}

		return playlistJobs(ref.URL, playlist), nil

	default:
		return nil, errUnsupportedYandexURLKind
	}
}

// trackAlbumByID returns the album from track metadata matching the parsed album ID.
func trackAlbumByID(track *model.Track, albumID string) *model.Album {
	if track == nil || len(track.Albums) == 0 || albumID == "" {
		return nil
	}

	for i := range track.Albums {
		album := track.Albums[i]
		if album.ID == nil {
			continue
		}

		if album.ID.String() == albumID {
			return &album
		}
	}

	return nil
}

// albumJobs expands album metadata into per-track download jobs.
func albumJobs(sourceURL string, album *model.Album) []*trackJob {
	if album == nil {
		return nil
	}

	jobs := make([]*trackJob, 0, album.TrackCount)
	collectionKind := collectionKindFromAlbum(album)
	trackNumber := 1

	trackCount := countAlbumTracks(album)
	for _, volume := range album.Volumes {
		for i := range volume {
			track := volume[i]

			trackCopy := track
			if len(trackCopy.Albums) == 0 {
				trackCopy.Albums = []model.Album{*album}
			}

			jobs = append(jobs, &trackJob{
				kind:            collectionKind,
				sourceURL:       sourceURL,
				collectionID:    albumID(album),
				collectionTitle: album.Title,
				track:           &trackCopy,
				album:           album,
				trackNumber:     trackNumber,
				trackCount:      trackCount,
			})
			trackNumber++
		}
	}

	return jobs
}

// playlistJobs expands playlist metadata into per-track download jobs.
func playlistJobs(sourceURL string, playlist *model.Playlist) []*trackJob {
	if playlist == nil {
		return nil
	}

	jobs := make([]*trackJob, 0, len(playlist.Tracks))

	trackCount := len(playlist.Tracks)
	for i := range playlist.Tracks {
		track := playlist.Tracks[i].Track
		trackCopy := track
		jobs = append(jobs, &trackJob{
			kind:            collectionPlaylist,
			sourceURL:       sourceURL,
			collectionID:    playlistID(playlist),
			collectionTitle: playlist.Title,
			track:           &trackCopy,
			album:           firstAlbum(&trackCopy),
			playlist:        playlist,
			trackNumber:     i + 1,
			trackCount:      trackCount,
		})
	}

	return jobs
}
