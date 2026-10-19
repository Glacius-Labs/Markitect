//go:build !unix && !windows

package bootstrap

import (
	"errors"
	"os"
)

// No best-effort lock on a Host without a supported native locking primitive.
func lockFile(*os.File) error {
	return errors.New("state directory locking is unsupported on this platform")
}
