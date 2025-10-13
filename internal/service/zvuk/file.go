package zvuk

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

const (
	// File options for overwriting an existing file.
	overwriteFileOptions = os.O_CREATE | os.O_TRUNC | os.O_WRONLY

	// File options for creating a new file (fails if the file already exists).
	createNewFileOptions = os.O_CREATE | os.O_EXCL | os.O_WRONLY
)

func (s *ServiceImpl) downloadAndSaveFile(
	ctx context.Context,
	url, destinationPath string,
	overwrite bool,
) (bool, error) {
	// Dry-run mode: simulate download without saving files.
	if s.cfg.DryRun {
		// Check if file would be skipped.
		if !overwrite {
			if _, err := os.Stat(destinationPath); err == nil {
				logger.Infof(ctx, "[DRY-RUN] File '%s' already exists, would skip", destinationPath)

				return true, nil
			}
		}

		logger.Infof(ctx, "[DRY-RUN] Would download file to: %s", destinationPath)

		return false, nil
	}

	// Choose file options based on whether we're allowed to overwrite the file.
	fileOptions := overwriteFileOptions
	if !overwrite {
		fileOptions = createNewFileOptions
	}

	// Open the file with the chosen options.
	file, err := os.OpenFile(filepath.Clean(destinationPath), fileOptions, constants.DefaultFilePermissions)
	if err != nil {
		// If the file already exists and we're not overwriting, log and skip.
		if os.IsExist(err) && !overwrite {
			logger.Infof(ctx, "File '%s' already exists, skipping download", destinationPath)

			return true, nil
		}

		return false, err
	}

	// Download the file content from the URL.
	reader, err := s.zvukClient.DownloadFromURL(ctx, url)
	if err != nil {
		_ = file.Close()
		return false, err
	}

	defer reader.Close()

	// Copy the downloaded content to the file.
	_, err = io.Copy(file, reader)
	if err != nil {
		_ = file.Close()
		return false, err
	}

	// Force flush all buffered data to disk.
	if syncErr := file.Sync(); syncErr != nil {
		_ = file.Close()
		return false, syncErr
	}

	// This ensures the file is fully written and closed before any other goroutine
	// tries to read it (e.g., for embedding covers).
	// Relying on defer is too late because it executes after the function returns,
	// creating a race condition where readers get partial/corrupted data.
	if closeErr := file.Close(); closeErr != nil {
		return false, closeErr
	}

	return false, nil
}

func (s *ServiceImpl) truncateFolderName(ctx context.Context, pattern, name string) string {
	// Check if the folder name exceeds the maximum allowed length.
	if s.cfg.MaxFolderNameLength > 0 && int64(len([]rune(name))) > s.cfg.MaxFolderNameLength {
		// Truncate the name to the maximum length.
		truncated := string([]rune(name)[:s.cfg.MaxFolderNameLength])
		logger.Infof(ctx, "%s folder name was truncated to %d characters", pattern, s.cfg.MaxFolderNameLength)

		return truncated
	}

	return name
}
