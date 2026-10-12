package projectcli

import "github.com/Glacius-Labs/Markitect/src/internal/host/projecthistory"

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
)

// briefListOutput is the `brief list` overview without --manager.
type briefListOutput struct {
	Notifications        []visibleNotification `json:"notifications"`
	HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
	StateDigest          string                `json:"stateDigest"`
}

func TestPlanOperationAndSingleManagerContracts(t *testing.T) {
	for _, operation := range []string{"apply", "cleanup", "reconcile"} {
		fields, err := parseFields(t, "plan", "--goal", "g", "--operation", operation)
		if err != nil || fields["operation"] != operation {
			t.Fatalf("plan --operation %s parsed to %v, err=%v", operation, fields, err)
		}
	}
	repo := t.TempDir()
	tests := []struct {
		name string
		args []string
		fail string
	}{
		{"unknown operation", []string{"plan", "--goal", "g", "--operation", "verify"}, "--operation must be apply, cleanup or reconcile"},
		{"context takes one manager operand", []string{"context", "a", "b"}, `unexpected argument "b"`},
		{"context has no manager flag", []string{"context", "--manager", "a"}, "flag provided but not defined: -manager"},
		{"brief list one manager", []string{"brief", "list", "--manager", "a", "--manager", "b"}, "may be given only once"},
		{"brief dismiss one manager", []string{"brief", "dismiss", "--event", "e", "--manager", "a", "--manager", "b", "--expect", "d", "--write"}, "may be given only once"},
		{"brief dismiss requires write", []string{"brief", "dismiss", "--event", "e", "--manager", "a", "--expect", "d"}, "--expect is valid only together with --write or --execute"},
		{"brief dismiss requires event", []string{"brief", "dismiss", "--manager", "a", "--expect", "d", "--write"}, "brief dismiss requires --event, --manager, --expect and --write"},
		{"brief write requires state digest", []string{"brief", "--since", strings.Repeat("a", 40), "--revision", strings.Repeat("b", 40), "--provenance", "ADR-1", "--write"}, "requires --expect"},
		{"brief requires its range", []string{"brief", "--provenance", "ADR-1"}, "brief requires --since, --revision and --provenance"},
		{"brief list takes only manager", []string{"brief", "list", "--provenance", "ADR-1"}, "brief list takes only --manager"},
		{"brief unknown action", []string{"brief", "remove"}, `unknown action "remove"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append(append([]string{}, test.args...), "--repo", repo)
			code, out, errout := runCLI(t, args...)
			if code != 2 || out != "" || !strings.Contains(errout, "markitect "+test.args[0]+": ") || !strings.Contains(errout, test.fail) {
				t.Fatalf("exit=%d stdout=%q stderr=%q, want 2 and %q", code, out, errout, test.fail)
			}
		})
	}
	if entries, err := os.ReadDir(repo); err != nil || len(entries) != 0 {
		t.Fatalf("rejected invocations wrote files: %v %v", entries, err)
	}
}

func TestBriefCreateListDismissAndManagerContext(t *testing.T) {
	repo := copyProjectWorld(t)
	base := gitOutput(t, repo, "rev-parse", "HEAD")
	modelPath := filepath.Join(repo, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	oldText := "Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled."
	newText := "Cancellation is valid only before shipment; a shipped order cannot be cancelled."
	updated := strings.Replace(string(content), oldText, newText, 1)
	if updated == string(content) {
		t.Fatal("fixture model text was not found")
	}
	if err := os.WriteFile(modelPath, []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "change accepted model")
	revision := gitOutput(t, repo, "rev-parse", "HEAD")
	args := []string{"brief", "--repo", repo, "--since", base, "--revision", revision, "--provenance", "ADR-briefing-1"}
	type briefOutput struct {
		Bundle      projectbriefing.Bundle `json:"bundle"`
		StateDigest string                 `json:"stateDigest"`
	}
	preview := decodeOutput[briefOutput](t, mustCLI(t, args...))
	if len(preview.Bundle.Events) != 1 || preview.Bundle.Events[0].Severity != "info" {
		t.Fatalf("preview events = %#v", preview.Bundle.Events)
	}
	if read, err := projecthistory.Read(repo); err != nil || len(read.Accepted.Briefings) != 0 {
		t.Fatalf("brief preview recorded a briefing: %+v %v", read.Accepted.Briefings, err)
	}
	// create is the default action and may also be named.
	if named := decodeOutput[briefOutput](t, mustCLI(t, append([]string{"brief", "create"}, args[1:]...)...)); named.StateDigest != preview.StateDigest || named.Bundle.Digest != preview.Bundle.Digest {
		t.Fatalf("brief create differs from brief: %+v", named)
	}
	written := decodeOutput[briefOutput](t, mustCLI(t, append(args, "--expect", preview.StateDigest, "--write")...))
	if written.StateDigest == "" || written.StateDigest == preview.StateDigest {
		t.Fatalf("brief write did not advance the state digest: %q", written.StateDigest)
	}
	overview := decodeOutput[briefListOutput](t, mustCLI(t, "brief", "list", "--repo", repo))
	if len(overview.Notifications) != 1 || overview.Notifications[0].ResolutionStatus != "unresolved" || overview.Notifications[0].Severity != "info" || overview.Notifications[0].CommitTime == "" || overview.StateDigest != written.StateDigest {
		t.Fatalf("overview notification = %#v", overview)
	}
	manager := managerWithEventContext(t, preview.Bundle, preview.Bundle.Events[0])
	managerContext := decodeOutput[struct {
		Events []managerEvent `json:"events"`
	}](t, mustCLI(t, "brief", "list", "--repo", repo, "--manager", manager))
	if len(managerContext.Events) != 1 || managerContext.Events[0].ResolutionStatus != "unresolved" {
		t.Fatalf("manager event context = %#v", managerContext.Events)
	}
	stateDigest := written.StateDigest
	if code, _, errout := runCLI(t, "brief", "dismiss", "--repo", repo, "--event", preview.Bundle.Events[0].ID, "--manager", manager, "--expect", "sha256:stale", "--write"); code != 2 || !strings.Contains(errout, "markitect brief:") {
		t.Fatalf("dismiss with a stale state digest exit=%d stderr=%s", code, errout)
	}
	for _, affectedManager := range preview.Bundle.Events[0].AffectedManagers {
		result := decodeOutput[struct {
			StateDigest string `json:"stateDigest"`
			Note        string `json:"note"`
		}](t, mustCLI(t, "brief", "dismiss", "--repo", repo, "--event", preview.Bundle.Events[0].ID, "--manager", affectedManager, "--expect", stateDigest, "--write"))
		if !strings.Contains(result.Note, "does not resolve") && !strings.Contains(result.Note, "remains unresolved") {
			t.Fatalf("dismissal note obscures resolution state: %q", result.Note)
		}
		stateDigest = result.StateDigest
	}
	dismissed := decodeOutput[briefListOutput](t, mustCLI(t, "brief", "list", "--repo", repo))
	if len(dismissed.Notifications) != 0 || dismissed.HiddenDismissedCount != 1 {
		t.Fatalf("dismissed overview = %#v", dismissed)
	}
	if contextAfterDismiss := mustCLI(t, "brief", "list", "--repo", repo, "--manager", manager); !strings.Contains(string(contextAfterDismiss), preview.Bundle.Events[0].ID) {
		t.Fatalf("dismissal hid manager briefing: %s", contextAfterDismiss)
	}
}

func TestBriefListUsesPredictedAcceptedHistoryAndStaleDismissDoesNotWrite(t *testing.T) {
	repo, base, revision := commitBriefingModelChange(t)
	accepted, err := projecthistory.ReadAcceptedHistory(repo, revision)
	if err != nil || len(accepted.State.Briefings) != 1 {
		t.Fatalf("accepted history = %+v, err=%v", accepted, err)
	}
	storePath := filepath.Join(repo, ".markitect", "state", "briefings", "history.json")
	listed, err := briefList(repo, "")
	if err != nil {
		t.Fatalf("read-only brief list: %v", err)
	}
	overview := listed.(struct {
		Notifications        []visibleNotification `json:"notifications"`
		HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
		StateDigest          string                `json:"stateDigest"`
	})
	if len(overview.Notifications) != 1 || overview.Notifications[0].ResolutionStatus != projectbriefing.StatusUnresolved || overview.StateDigest != accepted.Receipt.StoreDigest {
		t.Fatalf("brief list omitted accepted in-memory history or predicted token: overview=%+v acceptedBriefings=%+v dismissals=%+v", overview, accepted.State.Briefings, accepted.State.Dismissals)
	}
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("brief list persisted accepted history: %v", err)
	}
	bundle := accepted.State.Briefings[0]
	event := bundle.Events[0]
	manager := managerWithEventContext(t, bundle, event)
	if code, _, stderr := runCLI(t, "brief", "dismiss", "--repo", repo, "--event", event.ID, "--manager", manager, "--expect", "sha256:stale", "--write"); code != 2 || !strings.Contains(stderr, "changed since it was read") {
		t.Fatalf("stale dismissal exit=%d stderr=%q", code, stderr)
	}
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("stale dismissal persisted accepted history: %v", err)
	}
	if code, _, stderr := runCLI(t, "brief", "dismiss", "--repo", repo, "--event", event.ID, "--manager", manager, "--expect", accepted.Receipt.StoreDigest, "--write"); code != 0 {
		t.Fatalf("valid predicted-token dismissal exit=%d stderr=%q", code, stderr)
	}
	read, err := projecthistory.Read(repo)
	state := read.Accepted
	if err != nil || len(state.Dismissals) != 1 || state.History == nil || state.History.Revision != revision || base == revision {
		t.Fatalf("valid dismissal did not persist accepted event: state=%+v err=%v", read.Accepted, err)
	}
}

func TestBriefListPreservesDeliveredUncommittedResolutionStatus(t *testing.T) {
	repo, _, revision := commitBriefingModelChange(t)
	if _, err := projecthistory.EnsureAcceptedHistory(repo, revision); err != nil {
		t.Fatal(err)
	}
	read, err := projecthistory.Read(repo)
	state, digest := read.Accepted, read.PersistedDigest
	if err != nil || len(state.Briefings) != 1 {
		t.Fatalf("persisted briefing history = %+v err=%v", state, err)
	}
	project, err := projectwork.Load(repo, revision)
	if err != nil {
		t.Fatal(err)
	}
	bundle := state.Briefings[0]
	event := bundle.Events[0]
	content := []byte("# Delivered, not committed\n")
	delivered, err := projecthistory.NewDeliveredFile(repo, "docs/uncommitted-delivery.txt", "100644", content, false)
	if err != nil {
		t.Fatal(err)
	}
	managerIDs := make([]string, 0, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		managerIDs = append(managerIDs, manager.ID)
	}
	evidence := projectbriefing.VerifiedResolutionEvidence{
		EventIDs: []string{event.ID}, RunID: "run-verified", PlanDigest: "plan-digest",
		CandidateID: "candidate-verified", CandidateDigest: "candidate-digest",
		VerificationDigest: "full-verify-digest", ApplyDigest: "apply-digest", FullVerifyPassed: true,
		CoveredManagerIDs: managerIDs, EvidenceRefs: []string{"check:integration", "verify:all-managers"},
		Delivered: []projectbriefing.DeliveredFile{delivered},
	}
	if _, err := projecthistory.ResolveVerified(repo, revision, project.Report.ModelDigest, evidence, digest); err != nil {
		t.Fatalf("record verified resolution: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "uncommitted-delivery.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	listed, err := briefList(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	overview := listed.(struct {
		Notifications        []visibleNotification `json:"notifications"`
		HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
		StateDigest          string                `json:"stateDigest"`
	})
	if len(overview.Notifications) != 1 || overview.Notifications[0].ResolutionStatus != projectbriefing.StatusDeliveredUncommitted {
		t.Fatalf("overview lost delivered-uncommitted status: %+v", overview)
	}
	managerValue, err := briefList(repo, managerWithEventContext(t, bundle, event))
	if err != nil {
		t.Fatal(err)
	}
	managerList := managerValue.(struct {
		Briefings     []projectbriefing.Briefing `json:"briefings"`
		Events        []managerEvent             `json:"events"`
		ContextDigest string                     `json:"contextDigest"`
		StateDigest   string                     `json:"stateDigest"`
	})
	if len(managerList.Events) != 1 || managerList.Events[0].ResolutionStatus != projectbriefing.StatusDeliveredUncommitted {
		t.Fatalf("manager list lost delivered-uncommitted status: %+v", managerList)
	}
}

func commitBriefingModelChange(t *testing.T) (repo, base, revision string) {
	t.Helper()
	repo = copyProjectWorld(t)
	manifestPath := filepath.Join(repo, ".markitect", "project.yaml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedManifest := strings.Replace(string(manifest), "coverageMode: full\n", "coverageMode: full\nworkflowMode: guided\n", 1)
	if updatedManifest == string(manifest) {
		t.Fatal("fixture coverage mode was not found")
	}
	if err := os.WriteFile(manifestPath, []byte(updatedManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".markitect/project.yaml")
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "enable guided workflow")
	base = gitOutput(t, repo, "rev-parse", "HEAD")
	modelPath := filepath.Join(repo, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	oldText := "Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled."
	newText := "Cancellation is valid only before shipment; a shipped order cannot be cancelled."
	updated := strings.Replace(string(data), oldText, newText, 1)
	if updated == string(data) {
		t.Fatal("fixture model text was not found")
	}
	if err := os.WriteFile(modelPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "change accepted model")
	revision = gitOutput(t, repo, "rev-parse", "HEAD")
	return repo, base, revision
}

func managerWithEventContext(t *testing.T, bundle projectbriefing.Bundle, event projectbriefing.Event) string {
	t.Helper()
	for _, briefing := range bundle.Managers {
		if !containsString(event.AffectedManagers, briefing.ManagerID) {
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
