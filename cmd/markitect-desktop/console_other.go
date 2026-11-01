//go:build native && !windows

package main

// hideChildConsoles has nothing to do: only Windows opens a window for a
// console program.
func hideChildConsoles() error { return nil }
