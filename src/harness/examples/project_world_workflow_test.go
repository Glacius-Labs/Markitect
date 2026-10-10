package examples

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

// TestProjectWorldNativeCLIWorkflow runs the built native CLI against a fixed
// copy of the actual Shop example. The test uses only local source operations
// and a missing-runtime negative check; it does not call a provider. The
// agent-fixture protocol tests remain protocol fixtures, not evidence of a
// real provider or a later live Shop work-root run.
func TestProjectWorldNativeCLIWorkflow(t *testing.T) {
	repository, shop := projectWorldSource(t)
	sourceBefore, err := treeDigest(shop)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), executableName("markitect"))
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./src/cmd/markitect")
	build.Dir = repository
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build native CLI: %v\n%s", err, output)
	}

	repo := testkit.NewRepo(t)
	repo.Git("symbolic-ref", "HEAD", "refs/heads/feature-project-world")
	root := repo.Dir
	if err := copyShopTree(shop, root); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, ".markitect", "project.yaml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// Run the current legacy Shop source in guided mode in the isolated copy.
	// This keeps the checked-in example untouched while exercising the new
	// explicit workflow contract.
	if !bytes.Contains(manifest, []byte("coverageMode: full\n")) {
		t.Fatal("Shop fixture no longer declares full coverage")
	}
	manifest = bytes.Replace(manifest, []byte("coverageMode: full\n"), []byte("coverageMode: full\nworkflowMode: guided\nacceptancePolicy: committed-model\n"), 1)
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		t.Fatal(err)
	}
	repo.Commit("Freeze Shop source smoke fixture")
	smokeExploreReadiness(t, binary, root)

	for _, action := range [][]string{
		{"check"}, {"index"}, {"coverage"},
	} {
		stdout, stderr, code := cliSmoke(t, binary, root, action...)
		if code != 0 {
			t.Fatalf("project %s exit=%d: %s", action[0], code, stderr)
		}
		if action[0] == "coverage" {
			var report struct {
				Conforming bool     `json:"conforming"`
				Unknown    []string `json:"unknown"`
			}
			if err := json.Unmarshal(stdout, &report); err != nil || !report.Conforming || len(report.Unknown) != 0 {
				t.Fatalf("full Shop coverage report=%s err=%v", stdout, err)
			}
		} else if action[0] == "index" && !bytes.Contains(stdout, []byte(`"status": "succeeded"`)) {
			t.Fatalf("project index did not report succeeded: %s", stdout)
		}
	}

	for _, test := range []struct{ manager, contains string }{
		{`["project.markitect.example.org/v1alpha1","Manager","","shop"]`, `"name": "commerce"`},
		{`["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]`, "cancel-before-shipped"},
	} {
		stdout, stderr, code := cliSmoke(t, binary, root, "context", "--manager", test.manager)
		if code != 0 || !bytes.Contains(stdout, []byte(test.contains)) {
			t.Fatalf("project context manager=%s exit=%d stderr=%s output=%s", test.manager, code, stderr, stdout)
		}
	}

	viewPath := filepath.Join(root, "docs", "markitect", "project.md")
	wantView, stderr, code := cliSmoke(t, binary, root, "document")
	if code != 0 || !bytes.Contains(wantView, []byte("cancel-before-shipped")) {
		t.Fatalf("project document preview exit=%d stderr=%s", code, stderr)
	}
	if _, stderr, code := cliSmoke(t, binary, root, "document", "--write"); code != 0 {
		t.Fatalf("project document --write exit=%d: %s", code, stderr)
	}
	gotView, err := os.ReadFile(viewPath)
	if err != nil || !bytes.Equal(gotView, wantView) || !bytes.Contains(gotView, []byte("cancel-before-shipped")) {
		t.Fatalf("generated view differs from expected Shop bytes: err=%v", err)
	}

	// Both provider targets are generated as reviewed project-local files.
	// Applying the exact preview twice must converge to unchanged outputs.
	preview, stderr, code := cliSmoke(t, binary, root, "onboard", "--provider", "both")
	if code != 0 {
		t.Fatalf("onboard preview exit=%d: %s", code, stderr)
	}
	var onboarding struct {
		Digest string `json:"digest"`
		Files  []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"files"`
	}
	if err := json.Unmarshal(preview, &onboarding); err != nil || onboarding.Digest == "" || len(onboarding.Files) < 2 {
		t.Fatalf("onboarding preview=%s err=%v", preview, err)
	}
	if _, stderr, code := cliSmoke(t, binary, root, "onboard", "--provider", "both", "--expect", onboarding.Digest, "--write"); code != 0 {
		t.Fatalf("onboard apply exit=%d: %s", code, stderr)
	}
	secondPreview, stderr, code := cliSmoke(t, binary, root, "onboard", "--provider", "both")
	if code != 0 {
		t.Fatalf("second onboard preview exit=%d: %s", code, stderr)
	}
	var second struct {
		Digest string `json:"digest"`
		Files  []struct {
			Action string `json:"action"`
		} `json:"files"`
	}
	if err := json.Unmarshal(secondPreview, &second); err != nil || second.Digest == "" || len(second.Files) == 0 {
		t.Fatalf("second onboarding preview=%s err=%v", secondPreview, err)
	}
	for _, file := range second.Files {
		if file.Action != "unchanged" {
			t.Fatalf("onboarding was not idempotent: second action=%q output=%s", file.Action, secondPreview)
		}
	}
	if _, stderr, code := cliSmoke(t, binary, root, "onboard", "--provider", "both", "--expect", second.Digest, "--write"); code != 0 {
		t.Fatalf("idempotent onboard apply exit=%d: %s", code, stderr)
	}

	sourceAfter, err := treeDigest(shop)
	if err != nil || sourceAfter != sourceBefore {
		t.Fatalf("native CLI smoke changed checked-in Shop source: before=%s after=%s err=%v", sourceBefore, sourceAfter, err)
	}
}

// smokeExploreReadiness runs before generated onboarding files are written so
// the readiness binding sees only the clean committed project model.
func smokeExploreReadiness(t *testing.T, binary, root string) {
	t.Helper()
	// Explore record CRUD is model-only. Its open blocking decision remains
	// visible in status, while readiness cannot claim success without runtime.
	explorationInput := filepath.Join(root, ".markitect", "drafts", "native-smoke-exploration.json")
	if err := os.MkdirAll(filepath.Dir(explorationInput), 0755); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{
		"apiVersion": "markitect.example.org/project-exploration/v1alpha1",
		"id":         "native-shop-smoke", "status": "active", "request": "Review Shop cancellation behavior",
		"scopes":    []any{map[string]any{"id": "cancel", "name": "Cancel order", "goal": "Cancel a confirmed order and release its reservation atomically", "operation": "apply", "managerIds": []string{"[\"project.markitect.example.org/v1alpha1\",\"Manager\",\"commerce.sales.orders\",\"orders\"]"}}},
		"decisions": []any{map[string]any{"id": "public-outcome", "scopeIds": []string{"cancel"}, "question": "May cancellation outcomes be exposed publicly?", "blocking": true, "status": "open"}},
		"drafts":    []any{}, "structureAcknowledgements": []any{}, "completions": []any{},
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(explorationInput, inputBytes, 0644); err != nil {
		t.Fatal(err)
	}
	inputRelative, err := filepath.Rel(root, explorationInput)
	if err != nil {
		t.Fatal(err)
	}
	preview, stderr, code := cliSmoke(t, binary, root, "explore", "--input", filepath.ToSlash(inputRelative))
	if code != 0 {
		t.Fatalf("Explore preview exit=%d: %s", code, stderr)
	}
	var explorationPlan struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(preview, &explorationPlan); err != nil || explorationPlan.Digest == "" {
		t.Fatalf("Explore preview=%s err=%v", preview, err)
	}
	if _, stderr, code := cliSmoke(t, binary, root, "explore", "--input", filepath.ToSlash(inputRelative), "--expect", explorationPlan.Digest, "--write"); code != 0 {
		t.Fatalf("Explore write exit=%d: %s", code, stderr)
	}
	status, stderr, code := cliSmoke(t, binary, root, "explore", "--exploration", "native-shop-smoke")
	if code != 0 {
		t.Fatalf("Explore status exit=%d: %s", code, stderr)
	}
	var statusRecord struct {
		Status    string `json:"status"`
		Decisions []struct {
			ID       string `json:"id"`
			Blocking bool   `json:"blocking"`
			Status   string `json:"status"`
		} `json:"decisions"`
	}
	if err := json.Unmarshal(status, &statusRecord); err != nil || statusRecord.Status != "active" || len(statusRecord.Decisions) != 1 || statusRecord.Decisions[0].ID != "public-outcome" || !statusRecord.Decisions[0].Blocking || statusRecord.Decisions[0].Status != "open" {
		t.Fatalf("Explore status did not retain open blocking decision: output=%s err=%v", status, err)
	}
	_, readinessErr, readinessCode := cliSmoke(t, binary, root, "readiness", "--exploration", "native-shop-smoke", "--scope", "cancel")
	if readinessCode == 0 || !strings.Contains(strings.ToLower(readinessErr), "runtime") {
		t.Fatalf("readiness must require configured runtime; exit=%d stderr=%s", readinessCode, readinessErr)
	}

}

func projectWorldSource(t *testing.T) (repository, shop string) {
	t.Helper()
	repository = harnessRepositoryRoot(t)
	shop = filepath.Join(repository, "examples", "project-world")
	return repository, shop
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func copyShopTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0755)
		}
		if entry.IsDir() && (relative == ".git" || relative == filepath.Join(".markitect", "runs") || relative == filepath.Join(".markitect", "state", "runs")) {
			return filepath.SkipDir
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func treeDigest(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || filepath.Base(path) == "runs") {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	// WalkDir is lexical, but sort explicitly so this binding stays clear.
	sortStrings(files)
	h := sha256.New()
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%s\x00", file, data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cliSmoke(t *testing.T, binary, root string, args ...string) (stdout []byte, stderr string, code int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	commandArgs := []string{"project", args[0], "--repo", root}
	commandArgs = append(commandArgs, args[1:]...)
	command := exec.CommandContext(ctx, binary, commandArgs...)
	command.Dir = root
	var errout bytes.Buffer
	command.Stderr = &errout
	stdout, err := command.Output()
	if ctx.Err() != nil {
		t.Fatalf("markitect project %s timed out: %v", strings.Join(args, " "), ctx.Err())
	}
	if err == nil {
		return stdout, errout.String(), 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("start markitect project %s: %v", strings.Join(args, " "), err)
	}
	return stdout, errout.String(), exit.ExitCode()
}
