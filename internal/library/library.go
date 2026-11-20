// Package library is the library built into the tool: the packages of types,
// the templates that apply copies into a spec, and the report type, which the
// tool owns and never copies.
//
// Its contents are described in docs/guides/library.md.
package library

import (
	"embed"
	"io/fs"
)

//go:embed packages templates report.md
var files embed.FS

// ReportType is the path of the report type in FS.
const ReportType = "report.md"

// FS returns the library: packages/NAME/ for each package, templates/NAME/
// for each template, and the report type at ReportType.
func FS() fs.FS { return files }
