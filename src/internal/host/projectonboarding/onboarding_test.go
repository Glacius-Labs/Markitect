package projectonboarding

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
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
	if len(plan.Files) != 45 {
		t.Fatalf("plan has %d files, want one shared workflow, two entrypoints, ten skills and guides per provider plus recovery references", len(plan.Files))
	}
	if len(plan.Targets) != 47 {
		t.Fatalf("plan guards %d targets, want generated outputs plus two optional legacy migration paths", len(plan.Targets))
	}
	for _, file := range plan.Files {
		if file.Action != "create" {
			t.Fatalf("plan action for %s = %q, want create", file.Path, file.Action)
		}
	}
	if _, err := Apply(root, plan, "wrong-digest"); err == nil {
		t.Fatal("Apply accepted a different expected digest")
	}
	written, err := Apply(root, plan, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Written) != len(plan.Files) {
		t.Fatalf("written %d paths, want %d", len(written.Written), len(plan.Files))
	}
	for _, skillPath := range staleSkillPaths([]Provider{Codex, Claude}) {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(skillPath))); !os.IsNotExist(err) {
			t.Fatalf("obsolete router %s was installed (stat err %v)", skillPath, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(root, workflowPath))
	if err != nil || !bytes.Contains(content, []byte("ordinary Work Item")) || !bytes.Contains(content, []byte("ordinary repository write access")) {
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
	if strings.Contains(merged, "old generated text") || !strings.Contains(merged, "markitect-implement") {
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

func TestPreviewDeletesOnlyExactManagedLegacyRouterAndIsIdempotent(t *testing.T) {
	root, project := onboardingRepo(t)
	paths := staleSkillPaths([]Provider{Codex, Claude})
	for _, path := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(previouslyGeneratedRouter()), 0644); err != nil {
			t.Fatal(err)
		}
	}

	options := defaultOptions(Claude, Codex)
	plan, err := Preview(root, project.Report.ModelDigest, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if got := fileFor(t, plan, path); got.Action != "delete" || got.Content != "" {
			t.Errorf("legacy target %s = %+v, want a guarded deletion", path, got)
		}
	}
	for _, target := range plan.Targets {
		if strings.Contains(target.Path, "markitect-model-first") && (!target.Exists || target.ContentHash == "") {
			t.Errorf("legacy deletion target was not bound to observed bytes: %+v", target)
		}
	}
	if _, err := Apply(root, plan, plan.Digest); err != nil {
		t.Fatal(err)
	}
	second, err := Preview(root, project.Report.ModelDigest, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Errorf("legacy router %s remains after migration (stat err %v)", path, err)
		}
		for _, file := range second.Files {
			if file.Path == path {
				t.Errorf("second preview regenerated obsolete path %s", path)
			}
		}
	}
	for _, file := range second.Files {
		if file.Action != "unchanged" {
			t.Errorf("second preview action for %s = %q, want unchanged", file.Path, file.Action)
		}
	}
}

func TestPreviewPreservesCustomizedLegacyRouterAsConflict(t *testing.T) {
	root, project := onboardingRepo(t)
	path := staleSkillPaths([]Provider{Codex})[0]
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	original := previouslyGeneratedRouter() + "\n## Local instructions\nKeep me.\n"
	if err := os.WriteFile(fullPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex)); err == nil || !strings.Contains(err.Error(), "custom or mixed content") {
		t.Fatalf("customized same-name router was not rejected explicitly: %v", err)
	}
	actual, err := os.ReadFile(fullPath)
	if err != nil || string(actual) != original {
		t.Fatalf("customized router changed after conflict: err=%v content=%q", err, actual)
	}
}

func TestPreviewProtectsConfiguredDocumentAtLegacySkillPath(t *testing.T) {
	for _, test := range []struct {
		provider Provider
		path     string
	}{
		{provider: Codex, path: ".agents/skills/markitect-model-first/SKILL.md"},
		{provider: Claude, path: ".claude/skills/markitect-model-first/SKILL.md"},
	} {
		t.Run(string(test.provider), func(t *testing.T) {
			root, _ := onboardingRepo(t)
			manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
			manifest, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(string(manifest), "documentPath: "+projectwork.DefaultDocumentPath, "documentPath: "+test.path, 1)
			if updated == string(manifest) {
				t.Fatal("fixture did not contain its default documentPath")
			}
			if err := os.WriteFile(manifestPath, []byte(updated), 0644); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, filepath.FromSlash(test.path))
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				t.Fatal(err)
			}
			original := previouslyGeneratedRouter()
			if err := os.WriteFile(target, []byte(original), 0644); err != nil {
				t.Fatal(err)
			}
			project, err := projectwork.Load(root, "")
			if err != nil {
				t.Fatalf("load configured document collision fixture: %v", err)
			}
			if _, err := Preview(root, project.Report.ModelDigest, defaultOptions(test.provider)); err == nil || !strings.Contains(err.Error(), "legacy skill migration target") {
				t.Fatalf("Preview did not reject configured document/migration overlap: %v", err)
			}
			actual, err := os.ReadFile(target)
			if err != nil || string(actual) != original {
				t.Fatalf("configured project document changed during rejected onboarding: err=%v", err)
			}
		})
	}
}

func TestApplyRejectsChangedLegacyRouterAfterPreview(t *testing.T) {
	root, project := onboardingRepo(t)
	path := staleSkillPaths([]Provider{Codex})[0]
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(previouslyGeneratedRouter()), 0644); err != nil {
		t.Fatal(err)
	}
	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatal(err)
	}
	changed := previouslyGeneratedRouter() + "\nlocal bytes\n"
	if err := os.WriteFile(fullPath, []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, plan, plan.Digest); err == nil {
		t.Fatal("Apply removed legacy path changed after preview")
	}
	actual, err := os.ReadFile(fullPath)
	if err != nil || string(actual) != changed {
		t.Fatalf("stale deletion changed custom bytes: err=%v content=%q", err, actual)
	}
}

func TestPreviewRefusesUnmanagedSameNameSkillCollision(t *testing.T) {
	root, project := onboardingRepo(t)
	path := filepath.Join(root, filepath.FromSlash(".agents/skills/markitect-init/SKILL.md"))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	original := "---\nname: markitect-init\ndescription: project-owned setup skill\n---\n# Custom setup\nKeep this skill.\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex)); err == nil || !strings.Contains(err.Error(), "without a Markitect managed block") {
		t.Fatalf("Preview error = %v, want unmanaged same-name skill collision", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != original {
		t.Fatalf("collision changed the existing skill: err=%v content=%q", err, actual)
	}
}

func TestApplyRejectsChangedGeneratedReferenceAfterPreview(t *testing.T) {
	root, project := onboardingRepo(t)
	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatal(err)
	}
	target := ".agents/skills/markitect-check/references/operating-guide.md"
	fullPath := filepath.Join(root, filepath.FromSlash(target))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	userBytes := []byte("local edit after preview\n")
	if err := os.WriteFile(fullPath, userBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, plan, plan.Digest); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("Apply error = %v, want stale preview refusal", err)
	}
	actual, err := os.ReadFile(fullPath)
	if err != nil || !bytes.Equal(actual, userBytes) {
		t.Fatalf("stale Apply overwrote the changed reference: err=%v content=%q", err, actual)
	}
}

func TestCRLFSkillFrontmatterRejectsMalformedClosingDelimiter(t *testing.T) {
	_, err := mergeFile(".agents/skills/example/SKILL.md", "---\r\nname: local\r\nbody without closing delimiter\r\n", "---\nname: generated\n---\n"+beginMarker+"\nmanaged\n"+endMarker)
	if err == nil || !strings.Contains(err.Error(), "malformed YAML frontmatter") {
		t.Fatalf("mergeFile error = %v, want malformed frontmatter refusal", err)
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

func TestGeneratedSkillsHaveDiscoverableFrontmatterAndInstalledReferences(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{Files: files}
	wantNames := []string{"markitect-init", "markitect-extract", "markitect-design", "markitect-suggest", "markitect-configure", "markitect-implement", "markitect-cleanup", "markitect-verify", "markitect-apply", "markitect-check"}
	for _, providerRoot := range []string{".agents/skills", ".claude/skills"} {
		for _, name := range wantNames {
			skillPath := providerRoot + "/" + name + "/SKILL.md"
			content := fileFor(t, plan, skillPath).Content
			parts := strings.SplitN(content, "---\n", 3)
			if len(parts) != 3 || parts[0] != "" {
				t.Fatalf("%s has malformed SKILL.md frontmatter", skillPath)
			}
			var metadata struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
				t.Fatalf("%s frontmatter: %v", skillPath, err)
			}
			if metadata.Name != name || metadata.Description == "" {
				t.Fatalf("%s has incomplete skill metadata: %+v", skillPath, metadata)
			}
			guidePath := providerRoot + "/" + name + "/references/operating-guide.md"
			guide := fileFor(t, plan, guidePath).Content
			if !strings.Contains(parts[2], "references/operating-guide.md") {
				t.Errorf("%s does not link to its focused guide", skillPath)
			}
			if strings.Contains(guide, ".markitect/workflows/") {
				t.Errorf("%s requires a control-plane file outside the installed skill path", guidePath)
			}
			if providerRoot == ".claude/skills" {
				codexPath := strings.Replace(skillPath, ".claude/skills", ".agents/skills", 1)
				if content != fileFor(t, plan, codexPath).Content {
					t.Errorf("provider skill content differs for %s", name)
				}
				codexGuidePath := strings.Replace(guidePath, ".claude/skills", ".agents/skills", 1)
				if guide != fileFor(t, plan, codexGuidePath).Content {
					t.Errorf("provider guide content differs for %s", name)
				}
			}
		}
	}
	for _, providerRoot := range []string{".agents/skills", ".claude/skills"} {
		recoveryPath := providerRoot + "/markitect-implement/references/recovery.md"
		sharedRecovery := fileFor(t, plan, recoveryPath).Content
		if !strings.Contains(sharedRecovery, "Resume the existing run") || !strings.Contains(sharedRecovery, "unknown external outcome") {
			t.Fatalf("%s is missing persisted-state boundaries", recoveryPath)
		}
		if providerRoot == ".claude/skills" {
			codexRecovery := strings.Replace(recoveryPath, ".claude/skills", ".agents/skills", 1)
			if sharedRecovery != fileFor(t, plan, codexRecovery).Content {
				t.Error("provider recovery references differ")
			}
		}
		implementGuidePath := providerRoot + "/markitect-implement/references/operating-guide.md"
		implementGuide := fileFor(t, plan, implementGuidePath).Content
		if !strings.Contains(implementGuide, "](recovery.md)") || path.Clean(path.Join(path.Dir(implementGuidePath), "recovery.md")) != recoveryPath {
			t.Fatalf("%s does not resolve the shared recovery reference", implementGuidePath)
		}
		for _, name := range []string{"markitect-cleanup", "markitect-verify", "markitect-apply"} {
			guidePath := providerRoot + "/" + name + "/references/operating-guide.md"
			guide := fileFor(t, plan, guidePath).Content
			link := "../../markitect-implement/references/recovery.md"
			if !strings.Contains(guide, "]("+link+")") || path.Clean(path.Join(path.Dir(guidePath), link)) != recoveryPath {
				t.Errorf("%s does not resolve shared recovery path %s", guidePath, recoveryPath)
			}
		}
	}
}

func TestRenderedSkillPathsMatchProjectOwnershipRegistry(t *testing.T) {
	capabilities := capabilitySkills()
	registeredNames := projectwork.NativeSkillNames()
	if len(capabilities) != len(registeredNames) {
		t.Fatalf("renderer exposes %d operation skills, project ownership registers %d", len(capabilities), len(registeredNames))
	}
	nameSet := make(map[string]bool, len(registeredNames))
	for _, name := range registeredNames {
		nameSet[name] = true
	}
	for _, capability := range capabilities {
		if !nameSet[capability.name] {
			t.Errorf("renderer skill %s is missing from project ownership registry", capability.name)
		}
	}
	for _, provider := range []Provider{Codex, Claude} {
		files, err := renderFiles(Options{Providers: []Provider{provider}, DocumentationPath: "docs/markitect/project.md"})
		if err != nil {
			t.Fatal(err)
		}
		root, providerName := ".agents/skills", "codex"
		if provider == Claude {
			root, providerName = ".claude/skills", "claude"
		}
		registeredPaths := make(map[string]bool)
		for _, target := range projectwork.NativeSkillPaths(providerName) {
			registeredPaths[target] = true
			fileFor(t, Plan{Files: files}, target)
		}
		for _, file := range files {
			if strings.HasPrefix(file.Path, root+"/") && !registeredPaths[file.Path] {
				t.Errorf("rendered native skill output %s is missing from project ownership registry", file.Path)
			}
		}
	}
}

func TestCapabilityMetadataDistinguishesRepresentativeRequests(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{Files: files}
	cases := []struct {
		name    string
		query   string
		signals []string
	}{
		{name: "markitect-init", query: "Initialize Markitect and install repository-local guidance", signals: []string{"initialize", "project", "onboarding"}},
		{name: "markitect-implement", query: "Implement this Work Item end to end and continue through required checks and application", signals: []string{"implement", "work item", "end to end"}},
		{name: "markitect-implement", query: "A compiler diagnostic appeared while implementing the accepted scoped candidate", signals: []string{"compiler", "repair"}},
		{name: "markitect-implement", query: "The run was interrupted; resume without repeating completed work", signals: []string{"resume", "interruption"}},
		{name: "markitect-check", query: "The model compiler reports an unresolved reference and missing ownership", signals: []string{"structural", "model/compiler", "ownership"}},
		{name: "markitect-extract", query: "Extract a proposed model from an existing application codebase", signals: []string{"existing", "code", "brownfield"}},
		{name: "markitect-design", query: "Change accepted intent and ownership, without implementing the proposal", signals: []string{"design", "intent", "without implementing"}},
		{name: "markitect-suggest", query: "Suggest model improvements but leave canonical files untouched", signals: []string{"recommend", "proposal-only", "unchanged"}},
		{name: "markitect-configure", query: "Change this existing project's runtime without changing its budget", signals: []string{"existing", "runtime", "budget"}},
		{name: "markitect-cleanup", query: "Refactor the code while preserving accepted behavior and public contracts", signals: []string{"refactor", "preserving accepted behavior", "public contracts"}},
		{name: "markitect-verify", query: "Verify a candidate and report drift and missing evidence", signals: []string{"assess", "candidate", "report drift"}},
		{name: "markitect-apply", query: "Apply the already verified candidate after required reviews", signals: []string{"already verified", "preflight", "required reviews"}},
	}
	for _, test := range cases {
		for _, providerRoot := range []string{".agents/skills", ".claude/skills"} {
			content := fileFor(t, plan, providerRoot+"/"+test.name+"/SKILL.md").Content
			for _, signal := range test.signals {
				if !strings.Contains(strings.ToLower(content), signal) {
					t.Errorf("%s discovery metadata lacks intent signal %q for request %q", test.name, signal, test.query)
				}
			}
		}
	}
}

func TestModelFirstWorkflowCoversShortWorkItemsReadinessAndBrownfieldAdoption(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	workflow := fileFor(t, Plan{Files: files}, workflowPath).Content
	for _, required := range []string{
		"The selected project is .markitect/project.yaml",
		"ordinary Work Item",
		"project_explore",
		"project_readiness",
		"project_edit",
		"project_brownfield",
		"project_brownfield_run",
		"project_deliver",
		"project_status",
		"committed-model policy",
		"Initial adoption never changes application source",
		"Apply does not merge, publish, or deploy",
		"Technical checks, semantic evidence, and human acceptance are distinct",
		"ordinary repository write access",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("shared workflow is missing required guidance %q", required)
		}
	}
	for _, obsolete := range []string{"Use the native CLI stages below", "brownfield-action start", "markitect project deliver --repo", "--execution-mode native-work", "helperLimit"} {
		if strings.Contains(workflow, obsolete) {
			t.Errorf("shared workflow retains obsolete CLI/worker guidance %q", obsolete)
		}
	}
}

func TestRenderedExploreRecordDecodesAndCreatesBoundPreview(t *testing.T) {
	root, project := onboardingRepo(t)
	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Manager to select")
	}
	files, err := renderFiles(Options{Providers: []Provider{Codex}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	workflow := fileFor(t, Plan{Files: files}, workflowPath).Content
	fence := strings.Repeat(string(rune(96)), 3)
	marker := "Minimal new exploration input record (pass as record to project_explore):\n\n" + fence + "json\n"
	start := strings.Index(workflow, marker)
	if start < 0 {
		t.Fatal("shared workflow is missing its minimal Explore JSON example")
	}
	start += len(marker)
	end := strings.Index(workflow[start:], "\n"+fence)
	if end < 0 {
		t.Fatal("minimal Explore JSON example has no closing code fence")
	}
	input := strings.TrimSpace(workflow[start : start+end])
	managerJSON, err := json.Marshal(project.Report.Managers[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	input = strings.Replace(input, `"<existing-manager-id>"`, string(managerJSON), 1)
	record, err := projectexplore.DecodeRecordInput([]byte(input))
	if err != nil {
		t.Fatalf("rendered minimal Explore JSON did not decode: %v\n%s", err, input)
	}
	if record.Status != projectexplore.StatusActive || record.ID != "cancel-order" || len(record.Scopes) != 1 || record.CreatedAgainst != "" {
		t.Fatalf("rendered Explore input has unexpected initial state: %#v", record)
	}

	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("read fixture HEAD: %v", err)
	}
	branch, err := exec.Command("git", "-C", root, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	if err != nil {
		t.Fatalf("read fixture branch: %v", err)
	}
	managerIDs := append([]string{}, record.Scopes[0].ManagerIDs...)
	empty := []string{}
	binding := projectexplore.Binding{
		RepositoryRoot: root, Branch: strings.TrimSpace(string(branch)), Head: strings.TrimSpace(string(head)),
		ModelRevision: project.Revision, ModelAccepted: false, AcceptancePolicy: project.Config.AcceptancePolicy,
		ProjectDigest: testDigest(project.Digest), ModelDigest: testDigest(project.Report.ModelDigest),
		SnapshotDigest: testDigest(project.Snapshot.Digest()), SelectionDigest: testDigest("selection"),
		ScopeID: record.Scopes[0].ID, ScopeName: record.Scopes[0].Name, Goal: record.Scopes[0].Goal,
		Operation: record.Scopes[0].Operation, ManagerIDs: managerIDs, ResponsibleManagerIDs: empty,
		RequiredArtifacts: []string{}, FileStructure: []string{}, Checks: []string{}, BasisFiles: []projectexplore.BasisFile{},
	}
	preview, err := projectexplore.CreatePreview(root, record, binding)
	if err != nil {
		t.Fatalf("rendered minimal Explore record did not create a bound preview: %v", err)
	}
	if preview.Digest == "" || preview.Next.CreatedAgainst != preview.BindingDigest || preview.Target.Exists {
		t.Fatalf("CreatePreview did not bind the new active record: %#v", preview)
	}
	for _, required := range []string{
		"project_explore and write:false",
		"write:true and expectedDigest",
		"returned writePlan.digest",
		"Minimal new exploration input record",
		"Technical checks, semantic evidence, and human acceptance are distinct",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("shared workflow is missing usable lifecycle guidance %q", required)
		}
	}
}

func testDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestNativeProviderEntriesRouteOrdinaryWorkToOperationSkills(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"AGENTS.md", "CLAUDE.md"} {
		content := fileFor(t, Plan{Files: files}, path).Content
		for _, operation := range []string{"markitect-init", "markitect-extract", "markitect-design", "markitect-suggest", "markitect-configure", "markitect-implement", "markitect-cleanup", "markitect-verify", "markitect-apply", "markitect-check"} {
			if !strings.Contains(content, operation) {
				t.Errorf("%s does not route to %s", path, operation)
			}
		}
		if !strings.Contains(content, "ordinary issue, bug, feature, or backlog Work Item") || !strings.Contains(content, "Markitect MCP as the outer project-operation interface") {
			t.Errorf("%s does not route ordinary work to operation skills", path)
		}
	}
	codex := fileFor(t, Plan{Files: files}, "AGENTS.md").Content
	claude := fileFor(t, Plan{Files: files}, "CLAUDE.md").Content
	for _, required := range []string{"codex mcp add markitect", "Codex CLI 0.162.0 App Server", "gpt-6-luna", "at `high`"} {
		if !strings.Contains(codex, required) {
			t.Errorf("Codex entrypoint lacks supported outer/inner client guidance %q", required)
		}
	}
	for _, required := range []string{"Claude Code 2.1.295", "claude mcp add --transport stdio markitect", "Claude Code is not used as an inner worker", "Codex CLI 0.162.0 App Server"} {
		if !strings.Contains(claude, required) {
			t.Errorf("Claude entrypoint lacks supported outer/inner client guidance %q", required)
		}
	}
	if strings.Contains(claude, "--execution-mode native-work") || strings.Contains(claude, "helperLimit") {
		t.Fatal("Claude entrypoint contains obsolete Codex exec mode or helper guidance")
	}
	for _, path := range staleSkillPaths([]Provider{Codex, Claude}) {
		for _, file := range files {
			if file.Path == path {
				t.Errorf("root routing still generates obsolete compatibility skill %s", path)
			}
		}
	}
}
