package projectcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
)

func TestGeneratedDistillationRejectsUncommittedSelectedModelBeforeRuntimeLoad(t *testing.T) {
	repo := copyProjectWorld(t)
	writeDistillBasisDiscovery(t, repo)
	modelPath := filepath.Join(repo, filepath.FromSlash(".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml"))
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, append(modelBytes, []byte("# uncommitted model edit\n")...), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errout bytes.Buffer
	if code := Run(generatedDistillArgs(repo), &out, &errout); code == 0 || !strings.Contains(errout.String(), "selected project inputs differ from committed HEAD") {
		t.Fatalf("dirty generated distillation exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/drafts/generated.json"))); !os.IsNotExist(err) {
		t.Fatalf("rejected generated distillation emitted a report: %v", err)
	}
}

func TestGeneratedDistillationCleanBasisReachesRuntimeValidation(t *testing.T) {
	repo := copyProjectWorld(t)
	writeDistillBasisDiscovery(t, repo)
	var out, errout bytes.Buffer
	if code := Run(generatedDistillArgs(repo), &out, &errout); code == 0 || !strings.Contains(errout.String(), "runtime apiVersion") || strings.Contains(errout.String(), "selected project inputs differ from committed HEAD") {
		t.Fatalf("clean generated distillation exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/drafts/generated.json"))); !os.IsNotExist(err) {
		t.Fatalf("runtime validation failure emitted a report: %v", err)
	}
}

func generatedDistillArgs(repo string) []string {
	return []string{
		"distill", "--repo", repo, "--discovery", ".markitect/drafts/distill-fixed-basis.json",
		"--generate", "--write", "--output", ".markitect/drafts/generated.json",
		"--input-micros-per-million", "1", "--output-micros-per-million", "1", "--max-cost-micros", "1000",
	}
}

func writeDistillBasisDiscovery(t *testing.T, repo string) {
	t.Helper()
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion,
		ID:         "distill-fixed-basis",
		Purpose:    "Capture selected cancellation evidence",
		Review:     "distill-fixed-basis-test",
		Commit:     commit,
		ScopeRoots: []string{"docs"},
		Selected: []projectadoption.SelectedPath{{
			ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Use selected project evidence", Basis: "documentation",
		}},
		Exclusions: []projectadoption.PathReason{},
		Unselected: []projectadoption.PathReason{},
	}
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	discoveryBytes, err := projectadoption.EncodeDiscovery(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/distill-fixed-basis.json", discoveryBytes); err != nil {
		t.Fatal(err)
	}
}
