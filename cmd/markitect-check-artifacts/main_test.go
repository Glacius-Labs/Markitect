package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReportsPassFindingsAndInvalidInvocation(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "markitect.yaml", "apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata:\n  name: sample\nspec: {}\n")
	writeFixture(t, root, "managed/tool.txt", "explicitly owned\n")
	writeFixture(t, root, "meta/coverage.yaml", `apiVersion: markitect.example.org/artifact-coverage/v1alpha1
kind: ArtifactCoverage
spec:
  roots: [managed, meta]
  tooling:
    - path: managed/tool.txt
      owner: helper tool
    - path: meta/coverage.yaml
      owner: coverage policy
`)
	args := []string{"--repo", root, "--config", "meta/coverage.yaml"}
	var out, errout bytes.Buffer
	if code := run(args, &out, &errout); code != 0 || !strings.Contains(out.String(), "status: passed") {
		t.Fatalf("passing check: code=%d stdout=%s stderr=%s", code, out.String(), errout.String())
	}
	writeFixture(t, root, "managed/unowned.txt", "new file\n")
	out.Reset()
	errout.Reset()
	if code := run(args, &out, &errout); code != 1 || !strings.Contains(out.String(), "unmanaged") {
		t.Fatalf("coverage violation: code=%d stdout=%s stderr=%s", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := run([]string{"--unknown"}, &out, &errout); code != 2 || errout.Len() == 0 {
		t.Fatalf("invalid invocation: code=%d stdout=%s stderr=%s", code, out.String(), errout.String())
	}
}

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
