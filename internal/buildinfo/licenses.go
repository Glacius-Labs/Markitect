package buildinfo

import _ "embed"

// Notices is the upstream license notices shipped with Markitect, printed by
// `markitect licenses`.
//
//go:embed notices.md
var Notices string
