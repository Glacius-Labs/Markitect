//go:build windows

package government

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var promotionKernel32 = syscall.NewLazyDLL("kernel32.dll")
var promotionMoveFileExW = promotionKernel32.NewProc("MoveFileExW")

func promotionIsReparsePoint(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func normalizePromotionPath(path string) (string, error) {
	switch {
	case strings.HasPrefix(strings.ToLower(path), `\\?\unc\`):
		return `\\` + path[len(`\\?\UNC\`):], nil
	case strings.HasPrefix(path, `\\?\`):
		return strings.TrimPrefix(path, `\\?\`), nil
	case strings.HasPrefix(path, `\\.\`):
		return "", errors.New("device namespace paths are not supported for operational directories")
	default:
		return path, nil
	}
}

func writePromotionIntentDurable(path string, data []byte) error {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("create promotion intent staging name: %w", err)
	}
	temporary := path + ".pending-" + hex.EncodeToString(nonce[:])
	f, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(temporary)
		return err
	}
	// MOVEFILE_WRITE_THROUGH makes MoveFileExW wait for the same-directory
	// rename metadata to reach disk. Without this platform guarantee, promotion
	// fails before the Git ref can be changed.
	if err := movePromotionWriteThrough(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("durably publish promotion intent: %w", err)
	}
	return nil
}

func createOperationalDirectoryDurable(parent, prefix string) (string, error) {
	staging, err := os.MkdirTemp(parent, prefix+"-staging-")
	if err != nil {
		return "", err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		_ = os.Remove(staging)
		return "", fmt.Errorf("create operational directory name: %w", err)
	}
	destination := filepath.Join(parent, prefix+"-"+hex.EncodeToString(nonce[:]))
	if err := movePromotionWriteThrough(staging, destination); err != nil {
		_ = os.Remove(staging)
		return "", fmt.Errorf("durably publish operational directory: %w", err)
	}
	return destination, nil
}

func movePromotionWriteThrough(source, destination string) error {
	from, err := syscall.UTF16PtrFromString(promotionLongPath(source))
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(promotionLongPath(destination))
	if err != nil {
		return err
	}
	// MOVEFILE_WRITE_THROUGH is 0x00000008 in Win32.
	const moveFileWriteThrough = 0x00000008
	if moved, _, callErr := promotionMoveFileExW.Call(
		uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), moveFileWriteThrough,
	); moved == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("MoveFileExW returned failure without an error")
	}
	return nil
}

func promotionLongPath(path string) string {
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
	}
	if filepath.IsAbs(path) && filepath.VolumeName(path) != "" {
		return `\\?\` + path
	}
	return path
}
