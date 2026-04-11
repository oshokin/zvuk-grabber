package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// RenameFile renames sourcePath to destinationPath.
// If replace is true and destination exists, destination is atomically replaced.
func RenameFile(srcPath, dstPath string, replace bool) error {
	sourcePath := filepath.Clean(srcPath)
	destinationPath := filepath.Clean(dstPath)

	if sourcePath == "" || destinationPath == "" || sourcePath == destinationPath {
		return nil
	}

	if !replace {
		if _, err := os.Stat(destinationPath); err == nil {
			return fmt.Errorf("%w: destination already exists: %s", os.ErrExist, destinationPath)
		}
	}

	err := os.Rename(sourcePath, destinationPath)
	if err == nil {
		return nil
	}

	if !replace {
		return err
	}

	if _, statErr := os.Stat(destinationPath); statErr != nil {
		return err
	}

	return replaceExistingDestination(sourcePath, destinationPath)
}

func replaceExistingDestination(sourcePath, destinationPath string) error {
	backupPath := destinationPath + ".bak-" + uuid.NewString()

	if err := os.Rename(destinationPath, backupPath); err != nil {
		return fmt.Errorf("failed to move existing destination aside: %w", err)
	}

	moveSourceErr := os.Rename(sourcePath, destinationPath)
	if moveSourceErr == nil {
		_ = os.Remove(backupPath)
		return nil
	}

	restoreErr := os.Rename(backupPath, destinationPath)
	if restoreErr != nil && !errors.Is(restoreErr, os.ErrNotExist) {
		return fmt.Errorf("rename failed: %w (restore failed: %w)", moveSourceErr, restoreErr)
	}

	return moveSourceErr
}
