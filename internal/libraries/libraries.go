// Package libraries holds the standard Libraries that ship inside the
// markitect binary: std, rest and cli, each a folder of Kind Definitions.
package libraries

import (
	"bytes"
	"embed"
	"io/fs"
	"sort"
)

// Version is the version of every standard Library in this binary.
const Version = "1.0.0"

//go:embed std rest cli
var files embed.FS

// File is one file of a standard Library.
type File struct {
	Path string // relative to the Library folder
	Data []byte
}

var crlf, lf = []byte{13, 10}, []byte{10}

// Names returns the names of the standard Libraries, sorted.
func Names() []string { return []string{"cli", "rest", "std"} }

// Files returns the files of a standard Library, or false when there is no
// standard Library of that name.
func Files(name string) ([]File, bool) {
	entries, err := fs.ReadDir(files, name)
	if err != nil {
		return nil, false
	}
	var out []File
	for _, e := range entries {
		data, err := files.ReadFile(name + "/" + e.Name())
		if err != nil {
			return nil, false
		}
		// The lock pins these bytes; LF line ends keep them the same on every
		// platform, whatever a checkout did to them.
		out = append(out, File{Path: e.Name(), Data: bytes.ReplaceAll(data, crlf, lf)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, true
}
