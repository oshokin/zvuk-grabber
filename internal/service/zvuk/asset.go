package zvuk

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// saveDescription saves the description file for audiobooks and podcasts.
func (s *ServiceImpl) saveDescription(ctx context.Context, description, path, itemType string) string {
	// Check if final destination already exists (to avoid inconsistent state).
	finalDescFilename := defaultDescriptionBasename + extensionTXT
	finalDescPath := filepath.Join(path, finalDescFilename)

	// Check if file exists and should not be replaced.
	_, err := os.Stat(finalDescPath)
	if err == nil && !s.cfg.ReplaceDescriptions {
		logMessage := "%s description already exists, skipping save"
		if s.cfg.DryRun {
			logMessage = "[DRY-RUN] %s description file already exists, would skip"
		}

		logger.Infof(ctx, logMessage, itemType)

		return finalDescPath
	}

	// Generate UUID-based temp filename to avoid concurrent download conflicts.
	tempDescFilename := defaultDescriptionBasename + "_" + uuid.New().String() + extensionTXT
	tempDescPath := filepath.Join(path, tempDescFilename)

	// Dry-run mode: simulate description save.
	if s.cfg.DryRun {
		logger.Infof(ctx, "[DRY-RUN] Would save %s description to: %s", itemType, tempDescFilename)
		return tempDescPath
	}

	// Write description in UTF-8 encoding.
	err = os.WriteFile(tempDescPath, []byte(description), constants.DefaultFilePermissions)
	if err != nil {
		logger.Errorf(ctx, "Failed to save %s description: %v", itemType, err)
		return ""
	}

	logger.Infof(ctx, "Saved %s description to %s", itemType, tempDescFilename)

	return tempDescPath
}

// downloadCover downloads the cover art for albums, audiobooks, and podcasts.
func (s *ServiceImpl) downloadCover(ctx context.Context, imageURL, path, itemType string) (string, string) {
	// Trim and validate the cover art URL.
	bigImageURL := strings.TrimSpace(imageURL)
	if bigImageURL == "" {
		return "", ""
	}

	// Parse and process the cover URL.
	parsedCover := s.parseAlbumCoverURL(bigImageURL)
	coverURL := parsedCover.url

	coverExtension := parsedCover.extension
	if coverExtension == "" {
		coverExtension = extensionJPG
	}

	// Check if final destination already exists.
	finalCoverFilename := utils.SetFileExtension(defaultCoverBasename, coverExtension, false)
	finalCoverPath := filepath.Join(path, finalCoverFilename)

	if !s.cfg.ReplaceCovers {
		if _, err := os.Stat(finalCoverPath); err == nil {
			logger.Infof(ctx, "%s cover already exists, skipping download", itemType)
			return finalCoverPath, ""
		}
	}

	// Generate UUID-based temp filename.
	tempCoverFilename := defaultCoverBasename + "_" + uuid.New().String() + coverExtension
	tempCoverPath := filepath.Join(path, tempCoverFilename)

	// Download the cover art.
	skipped, err := s.downloadAndSaveFile(ctx, coverURL, tempCoverPath, s.cfg.ReplaceCovers)
	if err != nil {
		logger.Errorf(ctx, "Failed to download %s cover: %v", itemType, err)
		return "", ""
	}

	if skipped {
		logger.Infof(ctx, "%s cover already exists, skipping download", itemType)
	} else {
		logger.Infof(ctx, "Successfully downloaded %s cover", itemType)
	}

	return tempCoverPath, tempCoverPath
}

// finalizeDescription renames the description file for audiobooks and podcasts.
func (s *ServiceImpl) finalizeDescription(
	ctx context.Context,
	itemIndex int64,
	audioCollection *audioCollection,
	filename string,
) {
	// Only process on the last item.
	if itemIndex != audioCollection.tracksCount {
		return
	}

	// Check if we have a temp description path.
	if audioCollection.descriptionTempPath == "" {
		return
	}

	// Skip in dry-run mode.
	if s.cfg.DryRun {
		return
	}

	// Check if temp description file exists.
	if _, err := os.Stat(audioCollection.descriptionTempPath); err != nil {
		return
	}

	var newDescriptionFilename string
	// For single-item collections without a dedicated folder, rename to match the item filename.
	if !s.cfg.CreateFolderForSingles && audioCollection.tracksCount == 1 {
		newDescriptionFilename = utils.SetFileExtension(filename, extensionTXT, true)
	} else {
		// For multi-item or collections with folders, use standard name.
		newDescriptionFilename = defaultDescriptionBasename + extensionTXT
	}

	newDescriptionPath := filepath.Join(audioCollection.tracksPath, newDescriptionFilename)

	// Check if already renamed (same file).
	originalStat, err := os.Stat(audioCollection.descriptionTempPath)
	if err != nil {
		return
	}

	existingStat, err := os.Stat(newDescriptionPath)
	if err == nil && os.SameFile(originalStat, existingStat) {
		return
	}

	// Check ReplaceDescriptions flag if destination exists.
	if !s.cfg.ReplaceDescriptions && err == nil {
		logger.Infof(ctx, "Description already exists at final path, skipping rename")

		if removeErr := os.Remove(audioCollection.descriptionTempPath); removeErr != nil {
			logger.Warnf(
				ctx,
				"Failed to remove temp description '%s': %v",
				audioCollection.descriptionTempPath,
				removeErr,
			)
		}

		return
	}

	// Rename the description file.
	if renameErr := os.Rename(audioCollection.descriptionTempPath, newDescriptionPath); renameErr != nil {
		logger.Errorf(ctx, "Failed to rename description from '%s' to '%s': %v",
			audioCollection.descriptionTempPath, newDescriptionPath, renameErr)
	}
}
