//go:build native && windows

package main

import (
	"syscall"
	"unsafe"
)

// messageBoxStyle is MB_ICONERROR | MB_TOPMOST: an error icon and an OK button,
// on top of other windows. A user who waited up to 60 seconds may be working
// elsewhere by now, so it does not ask for the foreground, where a key meant
// for another window would dismiss it unread.
const messageBoxStyle = 0x00000010 | 0x00040000

var messageBoxW = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

// messageBox shows text in Windows' own message box and returns once it is
// dismissed. Wails' dialogs cannot: they need the app's window loop, and Wails
// ends the app from inside it. A text it cannot show stays in the error log.
func messageBox(text string) {
	title, err := syscall.UTF16PtrFromString("Markitect")
	if err != nil {
		return
	}
	body, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	_, _, _ = messageBoxW.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), messageBoxStyle)
}
