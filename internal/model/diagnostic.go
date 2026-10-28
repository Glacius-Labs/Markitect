package model

import (
	"fmt"
	"sort"
)

// Diagnostic is one finding of the compiler or the linter. File is
// repository-relative with forward slashes; Line is 1-based and 0 when the
// finding concerns the whole file or no file.
type Diagnostic struct {
	Code    string `json:"code"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

func (d Diagnostic) String() string {
	switch {
	case d.File == "":
		return fmt.Sprintf("%s: %s", d.Code, d.Message)
	case d.Line == 0:
		return fmt.Sprintf("%s: %s: %s", d.File, d.Code, d.Message)
	default:
		return fmt.Sprintf("%s:%d: %s: %s", d.File, d.Line, d.Code, d.Message)
	}
}

// Diagnostics collects findings in a stable order.
type Diagnostics []Diagnostic

// Add appends one finding.
func (ds *Diagnostics) Add(code, file string, line int, format string, args ...any) {
	*ds = append(*ds, Diagnostic{Code: code, File: file, Line: line, Message: fmt.Sprintf(format, args...)})
}

// Sorted returns the findings ordered by file, line, code and message, without
// exact duplicates.
func (ds Diagnostics) Sorted() Diagnostics {
	out := append(Diagnostics(nil), ds...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	unique := out[:0]
	for i, d := range out {
		if i == 0 || d != out[i-1] {
			unique = append(unique, d)
		}
	}
	return unique
}
