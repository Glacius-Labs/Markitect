//go:build !unix && !windows

package checks

import (
	"errors"
	"os"
)

func redirected(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }
func openExecutable(*os.Root, string) (*os.File, error) {
	return nil, errors.New("regular executable handle observation unsupported on this platform")
}
