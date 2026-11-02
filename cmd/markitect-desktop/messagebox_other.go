//go:build native && !windows

package main

// messageBox has nothing to do: the failures it tells about are those of
// WebView2, which only Windows has, and on the other platforms a fatal error
// stays a line in the error log.
func messageBox(string) {}
