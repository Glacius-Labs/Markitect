package buildinfo

import _ "embed"

// Notices is the upstream license notices shipped with Markitect, the single
// source for the read-only CLI notice display and source distribution.
//
//go:embed notices.md
var Notices string
