//go:build windows

package government

import (
	"path/filepath"
	"strings"
	"syscall"
)

func samePromotionPathSpelling(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func promotionExtendedPathForTest(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
	}
	return `\\?\` + path
}

func promotionShortPathForTest(path string) (string, bool) {
	input, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", false
	}
	buffer := make([]uint16, 260)
	for {
		n, callErr := syscall.GetShortPathName(input, &buffer[0], uint32(len(buffer)))
		if callErr != nil || n == 0 {
			return "", false
		}
		if int(n) >= len(buffer) {
			buffer = make([]uint16, int(n)+1)
			continue
		}
		short := filepath.Clean(syscall.UTF16ToString(buffer[:n]))
		return short, !strings.EqualFold(short, filepath.Clean(path))
	}
}
