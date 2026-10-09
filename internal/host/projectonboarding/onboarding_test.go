package projectonboarding

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"go.yaml.in/yaml/v3"
)

func onboardingRepo(t *testing.T) (string, *projectwork.Project) {
	t.Helper()
	root := t.TempDir()
	commands := [][]string{{"init", "-b", "codex/onboarding-test"}, {"config", "user.name", "Onboarding Test"}, {"config", "user.email", "onboarding@example.invalid"}}
	for _, args := range commands {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Example\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "initial"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if _, err := projectwork.Init(root, "onboarding-fixture", true); err != nil {
		t.Fatalf("init fixture project: %v", err)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load fixture project: %v", err)
	}
	return root, project
}

func defaultOptions(providers ...Provider) Options {
	return Options{Providers: providers}
}

func fileFor(t *testing.T, plan Plan, target string) FileChange {
	t.Helper()
	for _, file := range plan.Files {
		if file.Path == target {
			return file
		}
	}
	t.Fatalf("plan did not include %s", target)
	return FileChange{}
}

func TestPreviewAndApplyCodexAndClaude(t *testing.T) {
	root, project := onboardingRepo(t)
	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Claude, Codex))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest == "" || plan.Options.DocumentationPath != "docs/markitect/project.md" {
		t.Fatalf("plan did not bind digest and configured documentation path: %+v", plan)
	}
	want := []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md", ".markitect/workflows/model-first.md", "AGENTS.md", "CLAUDE.md"}
	if len(plan.Files) != len(want) {
		t.Fatalf("plan files = %v, want %v", plan.Files, want)
	}
	for i, target := range want {
		if plan.Files[i].Path != target || plan.Files[i].Action != "create" {
			t.Fatalf("plan file[%d] = %+v, want create %s", i, plan.Files[i], target)
		}
		if !strings.Contains(plan.Files[i].Content, workflowPath) && target != workflowPath {
			t.Fatalf("%s does not point to the canonical workflow", target)
		}
	}
	if _, err := Apply(root, plan, "wrong-digest"); err == nil {
		t.Fatal("Apply accepted a different expected digest")
	}
	written, err := Apply(root, plan, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Written) != len(want) {
		t.Fatalf("written paths = %v, want %v", written.Written, want)
	}
	for _, skillPath := range []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md"} {
		skill, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(skillPath)))
		if readErr != nil {
			t.Fatalf("read installed skill %s: %v", skillPath, readErr)
		}
		parts := strings.SplitN(string(skill), "---\n", 3)
		if len(parts) != 3 || parts[0] != "" {
			t.Fatalf("installed skill %s has malformed frontmatter", skillPath)
		}
		var metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil || metadata.Name != "markitect-model-first" || metadata.Description == "" {
			t.Fatalf("installed skill %s has invalid frontmatter: metadata=%+v err=%v", skillPath, metadata, err)
		}
		if !strings.Contains(parts[2], beginMarker) || !strings.Contains(parts[2], "repository-root .markitect/workflows/model-first.md") {
			t.Fatalf("installed skill %s lacks managed workflow instructions", skillPath)
		}
	}
	content, err := os.ReadFile(filepath.Join(root, workflowPath))
	if err != nil || !bytes.Contains(content, []byte("ordinary language")) || !bytes.Contains(content, []byte("ordinary repository write access")) {
		t.Fatalf("canonical workflow content missing required guidance: %v", err)
	}
}

func TestPreviewMergesNativeFilesWithoutChangingBytesOutsideManagedBlock(t *testing.T) {
	root, project := onboardingRepo(t)
	prefix := "# Existing repository guidance\r\nKeep this byte sequence.\r\n"
	suffix := "\r\n## Local policy\r\nKeep this too.\r\n"
	oldBlock := beginMarker + "\r\nold generated text\r\n" + endMarker
	original := prefix + oldBlock + suffix
	originalMode := os.FileMode(0644)
	if runtime.GOOS != "windows" {
		originalMode = 0755
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(original), originalMode); err != nil {
		t.Fatal(err)
	}
	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatal(err)
	}
	merged := fileFor(t, plan, "AGENTS.md").Content
	if !strings.HasPrefix(merged, prefix) || !strings.HasSuffix(merged, suffix) {
		t.Fatalf("preview changed bytes outside managed block:\n%s", merged)
	}
	if strings.Contains(merged, "old generated text") || !strings.Contains(merged, "ordinary conversation") {
		t.Fatalf("preview did not replace only the managed block:\n%s", merged)
	}
	if _, err := Apply(root, plan, plan.Digest); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(actual, []byte(prefix)) || !bytes.HasSuffix(actual, []byte(suffix)) {
		t.Fatal("write changed bytes outside the managed block")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != originalMode.Perm() {
			t.Fatalf("write changed AGENTS.md mode from %04o to %04o", originalMode.Perm(), info.Mode().Perm())
		}
	}
}

func TestMalformedManagedBlockRefusesPreviewWithoutWriting(t *testing.T) {
	root, project := onboardingRepo(t)
	original := "# Existing\n" + beginMarker + "\nunterminated\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex)); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("Preview error = %v, want malformed block refusal", err)
	}
	actual, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || string(actual) != original {
		t.Fatalf("malformed file changed: %v", err)
	}
}

func TestApplyRejectsStaleModelAndConfiguration(t *testing.T) {
	t.Run("model", func(t *testing.T) {
		root, project := onboardingRepo(t)
		plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
		if err != nil {
			t.Fatal(err)
		}
		modelPath := filepath.Join(root, filepath.FromSlash(projectwork.ModelRoot+"/manager.yaml"))
		model, err := os.ReadFile(modelPath)
		if err != nil {
			t.Fatal(err)
		}
		updated := bytes.Replace(model, []byte("repository-wide engineering mandate"), []byte("updated engineering mandate"), 1)
		if bytes.Equal(model, updated) {
			t.Fatal("fixture Manager purpose was not found")
		}
		if err := os.WriteFile(modelPath, updated, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Apply(root, plan, plan.Digest); err == nil || !strings.Contains(err.Error(), "digest") {
			t.Fatalf("Apply error = %v, want stale model digest refusal", err)
		}
	})
	t.Run("document destination option", func(t *testing.T) {
		root, project := onboardingRepo(t)
		plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Preview(root, project.Report.ModelDigest, Options{Providers: []Provider{Codex}, DocumentationPath: "docs/other.md"}); err == nil || !strings.Contains(err.Error(), "differs from the canonical project destination") {
			t.Fatalf("Preview error = %v, want destination/config mismatch", err)
		}
		if _, err := Apply(root, plan, plan.Digest); err != nil {
			t.Fatalf("unchanged canonical config should preserve preview: %v", err)
		}
	})
	t.Run("changed project config", func(t *testing.T) {
		root, project := onboardingRepo(t)
		plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
		if err != nil {
			t.Fatal(err)
		}
		manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
		manifest, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		updated := bytes.Replace(manifest, []byte("docs/markitect/project.md"), []byte("docs/markitect/changed.md"), 1)
		if bytes.Equal(manifest, updated) {
			t.Fatal("fixture documentPath was not found")
		}
		if err := os.WriteFile(manifestPath, updated, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Apply(root, plan, plan.Digest); err == nil {
			t.Fatal("Apply accepted a changed canonical project configuration")
		}
	})
}

func TestPreviewRejectsUnsafeOrUnconfiguredDocumentationDestinations(t *testing.T) {
	root, project := onboardingRepo(t)
	for _, destination := range []string{"../outside.md", ".markitect/leak.md", "docs\\escape.md", "docs/no-extension.txt", "docs/other.md"} {
		_, err := Preview(root, project.Report.ModelDigest, Options{Providers: []Provider{Codex}, DocumentationPath: destination})
		if err == nil {
			t.Errorf("Preview accepted documentation destination %q", destination)
		}
	}
}

func TestOnboardingUsesAndPreservesLegacyHiddenDocumentationDestination(t *testing.T) {
	root, _ := onboardingRepo(t)
	manifestPath := filepath.Join(root, projectwork.ManifestPath)
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest = bytes.Replace(manifest, []byte("documentPath: docs/markitect/project.md\n"), nil, 1)
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load legacy project without documentPath: %v", err)
	}
	if got := projectwork.DocumentPath(project.Config); got != projectwork.ViewPath {
		t.Fatalf("legacy document destination = %q, want %q", got, projectwork.ViewPath)
	}
	view, err := projectwork.Document(project, false)
	if err != nil {
		t.Fatalf("render legacy view: %v", err)
	}
	legacyViewPath := filepath.Join(root, filepath.FromSlash(projectwork.ViewPath))
	if err := os.MkdirAll(filepath.Dir(legacyViewPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyViewPath, []byte(view), 0644); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	docsPath := filepath.Join(root, "docs/markitect/project.md")
	docsBefore, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatalf("preview onboarding for legacy project: %v", err)
	}
	if plan.Options.DocumentationPath != projectwork.ViewPath {
		t.Fatalf("onboarding destination = %q, want legacy %q", plan.Options.DocumentationPath, projectwork.ViewPath)
	}
	if !strings.Contains(fileFor(t, plan, workflowPath).Content, projectwork.ViewPath) {
		t.Fatal("shared workflow does not retain the legacy documentation destination")
	}
	if _, err := Apply(root, plan, plan.Digest); err != nil {
		t.Fatalf("apply onboarding for legacy project: %v", err)
	}
	manifestAfter, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Fatal("onboarding changed the legacy project configuration")
	}
	docsAfter, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(docsBefore, docsAfter) {
		t.Fatal("onboarding rewrote the unrelated configured documentation destination")
	}
}

func TestGeneratedSkillsHaveDiscoverableFrontmatterAndRootBasedWorkflowPointer(t *testing.T) {
	for _, provider := range []Provider{Codex, Claude} {
		files, err := renderFiles(Options{Providers: []Provider{provider}, DocumentationPath: "docs/markitect/project.md"})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if !strings.HasSuffix(file.Path, "/SKILL.md") {
				continue
			}
			parts := strings.SplitN(file.Content, "---\n", 3)
			if len(parts) != 3 || parts[0] != "" {
				t.Fatalf("%s has malformed SKILL.md frontmatter", file.Path)
			}
			var metadata struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
				t.Fatalf("%s frontmatter: %v", file.Path, err)
			}
			if metadata.Name != "markitect-model-first" || metadata.Description == "" {
				t.Fatalf("%s has incomplete skill metadata: %+v", file.Path, metadata)
			}
			if !strings.Contains(parts[2], "repository-root .markitect/workflows/model-first.md") || strings.Contains(parts[2], "](.markitect/workflows/") {
				t.Fatalf("%s does not point to the canonical workflow from the repository root", file.Path)
			}
		}
	}
}
