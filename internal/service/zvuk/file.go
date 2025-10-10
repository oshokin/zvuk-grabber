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

func (s *ServiceImpl) downloadAndSaveFile(ctx context.Context, url, destinationPath string, overwrite bool) error {
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

			return nil
		}

		return err
	}

	defer file.Close()

	// Download the file content from the URL.
	reader, err := s.zvukClient.DownloadFromURL(ctx, url)
	if err != nil {
		return err
	}

	defer reader.Close()

	// Copy the downloaded content to the file.
	_, err = io.Copy(file, reader)

	return err
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
