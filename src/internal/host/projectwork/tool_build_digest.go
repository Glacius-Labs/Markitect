package projectwork

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

const maximumToolExecutableBytes = 1 << 30

// ToolBuildDigest binds a proposal to the actual running Host executable.
// Build metadata alone cannot distinguish different uncommitted source builds.
func ToolBuildDigest() (string, error) {
	name, err := os.Executable()
	if err != nil {
		return "", errors.New("identify running Host executable")
	}
	return toolExecutableDigest(name)
}

func toolExecutableDigest(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", errors.New("open Host executable for identity")
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > maximumToolExecutableBytes {
		return "", errors.New("Host executable is not a bounded nonempty regular file")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maximumToolExecutableBytes+1))
	if err != nil || n != before.Size() {
		return "", errors.New("Host executable could not be read consistently")
	}
	after, err := f.Stat()
	if err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return "", errors.New("Host executable changed while computing identity")
	}
	current, err := os.Stat(name)
	if err != nil || !os.SameFile(before, current) {
		return "", errors.New("Host executable identity changed")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
