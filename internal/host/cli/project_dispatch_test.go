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

func TestRootHelpDistinguishesProjectModelFromHistoricalAuthoring(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 {
		t.Fatalf("root help = %d: %s", code, errout.String())
	}
	for _, required := range []string{".markitect/project.yaml", "markitect project init", "markitect project schema", "Top-level 'markitect init' creates the historical Project/Domain format"} {
		if !strings.Contains(out.String(), required) {
			t.Errorf("root help missing %q", required)
		}
	}
}
