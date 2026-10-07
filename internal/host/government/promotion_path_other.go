//go:build !windows

package government

import (
	"fmt"
	"os"
	"path/filepath"
)

func promotionIsReparsePoint(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink != 0
}

func normalizePromotionPath(path string) (string, error) { return path, nil }

func writePromotionIntentDurable(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open promotion intent parent for sync: %w", err)
	}
	syncErr := directory.Sync()
	closeErr = directory.Close()
	if syncErr != nil {
		return fmt.Errorf("sync promotion intent parent directory: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close promotion intent parent directory: %w", closeErr)
	}
	return nil
}

func createOperationalDirectoryDurable(parent, prefix string) (string, error) {
	directory, err := os.MkdirTemp(parent, prefix)
	if err != nil {
		return "", err
	}
	parentHandle, err := os.Open(parent)
	if err != nil {
		return directory, fmt.Errorf("open operational directory parent for sync: %w", err)
	}
	syncErr := parentHandle.Sync()
	closeErr := parentHandle.Close()
	if syncErr != nil {
		return directory, fmt.Errorf("sync operational directory parent: %w", syncErr)
	}
	if closeErr != nil {
		return directory, fmt.Errorf("close operational directory parent: %w", closeErr)
	}
	return directory, nil
}
