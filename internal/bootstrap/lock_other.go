//go:build !unix

package bootstrap

import (
	"errors"
	"os"
)

// No best-effort lock on a host without a supported native locking primitive.
// Windows is such a host: it refuses native Apply, the lock's only user, first.
func lockFile(*os.File) error {
	return errors.New("state directory locking is unsupported on this platform")
}
