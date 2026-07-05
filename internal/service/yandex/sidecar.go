package yandex

import (
	"context"
	"os"
	"path/filepath"

	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// writeSidecarFile writes sidecar content atomically via temp file and rename.
func writeSidecarFile(path, content string, replace bool) error {
	if err := os.MkdirAll(filepath.Dir(path), files.DefaultFolderPermissions); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+tempFilePattern)
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	cleanup := true

	defer func() {
		_ = tempFile.Close()

		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err = tempFile.WriteString(content + "\n"); err != nil {
		return err
	}

	if err = tempFile.Close(); err != nil {
		return err
	}

	if err = utils.RenameFile(tempPath, path, replace); err != nil {
		return err
	}

	cleanup = false

	return nil
}

// useExistingSidecar reports whether an existing sidecar file should be reused.
func (s *ServiceImpl) useExistingSidecar(
	ctx context.Context,
	path string,
	replace bool,
	recordSkipped func(),
	checkFailedMessage string,
	alreadyExistsMessage string,
) bool {
	if replace {
		return false
	}

	exists, existsErr := utils.IsFileExist(path)
	if existsErr != nil {
		logger.Debugf(ctx, checkFailedMessage, path, existsErr)
		return false
	}

	if !exists {
		return false
	}

	if recordSkipped != nil {
		recordSkipped()
	}

	logger.Infof(ctx, alreadyExistsMessage)

	return true
}
