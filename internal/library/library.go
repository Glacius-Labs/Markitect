// Package library is what the tool knows of the default that ships with
// Markitect: per module its reference values and its question catalog, which
// the tool reads and never copies. Both are built into the tool. The default
// itself, the spec folder that a project copies, is the folder plugin/default/
// of the repository and is no part of the binary.
//
// Its contents are described in docs/guides/library.md.
package library

import (
	"embed"
	"io/fs"
)

//go:embed references questions
var files embed.FS

// FS returns what the tool reads at run time: references/NAME.md with the
// reference values and questions/NAME.md with the question catalog of each
// module. Every file of it is a document of the language.
func FS() fs.FS { return files }
