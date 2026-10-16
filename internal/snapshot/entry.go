package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Mode describes a portable regular file, not an OS permission bitmask or Git mode.
type Mode uint8

const (
	// Regular is a regular file without executable permission.
	Regular Mode = 1
	// Executable is a regular file with executable permission.
	Executable Mode = 2
)

// ErrInvalidMode identifies unsupported entry types or modes, including zero.
var ErrInvalidMode = errors.New("invalid snapshot mode")

// ContentID is SHA-256 of exact file bytes, independent of path and mode.
type ContentID [sha256.Size]byte

func (id ContentID) String() string { return hex.EncodeToString(id[:]) }

// Entry is an immutable regular file. Its zero value is invalid.
// Entries are comparable; equality includes exact path, mode and bytes.
type Entry struct {
	path    Path
	mode    Mode
	content string
}

// NewEntry fixes explicit bytes by copying them. It rejects zero paths and modes
// other than Regular and Executable; nil bytes mean empty file content.
func NewEntry(path Path, mode Mode, content []byte) (Entry, error) {
	if path.value == "" {
		return Entry{}, fmt.Errorf("%w: zero path", ErrInvalidPath)
	}
	if mode != Regular && mode != Executable {
		return Entry{}, fmt.Errorf("%w: %d at %q", ErrInvalidMode, mode, path)
	}
	return Entry{path: path, mode: mode, content: string(content)}, nil
}

func (e Entry) Path() Path { return e.path }
func (e Entry) Mode() Mode { return e.mode }

// Bytes returns a fresh copy of the content.
func (e Entry) Bytes() []byte { return []byte(e.content) }

func (e Entry) ContentID() ContentID { return ContentID(sha256.Sum256([]byte(e.content))) }
