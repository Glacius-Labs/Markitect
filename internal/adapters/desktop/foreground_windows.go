package desktop

import "syscall"

var allowSetForegroundWindow = syscall.NewLazyDLL("user32.dll").NewProc("AllowSetForegroundWindow")

// allowForeground lets the running service bring the window it opens for
// this start to the front. Windows lets only the program the user just
// started do that, and this start is that program, not the service.
func allowForeground(pid int) {
	if pid > 0 && allowSetForegroundWindow.Find() == nil {
		_, _, _ = allowSetForegroundWindow.Call(uintptr(pid))
	}
}
