// Package library is the library built into the tool: the packages of types,
// the templates that apply copies into a spec, the types the tool owns and
// never copies, of reports, settings and control samples, and per package
// its reference values and its question catalog, which the tool reads and
// never copies.
//
// Its contents are described in docs/guides/library.md.
package library

import (
	"embed"
	"io/fs"
)

//go:embed packages templates references questions report.md settings.md sample.md
var files embed.FS

// The paths in FS of the types the tool owns: of a report, of the settings
// and of the result of a control sample.
const (
	ReportType   = "report.md"
	SettingsType = "settings.md"
	SampleType   = "sample.md"
)

// FS returns the library: packages/NAME/ for each package, templates/NAME/
// for each template, references/NAME.md with the reference values and
// questions/NAME.md with the question catalog of each package, and the types
// the tool owns at ReportType, SettingsType and SampleType.
func FS() fs.FS { return files }
