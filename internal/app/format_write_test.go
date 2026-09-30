package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestFormatWriteRefusesSharedWriterLock(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	skillFile := filepath.Join(root, filepath.FromSlash(skillPath))
	if err := os.WriteFile(skillFile, append(mustRead(t, skillFile), []byte("# formatting comment\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, skillFile)
	lock := filepath.Join(root, ".artifacts", "markitect", "write.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("held\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Format(root, project, true); err == nil || !strings.Contains(err.Error(), "write.lock") {
		t.Fatalf("Format error = %v, want shared writer lock refusal", err)
	}
	if after := mustRead(t, skillFile); string(after) != string(before) {
		t.Fatal("format changed the source while another writer held the shared lock")
	}
}

func TestFormatWriteRefusesStaleSource(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	skillFile := filepath.Join(root, filepath.FromSlash(skillPath))
	if err := os.WriteFile(skillFile, append(mustRead(t, skillFile), []byte("# formatting comment\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := Format(root, project, false); err != nil || len(changed) == 0 {
		t.Fatalf("fixture does not require canonicalization: changed=%v err=%v", changed, err)
	}
	if err := os.WriteFile(skillFile, append(mustRead(t, skillFile), []byte("# concurrent edit\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, skillFile)
	if _, err := Format(root, project, true); err == nil || !strings.Contains(err.Error(), "source changed since capture") {
		t.Fatalf("Format error = %v, want stale-source refusal", err)
	}
	if after := mustRead(t, skillFile); string(after) != string(before) {
		t.Fatal("format overwrote a source edit made after capture")
	}
}

func TestEnsureWriteBranchDetectsSameSHABranchSwitch(t *testing.T) {
	root := installTestRepo(t, "feature/captured")
	expected, err := writeBranchName(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := source.GitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	runWriterGit(t, root, "branch", "feature/other")
	runWriterGit(t, root, "checkout", "feature/other")
	after, err := source.GitOutput(root, "rev-parse", "HEAD")
	if err != nil || string(before) != string(after) {
		t.Fatalf("branch-switch fixture changed HEAD: before=%q after=%q err=%v", before, after, err)
	}
	if err := ensureWriteBranch(root, expected); err == nil || !strings.Contains(err.Error(), "branch changed") {
		t.Fatalf("same-SHA branch switch error = %v, want branch-change refusal", err)
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
