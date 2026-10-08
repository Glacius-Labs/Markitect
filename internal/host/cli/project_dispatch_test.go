package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProjectFacadeDispatchesNestedHelp(t *testing.T) {
	for _, args := range [][]string{{"project", "init", "--help"}, {"help", "project"}} {
		var out, errout bytes.Buffer
		if code := Run(args, &out, &errout); code != 0 {
			t.Fatalf("Run(%q) = %d: %s", args, code, errout.String())
		}
		if !strings.Contains(out.String(), "markitect project") {
			t.Fatalf("nested help missing: %s", out.String())
		}
	}
}
