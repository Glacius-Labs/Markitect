// Package snapshot defines a fixed collection of source files and deterministic
// operations over snapshots, independent of how their bytes were obtained.
package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

// RegularMode and ExecutableMode are portable regular-file mode tokens in the
// legacy Git mode encoding used by Markitect snapshots.
const (
	RegularMode    = "100644"
	ExecutableMode = "100755"
)

// Snapshot contains file bytes keyed by slash-separated repository paths.
// ID is an opaque identity supplied by the source; it and Provisional do not
// affect Digest. Modes uses portable regular-file mode tokens in the legacy
// Git mode encoding.
type Snapshot struct {
	ID          string
	Provisional bool
	Files       map[string][]byte
	Modes       map[string]string
}

// Digest returns the legacy stable SHA-256 fingerprint of sorted paths, mode
// strings and bytes. It intentionally excludes ID and Provisional.
func (s *Snapshot) Digest() string {
	if s == nil {
		return ""
	}
	h := sha256.New()
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var n [8]byte
	for _, p := range paths {
		writeField := func(b []byte) {
			binary.BigEndian.PutUint64(n[:], uint64(len(b)))
			_, _ = h.Write(n[:])
			_, _ = h.Write(b)
		}
		writeField([]byte(p))
		writeField([]byte(s.Modes[p]))
		writeField(s.Files[p])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ChangeSet records paths whose bytes or mode changed between snapshots.
type ChangeSet struct {
	Added    []string
	Modified []string
	Removed  []string
}

// Paths returns the recorded changed paths in lexicographic order.
func (c ChangeSet) Paths() []string {
	if len(c.Added)+len(c.Modified)+len(c.Removed) == 0 {
		return nil
	}
	paths := make([]string, 0, len(c.Added)+len(c.Modified)+len(c.Removed))
	paths = append(paths, c.Added...)
	paths = append(paths, c.Modified...)
	paths = append(paths, c.Removed...)
	sort.Strings(paths)
	return paths
}

// Compare classifies file additions, removals, and byte or mode modifications.
// Nil snapshots are treated as empty snapshots.
func Compare(before, after *Snapshot) ChangeSet {
	var result ChangeSet
	var beforeFiles, afterFiles map[string][]byte
	var beforeModes, afterModes map[string]string
	if before != nil {
		beforeFiles, beforeModes = before.Files, before.Modes
	}
	if after != nil {
		afterFiles, afterModes = after.Files, after.Modes
	}
	for name, oldBytes := range beforeFiles {
		newBytes, exists := afterFiles[name]
		if !exists {
			result.Removed = append(result.Removed, name)
		} else if !bytes.Equal(oldBytes, newBytes) || beforeModes[name] != afterModes[name] {
			result.Modified = append(result.Modified, name)
		}
	}
	for name := range afterFiles {
		if _, exists := beforeFiles[name]; !exists {
			result.Added = append(result.Added, name)
		}
	}
	sort.Strings(result.Added)
	sort.Strings(result.Modified)
	sort.Strings(result.Removed)
	return result
}
