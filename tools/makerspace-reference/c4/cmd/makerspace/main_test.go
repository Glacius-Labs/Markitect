package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgramSession(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "makerspace.exe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	in := `{"id":"1","op":"listMachines","args":{"actor":"admin","at":"2026-03-02T09:00Z"}}` + "\n" +
		`garbage` + "\n" +
		`{"id":"3","op":"listMachines","args":{"actor":"admin","at":"2026-03-02T09:00Z"}}` // no final line feed
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("exit: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], `"id":"1"`) || !strings.Contains(lines[1], "invalidRequest") ||
		!strings.Contains(lines[2], `"id":"3"`) || stderr.Len() != 0 {
		t.Fatalf("stdout %q stderr %q", stdout.String(), stderr.String())
	}
}
