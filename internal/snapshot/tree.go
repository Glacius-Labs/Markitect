package snapshot

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrInvalidEntry identifies a zero Entry supplied to NewTree.
var ErrInvalidEntry = errors.New("invalid snapshot entry")

// ErrPathCollision identifies exact duplicate paths and file/directory
// conflicts. NewTree sorts first, so the selected error is input-order independent.
var ErrPathCollision = errors.New("snapshot path collision")

// TreeID identifies a fixed tree, independently of acquisition or workspace state.
type TreeID [sha256.Size]byte

func (id TreeID) String() string { return hex.EncodeToString(id[:]) }

// Tree is an immutable, sorted collection of entries. Its zero value is empty.
type Tree struct{ entries []Entry }

// NewTree fixes entries in exact-path byte order and checks the complete tree.
// Case and Unicode normalization forms are distinct, including directory names.
// No partial Tree is returned on failure.
func NewTree(entries []Entry) (Tree, error) {
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.path.value, b.path.value) })
	paths := make(map[string]bool)
	files := make(map[string]bool)
	for _, entry := range entries {
		if entry.path.value == "" {
			return Tree{}, ErrInvalidEntry
		}
		parts := strings.Split(entry.path.value, "/")
		for i := range parts {
			prefix := strings.Join(parts[:i+1], "/")
			if paths[prefix] {
				if files[prefix] || i == len(parts)-1 {
					return Tree{}, fmt.Errorf("%w: %q", ErrPathCollision, prefix)
				}
			}
			paths[prefix] = true
			if i == len(parts)-1 {
				files[prefix] = true
			}
		}
	}
	return Tree{entries: entries}, nil
}

// EntryCount returns the number of files without allocating a manifest copy.
func (t Tree) EntryCount() int { return len(t.entries) }

// Entries returns a caller-owned slice in exact-path byte order.
func (t Tree) Entries() []Entry { return slices.Clone(t.entries) }

// Lookup finds an exact path, without case folding or normalization.
func (t Tree) Lookup(path Path) (Entry, bool) {
	i, found := slices.BinarySearchFunc(t.entries, path.value, func(e Entry, p string) int {
		return strings.Compare(e.path.value, p)
	})
	if !found {
		return Entry{}, false
	}
	return t.entries[i], true
}

// Equal compares exact path, mode and bytes without relying on hash equality.
func (t Tree) Equal(other Tree) bool { return slices.Equal(t.entries, other.entries) }

// Digest returns SHA-256 of "markitect.snapshot.tree.v1\x00", then the entry
// count as uint64 big endian, then each sorted entry's path, one-byte Mode and
// raw 32-byte ContentID, each prefixed by its uint64 big-endian byte length.
// The encoding is separate from canonical model and Git identities.
func (t Tree) Digest() TreeID {
	h := sha256.New()
	_, _ = h.Write([]byte("markitect.snapshot.tree.v1\x00"))
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(t.entries)))
	_, _ = h.Write(size[:])
	write := func(value []byte) {
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = h.Write(value)
	}
	for _, entry := range t.entries {
		write([]byte(entry.path.value))
		write([]byte{byte(entry.mode)})
		content := entry.ContentID()
		write(content[:])
	}
	var id TreeID
	copy(id[:], h.Sum(nil))
	return id
}
