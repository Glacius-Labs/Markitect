//go:build native && windows

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// consoleVisible starts a console program and has it say whether its console
// has a window one can see.
func consoleVisible(t *testing.T) bool {
	t.Helper()
	const script = `Add-Type -Namespace Probe -Name Console -MemberDefinition '` +
		`[DllImport("kernel32.dll")] public static extern IntPtr GetConsoleWindow();` +
		`[DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr window);';` +
		`[Probe.Console]::IsWindowVisible([Probe.Console]::GetConsoleWindow())`
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		t.Fatalf("the probe: %v\n%s", err, out)
	}
	switch strings.TrimSpace(string(out)) {
	case "True":
		return true
	case "False":
		return false
	}
	t.Fatalf("the probe said %q", out)
	return false
}

// Run it as the app is built, a GUI program, or it has a console already:
//
//	go test -tags native -ldflags=-H=windowsgui ./cmd/markitect-desktop
func TestAProgramTheAppStartsShowsNoWindow(t *testing.T) {
	// Without the hidden console a child shows a window, so the probe can see
	// the fault. Only CI shows it; on a person's screen it is one more flash.
	if os.Getenv("CI") != "" {
		t.Logf("before the hidden console, a child's window is visible: %v", consoleVisible(t))
	}
	if err := hideChildConsoles(); err != nil {
		t.Fatal(err)
	}
	if consoleVisible(t) {
		t.Fatal("a program the app starts shows a console window")
	}
	// The shell that made the console has ended, and the console lives on.
	if consoleVisible(t) {
		t.Fatal("a later program shows a console window")
	}
}
