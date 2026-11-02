//go:build native && windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	genericAll       = 0x10000000
	wmClose          = 0x0010
	dialogClass      = "#32770" // the window class of a message box
	messageBoxTextID = 0xFFFF   // the control in a message box that holds its text
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	createDesktop      = user32.NewProc("CreateDesktopW")
	openDesktop        = user32.NewProc("OpenDesktopW")
	closeDesktop       = user32.NewProc("CloseDesktop")
	setThreadDesktop   = user32.NewProc("SetThreadDesktop")
	enumDesktopWindows = user32.NewProc("EnumDesktopWindows")
	getClassName       = user32.NewProc("GetClassNameW")
	getWindowText      = user32.NewProc("GetWindowTextW")
	getDlgItemText     = user32.NewProc("GetDlgItemTextW")
	postMessage        = user32.NewProc("PostMessageW")
)

func utf16Arg(s string) uintptr {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return uintptr(unsafe.Pointer(p))
}

// hiddenDesktop makes a desktop that no screen shows, and returns its name and
// handle. A window on it behaves as on any other, and nobody sees it.
func hiddenDesktop(t *testing.T) (string, uintptr) {
	t.Helper()
	name := fmt.Sprintf("markitect-test-%d", os.Getpid())
	desktop, _, err := createDesktop.Call(utf16Arg(name), 0, 0, 0, genericAll, 0)
	if desktop == 0 {
		t.Fatalf("no hidden desktop: %v", err)
	}
	t.Cleanup(func() { _, _, _ = closeDesktop.Call(desktop) })
	return name, desktop
}

// showOnDesktop is messageBox on the named hidden desktop. When the box cannot
// go there the process ends, so that it never shows on a person's screen.
func showOnDesktop(name string) func(string) {
	return func(text string) {
		// Never unlocked: the thread on that desktop ends with this goroutine.
		runtime.LockOSThread()
		desktop, _, err := openDesktop.Call(utf16Arg(name), 0, 0, genericAll)
		if desktop == 0 {
			fmt.Fprintln(os.Stderr, "no hidden desktop:", err)
			os.Exit(3)
		}
		if ok, _, err := setThreadDesktop.Call(desktop); ok == 0 {
			fmt.Fprintln(os.Stderr, "not on the hidden desktop:", err)
			os.Exit(3)
		}
		messageBox(text)
	}
}

// messageBoxShown waits on desktop for a message box until ended closes. It
// dismisses the box it finds and returns its title and text, or nothing when
// none showed.
func messageBoxShown(desktop uintptr, ended <-chan struct{}) (title, text string) {
	var box uintptr
	look := syscall.NewCallback(func(window, _ uintptr) uintptr {
		var class [16]uint16
		n, _, _ := getClassName.Call(window, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
		if syscall.UTF16ToString(class[:n]) == dialogClass {
			box = window
			return 0
		}
		return 1
	})
	// The box is on the desktop before it holds its text.
	for text == "" {
		select {
		case <-ended:
			return "", ""
		case <-time.After(50 * time.Millisecond):
		}
		_, _, _ = enumDesktopWindows.Call(desktop, look, 0)
		if box != 0 {
			var buffer [2048]uint16
			n, _, _ := getDlgItemText.Call(box, messageBoxTextID, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
			text = syscall.UTF16ToString(buffer[:n])
		}
	}
	var buffer [64]uint16
	n, _, _ := getWindowText.Call(box, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	// What the box's close button sends; it ignores a posted IDOK.
	_, _, _ = postMessage.Call(box, wmClose, 0, 0)
	return syscall.UTF16ToString(buffer[:n]), text
}
