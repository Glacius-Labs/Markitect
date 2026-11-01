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

func TestAProgramTheAppStartsShowsNoWindow(t *testing.T) {
	if hasConsole() {
		t.Fatal("run this as the app is built, as a GUI program: go test -tags native -ldflags=-H=windowsgui ./cmd/markitect-desktop")
	}
	// Without the hidden console a child shows a window, which proves that the
	// probe can see the fault. Only CI shows it: on a person's screen it is one
	// more flash, and under Windows Terminal the probe sees no window at all.
	if os.Getenv("CI") != "" && !consoleVisible(t) {
		t.Fatal("before the hidden console, the probe sees no window; it cannot see the fault here")
	}
	if err := hideChildConsoles(); err != nil {
		t.Fatal(err)
	}
	if !hasConsole() {
		t.Fatal("the app has not joined the hidden console")
	}
	if consoleVisible(t) {
		t.Fatal("a program the app starts shows a console window")
	}
}
