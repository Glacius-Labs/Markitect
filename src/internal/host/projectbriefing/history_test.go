package projectbriefing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestEnsureAcceptedHistoryBootstrapsAndAutomaticallyRecordsCommittedChanges(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	receipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.BaselineRevision != baseline || receipt.Revision != changed || len(receipt.Bundles) != 1 {
		t.Fatalf("ensure receipt = %#v", receipt)
	}
	bundle := receipt.Bundles[0]
	if bundle.SinceRevision != baseline || bundle.Revision != changed || len(bundle.Events) == 0 {
		t.Fatalf("automatic briefing did not bind adjacent accepted revisions: %#v", bundle)
	}
	if bundle.Provenance.DecisionReference != "git-commit:"+changed || !strings.HasPrefix(bundle.Provenance.Actor, "git-commit-author:") || !strings.Contains(bundle.Provenance.Authority, "Git identity is not authenticated") {
		t.Fatalf("automatic provenance overclaims or omits source: %#v", bundle.Provenance)
	}
	state, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if digest != receipt.StoreDigest || state.History == nil || state.History.BaselineRevision != baseline || state.History.Revision != changed || len(state.Briefings) != 1 {
		t.Fatalf("persisted history = %#v, digest=%s", state, digest)
	}
	again, err := EnsureAcceptedHistory(root, changed)
	if err != nil || again.StoreDigest != receipt.StoreDigest || len(again.Bundles) != 0 {
		t.Fatalf("idempotent ensure = %#v err=%v", again, err)
	}
	project, err := projectwork.Load(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	manager := bundle.Events[0].AffectedManagers[0]
	briefings, events, _, err := LoadForManager(root, project.Model.Digest, manager, changed)
	if err != nil || len(briefings) != 1 || len(events) != 1 {
		t.Fatalf("ensured manager history briefings=%d events=%d err=%v", len(briefings), len(events), err)
	}
}

func TestEnsureAcceptedHistoryAdvancesCodeOnlyAndIgnoresWorkingDraft(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	receipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	original, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	draft := strings.Replace(string(original), "before shipment", "before any dispatch", 1)
	if draft == string(original) {
		t.Fatal("could not create model draft")
	}
	if err := os.WriteFile(modelPath, []byte(draft), 0644); err != nil {
		t.Fatal(err)
	}
	draftReceipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil || draftReceipt.StoreDigest != receipt.StoreDigest || len(draftReceipt.Bundles) != 0 {
		t.Fatalf("working draft changed accepted history receipt=%#v err=%v", draftReceipt, err)
	}
	if err := os.WriteFile(modelPath, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Committed code-only change.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "documentation only")
	codeOnly := gitOutputTest(t, root, "rev-parse", "HEAD")
	advanced, err := EnsureAcceptedHistory(root, codeOnly)
	if err != nil || len(advanced.Bundles) != 0 || advanced.BaselineRevision != baseline || advanced.Revision != codeOnly {
		t.Fatalf("code-only commit did not advance cursor without event receipt=%#v err=%v", advanced, err)
	}
	state, _, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != codeOnly || len(state.Briefings) != 1 {
		t.Fatalf("code-only cursor state=%#v err=%v", state, err)
	}
}

func TestEnsureAcceptedHistorySurvivesCodeOnlyTopicBranchAndNoFFMerge(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Code-only topic change.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "code-only topic change")
	topic := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatalf("precondition: ensure on topic: %v", err)
	}
	gitTest(t, root, "checkout", mainBranch)
	_, onTopic, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	view, err := ReadAcceptedHistory(root, changed)
	if err != nil || view.State.History == nil || view.State.History.Revision != changed || view.PersistedDigest != onTopic {
		t.Errorf("read-only view did not derive main's cursor: view=%#v err=%v", view.State.History, err)
	}
	if _, after, err := Read(root); err != nil || after != onTopic {
		t.Fatalf("read-only view changed the store: before=%s after=%s err=%v", onTopic, after, err)
	}
	if receipt, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Errorf("ensure back on %s at unchanged model %s: %v", mainBranch, changed, err)
	} else if receipt.StoreDigest != view.Receipt.StoreDigest {
		t.Errorf("ensure persisted %s, read-only view computed %s", receipt.StoreDigest, view.Receipt.StoreDigest)
	}
	gitTest(t, root, "merge", "--no-ff", "-m", "merge topic", "topic")
	merged := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, merged); err != nil {
		t.Errorf("ensure after --no-ff merge %s: %v", merged, err)
	}
	state, _, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != merged || len(state.Briefings) != 1 {
		t.Fatalf("merged cursor state=%#v err=%v", state.History, err)
	}
}

func TestEnsureAcceptedHistoryKeepsTopicModelChangeProvisionalOnDivergedMain(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	topic := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "model change accepted only on topic")
	if receipt, err := EnsureAcceptedHistory(root, topic); err != nil || len(receipt.Bundles) != 1 {
		t.Fatalf("precondition: topic model change was not briefed: receipt=%#v err=%v", receipt, err)
	}
	gitTest(t, root, "checkout", mainBranch)
	diverged := commitReadmeTest(t, root, "Main diverges.\n", "main diverges")
	view, err := ReadAcceptedHistory(root, diverged)
	if err != nil || len(view.State.Briefings) != 1 || view.State.Briefings[0].Revision != changed {
		t.Fatalf("read-only view counted the topic's provisional briefing or was blocked by it: %#v err=%v", view.State.Briefings, err)
	}
	if receipt, err := EnsureAcceptedHistory(root, diverged); err != nil {
		t.Fatalf("a model change briefed only on an unmerged topic blocked main: %v", err)
	} else if receipt.StoreDigest != view.Receipt.StoreDigest {
		t.Fatalf("ensure persisted %s, read-only view computed %s", receipt.StoreDigest, view.Receipt.StoreDigest)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Briefings) != 1 || state.Briefings[0].Revision != changed {
		t.Fatalf("main counted the topic's provisional briefing: %#v err=%v", state.Briefings, err)
	}
	if stored := storedBriefingsTest(t, root); len(stored) != 2 {
		t.Fatalf("provisional topic briefing was not kept: %d stored", len(stored))
	}
	gitTest(t, root, "checkout", "topic")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatalf("ensure back on the topic: %v", err)
	}
	if state, _, err := Read(root); err != nil || len(state.Briefings) != 2 {
		t.Fatalf("topic lost its own briefing: %#v err=%v", state.Briefings, err)
	}
}

func TestEnsureAcceptedHistoryAcceptsMergedTopicModelChangeOnce(t *testing.T) {
	for _, merge := range []string{"no-ff", "squash", "rebase", "fast-forward"} {
		t.Run(merge, func(t *testing.T) {
			root, _, changed := committedModelFixture(t)
			mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
			if _, err := EnsureAcceptedHistory(root, changed); err != nil {
				t.Fatal(err)
			}
			gitTest(t, root, "checkout", "-b", "topic")
			topic := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "topic model change")
			receipt, err := EnsureAcceptedHistory(root, topic)
			if err != nil || len(receipt.Bundles) != 1 {
				t.Fatalf("precondition: topic model change was not briefed: receipt=%#v err=%v", receipt, err)
			}
			topicDigest := receipt.ModelDigest
			gitTest(t, root, "checkout", mainBranch)
			if merge != "fast-forward" {
				commitReadmeTest(t, root, "Main moves on.\n", "main moves on")
			}
			switch merge {
			case "no-ff":
				gitTest(t, root, "merge", "--no-ff", "-m", "merge topic", "topic")
			case "squash":
				gitTest(t, root, "merge", "--squash", "topic")
				gitCommitTest(t, root, "squash topic")
			case "rebase":
				gitTest(t, root, "checkout", "topic")
				gitTest(t, root, "rebase", mainBranch)
				gitTest(t, root, "checkout", mainBranch)
				gitTest(t, root, "merge", "--ff-only", "topic")
			case "fast-forward":
				gitTest(t, root, "merge", "--ff-only", "topic")
			}
			merged := gitOutputTest(t, root, "rev-parse", "HEAD")
			if _, err := EnsureAcceptedHistory(root, merged); err != nil {
				t.Fatalf("ensure on %s after %s merge: %v", mainBranch, merge, err)
			}
			if again, err := EnsureAcceptedHistory(root, merged); err != nil || len(again.Bundles) != 0 {
				t.Fatalf("repeated ensure after %s merge: receipt=%#v err=%v", merge, again, err)
			}
			state, _, err := Read(root)
			if err != nil {
				t.Fatal(err)
			}
			var accepted []Bundle
			for _, bundle := range state.Briefings {
				if bundle.ModelDigest == topicDigest {
					accepted = append(accepted, bundle)
				}
			}
			// Only a fast-forward puts the topic commit on main's first-parent
			// line; every other merge records main's own briefing.
			wantRevision := merged
			if merge == "fast-forward" {
				wantRevision = topic
			}
			if len(state.Briefings) != 2 || len(accepted) != 1 || accepted[0].Revision != wantRevision {
				t.Fatalf("merged model change was not accepted exactly once at %s: %#v", wantRevision, state.Briefings)
			}
			wantStored := 3
			if merge == "fast-forward" {
				wantStored = 2
			}
			if stored := storedBriefingsTest(t, root); len(stored) != wantStored {
				t.Fatalf("stored briefings = %d, want %d (the topic's stays provisional)", len(stored), wantStored)
			}
		})
	}
}

func TestEnsureAcceptedHistoryStartsOwnBaselineAfterSquashedProjectInit(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "--initial-branch=feature-squashed-init")
	commitReadmeTest(t, root, "Before the project model.\n", "repository before the project")
	gitTest(t, root, "checkout", "-b", "topic")
	copyProjectWorldTest(t, root)
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "introduce the project model on a topic")
	topic := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatalf("precondition: topic baseline: %v", err)
	}
	gitTest(t, root, "checkout", "feature-squashed-init")
	gitTest(t, root, "merge", "--squash", "topic")
	gitCommitTest(t, root, "squash the project model")
	squashed := gitOutputTest(t, root, "rev-parse", "HEAD")
	receipt, err := EnsureAcceptedHistory(root, squashed)
	if err != nil || receipt.BaselineRevision != squashed || len(receipt.Bundles) != 0 {
		t.Fatalf("squashed project model did not start its own baseline: receipt=%#v err=%v", receipt, err)
	}
}

func TestEnsureAcceptedHistoryRejectsConflictingBriefingOnActiveLine(t *testing.T) {
	root, base, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	third := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "third model value")
	aggregate, err := Generate(root, base, third, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored Store
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.Briefings = append(stored.Briefings, aggregate)
	if data, err = json.Marshal(stored); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, before, err := Read(root)
	if err != nil {
		t.Fatalf("precondition: injected briefing must be a valid store entry: %v", err)
	}
	if _, err := ReadAcceptedHistory(root, third); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("read-only view accepted a briefing that skips a transition on the active line: %v", err)
	}
	if _, err := EnsureAcceptedHistory(root, third); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("briefing that skips a transition on the active line was accepted: %v", err)
	}
	if _, after, err := Read(root); err != nil || after != before {
		t.Fatalf("rejected ambiguous history changed the store: before=%s after=%s err=%v", before, after, err)
	}
}

func TestReadAcceptedHistoryWritesNothingAndMatchesEnsure(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	stateDir := filepath.Join(root, ".markitect", "state")
	view, err := ReadAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("read-only accepted history created operational state: %v", err)
	}
	if view.PersistedDigest != StoreDigest(emptyStore()) || view.Receipt.BaselineRevision != baseline || view.Receipt.Revision != changed || len(view.Receipt.Bundles) != 1 || len(view.State.Briefings) != 1 {
		t.Fatalf("read-only view of a fresh repository = %#v", view.Receipt)
	}
	project, err := projectwork.Load(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	manager := view.Receipt.Bundles[0].Events[0].AffectedManagers[0]
	_, events, viewBinding, err := LoadForManagerFrom(root, view.State, project.Model.Digest, manager, changed)
	if err != nil || len(events) != 1 {
		t.Fatalf("manager briefing from the read-only view events=%d err=%v", len(events), err)
	}
	receipt, err := EnsureAcceptedHistory(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	if hash(receipt) != hash(view.Receipt) {
		t.Fatalf("ensure persisted a different answer than the read-only view:\nensure=%#v\nview=%#v", receipt, view.Receipt)
	}
	state, digest, err := Read(root)
	if err != nil || digest != view.Receipt.StoreDigest || hash(state) != hash(view.State) {
		t.Fatalf("persisted history differs from the read-only view: digest=%s view=%s err=%v", digest, view.Receipt.StoreDigest, err)
	}
	if _, _, binding, err := LoadForManager(root, project.Model.Digest, manager, changed); err != nil || binding != viewBinding {
		t.Fatalf("persisted manager briefing %s differs from the read-only view %s: %v", binding, viewBinding, err)
	}
	older, err := ReadAcceptedHistory(root, baseline)
	if err != nil || older.PersistedDigest != digest || older.Receipt.StoreDigest != digest || older.Receipt.Revision != baseline || len(older.Receipt.Bundles) != 0 {
		t.Fatalf("read-only view behind the persisted cursor = %#v err=%v", older.Receipt, err)
	}
	gitTest(t, root, "commit", "--allow-empty", "-m", "code-only change after ensure")
	head := gitOutputTest(t, root, "rev-parse", "HEAD")
	newer, err := ReadAcceptedHistory(root, head)
	if err != nil || newer.PersistedDigest != digest || newer.State.History == nil || newer.State.History.Revision != head || len(newer.Receipt.Bundles) != 0 {
		t.Fatalf("read-only view ahead of the persisted cursor = %#v err=%v", newer.Receipt, err)
	}
	if _, after, err := Read(root); err != nil || after != digest {
		t.Fatalf("read-only view changed the persisted store: before=%s after=%s err=%v", digest, after, err)
	}
	if ensured, err := EnsureAcceptedHistory(root, head); err != nil || ensured.StoreDigest != newer.Receipt.StoreDigest {
		t.Fatalf("ensure after the read-only view = %#v err=%v, view digest %s", ensured, err, newer.Receipt.StoreDigest)
	}
}

func TestReadAcceptedHistoryChainsSuccessiveUnpersistedChangesLikeEnsure(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	first, err := EnsureAcceptedHistory(root, changed)
	if err != nil || len(first.Bundles) != 1 || len(first.Bundles[0].Events) != 1 {
		t.Fatalf("precondition: first accepted change receipt=%#v err=%v", first, err)
	}
	commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "second change, not yet persisted")
	head := commitModelChangeTest(t, root, "prior to fulfillment", "before any dispatch", "third change, not yet persisted")
	view, err := ReadAcceptedHistory(root, head)
	if err != nil || len(view.Receipt.Bundles) != 2 {
		t.Fatalf("read-only view of two unpersisted changes = %#v err=%v", view.Receipt, err)
	}
	second, third := view.Receipt.Bundles[0].Events, view.Receipt.Bundles[1].Events
	if len(second) != 1 || len(third) != 1 || second[0].Predecessor != first.Bundles[0].Events[0].ID || third[0].Predecessor != second[0].ID {
		t.Fatalf("read-only view did not chain each change to the one before it: second=%#v third=%#v", second, third)
	}
	receipt, err := EnsureAcceptedHistory(root, head)
	if err != nil || hash(receipt) != hash(view.Receipt) {
		t.Fatalf("ensure persisted other events than the read-only view: ensure=%#v err=%v", receipt, err)
	}
	if state, _, err := Read(root); err != nil || hash(state) != hash(view.State) {
		t.Fatalf("persisted history differs from the read-only view: err=%v", err)
	}
	if again, err := EnsureAcceptedHistory(root, head); err != nil || len(again.Bundles) != 0 {
		t.Fatalf("persisted chain does not validate on the next ensure: receipt=%#v err=%v", again, err)
	}
}

func TestEnsureAcceptedHistoryChainsChangeRevertAndReapplyInOneCall(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	persisted, err := EnsureAcceptedHistory(root, changed)
	if err != nil || len(persisted.Bundles) != 1 || len(persisted.Bundles[0].Events) != 1 {
		t.Fatalf("precondition: persisted change receipt=%#v err=%v", persisted, err)
	}
	commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "change A to B")
	commitModelChangeTest(t, root, "prior to fulfillment", "before shipment", "revert B to A")
	head := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "re-apply A to B")
	view, err := ReadAcceptedHistory(root, head)
	if err != nil {
		t.Fatalf("read-only view of change, revert and re-apply: %v", err)
	}
	receipt, err := EnsureAcceptedHistory(root, head)
	if err != nil || len(receipt.Bundles) != 3 {
		t.Fatalf("one ensure over change, revert and re-apply: receipt=%#v err=%v", receipt, err)
	}
	events := make([]Event, 0, 3)
	for i, bundle := range receipt.Bundles {
		if len(bundle.Events) != 1 || len(view.Receipt.Bundles) != 3 || len(view.Receipt.Bundles[i].Events) != 1 || view.Receipt.Bundles[i].Events[0].ID != bundle.Events[0].ID {
			t.Fatalf("briefing %d: ensure events %#v, read-only view %#v", i, bundle.Events, view.Receipt.Bundles)
		}
		events = append(events, bundle.Events[0])
	}
	change, revert, reapply := events[0], events[1], events[2]
	if change.ID == revert.ID || revert.ID == reapply.ID || change.ID == reapply.ID {
		t.Fatalf("change, revert and re-apply share event IDs: %s %s %s", change.ID, revert.ID, reapply.ID)
	}
	if change.Predecessor != persisted.Bundles[0].Events[0].ID || revert.Predecessor != change.ID || reapply.Predecessor != revert.ID {
		t.Fatalf("predecessors do not chain re-apply -> revert -> change -> persisted: %q %q %q", change.Predecessor, revert.Predecessor, reapply.Predecessor)
	}
	if again, err := EnsureAcceptedHistory(root, head); err != nil || len(again.Bundles) != 0 {
		t.Fatalf("ensure after the chained call: receipt=%#v err=%v", again, err)
	}
	after, err := ReadAcceptedHistory(root, head)
	if err != nil || len(after.Receipt.Bundles) != 0 {
		t.Fatalf("read-only view after the chained call: receipt=%#v err=%v", after.Receipt, err)
	}
	stored := map[string]bool{}
	for _, bundle := range after.State.Briefings {
		for _, event := range bundle.Events {
			stored[event.ID] = true
		}
	}
	for _, event := range events {
		if !stored[event.ID] {
			t.Fatalf("read-only view after the call lacks event %s", event.ID)
		}
	}
}

func TestReadAcceptedHistorySucceedsInParallelAfterNewCommits(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	_, persisted, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	head := commitNewAcceptedHistoryTest(t, root)
	const parallel = 6
	views := make([]AcceptedHistory, parallel)
	errs := make([]error, parallel)
	var wait sync.WaitGroup
	for i := range parallel {
		wait.Add(1)
		go func() {
			defer wait.Done()
			views[i], errs[i] = ReadAcceptedHistory(root, head)
		}()
	}
	wait.Wait()
	for i := range parallel {
		if errs[i] != nil {
			t.Errorf("parallel read-only accepted history %d failed: %v", i, errs[i])
			continue
		}
		if views[i].Receipt.StoreDigest != views[0].Receipt.StoreDigest || views[i].State.History == nil || views[i].State.History.Revision != head || len(views[i].Receipt.Bundles) != 1 {
			t.Errorf("parallel read-only view %d = %#v", i, views[i].Receipt)
		}
	}
	if _, after, err := Read(root); err != nil || after != persisted {
		t.Fatalf("parallel reads changed the persisted store: before=%s after=%s err=%v", persisted, after, err)
	}
	assertOnlyBriefingStoreFileTest(t, root)
}

func TestEnsureAcceptedHistoryParallelWritersConvergeOnOneStore(t *testing.T) {
	root, _, _ := committedModelFixture(t)
	head := commitNewAcceptedHistoryTest(t, root)
	const writers, readers = 6, 2
	receipts := make([]EnsureReceipt, writers)
	views := make([]AcceptedHistory, readers)
	errs := make([]error, writers+readers)
	var wait sync.WaitGroup
	for i := range writers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			receipts[i], errs[i] = EnsureAcceptedHistory(root, head)
		}()
	}
	for i := range readers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			views[i], errs[writers+i] = ReadAcceptedHistory(root, head)
		}()
	}
	wait.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("parallel accepted-history call %d failed: %v", i, err)
		}
	}
	if t.Failed() {
		return
	}
	state, digest, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != head || len(state.Briefings) != 2 {
		t.Fatalf("parallel writers left history=%#v briefings=%d err=%v", state.History, len(state.Briefings), err)
	}
	added := 0
	for i, receipt := range receipts {
		if receipt.StoreDigest != digest || receipt.Revision != head {
			t.Errorf("writer %d receipt digest=%s revision=%s, store digest=%s", i, receipt.StoreDigest, receipt.Revision, digest)
		}
		added += len(receipt.Bundles)
	}
	if added != len(state.Briefings) {
		t.Errorf("writers reported %d added briefings, want each of %d exactly once", added, len(state.Briefings))
	}
	for i, view := range views {
		if view.Receipt.StoreDigest != digest {
			t.Errorf("reader %d computed %s, writers persisted %s", i, view.Receipt.StoreDigest, digest)
		}
	}
	assertOnlyBriefingStoreFileTest(t, root)
}

func TestBriefingStoreReadersAndWritersSurviveConcurrentReplacement(t *testing.T) {
	root := t.TempDir()
	const writers, writes, readers, reads = 4, 25, 4, 250
	errs := make(chan error, writers*writes+readers*reads)
	var wait sync.WaitGroup
	for w := range writers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for n := range writes {
				_, digest, err := readStore(root)
				if err != nil {
					errs <- err
					continue
				}
				revision := fmt.Sprintf("%040x", w*writes+n+1)
				if _, err := update(root, digest, func(state *Store) error {
					state.History = &HistoryCursor{Policy: acceptedPolicy, BaselineRevision: revision, BaselineModelDigest: "baseline", Revision: revision, ModelDigest: "model"}
					return nil
				}); err != nil && !errors.Is(err, ErrStaleStore) {
					errs <- err
				}
			}
		}()
	}
	for range readers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range reads {
				if _, _, err := readStore(root); err != nil {
					errs <- err
				}
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent store access failed: %v", err)
	}
	assertOnlyBriefingStoreFileTest(t, root)
}

// commitNewAcceptedHistoryTest commits code-only changes around one more
// accepted model change and returns the new HEAD.
func commitNewAcceptedHistoryTest(t *testing.T, root string) string {
	t.Helper()
	for i := range 3 {
		gitTest(t, root, "commit", "--allow-empty", "-m", fmt.Sprintf("code-only change %d", i))
	}
	commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "further accepted model change")
	for i := range 3 {
		gitTest(t, root, "commit", "--allow-empty", "-m", fmt.Sprintf("later code-only change %d", i))
	}
	return gitOutputTest(t, root, "rev-parse", "HEAD")
}

func assertOnlyBriefingStoreFileTest(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".markitect", "state", "briefings"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "history.json" {
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("briefing store directory holds %v, want only history.json", names)
	}
}

func commitModelChangeTest(t *testing.T, root, old, replacement, message string) string {
	t.Helper()
	const relative = ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml"
	path := filepath.Join(root, filepath.FromSlash(relative))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(content), old, replacement, 1)
	if updated == string(content) {
		t.Fatalf("could not replace %q in the fixture model", old)
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", relative)
	gitCommitTest(t, root, message)
	return gitOutputTest(t, root, "rev-parse", "HEAD")
}

func commitReadmeTest(t *testing.T, root, content, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, message)
	return gitOutputTest(t, root, "rev-parse", "HEAD")
}

// storedBriefingsTest returns every persisted briefing, provisional ones
// included, straight from the store file.
func storedBriefingsTest(t *testing.T, root string) []Bundle {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(storePath)))
	if err != nil {
		t.Fatal(err)
	}
	var stored Store
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	return stored.Briefings
}

func TestEnsureAcceptedHistoryPreservesRevertAndRejectsManualAggregateWrite(t *testing.T) {
	root, baseline, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	original, err := sourceFileAtRevision(t, root, baseline, ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, original, 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "revert accepted change")
	reverted := gitOutputTest(t, root, "rev-parse", "HEAD")
	receipt, err := EnsureAcceptedHistory(root, reverted)
	if err != nil || len(receipt.Bundles) != 1 || receipt.Bundles[0].SinceRevision != changed || receipt.Bundles[0].Revision != reverted {
		t.Fatalf("revert was not recorded as its own adjacent event: receipt=%#v err=%v", receipt, err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 2 {
		t.Fatalf("history did not retain both transitions: %#v err=%v", state, err)
	}
	aggregate, err := Generate(root, baseline, reverted, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, aggregate, digest); !errors.Is(err, ErrAmbiguousHistory) && !errors.Is(err, ErrStaleModel) {
		t.Fatalf("manual aggregate write crossed accepted cursor: %v", err)
	}
}

func TestEnsureAcceptedHistoryRequiresCommittedCompleteValidActiveHistory(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, strings.Repeat("a", 40)); !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("non-HEAD revision error = %v", err)
	}
	path := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	if err := os.WriteFile(path, []byte("not: [valid"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "invalid committed model")
	invalid := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, invalid); err == nil || errors.Is(err, ErrNoAcceptedModel) {
		t.Fatalf("invalid accepted intermediate model was not rejected: %v", err)
	}
	historical, err := EnsureAcceptedHistory(root, revision)
	if err != nil || historical.Revision != revision {
		t.Fatalf("valid selected ancestor on the active branch was rejected: receipt=%#v err=%v", historical, err)
	}
}

func TestEnsureAcceptedHistoryRejectsBrokenStatementReferencesAtBaselineAndIntermediate(t *testing.T) {
	setMissingStatementReference := func(t *testing.T, root string) {
		t.Helper()
		path := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(content), "  requires: []", "  requires:\n    - namespace: commerce.sales.orders\n      name: missing-statement", 1)
		if updated == string(content) {
			t.Fatal("could not add missing Statement reference")
		}
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("baseline", func(t *testing.T) {
		root, baseline, _ := committedModelFixture(t)
		gitTest(t, root, "checkout", "--orphan", "feature-invalid-baseline")
		setMissingStatementReference(t, root)
		gitTest(t, root, "add", ".")
		gitCommitTest(t, root, "invalid first project model")
		revision := gitOutputTest(t, root, "rev-parse", "HEAD")
		if _, err := EnsureAcceptedHistory(root, revision); err == nil {
			t.Fatalf("broken baseline model was accepted (prior valid revision %s): %v", baseline, err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
			t.Fatalf("invalid baseline created accepted history: %v", err)
		}
	})

	t.Run("intermediate", func(t *testing.T) {
		root, _, _ := committedModelFixture(t)
		setMissingStatementReference(t, root)
		gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
		gitCommitTest(t, root, "invalid committed reference")
		revision := gitOutputTest(t, root, "rev-parse", "HEAD")
		if _, err := EnsureAcceptedHistory(root, revision); err == nil {
			t.Fatalf("broken intermediate model was accepted: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
			t.Fatalf("invalid intermediate created accepted history: %v", err)
		}
	})
}

func TestEnsureAcceptedHistoryRejectsAnalyzedStructuralErrorsButAllowsIncompleteFindings(t *testing.T) {
	root, _, _ := committedModelFixture(t)
	artifactPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "artifacts.yaml")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	withMissingImplementation := strings.Replace(string(artifact), "src/shop/orders/", "src/shop/not-yet-implemented-orders/", 1)
	if withMissingImplementation == string(artifact) {
		t.Fatal("could not create expected incomplete implementation path")
	}
	if err := os.WriteFile(artifactPath, []byte(withMissingImplementation), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/artifacts.yaml")
	gitCommitTest(t, root, "expected implementation gap")
	incompleteRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	incompleteProject, err := projectwork.Load(root, incompleteRevision)
	if err != nil {
		t.Fatal(err)
	}
	if !hasIncompleteFinding(incompleteProject.Report.Findings) {
		t.Fatalf("fixture no longer covers implementation/coverage incompleteness: %#v", incompleteProject.Report.Findings)
	}
	if _, err := EnsureAcceptedHistory(root, incompleteRevision); err != nil {
		t.Fatalf("incomplete implementation/coverage was treated as an invalid canonical model: %v", err)
	}
	priorState, priorDigest, err := Read(root)
	if err != nil || priorState.History == nil || priorState.History.Revision != incompleteRevision {
		t.Fatalf("incomplete valid model did not establish accepted cursor: state=%#v err=%v", priorState.History, err)
	}
	statementPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "inventory", "reservations", "release-reservation.yaml")
	statement, err := os.ReadFile(statementPath)
	if err != nil {
		t.Fatal(err)
	}
	privateStatement := strings.Replace(string(statement), "  public: true", "  public: false", 1)
	if privateStatement == string(statement) {
		t.Fatal("could not make referenced Statement private")
	}
	if err := os.WriteFile(statementPath, []byte(privateStatement), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/inventory/reservations/release-reservation.yaml")
	gitCommitTest(t, root, "cross-manager private reference")
	revision := gitOutputTest(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatalf("Core-valid model should reach project structural analysis: %v", err)
	}
	if project.Model.Digest == "" || !hasStructuralError(project.Report.Findings) {
		t.Fatalf("fixture did not produce a compiled model with a structural report error: digest=%q findings=%#v", project.Model.Digest, project.Report.Findings)
	}
	if _, err := EnsureAcceptedHistory(root, revision); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("structurally invalid committed model was accepted: %v", err)
	}
	state, digest, err := Read(root)
	if err != nil || state.History == nil || state.History.Revision != incompleteRevision || digest != priorDigest {
		t.Fatalf("structurally invalid model advanced accepted history: state=%#v digest=%s err=%v", state.History, digest, err)
	}
}

func hasStructuralError(findings []projectmodel.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func hasIncompleteFinding(findings []projectmodel.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "incomplete" {
			return true
		}
	}
	return false
}

func TestEnsureAcceptedHistoryRejectsMissingProjectAndShallowAncestry(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "--initial-branch=feature-no-project")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("not a project\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "ordinary repository")
	noProject := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, noProject); !errors.Is(err, ErrNoAcceptedModel) {
		t.Fatalf("repository without project model error = %v", err)
	}
	root, _, revision := committedModelFixture(t)
	shallowFile := filepath.Join(root, ".git", "shallow")
	if err := os.WriteFile(shallowFile, []byte(revision+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureAcceptedHistory(root, revision); !errors.Is(err, ErrAmbiguousHistory) {
		t.Fatalf("shallow history error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("incomplete history created an accepted baseline: %v", err)
	}
}

func TestEnsureAcceptedHistoryFirstModelIsBaselineWithoutInventedChange(t *testing.T) {
	root, baseline, _ := committedModelFixture(t)
	gitTest(t, root, "checkout", baseline)
	project, err := projectwork.Load(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := EnsureAcceptedHistory(root, baseline)
	if err != nil || receipt.BaselineRevision != baseline || receipt.Revision != baseline || len(receipt.Bundles) != 0 {
		t.Fatalf("first committed project model bootstrap receipt=%#v err=%v", receipt, err)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Briefings) != 0 || state.History == nil {
		t.Fatalf("initial model was invented as a change: %#v err=%v", state, err)
	}
	manager := project.Report.Managers[0].ID
	briefings, events, _, err := LoadForManager(root, project.Model.Digest, manager, baseline)
	if err != nil || len(briefings) != 0 || len(events) != 0 {
		t.Fatalf("baseline manager context briefings=%d events=%d err=%v", len(briefings), len(events), err)
	}
}

func TestEnsureAcceptedHistoryInitWaitsForFirstCommittedModel(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "--initial-branch=feature-init-history")
	if _, err := EnsureAcceptedHistory(root, strings.Repeat("a", 40)); !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("unborn repository error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := projectwork.Init(root, "history-bootstrap", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("uncommitted init unexpectedly seeded accepted history: %v", err)
	}
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "initialize project")
	revision := gitOutputTest(t, root, "rev-parse", "HEAD")
	receipt, err := EnsureAcceptedHistory(root, revision)
	if err != nil || receipt.BaselineRevision != revision || receipt.Revision != revision || len(receipt.Bundles) != 0 {
		t.Fatalf("first committed init baseline receipt=%#v err=%v", receipt, err)
	}
}

func TestVerifiedResolutionIsCandidateBoundSeparateFromDismissal(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 1 || len(state.Briefings[0].Events) == 0 {
		t.Fatalf("briefing history state=%#v err=%v", state, err)
	}
	event := state.Briefings[0].Events[0]
	manager := event.AffectedManagers[0]
	digest, err = Dismiss(root, event.ID, manager, digest)
	if err != nil {
		t.Fatal(err)
	}
	state, digest, err = Read(root)
	if err != nil || EventResolutionStatus(state, event.ID).Status != "unresolved" {
		t.Fatalf("dismissal changed resolution status: status=%#v err=%v", EventResolutionStatus(state, event.ID), err)
	}
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(event.ID, project.Report.Managers)
	if _, err := ResolveVerified(root, revision, "stale-model-digest", evidence, digest); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("stale selected model resolved event: %v", err)
	}
	incomplete := evidence
	incomplete.FullVerifyPassed = false
	if _, err := ResolveVerified(root, revision, project.Model.Digest, incomplete, digest); !errors.Is(err, ErrResolution) {
		t.Fatalf("failed full Verify resolved event: %v", err)
	}
	missingManager := evidence
	missingManager.CoveredManagerIDs = missingManager.CoveredManagerIDs[:len(missingManager.CoveredManagerIDs)-1]
	if _, err := ResolveVerified(root, revision, project.Model.Digest, missingManager, digest); !errors.Is(err, ErrResolution) {
		t.Fatalf("partial Manager coverage resolved event: %v", err)
	}
	newDigest, err := ResolveVerified(root, revision, project.Model.Digest, evidence, digest)
	if err != nil {
		t.Fatal(err)
	}
	state, current, err := Read(root)
	if err != nil || current != newDigest || len(state.Dismissals) != 1 || len(state.Resolutions) != 1 {
		t.Fatalf("resolution/dismissal store state=%#v digest=%s err=%v", state, current, err)
	}
	status := EventResolutionStatus(state, event.ID)
	if status.Status != "resolved" || status.Resolution == nil || status.Resolution.Evidence.ApplyDigest != evidence.ApplyDigest {
		t.Fatalf("event resolution status=%#v", status)
	}
	repeatedDigest, err := ResolveVerified(root, revision, project.Model.Digest, evidence, current)
	if err != nil || repeatedDigest != current {
		t.Fatalf("identical resolution was not idempotent: digest=%s err=%v", repeatedDigest, err)
	}
	conflicting := evidence
	conflicting.RunID = "another-run"
	if _, err := ResolveVerified(root, revision, project.Model.Digest, conflicting, current); !errors.Is(err, ErrAlreadyResolved) {
		t.Fatalf("different candidate evidence replaced resolution: %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state.Resolutions[0].Evidence.ApplyDigest = "tampered-apply-digest"
	data, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(root); !errors.Is(err, ErrResolution) {
		t.Fatalf("tampered resolution evidence passed store validation: %v", err)
	}
}

func TestVerifiedResolutionRejectsFutureEventAtOldModelRevision(t *testing.T) {
	root, _, firstRevision := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, firstRevision); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	content, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	thirdValue := strings.Replace(string(content), "before shipment", "prior to fulfillment", 1)
	if thirdValue == string(content) {
		t.Fatal("could not create future model revision")
	}
	if err := os.WriteFile(firstPath, []byte(thirdValue), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "later model change")
	futureRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	if _, err := EnsureAcceptedHistory(root, futureRevision); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 2 {
		t.Fatalf("future history state=%#v err=%v", state, err)
	}
	var futureEvent Event
	for _, bundle := range state.Briefings {
		if bundle.Revision == futureRevision && len(bundle.Events) > 0 {
			futureEvent = bundle.Events[0]
		}
	}
	if futureEvent.ID == "" {
		t.Fatal("future accepted event was not persisted")
	}
	oldProject, err := projectwork.Load(root, firstRevision)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(futureEvent.ID, oldProject.Report.Managers)
	if _, err := ResolveVerified(root, firstRevision, oldProject.Model.Digest, evidence, digest); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("future event was resolved against an older model: %v", err)
	}
}

func TestUnmergedTopicResolutionStaysProvisionalWhileDismissalFollowsEvent(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	topic := commitReadmeTest(t, root, "Code-only topic change.\n", "code-only topic change")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil || len(state.Briefings) != 1 {
		t.Fatalf("topic history state=%#v err=%v", state.Briefings, err)
	}
	event := state.Briefings[0].Events[0]
	if digest, err = Dismiss(root, event.ID, event.AffectedManagers[0], digest); err != nil {
		t.Fatal(err)
	}
	resolveDeliveredTest(t, root, topic, event.ID, "run-topic", deliveredPathTest, "# Delivered on the topic.\n")
	gitTest(t, root, "add", deliveredPathTest)
	gitCommitTest(t, root, "commit the delivery")
	gitTest(t, root, "checkout", mainBranch)
	state, _, err = Read(root)
	if err != nil {
		t.Fatal(err)
	}
	// The event is accepted on main too, so its dismissal is; the delivery is
	// not.
	if status := EventResolutionStatus(state, event.ID); status.Status != "unresolved" || len(state.Dismissals) != 1 || len(state.Briefings) != 1 {
		t.Fatalf("unmerged topic resolution counted or dismissal lost on %s: status=%#v dismissals=%#v", mainBranch, status, state.Dismissals)
	}
	gitTest(t, root, "merge", "--no-ff", "-m", "merge topic", "topic")
	state, _, err = Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if status := EventResolutionStatus(state, event.ID); status.Status != "resolved" || status.Resolution.ModelRevision != topic || len(state.Dismissals) != 1 {
		t.Fatalf("merged topic resolution or dismissal not accepted: status=%#v dismissals=%#v", status, state.Dismissals)
	}
}

func TestVerifiedResolutionOnUnmergedTopicDoesNotBlockMain(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	topic := commitReadmeTest(t, root, "Code-only topic change.\n", "code-only topic change")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatal(err)
	}
	state, _, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	project, err := projectwork.Load(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	resolveDeliveredTest(t, root, topic, event.ID, "run-topic", deliveredPathTest, "# Delivered on the topic.\n")
	gitTest(t, root, "add", deliveredPathTest)
	gitCommitTest(t, root, "commit the delivery")
	gitTest(t, root, "checkout", mainBranch)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(event.ID, project.Report.Managers)
	evidence.RunID = "run-on-main"
	if _, err := ResolveVerified(root, changed, project.Model.Digest, evidence, digest); err != nil {
		t.Fatalf("provisional topic resolution blocked resolving on %s: %v", mainBranch, err)
	}
	state, _, err = Read(root)
	if status := EventResolutionStatus(state, event.ID); err != nil || status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-on-main" {
		t.Fatalf("main resolution status=%#v err=%v", status, err)
	}
}

func TestDeliveryOnFreshTopicResolvesOnlyWhereItsResultIsCommitted(t *testing.T) {
	for _, merge := range []string{"merge-commit", "squash"} {
		t.Run(merge, func(t *testing.T) {
			root, _, changed := committedModelFixture(t)
			mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
			if _, err := EnsureAcceptedHistory(root, changed); err != nil {
				t.Fatal(err)
			}
			gitTest(t, root, "checkout", "-b", "topic")
			state, _, err := Read(root)
			if err != nil {
				t.Fatal(err)
			}
			event := state.Briefings[0].Events[0]
			resolveDeliveredTest(t, root, changed, event.ID, "run-topic", deliveredPathTest, "# Delivered on the topic.\n")
			gitTest(t, root, "add", deliveredPathTest)
			gitCommitTest(t, root, "commit the delivery")
			if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" {
				t.Fatalf("delivery is not resolved on its own topic: %#v", status)
			}
			gitTest(t, root, "checkout", mainBranch)
			if status := resolutionStatusTest(t, root, event.ID); status.Status != "unresolved" {
				t.Fatalf("delivery from a topic without commits of its own counted on %s: %#v", mainBranch, status)
			}
			mergeTopicTest(t, root, merge)
			if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-topic" {
				t.Fatalf("delivery is not resolved on %s after a %s: %#v", mainBranch, merge, status)
			}
		})
	}
}

func TestResolutionOfMergedTopicModelChangeCarriesToMain(t *testing.T) {
	for _, merge := range []string{"merge-commit", "squash"} {
		t.Run(merge, func(t *testing.T) {
			root, _, changed := committedModelFixture(t)
			mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
			if _, err := EnsureAcceptedHistory(root, changed); err != nil {
				t.Fatal(err)
			}
			gitTest(t, root, "checkout", "-b", "topic")
			topic := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "topic model change")
			receipt, err := EnsureAcceptedHistory(root, topic)
			if err != nil || len(receipt.Bundles) != 1 {
				t.Fatalf("precondition: topic model change was not briefed: receipt=%#v err=%v", receipt, err)
			}
			event := receipt.Bundles[0].Events[0]
			resolveDeliveredTest(t, root, topic, event.ID, "run-topic", deliveredPathTest, "# Delivered for the topic model change.\n")
			gitTest(t, root, "add", deliveredPathTest)
			gitCommitTest(t, root, "commit the delivery")
			gitTest(t, root, "checkout", mainBranch)
			commitReadmeTest(t, root, "Main moves on.\n", "main moves on")
			mergeTopicTest(t, root, merge)
			merged := gitOutputTest(t, root, "rev-parse", "HEAD")
			receipt, err = EnsureAcceptedHistory(root, merged)
			if err != nil || len(receipt.Bundles) != 1 || receipt.Bundles[0].Revision != merged {
				t.Fatalf("main did not brief the merged model change itself: receipt=%#v err=%v", receipt, err)
			}
			if ids := receipt.Bundles[0].Global.EventIDs; len(ids) != 1 || ids[0] != event.ID {
				t.Fatalf("the same model change got another identity on %s: %v, topic %s", mainBranch, ids, event.ID)
			}
			if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-topic" {
				t.Fatalf("topic resolution does not carry to %s after a %s: %#v", mainBranch, merge, status)
			}
		})
	}
}

func TestSquashedTopicResolutionRanksWhereItsResultEnteredTheLine(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	state, _, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	const topicPath = "src/shop/inventory/reservations.py"
	gitTest(t, root, "checkout", "-b", "topic")
	topicBase := commitReadmeTest(t, root, "Topic work.\n", "topic work")
	if _, err := EnsureAcceptedHistory(root, topicBase); err != nil {
		t.Fatal(err)
	}
	resolveDeliveredTest(t, root, topicBase, event.ID, "run-topic", topicPath, "# Topic delivery.\n")
	gitTest(t, root, "add", topicPath)
	gitCommitTest(t, root, "commit the topic delivery")
	gitTest(t, root, "checkout", mainBranch)
	resolveDeliveredTest(t, root, changed, event.ID, "run-main", deliveredPathTest, "# Main delivery.\n")
	gitTest(t, root, "add", deliveredPathTest)
	gitCommitTest(t, root, "commit the main delivery")
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-main" {
		t.Fatalf("precondition: main delivery status = %#v", status)
	}
	// The squash brings the topic's delivery onto the line after main's.
	mergeTopicTest(t, root, "squash")
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-topic" {
		t.Fatalf("squashed topic delivery did not rank where it entered the line: %#v", status)
	}
}

func TestEarlierStoreFormatIsRefusedWithRebuildInstruction(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const earlier = "markitect.example.org/project-model-briefing/v1alpha1"
	older := strings.ReplaceAll(string(data), APIVersion, earlier)
	if err := os.WriteFile(path, []byte(older), 0600); err != nil {
		t.Fatal(err)
	}
	read := func() error { _, _, err := Read(root); return err }
	ensure := func() error { _, err := EnsureAcceptedHistory(root, changed); return err }
	for name, call := range map[string]func() error{"read": read, "ensure": ensure} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), earlier) || !strings.Contains(err.Error(), "re-briefed from the first committed model") || !strings.Contains(err.Error(), "dismissals and resolutions are lost") {
			t.Fatalf("%s of an earlier store format: %v", name, err)
		}
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != older {
		t.Fatalf("refused store was changed: err=%v", err)
	}
}

func TestReappliedModelChangeGetsItsOwnEventID(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	reverted := commitModelChangeTest(t, root, "before shipment", "while the order is confirmed", "revert the model change")
	reapplied := commitModelChangeTest(t, root, "while the order is confirmed", "before shipment", "re-apply the model change")
	if _, err := EnsureAcceptedHistory(root, reapplied); err != nil {
		t.Fatal(err)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Briefings) != 3 {
		t.Fatalf("change, revert and re-apply were not briefed: %#v err=%v", state.Briefings, err)
	}
	// The briefing overview requires every event to belong to one revision.
	revisionOf := map[string]string{}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if prior, ok := revisionOf[event.ID]; ok && prior != bundle.Revision {
				t.Fatalf("event %s is attached to %s and %s (revert at %s)", event.ID, prior, bundle.Revision, reverted)
			}
			revisionOf[event.ID] = bundle.Revision
		}
	}
	if len(revisionOf) != 3 {
		t.Fatalf("event IDs are not distinct: %v", revisionOf)
	}
}

func TestUnreadableHeadTreeNeverResolvesDelivery(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	state, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	const removedPath = "src/shop/inventory/reservations.py"
	removed, err := NewDeliveredFile(root, removedPath, "", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(event.ID, project.Report.Managers)
	evidence.Delivered = []DeliveredFile{removed}
	if _, err := ResolveVerified(root, changed, project.Model.Digest, evidence, digest); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "rm", "-q", "--", removedPath)
	gitCommitTest(t, root, "commit the removal")
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" {
		t.Fatalf("precondition: committed removal status = %#v", status)
	}
	// Drop HEAD's root tree object so that reading HEAD's tree fails.
	tree := gitOutputTest(t, root, "rev-parse", "HEAD^{tree}")
	object := filepath.Join(root, ".git", "objects", tree[:2], tree[2:])
	if err := os.Chmod(object, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(object); err != nil {
		t.Fatal(err)
	}
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "unresolved" {
		t.Fatalf("delivery counted although HEAD's tree could not be read: %#v", status)
	}
}

func TestUncommittedDeliveryIsReportedButNotResolved(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	state, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	const removedPath = "src/shop/inventory/reservations.py"
	const content = "# Delivered, not yet committed.\n"
	written, err := NewDeliveredFile(root, deliveredPathTest, "100644", []byte(content), false)
	if err != nil {
		t.Fatal(err)
	}
	removed, err := NewDeliveredFile(root, removedPath, "", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(event.ID, project.Report.Managers)
	evidence.Delivered = []DeliveredFile{written, removed}
	recorded, err := ResolveVerified(root, changed, project.Model.Digest, evidence, digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(deliveredPathTest)), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "unresolved" {
		t.Fatalf("delivery whose removal is not in the working tree was reported: %#v", status)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(removedPath))); err != nil {
		t.Fatal(err)
	}
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "delivered-uncommitted" || status.Resolution == nil || status.Resolution.Evidence.RunID != evidence.RunID {
		t.Fatalf("uncommitted delivery status = %#v, want delivered-uncommitted", status)
	}
	if _, current, err := Read(root); err != nil || current != recorded {
		t.Fatalf("status hint changed the store: recorded=%s current=%s err=%v", recorded, current, err)
	}
	gitTest(t, root, "add", "--", deliveredPathTest, removedPath)
	gitCommitTest(t, root, "commit the delivery")
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" {
		t.Fatalf("committed delivery status = %#v, want resolved", status)
	}
	gitTest(t, root, "checkout", mainBranch)
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "unresolved" {
		t.Fatalf("delivery committed only on the topic reported on %s: %#v", mainBranch, status)
	}
}

func TestLatestAcceptedResolutionOnLineWins(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	state, _, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	later := commitReadmeTest(t, root, "Main moves on.\n", "main moves on")
	if _, err := EnsureAcceptedHistory(root, later); err != nil {
		t.Fatal(err)
	}
	// Record the later commit's delivery first, so recording order and line
	// order disagree.
	const otherPath = "src/shop/inventory/reservations.py"
	resolveDeliveredTest(t, root, later, event.ID, "run-at-later-commit", deliveredPathTest, "# Delivery at the later commit.\n")
	resolveDeliveredTest(t, root, changed, event.ID, "run-at-earlier-commit", otherPath, "# Delivery at the earlier commit.\n")
	gitTest(t, root, "add", deliveredPathTest, otherPath)
	gitCommitTest(t, root, "commit both deliveries")
	if status := resolutionStatusTest(t, root, event.ID); status.Status != "resolved" || status.Resolution.Evidence.RunID != "run-at-later-commit" {
		t.Fatalf("latest resolution on the line did not win: %#v", status)
	}
}

func TestDismissalFollowsItsEvent(t *testing.T) {
	root, base, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	state, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	event := state.Briefings[0].Events[0]
	if _, err := Dismiss(root, event.ID, event.AffectedManagers[0], digest); err != nil {
		t.Fatal(err)
	}
	// A dismissal counts wherever its event is accepted: on any branch or a
	// detached HEAD whose line holds the event, but not before the event.
	for _, check := range []struct {
		checkout []string
		want     int
	}{{[]string{"topic"}, 1}, {[]string{mainBranch}, 1}, {[]string{"--detach", changed}, 1}, {[]string{"--detach", base}, 0}} {
		gitTest(t, root, append([]string{"checkout", "-q"}, check.checkout...)...)
		if state, _, err := Read(root); err != nil || len(state.Dismissals) != check.want {
			t.Fatalf("dismissals after checkout %v = %#v, want %d (err=%v)", check.checkout, state.Dismissals, check.want, err)
		}
	}
}

// deliveredPathTest is an existing implementation file that test deliveries
// rewrite.
const deliveredPathTest = "src/shop/orders/order.py"

// resolveDeliveredTest records a verified resolution at revision for a
// delivery that wrote content to path, and leaves that content uncommitted in
// the working tree, as Apply does.
func resolveDeliveredTest(t *testing.T, root, revision, eventID, runID, path, content string) {
	t.Helper()
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validResolutionEvidence(eventID, project.Report.Managers)
	evidence.RunID = runID
	delivered, err := NewDeliveredFile(root, path, "100644", []byte(content), false)
	if err != nil {
		t.Fatal(err)
	}
	evidence.Delivered = []DeliveredFile{delivered}
	if _, err := ResolveVerified(root, revision, project.Model.Digest, evidence, digest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func resolutionStatusTest(t *testing.T, root, eventID string) ResolutionStatus {
	t.Helper()
	state, _, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	return EventResolutionStatus(state, eventID)
}

func mergeTopicTest(t *testing.T, root, merge string) {
	t.Helper()
	switch merge {
	case "merge-commit":
		gitTest(t, root, "merge", "--no-ff", "-m", "merge topic", "topic")
	case "squash":
		gitTest(t, root, "merge", "--squash", "topic")
		gitCommitTest(t, root, "squash topic")
	default:
		t.Fatalf("unknown merge %q", merge)
	}
}

func validResolutionEvidence(eventID string, managers []projectmodel.Manager) VerifiedResolutionEvidence {
	managerIDs := make([]string, 0, len(managers))
	for _, manager := range managers {
		managerIDs = append(managerIDs, manager.ID)
	}
	sort.Strings(managerIDs)
	return VerifiedResolutionEvidence{
		EventIDs: []string{eventID}, RunID: "run-verified", PlanDigest: "plan-digest",
		CandidateID: "candidate-verified", CandidateDigest: "candidate-digest",
		VerificationDigest: "full-verify-digest", ApplyDigest: "apply-digest", FullVerifyPassed: true,
		CoveredManagerIDs: managerIDs, EvidenceRefs: []string{"check:integration", "verify:all-managers"},
		Delivered: []DeliveredFile{},
	}
}
