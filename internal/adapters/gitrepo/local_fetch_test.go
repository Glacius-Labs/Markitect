package gitrepo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestLocalFetchPackedNoChange(t *testing.T) {
	// Host configuration must not disable physical long paths in acquisition.
	// Override only this test-owned home; the shared testkit config is immutable.
	isolatedHome := t.TempDir()
	globalPath := filepath.Join(isolatedHome, ".gitconfig")
	globalConfig := []byte(testkit.GlobalConfig + "\n[core]\n\tlongpaths = false\n")
	if err := os.WriteFile(globalPath, globalConfig, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", isolatedHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(isolatedHome, ".config"))
	_, _, original, input := runFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// Both ordinary file transport and the explicit upload-pack command must
	// preserve a repository name containing spaces and shell metacharacters.
	source := filepath.Join(t.TempDir(), "source ' ; & $()")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := original.Reopen(ctx, input, source); err != nil {
		t.Fatal(err)
	}
	if _, err := localCommand(ctx, source, nil, "switch", "--create", "codex/packed-nochange-source"); err != nil {
		t.Fatal(err)
	}
	// Reachable history exceeds Git's default unpack limit without adding
	// artifacts to every Manager's Census. All commits keep the complete tree.
	head := input.Commit
	for i := 0; i < 101; i++ {
		var err error
		head, err = localCommand(ctx, source, []byte(fmt.Sprintf("Packed history %d\n", i)), "commit-tree", input.Tree, "-p", head)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := localCommand(ctx, source, nil, "update-ref", "HEAD", head); err != nil {
		t.Fatal(err)
	}
	if _, err := localCommand(ctx, source, nil, "repack", "-ad"); err != nil {
		t.Fatal(err)
	}
	selection := project.InputSelection{Repository: source,
		Local:        project.ModelSelection{Boundary: ".", ManifestPath: fixturePath(t, ".markitect/model.yaml")},
		Dependencies: []project.ModelSelection{{Boundary: ".markitect/dependencies/core", ManifestPath: fixturePath(t, "model.yaml")}},
	}
	captured, err := project.NewPreparationService(project.NewModelService(adapter(t), adapter(t))).Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok := captured.Prepared()
	if !ok {
		t.Fatal("complete committed input did not prepare", captured.Status())
	}
	prepared, err := project.Prepare(attempt)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(source, ".git/config")
	beforeConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(beforeConfig, []byte("longpaths")) {
		t.Fatal("fixture unexpectedly persisted a user-repository longpaths setting")
	}
	// Read the fixture explicitly; acquisition intentionally ignores globals.
	if value, err := localCommand(ctx, source, nil, "config", "--file", globalPath, "--get", "core.longpaths"); err != nil || value != "false" {
		t.Fatal("test-owned global longpaths=false control was not observed", string(value), err)
	}
	if packs, err := filepath.Glob(filepath.Join(source, ".git/objects/pack/*.pack")); err != nil || len(packs) == 0 {
		t.Fatal("fixture did not use packed objects", packs, err)
	}
	root := t.TempDir()
	for len(root) < 164 {
		root = filepath.Join(root, "p")
	}
	git := &LocalGit{StateRoot: filepath.Join(root, "state"), TempRoot: filepath.Join(root, "temporary")}
	if err := os.MkdirAll(git.TempRoot, 0700); err != nil {
		t.Fatal(err)
	}
	// Even the shortest MkdirTemp suffix leaves the seed's pack path above
	// Windows MAX_PATH. A loose-object fixture would miss this boundary.
	if len(filepath.Join(git.TempRoot, "markitect-input-1", ".git/objects/pack/pack-"+strings.Repeat("a", 40)+".pack")) < 260 {
		t.Fatal("regression does not cross the packed-object path boundary")
	}
	// Isolate packed-object reads from Git for Windows' separate -C limit:
	// even the longest MkdirTemp suffix keeps Manager Git directories below
	// 250, allowing room for Windows to expand a short user-directory name.
	if len(filepath.Join(git.TempRoot, "markitect-run-"+strings.Repeat("9", 10), "manager-"+strings.Repeat("a", 32), ".git")) >= 250 {
		t.Fatal("fixture crosses the separate Git working-directory boundary")
	}
	ref, err := git.Capture(ctx, "fixture", prepared)
	if err != nil {
		t.Fatal("packed no-change capture and retention failed", err)
	}
	if ref.Commit != head || ref.Tree != input.Tree || ref.Commit != prepared.Attempt().Model().Inputs().Head {
		t.Fatal("no-change capture rewrote the committed input", ref, input)
	}
	retainedPath, err := git.candidatePath(ref)
	if err != nil {
		t.Fatal(err)
	}
	if heads, err := localCommand(ctx, source, nil, "bundle", "list-heads", filepath.Join(retainedPath, "candidate.bundle")); err != nil || heads != head+" refs/markitect/retain/"+ref.Key {
		t.Fatal("retained bundle did not advertise exactly its original ref and commit", heads, err)
	}
	store, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	executor := &applyExecutor{t: t, git: git, store: store, noChange: true}
	command := apply.Command{Executor: executor, Candidates: git, Records: store, Publisher: &Publisher{Repository: adapter(t), Store: git}}
	report, err := command.Execute(ctx, run.Request{ID: "packed-nochange", Project: "fixture", Prepared: prepared, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(2*time.Minute), 150), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || report.Status != model.Passed || report.Publication == nil || !report.Publication.NoChange || !report.Publication.Readback || len(report.Publication.CompletedPaths) != 0 || len(executor.requests) != 3 {
		t.Fatal("complete no-change source consumer failed", report.Status, report.Publication, err)
	}
	for i, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		if executor.requests[i].Mandate != mandate {
			t.Fatal("no-change skipped a mandate", executor.requests[i].Mandate)
		}
	}
	afterConfig, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(beforeConfig, afterConfig) {
		t.Fatal("transport changed the user repository configuration", err)
	}
	retained := report.Publication.Candidate
	for _, owned := range []string{source, original.StateRoot, git.TempRoot} {
		if err := os.RemoveAll(owned); err != nil {
			t.Fatal(err)
		}
	}
	reopened := t.TempDir()
	fresh := &LocalGit{StateRoot: git.StateRoot}
	if err := fresh.Reopen(ctx, retained, reopened); err != nil {
		t.Fatal("candidate depended on old copies or export state", err)
	}
	actual, err := object(ctx, reopened, "HEAD")
	if err != nil || actual.ID != head || actual.Tree != input.Tree {
		t.Fatal("independent reopening changed the commit or complete tree", actual, err)
	}
	for _, entry := range prepared.Attempt().Model().Inputs().Raw.Census.Entries() {
		data, err := os.ReadFile(filepath.Join(reopened, filepath.FromSlash(entry.Path().String())))
		if err != nil || !bytes.Equal(data, entry.Bytes()) {
			t.Fatal("reopening lost governed source", entry.Path(), err)
		}
	}
	if info, err := os.Stat(filepath.Join(reopened, ".markitect/model/b/empty")); err != nil || !info.IsDir() {
		t.Fatal("reopening lost the empty Package", err)
	}
	if actual, err := os.ReadFile(globalPath); err != nil || !bytes.Equal(actual, globalConfig) {
		t.Fatal("transport or acquisition changed the isolated global configuration", err)
	}
}
