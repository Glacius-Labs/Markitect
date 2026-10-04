// Package licenses provides the upstream notices shipped with Markitect.
package licenses

import _ "embed"

// Text is the single source for the read-only CLI notice display and source distribution.
//
//go:embed notices.md
var Text string
