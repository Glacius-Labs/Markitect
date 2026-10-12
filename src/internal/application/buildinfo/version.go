// Package buildinfo owns the version shared by product and release tools.
package buildinfo

var version = "0.14.1"

// Version returns the source version or the immutable build's injected version.
func Version() string { return version }
