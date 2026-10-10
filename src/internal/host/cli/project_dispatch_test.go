package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The legacy tree no longer dispatches the product verbs; it points to them.
func TestLegacyTreeHasNoProjectNoun(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"project", "init", "--help"}, &out, &errout); code != 2 || !strings.Contains(errout.String(), `unknown command "project"`) {
		t.Fatalf("project = %d, stderr %q", code, errout.String())
	}
	out.Reset()
	if code := Run([]string{"--help"}, &out, &errout); code != 0 {
		t.Fatalf("root help = %d: %s", code, errout.String())
	}
	for _, required := range []string{"legacy Project/Domain", "'markitect init'", "'markitect schema'"} {
		if !strings.Contains(out.String(), required) {
			t.Errorf("root help missing %q: %s", required, out.String())
		}
	}
	if code := Run([]string{"init", "--help"}, &out, &errout); code != 2 || !strings.Contains(errout.String(), `unknown command "init"`) {
		t.Fatalf("top-level init = %d, stderr %q", code, errout.String())
	}
}
