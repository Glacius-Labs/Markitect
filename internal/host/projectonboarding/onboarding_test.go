package projectonboarding

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
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

func TestPreviewMergesCRLFSkillFrontmatterForBothProvidersIdempotently(t *testing.T) {
	root, project := onboardingRepo(t)
	frontmatter := "---\r\nname: local-skill\r\ndescription: preserve these bytes\r\n---\r\n# Local skill\r\n"
	suffix := "\r\n## Local instructions\r\nPreserve this too.\r\n"
	original := frontmatter + beginMarker + "\r\nold generated text\r\n" + endMarker + suffix
	paths := []string{
		".agents/skills/markitect-model-first/SKILL.md",
		".claude/skills/markitect-model-first/SKILL.md",
	}
	for _, path := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}
	}

	options := defaultOptions(Claude, Codex)
	plan, err := Preview(root, project.Report.ModelDigest, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		merged := fileFor(t, plan, path).Content
		if !strings.HasPrefix(merged, frontmatter) || !strings.HasSuffix(merged, suffix) {
			t.Errorf("%s did not preserve CRLF frontmatter and local bytes", path)
		}
		if strings.Count(merged, "---\r\n") != 2 || strings.Contains(merged, "old generated text") || !strings.Contains(merged, "When a short Work Item") {
			t.Errorf("%s has duplicate frontmatter or incorrect managed content", path)
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
		if got := fileFor(t, second, path); got.Action != "unchanged" {
			t.Errorf("second preview action for %s = %q, want unchanged", path, got.Action)
		}
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

func TestModelFirstWorkflowCoversShortWorkItemsReadinessAndBrownfieldAdoption(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	workflow := fileFor(t, Plan{Files: files}, workflowPath).Content
	for _, required := range []string{
		"short Work Item, issue, bug, idea",
		"explicit decision ledger",
		"recover durable state",
		"compute its readiness from the current fixed project snapshot",
		"obtain the review or acknowledgement required by repository policy",
		"responsible Manager may review and acknowledge",
		"policy explicitly requires human review",
		"project-model tool classification only marks canonical model paths",
		"own Manager ID in the mutation actor field",
		"Explore CRUD path stores ModelAccepted:false",
		"current committed HEAD is the accepted repository specification",
		"committing a noncanonical draft file does not change the accepted model",
		"markitect project edit --repo PATH --input MUTATION.json",
		"never impersonate user or a human",
		"Ask the contributor only when material intent or authority is outside the delegation",
		"reverse-model it iteratively",
		"explicit transient scopes",
		"Keep the initial adoption model-only",
		"Brownfield Work Item",
		"brownfield-action start",
		"brownfield-action begin",
		"brownfield-action context",
		"brownfield-action run",
		"phase\":\"propose",
		"phase integrate",
		"Each Manager is a distinct invocation",
		"delegationEvidenceIDs",
		"its context exposes them as metadata (ID, path, basis, and digest)",
		"An omitted root pool retains legacy broad routing",
		"explicit-empty marker",
		"parent receives each direct child's exact final report",
		"begin input is a bare ReverseIterationRequest",
		"previewDigest",
		"requestContractDigest",
		"fails closed and is not automatically replayable",
		"process exits",
		"latest known failed attempt",
		"authorized owner or delegated Manager review the ledger",
		"repository policy requires a human decision",
		"brownfield-action resolve",
		"brownfield-action apply-adoption",
		"do not authenticate a human",
		"cleanup as a separate operation",
		"bounded goal against the selected accepted revision",
		"leaf Managers implement their files",
		"independent reviewer assesses the exact scoped candidate bytes",
		"first successful Apply",
		"resume the existing persisted run",
		"do not replay completed Manager work",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("shared workflow is missing required guidance %q", required)
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
	marker := "Minimal new exploration input:\n\n" + fence + "json\n"
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
		"markitect project explore --repo PATH --input .markitect/drafts/work-item.json",
		"--acknowledged-at RFC3339_TIME",
		"Generate one explicit UTC RFC3339 --acknowledged-at value and reuse that exact value",
		"Use the returned writePlan.digest for WRITE_PLAN_DIGEST",
		"an authorized Manager may assert its own delegated authority",
		"never claim that assertion is a human acknowledgement",
		"The Host reconciles committed accepted-model history automatically",
		"markitect project briefings --repo PATH --manager MANAGER_ID",
		"Do not invent or manually aggregate a model delta that the accepted-history mechanism already supplies",
		"explicitly preserve the already authorized time, start/retry, and cost limits",
		"never accept a refresh that silently resets those bounds",
		"integrated work advances through verification and Apply",
		"an already applied run recovers the exploration completion",
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

func TestNativeProviderEntriesSelectSkillForShortWorkItems(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"AGENTS.md", "CLAUDE.md"} {
		content := fileFor(t, Plan{Files: files}, path).Content
		if !strings.Contains(content, "short Work Item, issue, bug, idea") || !strings.Contains(content, "repository-local Markitect model-first skill") {
			t.Errorf("%s does not trigger the native model-first skill for a short request", path)
		}
	}
	for _, path := range []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md"} {
		content := fileFor(t, Plan{Files: files}, path).Content
		if !strings.Contains(content, "description: Use for ordinary short Work Items, issues, bugs, and ideas") {
			t.Errorf("%s metadata does not support native discovery for short work requests", path)
		}
	}
}
