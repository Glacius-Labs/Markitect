package cli

import (
	"strings"
	"testing"
)

func TestLegacyHelpStillCoversOwnedCommands(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}, {"help", "schema"}, {"schema", "--help"}, {"authoring", "--help"}} {
		code, out, stderr := invoke(args...)
		if code != 0 || !strings.Contains(out, "usage:") || stderr != "" {
			t.Fatalf("help %v: code=%d stdout=%q stderr=%q", args, code, out, stderr)
		}
	}
	for _, args := range [][]string{{"help", "missing"}, {"missing-command"}, {"migrate"}, {"check", "--profile", "generic"}} {
		if code, _, _ := invoke(args...); code != 2 {
			t.Fatalf("%v returned %d, want 2", args, code)
		}
	}
}

func TestLocalReleaseCommandsAreNotLegacyDispatcherRoutes(t *testing.T) {
	for _, command := range []string{"package", "bundle", "install", "version", "licenses"} {
		t.Run(command, func(t *testing.T) {
			code, _, stderr := invoke(command)
			if code != 2 || !strings.Contains(stderr, "unknown command") {
				t.Fatalf("legacy dispatcher accepted %s: code=%d stderr=%q", command, code, stderr)
			}
		})
	}
}
