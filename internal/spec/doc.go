// Package spec compiles the spec from explicitly supplied Markdown files:
// Definitions with front matter, Packages as folders, Libraries of Kinds,
// links by path, visibility and the Package dependency graph.
//
// The package reads no files itself. Its callers hand it the bytes of the
// spec folder and of every Library the manifest names, and it returns the
// compiled Spec with every compile error as a Diagnostic. The language is
// described in docs/guides/spec-language.md.
package spec
