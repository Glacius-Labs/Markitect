//go:build !windows

package desktop

// allowForeground does nothing: only Windows keeps a program from bringing
// its window to the front unless the user just started it.
func allowForeground(int) {}
