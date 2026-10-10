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

func TestRootHelpPromotesTheModelFirstProjectWorkflow(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 {
		t.Fatalf("root help = %d: %s", code, errout.String())
	}
	for _, required := range []string{".markitect/project.yaml", "markitect project init", "markitect project schema"} {
		if !strings.Contains(out.String(), required) {
			t.Errorf("root help missing %q", required)
		}
	}
	if strings.Contains(out.String(), "Top-level 'markitect init'") || strings.Contains(out.String(), "  init,") {
		t.Errorf("root help exposes the removed historical initializer: %s", out.String())
	}
	if code := Run([]string{"init", "--help"}, &out, &errout); code != 2 || !strings.Contains(errout.String(), `unknown command "init"`) {
		t.Fatalf("top-level init = %d, stderr %q", code, errout.String())
	}
}
