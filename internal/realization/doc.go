// Package realization checks the realization record beside the spec: one
// registry page per Package with the path patterns it owns and the files it
// realizes, and the ignore file. It gives every repository file exactly one
// owning Package and reports files that are neither registered nor ignored.
//
// The record is described in docs/guides/spec-language.md.
package realization
