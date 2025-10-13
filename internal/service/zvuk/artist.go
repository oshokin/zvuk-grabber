package zvuk

import (
	"context"
	"slices"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// weightedAverageNumberOfAlbumsPerArtist is an estimated average number of albums per artist.
const weightedAverageNumberOfAlbumsPerArtist = 46

// fetchArtistAlbums fetches all albums for a given list of artists.
func (s *ServiceImpl) fetchArtistAlbums(ctx context.Context, artistItems []*DownloadItem) []*DownloadItem {
	var (
		artistsCount = len(artistItems)
		result       = make([]*DownloadItem, 0, artistsCount*weightedAverageNumberOfAlbumsPerArtist)
	)

	// Iterate over each artist and fetch their albums.
	for itemIndex, v := range artistItems {
		// Check if context was canceled (CTRL+C pressed) - stop immediately.
		select {
		case <-ctx.Done():
			return result
		default:
		}

		logger.Infof(ctx, "Fetching releases for artist with ID %s (%d out of %d)", v.ItemID, itemIndex+1, artistsCount)

		// Get the list of album IDs for the current artist.
		albumIDs, err := s.getArtistReleaseIDs(ctx, v.ItemID)
		if err != nil {
			logger.Error(ctx, "Failed to fetch artist releases: %v", err)
			s.recordError(&ErrorContext{
				Category:  DownloadCategoryArtist,
				ItemID:    v.ItemID,
				ItemTitle: "Artist ID: " + v.ItemID,
				ItemURL:   v.URL,
				Phase:     "fetching artist releases",
			}, err)

			continue
		}

		// Skip if no albums are found for the artist.
		if len(albumIDs) == 0 {
			logger.Info(ctx, "No albums found for this artist")

			continue
		}

		// Generate download-ready items for each album.
		for _, albumID := range albumIDs {
			var albumURL string

			albumURL, err = s.zvukClient.GetAlbumURL(albumID)
			if err != nil {
				logger.Error(ctx, "Failed to generate URL for album with ID %s: %v", albumID, err)

				continue
			}

			result = append(result, &DownloadItem{
				Category: DownloadCategoryAlbum,
				URL:      albumURL,
				ItemID:   albumID,
			})
		}
	}

	return result
}

// getArtistReleaseIDs fetches all release IDs for a given artist.
func (s *ServiceImpl) getArtistReleaseIDs(ctx context.Context, artistURL string) ([]string, error) {
	var (
		limit       = 50
		allAlbumIDs []string
		offset      int
	)

	// Fetch albums in batches until no more are returned.
	for {
		// Check if context was canceled (CTRL+C pressed) - stop immediately.
		select {
		case <-ctx.Done():
			return allAlbumIDs, ctx.Err()
		default:
		}

		albumIDs, err := s.zvukClient.GetArtistReleaseIDs(ctx, artistURL, offset, limit)
		if err != nil {
			return nil, err
		}

		// Stop if the response is empty (no more albums to fetch).
		if len(albumIDs) == 0 {
			break
		}

		// Append the fetched album IDs to the result slice.
		allAlbumIDs = append(allAlbumIDs, albumIDs...)
		offset += limit // Move to the next batch
	}

	// Remove duplicate album IDs and return the result.
	return slices.Compact(allAlbumIDs), nil
}
