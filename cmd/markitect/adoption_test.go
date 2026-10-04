package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdoptionCLIHasSeparateExplicitReadAndWriteBoundaries(t *testing.T) {
	for _, args := range [][]string{
		{"prepare", "--write"},
		{"prepare", "--scope", "scope.yaml", "--output", "workspace", "--write"},
		{"prepare", "--repo", "."},
		{"prepare", "--scope", "scope.yaml", "--output", "workspace", "--expect", strings.Repeat("a", 64)},
		{"copy-me", "--workspace", "workspace", "--queue", "queue.yaml", "--write"},
		{"copy-me", "--revision", strings.Repeat("a", 40)},
		{"init", "--scope", "scope.yaml"},
	} {
		var out, errout bytes.Buffer
		if code := run(args, &out, &errout); code != 2 || errout.Len() == 0 {
			t.Fatalf("must refuse %v without acquisition/writes: exit %d %s", args, code, out.String())
		}
	}
	for _, command := range []string{"prepare", "copy-me"} {
		var out, errout bytes.Buffer
		if code := run([]string{command, "--help"}, &out, &errout); code != 0 {
			t.Fatalf("help: %s", errout.String())
		}
		if strings.Contains(out.String(), "--repo") || strings.Contains(out.String(), "--revision") {
			t.Fatalf("adoption command exposes ambient project acquisition: %s", out.String())
		}
	}
}
