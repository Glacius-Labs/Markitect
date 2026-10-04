package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

func TestInitCLIPlansThenCreatesProjectWithoutInventingVerification(t *testing.T) {
	root := t.TempDir()
	args := []string{"init", "--repo", root, "--name", "example", "--namespace", "engineering", "--path", "docs/engineering"}
	code, output, stderr := invoke(args...)
	if code != 0 || stderr != "" {
		t.Fatalf("preview: code=%d stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[struct {
		Status string       `yaml:"status"`
		Plan   host.InitPlan `yaml:"plan"`
	}](t, output)
	if result.Status != "planned" || result.Plan.Applied || len(result.Plan.Files) != 2 {
		t.Fatalf("unexpected plan: %#v", result)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preview changed destination: %v, %v", entries, err)
	}
	code, repeated, stderr := invoke(args...)
	if code != 0 || stderr != "" || repeated != output {
		t.Fatalf("preview is not deterministic: %d %s %s", code, repeated, stderr)
	}
	git(t, root, "init", "-b", "feature/authoring")
	git(t, root, "config", "user.name", "Markitect Test")
	git(t, root, "config", "user.email", "markitect-test@example.invalid")
	code, output, stderr = invoke(append(args, "--write")...)
	if code != 0 || stderr != "" || !strings.Contains(output, "status: initialized") {
		t.Fatalf("write: %d %s %s", code, output, stderr)
	}
	for _, file := range result.Plan.Files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil || string(data) != file.Text {
			t.Fatalf("written file differs from preview %s: %v", file.Path, err)
		}
	}
	code, output, stderr = invoke("check", "--repo", root)
	if code != 0 || decodeYAML[report](t, output).Status != "passed" {
		t.Fatalf("initialized Project is not structurally valid: %d %s %s", code, output, stderr)
	}
	git(t, root, "add", "markitect.yaml", "docs/engineering/README.md")
	git(t, root, "commit", "-m", "initialize project")
	revision := git(t, root, "rev-parse", "HEAD")
	code, output, stderr = invoke("verify", "--repo", root, "--revision", revision)
	if code != 2 || decodeYAML[report](t, output).Status != "incomplete" || !strings.Contains(output, "incomplete-evidence") {
		t.Fatalf("empty Project claimed verification: %d %s %s", code, output, stderr)
	}
}

func TestInitCLIRequiresExplicitOptionsAndRejectsUnrelatedFlags(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "--repo", root},
		{"init", "--repo", root, "--name", "example", "--path", "docs/engineering"},
		{"init", "--repo", root, "--namespace", "engineering", "--path", "docs/engineering"},
		{"init", "--revision", "HEAD"},
		{"init", "--check"},
		{"init", "--kind", "Rule"},
		{"check", "--path", "docs/engineering"},
	} {
		code, _, stderr := invoke(args...)
		if code != 2 || stderr == "" {
			t.Fatalf("invalid arguments %v: %d %s", args, code, stderr)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid invocation changed destination: %v %v", entries, err)
	}
	code, output, stderr := invoke("help", "init")
	if code != 0 || stderr != "" || !strings.Contains(output, "--path") || strings.Contains(output, "--revision") {
		t.Fatalf("init help: %d %s %s", code, output, stderr)
	}
}

func TestInitCLIDefaultsAreaPathWhenOmitted(t *testing.T) {
	root := t.TempDir()
	code, output, stderr := invoke("init", "--repo", root, "--name", "example", "--namespace", "engineering")
	if code != 0 || stderr != "" {
		t.Fatalf("default-path preview: code=%d stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[struct {
		Status string       `yaml:"status"`
		Plan   host.InitPlan `yaml:"plan"`
	}](t, output)
	if result.Status != "planned" || result.Plan.Area.Path != ".markitect/areas/engineering" {
		t.Fatalf("unexpected default-path plan: %#v", result)
	}
	if len(result.Plan.Files) != 2 || result.Plan.Files[0].Path != ".markitect/areas/engineering/README.md" {
		t.Fatalf("unexpected default-path files: %#v", result.Plan.Files)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("default-path preview changed destination: %v, %v", entries, err)
	}
}
