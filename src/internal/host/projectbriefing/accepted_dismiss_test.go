package projectbriefing

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDismissAcceptedHistoryPersistsPredictedHistoryAndDismissalTogether(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	accepted, err := ReadAcceptedHistory(root, revision)
	if err != nil || len(accepted.State.Briefings) != 1 || len(accepted.State.Briefings[0].Events) == 0 {
		t.Fatalf("accepted history preview = %+v, err=%v", accepted, err)
	}
	bundle := accepted.State.Briefings[0]
	event := bundle.Events[0]
	manager := managerWithEventContextTest(t, bundle, event)
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("history preview unexpectedly created its store: %v", err)
	}
	digest, err := DismissAcceptedHistory(root, revision, event.ID, manager, accepted.Receipt.StoreDigest)
	if err != nil {
		t.Fatalf("dismiss with predicted store token: %v", err)
	}
	state, persistedDigest, err := Read(root)
	if err != nil || persistedDigest != digest || state.History == nil || state.History.Revision != revision || len(state.Briefings) != 1 || len(state.Dismissals) != 1 || state.Dismissals[0].EventID != event.ID || state.Dismissals[0].ManagerID != manager {
		t.Fatalf("accepted history and dismissal were not persisted together: state=%+v digest=%s err=%v", state, persistedDigest, err)
	}
}

func TestDismissAcceptedHistoryRejectsStoreAdvanceAfterHistoryReadWithoutOverwriting(t *testing.T) {
	root, _, revision := committedModelFixture(t)
	active, next, receipt, persistedDigest, err := acceptedDismissalPreview(root, revision)
	if err != nil || len(active.accepted(next).Briefings) != 1 || len(active.accepted(next).Briefings[0].Events) == 0 {
		t.Fatalf("accepted history preview = %+v, err=%v", next, err)
	}
	bundle := active.accepted(next).Briefings[0]
	event := bundle.Events[0]
	manager := managerWithEventContextTest(t, bundle, event)
	if _, err := EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatalf("concurrent writer materializes accepted history: %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dismissAcceptedHistoryFrom(root, active, next, receipt, persistedDigest, event.ID, manager, receipt.StoreDigest); !errors.Is(err, ErrStaleStore) {
		t.Fatalf("dismissal after backing-store advance = %v, want ErrStaleStore", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("stale dismissal replaced the advanced store: err=%v", err)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Dismissals) != 0 {
		t.Fatalf("stale dismissal changed dismissal state: %+v err=%v", state.Dismissals, err)
	}
}

func TestDismissAcceptedHistoryPreservesProvisionalBriefings(t *testing.T) {
	root, _, changed := committedModelFixture(t)
	mainBranch := gitOutputTest(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := EnsureAcceptedHistory(root, changed); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "-b", "topic")
	topic := commitModelChangeTest(t, root, "before shipment", "prior to fulfillment", "topic-only model change")
	if _, err := EnsureAcceptedHistory(root, topic); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", mainBranch)
	diverged := commitReadmeTest(t, root, "Main diverges before dismissal.\n", "main diverges")
	accepted, err := ReadAcceptedHistory(root, diverged)
	if err != nil || len(accepted.State.Briefings) != 1 || len(storedBriefingsTest(t, root)) != 2 {
		t.Fatalf("precondition should have one visible and two stored histories: accepted=%+v stored=%d err=%v", accepted.State.Briefings, len(storedBriefingsTest(t, root)), err)
	}
	bundle := accepted.State.Briefings[0]
	event := bundle.Events[0]
	if _, err := DismissAcceptedHistory(root, diverged, event.ID, managerWithEventContextTest(t, bundle, event), accepted.Receipt.StoreDigest); err != nil {
		t.Fatalf("dismiss main-line event: %v", err)
	}
	stored := storedBriefingsTest(t, root)
	if len(stored) != 2 {
		t.Fatalf("dismissal replaced the provisional topic briefing: stored=%d", len(stored))
	}
}

func managerWithEventContextTest(t *testing.T, bundle Bundle, event Event) string {
	t.Helper()
	for _, briefing := range bundle.Managers {
		if !contains(event.AffectedManagers, briefing.ManagerID) {
			continue
		}
		for _, contract := range briefing.Contracts {
			if contract.ID == event.DefinitionID.Key() {
				return briefing.ManagerID
			}
		}
	}
	t.Fatalf("no affected Manager has historical context for event %s", event.ID)
	return ""
}
