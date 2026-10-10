//go:build windows

package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func windowsShortPath(longPath string) (string, error) {
	input, err := syscall.UTF16PtrFromString(windowsLongPathAPISpelling(longPath))
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, 260)
	n, err := syscall.GetShortPathName(input, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", syscall.EINVAL
	}
	if int(n) >= len(buffer) {
		buffer = make([]uint16, int(n)+1)
		n, err = syscall.GetShortPathName(input, &buffer[0], uint32(len(buffer)))
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(buffer[:n]), nil
}

func windowsLongPathAPISpelling(path string) string {
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

func makeWindowsJunction(link, target string) error {
	command := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target)
	_, junctionErr := command.CombinedOutput()
	if junctionErr == nil {
		return nil
	}
	if err := os.Symlink(target, link); err == nil {
		return nil
	}
	return junctionErr
}
