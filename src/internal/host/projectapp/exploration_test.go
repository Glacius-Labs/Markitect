package projectapp

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestExploreOperationPreviewsAndWritesWithExpectedDigest(t *testing.T) {
	root := makePlanFixture(t)
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Managers")
	}
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "facade-exploration", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded", Name: "Bounded", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	operations := Operations{Host: projectrun.Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit}}
	selection := Selection{Root: root}
	preview, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Plan == nil || preview.Plan.Digest == "" || preview.Persisted != nil {
		t.Fatalf("preview result = %#v", preview)
	}
	if _, err := projectexplore.Load(root, record.ID); err == nil {
		t.Fatal("preview persisted the record")
	}
	if record.Digest != "" || record.CreatedAgainst != "" {
		t.Fatalf("preview mutated caller-owned record: %#v", record)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: "sha256:stale"}); err == nil {
		t.Fatal("write with a stale expected digest succeeded")
	}
	if _, err := projectexplore.Load(root, record.ID); err == nil {
		t.Fatal("stale digest write persisted the record")
	}
	written, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest})
	if err != nil {
		t.Fatalf("write exact preview: %v", err)
	}
	if written.Persisted == nil || written.Persisted.ID != record.ID || written.Plan == nil || written.Plan.Digest != preview.Plan.Digest {
		t.Fatalf("write result = %#v", written)
	}
	loaded, err := projectexplore.Load(root, record.ID)
	if err != nil || loaded.Digest != written.Persisted.Digest {
		t.Fatalf("loaded record = %#v err=%v", loaded, err)
	}
	listed, err := operations.Explore(ExploreOperation{Selection: selection})
	if err != nil || len(listed.Records) != 1 || listed.Records[0].ID != record.ID {
		t.Fatalf("list result = %#v err=%v", listed, err)
	}
	read, err := operations.Explore(ExploreOperation{Selection: selection, ExplorationID: record.ID})
	if err != nil || read.Record == nil || read.Record.Digest != loaded.Digest {
		t.Fatalf("read result = %#v err=%v", read, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "explorations", record.ID+".json")); err != nil {
		t.Fatalf("expected durable exploration file: %v", err)
	}
}

func TestExploreAcceptsGitCleanAutocrlfAndRejectsCapturedSourceChanges(t *testing.T) {
	root := makePlanFixture(t)
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedManifest := strings.Replace(string(manifest), "inventoryRoots: []", "inventoryRoots:\n  - docs", 1)
	if updatedManifest == string(manifest) {
		t.Fatal("could not select docs inventory for the exploration fixture")
	}
	if err := os.WriteFile(manifestPath, []byte(updatedManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	writeExploreFile(t, root, "docs/.gitattributes", "binary.dat -text\n")
	writeExploreFile(t, root, "docs/README.md", "selected source\nsecond line\n")
	writeExploreFile(t, root, "docs/binary.dat", "binary source\nsecond line\n")
	runExploreGit(t, root, "add", ".")
	runExploreGit(t, root, "commit", "-m", "select exploration source files")
	runExploreGit(t, root, "config", "core.autocrlf", "true")

	readmePath := filepath.Join(root, "docs", "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, bytes.ReplaceAll(readme, []byte("\n"), []byte("\r\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	revision := runExploreGit(t, root, "rev-parse", "HEAD")
	fixed, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fixed.Snapshot.Files["docs/README.md"]; !ok {
		t.Fatal("fixture README is not part of the fixed selected snapshot")
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Snapshot.Digest() == project.Snapshot.Digest() {
		t.Fatal("fixture did not preserve distinct LF fixed and CRLF working bytes")
	}
	if !bytes.Equal(project.Snapshot.Files["docs/README.md"], []byte("selected source\r\nsecond line\r\n")) {
		t.Fatalf("working README bytes = %q", project.Snapshot.Files["docs/README.md"])
	}

	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Managers")
	}
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "autocrlf-exploration", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded", Name: "Bounded", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	operations := Operations{Host: projectrun.Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit}}
	selection := Selection{Root: root, Revision: revision}
	preview, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatalf("preview over Git-clean CRLF checkout: %v", err)
	}
	if preview.Plan == nil || !preview.Plan.VerifyBasis {
		t.Fatalf("preview did not retain basis verification: %#v", preview.Plan)
	}

	if err := os.WriteFile(readmePath, []byte("substantive source edit\r\nsecond line\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest}); err == nil {
		t.Fatal("write accepted a substantive source change after preview")
	}
	if _, err := projectexplore.Load(root, record.ID); err == nil {
		t.Fatal("rejected stale write persisted the exploration record")
	}

	runExploreGit(t, root, "checkout", "--", "docs/README.md")
	binaryPath := filepath.Join(root, "docs", "binary.dat")
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, bytes.ReplaceAll(binary, []byte("\n"), []byte("\r\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record}); err == nil {
		t.Fatal("Git -text binary path accepted CRLF changes")
	}
}

func TestReadinessAcknowledgementUsesRawCapturedModelCheckoutBytes(t *testing.T) {
	root := makePlanFixture(t)
	revision := runExploreGit(t, root, "rev-parse", "HEAD")
	runExploreGit(t, root, "config", "core.autocrlf", "true")
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Config.ModelFiles) == 0 || len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no selected model file or Manager")
	}
	modelPath := project.Config.ModelFiles[0]
	absoluteModelPath := filepath.Join(root, filepath.FromSlash(modelPath))
	modelBytes, err := os.ReadFile(absoluteModelPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absoluteModelPath, bytes.ReplaceAll(modelBytes, []byte("\n"), []byte("\r\n")), 0o600); err != nil {
		t.Fatal(err)
	}

	operations := Operations{Host: projectrun.Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot, PlanEdit: projectwork.PlanEdit}}
	selection := Selection{Root: root, Revision: revision}
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "autocrlf-readiness", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded", Name: "Bounded", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	create, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatalf("preview exploration over CRLF YAML checkout: %v", err)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: create.Plan.Digest}); err != nil {
		t.Fatalf("persist exploration over CRLF YAML checkout: %v", err)
	}

	acknowledgement := &StructureAcknowledgementInput{
		Actor: "fixture owner", Authority: "project owner", Provenance: "explicit fixture decision",
		RecordedAt: time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
	}
	readinessOperation := ReadinessOperation{
		Selection: selection, ExplorationID: record.ID, ScopeID: "bounded", Acknowledgement: acknowledgement,
	}
	preview, err := operations.Readiness(readinessOperation)
	if err != nil {
		t.Fatalf("preview readiness acknowledgement over CRLF YAML checkout: %v", err)
	}
	if preview.WritePlan == nil {
		t.Fatal("readiness acknowledgement preview omitted its write plan")
	}
	if err := os.WriteFile(absoluteModelPath, append(modelBytes, []byte("# substantive change\r\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	readinessOperation.Write = true
	readinessOperation.ExpectedDigest = preview.WritePlan.Digest
	if _, err := operations.Readiness(readinessOperation); err == nil {
		t.Fatal("readiness write accepted a model change after preview")
	}
	unchanged, err := projectexplore.Load(root, record.ID)
	if err != nil || len(unchanged.Acknowledgements) != 0 {
		t.Fatalf("stale readiness write changed the record: acknowledgements=%d err=%v", len(unchanged.Acknowledgements), err)
	}

	runExploreGit(t, root, "checkout", "--", filepath.ToSlash(modelPath))
	if _, err := operations.Readiness(readinessOperation); err != nil {
		t.Fatalf("write exact readiness acknowledgement after restoring clean checkout: %v", err)
	}
	acknowledged, err := projectexplore.Load(root, record.ID)
	if err != nil || len(acknowledged.Acknowledgements) != 1 {
		t.Fatalf("acknowledged record = %#v err=%v", acknowledged, err)
	}
}

func writeExploreFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runExploreGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = source.CleanGitEnv()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestReadinessRequiresExplicitAcknowledgementForWrite(t *testing.T) {
	loads := 0
	operations := Operations{Host: projectrun.Host{Load: func(string, string) (*projectrun.Project, error) {
		loads++
		return nil, nil
	}}}
	_, err := operations.Readiness(ReadinessOperation{Selection: Selection{Root: t.TempDir()}, Write: true})
	if err == nil || err.Error() != "ready --write requires --acknowledge" {
		t.Fatalf("Readiness write without acknowledgement error = %v", err)
	}
	if loads != 0 {
		t.Fatalf("invalid write reached project Host %d times", loads)
	}
}

func TestExplorationOperationJSONUsesCamelCaseAndTypedRecord(t *testing.T) {
	record := &projectexplore.Record{ID: "typed-record"}
	payload, err := json.Marshal(ExploreOperation{
		Selection: Selection{Root: "repo"}, ExplorationID: "typed-record", Record: record, ExpectedDigest: "sha256:expected",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	for _, field := range []string{`"selection":{"root":"repo","revision":""}`, `"explorationId"`, `"expectedDigest"`, `"record":{"apiVersion"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("ExploreOperation JSON %s omits %s", encoded, field)
		}
	}
	if strings.Contains(encoded, `"input"`) {
		t.Fatalf("ExploreOperation JSON retained byte input: %s", encoded)
	}

	acknowledgement := &StructureAcknowledgementInput{Actor: "caller", Authority: "owner", Provenance: "request"}
	readiness, err := json.Marshal(ReadinessOperation{ExplorationID: "typed-record", ScopeID: "scope", Acknowledgement: acknowledgement})
	if err != nil {
		t.Fatal(err)
	}
	encoded = string(readiness)
	for _, field := range []string{`"explorationId"`, `"scopeId"`, `"acknowledgement"`, `"recordedAt"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("ReadinessOperation JSON %s omits %s", encoded, field)
		}
	}
}
