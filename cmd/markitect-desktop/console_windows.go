//go:build native && windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// createNoWindow starts a console program with a console that has no window.
const createNoWindow = 0x08000000

var attachConsole = syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")

// hideChildConsoles gives the app a console nobody sees. A GUI program has no
// console, so Windows opens a new console window for every console program it
// starts: each Git call of the service would flash one. A flash also takes the
// focus from the app's window, and a window that gets it back reads the
// project again, which starts Git again. Every program the app starts inherits
// this console instead, whichever package starts it: Git, Docker and what they
// start in turn.
//
// A console without a window comes only with a new process, so a shell starts
// with one, the app joins it, and the shell ends; the console lives on with the
// app.
func hideChildConsoles() error {
	shell := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"), "/d", "/q", "/k")
	shell.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	// It reads commands from this pipe, so it waits until the app has joined.
	in, err := shell.StdinPipe()
	if err != nil {
		return fmt.Errorf("the hidden console: %w", err)
	}
	if err := shell.Start(); err != nil {
		return fmt.Errorf("the hidden console: %w", err)
	}
	defer func() {
		in.Close()
		_ = shell.Process.Kill()
		_ = shell.Wait()
	}()
	// The console appears shortly after the process does.
	var last error
	for range 200 {
		ok, _, err := attachConsole.Call(uintptr(shell.Process.Pid))
		// Access denied: this process has a console already, as a test run
		// from a terminal may.
		if ok != 0 || errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return nil
		}
		last = err
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("the hidden console: %w", last)
}
