// Package library is the library of Markitect: the module core, the types that
// a person copies into a spec by hand, and per module its reference values and
// its question catalog, which the tool reads and never copies. Only the
// reference values and the question catalog are built into the tool; the module
// is a plain folder of this folder.
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
