package cli

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
)

// markitect help names every flag of each command, so that a session that
// reads the help finds each command the skills and guides tell it to run. The
// flags are read from the commands themselves, as -h prints them, so a new
// flag fails here until the help names it. config and mark print usages of
// their own.
func TestHelpNamesEveryFlag(t *testing.T) {
	flag := regexp.MustCompile(`(?m)^  -(\w+)`)
	for _, cmd := range []string{"update", "init", "tidy", "check", "vet", "list", "doc", "status", "version"} {
		var out, errOut bytes.Buffer
		if code := (CLI{Stdout: &out, Stderr: &errOut}).Run(context.Background(), []string{cmd, "-h"}); code != Misused {
			t.Fatalf("%s -h: exit %d", cmd, code)
		}
		flags := flag.FindAllStringSubmatch(errOut.String(), -1)
		if len(flags) == 0 {
			t.Fatalf("%s -h prints no flag:\n%s", cmd, errOut.String())
		}
		usage := helpOf(cmd)
		if usage == "" {
			t.Fatalf("the help has no line for %s", cmd)
		}
		for _, f := range flags {
			if !strings.Contains(usage, "--"+f[1]) {
				t.Errorf("the help of %s does not name --%s:\n%s", cmd, f[1], usage)
			}
		}
	}
}

// helpOf returns the lines of the help that are about one command: each line
// that starts with it, with the lines that continue it.
func helpOf(cmd string) string {
	var b strings.Builder
	in := false
	for _, line := range strings.Split(usageTop, "\n") {
		switch {
		case strings.HasPrefix(line, "  markitect "+cmd+" ") || line == "  markitect "+cmd:
			in = true
		case strings.HasPrefix(line, "  markitect ") || !strings.HasPrefix(line, "    "):
			in = false
		}
		if in {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
