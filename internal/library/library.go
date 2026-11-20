// Package library is the library built into the tool: the packages of types,
// the templates that apply copies into a spec, the report type, which the
// tool owns and never copies, and per package its reference values and its
// question catalog, which the tool reads and never copies.
//
// Its contents are described in docs/guides/library.md.
package library

import (
	"embed"
	"io/fs"
)

//go:embed packages templates references questions report.md
var files embed.FS

// ReportType is the path of the report type in FS.
const ReportType = "report.md"

// FS returns the library: packages/NAME/ for each package, templates/NAME/
// for each template, references/NAME.md with the reference values and
// questions/NAME.md with the question catalog of each package, and the report
// type at ReportType.
func FS() fs.FS { return files }
