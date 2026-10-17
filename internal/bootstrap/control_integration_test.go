package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/adapters/journal"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	runcontrol "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// Only profile metadata is fake. These source-bound port observations establish
// no driver, image, protocol or provider readiness. Git acquisition, compilation,
// accounting, settings, process lock, SQLite and RUN restoration are real.
type fakeControlMetadata struct{}

func (fakeControlMetadata) PrepareAgentProfile(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	file := func(boundary, name, data string) apply.ProfileMetadataFile {
		path, _ := snapshot.NewPath(name)
		return apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: boundary, Path: path},
			ContentID: snapshot.ContentID(sha256.Sum256([]byte(data))), Size: int64(len(data)), Mode: snapshot.Regular}
	}
	return apply.ProfileObservation{InputBinding: input.Binding(), Status: model.Passed,
		Contract: apply.ProfileContract{Name: "fake-bootstrap-control-metadata", Version: 1},
		Files:    []apply.ProfileMetadataFile{file("fake-program", "program", "fake static executable metadata")},
		Inventories: []apply.ProfileMetadataInventory{{Boundary: "fake-schema", Files: []apply.ProfileMetadataFile{
			file("fake-schema", "schema.json", "fake static schema metadata"),
		}}},
	}, nil
}

type gitControlFixture struct {
	repo     *testkit.Repo
	source   *gitrepo.Repository
	prepared apply.CapturedSettingsResult
	facts    apply.RunBindingFacts
	options  apply.ControlOptions
}

func newGitControlFixture(t *testing.T) gitControlFixture {
	t.Helper()
	repo := productFixture(t)
	execution, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(execution), "\r\n", "\n")
	start, end := strings.Index(text, "\nchecks:\n"), strings.Index(text, "\nacceptance:\n")
	if start < 0 || end <= start {
		t.Fatal("fixture execution sections missing")
	}
	// No check or execution adapter participates in this storage seam.
	repo.Write(".markitect/execution.yaml", text[:start]+"\nchecks: []"+text[end:])
	repo.Write(".gitignore", ".markitect/state/\n")
	repo.Write(".markitect/ignore.yaml", "apiVersion: markitect.ignore/v1\nfiles:\n  - path: .gitignore\n    reason: Fixture Git control policy.\n")
	head := repo.Commit("committed Git control integration inputs")
	path := func(name string) snapshot.Path {
		p, err := snapshot.NewPath(name)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	host := testkit.TempDir(t)
	if err := os.WriteFile(filepath.Join(host, "host.yaml"), []byte("apiVersion: markitect.host/v1\nagentProfiles:\n  markitect-agent:\n    codexAppServer: {profileVersion: 1}\ncheckProfiles: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	selection := apply.InputSelection{Repository: repo.Dir,
		Local:        apply.ModelSelection{Boundary: filepath.Join(repo.Dir, ".markitect"), ManifestPath: path("model.yaml")},
		HostProfiles: []apply.SourceLocation{{Boundary: host, Path: path("host.yaml")}}}
	prepared, err := apply.NewPreparationService(apply.NewModelService(source, source)).CaptureSettings(
		context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, fakeControlMetadata{}, nil)
	attempt, ok := prepared.Prepared()
	if err != nil || !ok || prepared.Status() != model.Passed {
		for _, a := range prepared.Attempts() {
			t.Logf("compile=%+v execution=%+v registry=%+v files=%+v settings=%+v", a.Structural().Model().Compilation(),
				a.Structural().Execution().Diagnostics(), a.Structural().Registry().Diagnostics(), a.Structural().Files().Diagnostics(), a.Settings().Diagnostics())
		}
		t.Fatalf("actual Git preparation: %s %v", prepared.Status(), err)
	}
	input := attempt.Structural().Model().Inputs()
	if input.Head != head || input.Guard == "" {
		t.Fatal("preparation lost the actual committed capture")
	}
	owner, err := runcontrol.NewPayload("text/plain", "utf-8", []byte("Prepare the captured model; do not dispatch work.\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	// Explicit Host assertions, not inferred assignment choices or capabilities.
	facts := apply.RunBindingFacts{RepositoryIdentity: "bootstrap-git-control-fixture", ProductBuild: "bootstrap-source-test",
		OwnerRequest: owner, Allocation: runcontrol.Capacity{Runs: 2, SessionStarts: 70, Steps: 140, ExecutionTime: 7 * time.Hour}}
	if _, err := prepared.BindRun(facts); err != nil {
		t.Fatal("actual prepared binding", err)
	}
	return gitControlFixture{repo: repo, source: source, prepared: prepared, facts: facts,
		options: apply.ControlOptions{StateDirectory: filepath.Join(testkit.TempDir(t), "store"),
			RestoreLimits: runcontrol.RestoreLimits{PageSize: 1, Batches: 20, Events: 40, Bytes: 16 << 20}}}
}

func (f gitControlFixture) open(t *testing.T) (apply.ControlOpenResult, *apply.ControlledRun) {
	t.Helper()
	result, err := apply.NewControlService(f.source, journal.New()).Open(context.Background(), f.prepared, f.facts, f.options)
	c, ok := result.Controlled()
	if err != nil || !ok {
		t.Fatal("actual Git/SQLite control open", err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	return result, c
}

func (f gitControlFixture) request() apply.ControlRequest {
	a, _ := f.prepared.Prepared()
	m := a.Structural().Model()
	i := m.Inputs()
	return apply.ControlRequest{Repository: m.Request().Selection.Repository, RepositoryIdentity: f.facts.RepositoryIdentity,
		Revision: i.Head, Guard: i.Guard, StateDirectory: f.options.StateDirectory}
}

func commitGitControlPreparation(t *testing.T, c *apply.ControlledRun) (runcontrol.Command, runcontrol.AppendBatch, runcontrol.AppendReceipt) {
	t.Helper()
	command := runcontrol.PrepareCommand("prepare", "explicit-assignment", "explicit-run", "",
		time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC), c.Basis())
	proposal, err := c.Coordinator().Propose(command)
	if err != nil {
		t.Fatal(err)
	}
	if c.Coordinator().Revision() != 0 || c.Coordinator().Current() != (runcontrol.RunView{}) || len(c.Coordinator().Assignments()) != 0 {
		t.Fatal("proposal published an unconfirmed run or assignment")
	}
	receipt, err := c.Coordinator().Commit(context.Background(), c.Journal(), proposal)
	if err != nil || !receipt.Matches(proposal.Batch()) {
		t.Fatal("explicit preparation was not exactly acknowledged", receipt, err)
	}
	return command, proposal.Batch(), receipt
}

func TestGitControlOpenAndExplicitPreparationSurviveReopen(t *testing.T) {
	f := newGitControlFixture(t)
	state := filepath.Join(f.repo.Dir, ".markitect", "state")
	// Public read-only commands and captured preparation must not acquire control.
	invoke(t, f.repo.Dir, 0, "check")
	invoke(t, f.repo.Dir, 0, "context", "product/cli")
	invoke(t, f.repo.Dir, 3, "apply")
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read-only preparation or unavailable Apply created control state", err)
	}
	result, c := f.open(t)
	a, _ := f.prepared.Prepared()
	original := a.Structural().Model().Inputs()
	observed, ok := result.Recapture()
	if !ok || !reflect.DeepEqual(original.Raw, observed.Raw) || original.Head != observed.Head || observed.Guard == "" {
		t.Fatal("creating real control metadata caused false source drift")
	}
	if _, err := os.Stat(filepath.Join(state, "journal.json")); err != nil {
		t.Fatal("actual store registration missing", err)
	}
	for _, file := range observed.Raw.Census.Entries() {
		if strings.HasPrefix(file.Path().String(), ".markitect/state/") {
			t.Fatal("control state leaked into real capture", file.Path())
		}
	}
	page, err := c.Journal().Read(context.Background(), runcontrol.EventQuery{Limit: 1})
	if err != nil || page.Revision != 0 || len(page.Batches) != 0 || page.Next != 0 ||
		c.Coordinator().Revision() != 0 || c.Coordinator().Current() != (runcontrol.RunView{}) || len(c.Coordinator().Assignments()) != 0 {
		t.Fatal("first checked store open silently appended or selected an assignment", page, err)
	}
	if competing, err := journal.New().Acquire(context.Background(), f.request()); !errors.Is(err, apply.ErrControlUnavailable) || competing != nil {
		if competing != nil {
			_ = competing.Close()
		}
		t.Fatal("successful open did not hold the real repository process lock", err)
	}
	command, batch, receipt := commitGitControlPreparation(t, c)
	binding, basis, view := c.Binding(), c.Basis(), c.Coordinator().Current()
	assignments := c.Coordinator().Assignments()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	_, reopened := f.open(t)
	if reopened.Binding().StoreID != binding.StoreID || reopened.Binding().Epoch <= binding.Epoch ||
		reopened.Basis().Ref() != basis.Ref() || reopened.Coordinator().Revision() != receipt.Revision ||
		reopened.Coordinator().Current() != view || view.State != runcontrol.Prepared || view.Basis != basis.Ref() ||
		!reflect.DeepEqual(assignments, reopened.Coordinator().Assignments()) || reopened.Coordinator().RecoveryRequired() || len(reopened.Coordinator().Invocations()) != 0 {
		t.Fatal("reopen lost original preparation or automatically continued execution")
	}
	for _, payload := range []runcontrol.Payload{basis.Payload(), f.facts.OwnerRequest} {
		got, err := reopened.Journal().ReadBlob(context.Background(), payload.Ref())
		if err != nil || !bytes.Equal(got.Bytes(), payload.Bytes()) || got.Ref() != payload.Ref() {
			t.Fatal("original basis/owner bytes not restored", err)
		}
	}
	page, err = reopened.Journal().Read(context.Background(), runcontrol.EventQuery{Limit: 2})
	if err != nil || page.Revision != receipt.Revision || len(page.Batches) != 1 || page.Next != 0 || !bytes.Equal(page.Batches[0].Bytes(), batch.Bytes()) {
		t.Fatal("reopen changed the confirmed stream", page, err)
	}
	// An explicit repeat receives the original acknowledgement; opening alone
	// neither repeats this command nor invents a continuation decision.
	proposal, err := reopened.Coordinator().Propose(command)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Coordinator().Commit(context.Background(), reopened.Journal(), proposal)
	if err != nil || got != receipt || !got.Matches(batch) || reopened.Coordinator().Revision() != receipt.Revision {
		t.Fatal("original receipt was not retained", got, err)
	}
}

func TestGitControlRefusesCommittedInputDriftAndRetainsRegisteredHistory(t *testing.T) {
	f := newGitControlFixture(t)
	_, c := f.open(t)
	_, batch, receipt := commitGitControlPreparation(t, c)
	binding, basis, view := c.Binding(), c.Basis(), c.Coordinator().Current()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	a, _ := f.prepared.Prepared()
	original := a.Structural().Model().Inputs()
	const changedPath = ".markitect/execution.yaml"
	data, err := os.ReadFile(filepath.Join(f.repo.Dir, filepath.FromSlash(changedPath)))
	if err != nil {
		t.Fatal(err)
	}
	f.repo.Write(changedPath, strings.Replace(string(data), "maxReviewRounds: 3", "maxReviewRounds: 2", 1))
	head := f.repo.Commit("tracked capture input changed after preparation")
	result, err := apply.NewControlService(f.source, journal.New()).Open(context.Background(), f.prepared, f.facts, f.options)
	if _, ok := result.Controlled(); !errors.Is(err, apply.ErrCaptureChanged) || ok {
		t.Fatal("actual changed Git source obtained a restored projection", err)
	}
	observed, ok := result.Recapture()
	if !ok || observed.Head != head || observed.Head == original.Head || observed.Guard == "" || reflect.DeepEqual(observed.Raw, original.Raw) {
		t.Fatal("failed under-lock source observation was lost")
	}
	before, found := original.Raw.Census.Lookup(mustControlPath(t, changedPath))
	after, changed := observed.Raw.Census.Lookup(mustControlPath(t, changedPath))
	if !found || !changed || before.ContentID() == after.ContentID() {
		t.Fatal("test did not change a captured tracked input")
	}
	retained, err := f.prepared.BindRun(f.facts)
	if err != nil || retained.Ref() != basis.Ref() || !bytes.Equal(retained.Payload().Bytes(), basis.Payload().Bytes()) ||
		result.Binding().StoreID != binding.StoreID || result.Binding().Epoch <= binding.Epoch {
		t.Fatal("failed Open substituted new facts or replaced registered identity", err)
	}
	// Reading the actual registered store after failure proves lock cleanup and
	// retained history. It grants no new source-bound ControlledRun capability.
	session, err := journal.New().Acquire(context.Background(), f.request())
	if err != nil {
		t.Fatal("failed recapture retained the actual lock", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()
	if session.Binding().StoreID != binding.StoreID || session.Binding().Epoch <= result.Binding().Epoch {
		t.Fatal("failure lost the registered store lifecycle")
	}
	page, err := session.Journal().Read(context.Background(), runcontrol.EventQuery{Limit: 2})
	if err != nil || page.Revision != receipt.Revision || len(page.Batches) != 1 || page.Next != 0 || !bytes.Equal(page.Batches[0].Bytes(), batch.Bytes()) {
		t.Fatal("refused recapture appended or replaced registered history", page, err)
	}
	restored, err := runcontrol.Restore(context.Background(), session.Journal(), f.facts.RepositoryIdentity, f.options.RestoreLimits)
	if err != nil || restored.Revision() != receipt.Revision || restored.Current() != view {
		t.Fatal("original registered preparation no longer readable", err)
	}
}

func mustControlPath(t *testing.T, name string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
