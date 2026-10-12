package projectrun

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// These filesystem primitives support method-owned workspace journals and
// helper delivery. Native transport journal storage lives in projectexecutor.
func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("private directory is unavailable: %s", path)
	}
	if err := os.Chmod(path, 0700); err != nil {
		return fmt.Errorf("restrict private directory: %w", err)
	}
	return nil
}

func createPrivateFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create private file: %w", err)
	}
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return fmt.Errorf("restrict private file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flush private file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close private file: %w", err)
	}
	return nil
}

func readPrivateRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limit {
		return nil, errors.New("trusted private record is unavailable or exceeds its bound")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	wire, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(wire)) > limit {
		return nil, errors.New("trusted private record exceeds its bound")
	}
	return wire, nil
}

func pathIsWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func nativeRandomID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate private identity: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
