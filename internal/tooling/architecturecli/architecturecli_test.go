package architecturecli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunArgumentAndRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown flag", args: []string{"--unknown"}, want: "flag provided but not defined"},
		{name: "positional argument", args: []string{"extra"}, want: "unexpected positional arguments"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out, errout bytes.Buffer
			if got := Run(test.args, &out, &errout); got != 2 {
				t.Fatalf("exit = %d, want 2", got)
			}
			if out.Len() != 0 || !strings.Contains(errout.String(), test.want) {
				t.Fatalf("stdout = %q, stderr = %q; want error containing %q", out.String(), errout.String(), test.want)
			}
		})
	}

	var out, errout bytes.Buffer
	if got := Run([]string{"--repo", t.TempDir()}, &out, &errout); got != 2 {
		t.Fatalf("exit = %d, want 2", got)
	}
	if out.Len() != 0 || !strings.Contains(errout.String(), "codemap.yaml") {
		t.Fatalf("stdout = %q, stderr = %q; want missing code map error", out.String(), errout.String())
	}
}

func TestRunPassesCleanRepository(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# Fixture\n")
	write("internal/model/core.go", "package core\n")
	write("internal/tooling/architecture/codemap.yaml", `packages:
  - path: internal/model
    layer: apply-model
    purpose: Core fixture.
    doc: README.md
`)

	var out, errout bytes.Buffer
	if got := Run([]string{"--repo", root}, &out, &errout); got != 0 {
		t.Fatalf("exit = %d, want 0; stdout = %q, stderr = %q", got, out.String(), errout.String())
	}
	if errout.Len() != 0 || out.String() != "architecture imports: passed\n" {
		t.Fatalf("stdout = %q, stderr = %q; want successful architecture report", out.String(), errout.String())
	}
}

func TestRunReportsArchitectureViolations(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# Fixture\n")
	write("internal/model/core.go", "package core\nimport _ \"github.com/Glacius-Labs/Markitect/internal/apply\"\n")
	write("internal/apply/host.go", "package host\n")
	write("internal/tooling/architecture/codemap.yaml", `packages:
  - path: internal/model
    layer: apply-model
    purpose: Core fixture.
    doc: README.md
  - path: internal/apply
    layer: apply-usecase
    purpose: Host fixture.
    doc: README.md
`)

	var out, errout bytes.Buffer
	if got := Run([]string{"--repo", root}, &out, &errout); got != 1 {
		t.Fatalf("exit = %d, want 1; stderr = %q", got, errout.String())
	}
	if errout.Len() != 0 || !strings.Contains(out.String(), "responsibility direction forbids") {
		t.Fatalf("stdout = %q, stderr = %q; want architecture violation", out.String(), errout.String())
	}
}
