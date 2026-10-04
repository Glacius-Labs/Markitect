package cli

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

func TestPackUsesFixedGitRevisionAndRefusesOverwrite(t *testing.T) {
	repo := newPackageSourceRepo(t)
	archivePath := filepath.Join(t.TempDir(), "review-guidance.zip")
	args := []string{"pack", "--repo", repo.root, "--revision", repo.base, "--output", archivePath}
	code, output, stderr := invoke(args...)
	if code != 0 {
		t.Fatalf("pack exit=%d stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[struct {
		Status       string               `yaml:"status"`
		SourceCommit string               `yaml:"sourceCommit"`
		Output       string               `yaml:"output"`
		Pin          authoring.PackagePin `yaml:"pin"`
	}](t, output)
	if result.Status != "packed" || result.SourceCommit != repo.base || result.Output != archivePath {
		t.Fatalf("pack did not identify the fixed commit and output: %#v", result)
	}
	if result.Pin.Name != "review-guidance" || result.Pin.Version != "1.0.0" || len(result.Pin.SHA256) != 64 {
		t.Fatalf("pack returned an invalid exact pin suggestion: %#v", result.Pin)
	}

	before, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	code, _, stderr = invoke(args...)
	if code != 2 || !strings.Contains(stderr, "file exists") {
		t.Fatalf("overwrite attempt exit=%d stderr=%s, want refusal", code, stderr)
	}
	after, err := os.ReadFile(archivePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("refused pack changed the existing archive: err=%v", err)
	}
}

func TestPackageConsumerCLIContextAndArchiveIntegrity(t *testing.T) {
	root := packageConsumerExample(t)
	code, output, stderr := invoke("check", "--repo", root)
	if code != 0 {
		t.Fatalf("unchanged consumer check exit=%d stderr=%s output=%s", code, stderr, output)
	}
	if result := decodeYAML[report](t, output); result.Status != "passed" {
		t.Fatalf("unchanged package consumer status = %q, want passed", result.Status)
	}

	code, output, stderr = invoke("context", "--repo", root, "--package", "review-guidance", "--namespace", "review", "--kind", "Workflow", "--name", "review-change")
	if code != 0 {
		t.Fatalf("package context exit=%d stderr=%s output=%s", code, stderr, output)
	}
	packageContext := decodeYAML[host.Context](t, output)
	if packageContext.Entry != "review-guidance::review/Workflow/review-change" {
		t.Fatalf("package context entry = %q", packageContext.Entry)
	}
	for _, want := range []string{
		"review-guidance::review/Skill/evidence-review",
		"review-guidance::review/Text/review-principles",
		"package:review-guidance/file:.markitect/areas/review/review-evidence.md",
	} {
		found := false
		for _, input := range packageContext.Inputs {
			if input.Key == want {
				found = true
				if input.Package != "review-guidance" || input.PackageVersion != "1.0.0" {
					t.Errorf("package input %s lost its origin/version: %#v", want, input)
				}
			}
		}
		if !found {
			t.Errorf("package context omitted %s", want)
		}
	}

	code, output, stderr = invoke("context", "--repo", root, "--namespace", "consumer", "--kind", "Skill", "--name", "local-review-entry")
	if code != 0 {
		t.Fatalf("local wrapper context exit=%d stderr=%s output=%s", code, stderr, output)
	}
	localContext := decodeYAML[host.Context](t, output)
	if localContext.Entry != "consumer/Skill/local-review-entry" {
		t.Fatalf("local wrapper context entry = %q", localContext.Entry)
	}
	foundPackageWorkflow := false
	for _, input := range localContext.Inputs {
		if input.Key == "review-guidance::review/Workflow/review-change" {
			foundPackageWorkflow = input.PackageVersion == "1.0.0"
		}
	}
	if !foundPackageWorkflow {
		t.Fatalf("local wrapper context omitted its versioned package dependency: %#v", localContext.Inputs)
	}

	project, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := host.GenerateOutputs(project)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := outputs[".agents/skills/local-review-entry/SKILL.md"]; !ok {
		t.Fatal("consumer-owned wrapper did not receive its local provider view")
	}
	if _, ok := outputs[".agents/skills/evidence-review/SKILL.md"]; ok {
		t.Fatal("imported Skill received a consumer-owned provider view")
	}

	changedRoot := filepath.Join(t.TempDir(), "consumer")
	copyPackageTestTree(t, root, changedRoot)
	archive := filepath.Join(changedRoot, filepath.FromSlash(".markitect/packages/review-guidance-1.0.0.zip"))
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, 0)
	if err := os.WriteFile(archive, data, 0644); err != nil {
		t.Fatal(err)
	}
	code, output, stderr = invoke("check", "--repo", changedRoot)
	if code != 2 || !strings.Contains(stderr, "archive sha256 does not match pin") {
		t.Fatalf("changed archive exit=%d stderr=%s output=%s, want fail-closed digest error", code, stderr, output)
	}
}

func packageExampleRoot(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(projectRoot(t), "examples", name)
}

func newPackageSourceRepo(t *testing.T) cliRepo {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	copyPackageTestTree(t, packageExampleRoot(t, "content-package"), root)
	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.name", "Markitect Test")
	git(t, root, "config", "user.email", "markitect-test@example.invalid")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "package fixture")
	return cliRepo{root: root, base: git(t, root, "rev-parse", "HEAD")}
}

func packageConsumerExample(t *testing.T) string {
	t.Helper()
	return packageExampleRoot(t, "package-consumer")
}

func copyPackageTestTree(t *testing.T, sourceRoot, destinationRoot string) {
	t.Helper()
	err := filepath.WalkDir(sourceRoot, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, file)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
