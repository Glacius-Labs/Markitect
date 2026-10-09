package projectcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
)

func TestParseOperationAndSingleManagerContracts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		fail string
	}{
		{"apply operation", []string{"plan", "--repo", ".", "--goal", "g", "--operation", "apply"}, "apply", ""},
		{"cleanup operation", []string{"plan", "--repo", ".", "--goal", "g", "--operation", "cleanup"}, "cleanup", ""},
		{"reconcile operation", []string{"plan", "--repo", ".", "--goal", "g", "--operation", "reconcile"}, "reconcile", ""},
		{"unknown operation", []string{"plan", "--repo", ".", "--goal", "g", "--operation", "verify"}, "", "must be apply, cleanup or reconcile"},
		{"context multiple managers", []string{"context", "--repo", ".", "--manager", "a", "--manager", "b"}, "", "accepts only one --manager"},
		{"briefings multiple managers", []string{"briefings", "--repo", ".", "--manager", "a", "--manager", "b"}, "", "accepts only one --manager"},
		{"dismiss multiple managers", []string{"dismiss", "--repo", ".", "--event", "e", "--manager", "a", "--manager", "b", "--expect", "d", "--write"}, "", "accepts only one --manager"},
		{"dismiss requires write", []string{"dismiss", "--repo", ".", "--event", "e", "--manager", "a", "--expect", "d"}, "", "requires --write"},
		{"brief write requires state digest", []string{"brief", "--repo", ".", "--since", strings.Repeat("a", 40), "--revision", strings.Repeat("b", 40), "--provenance", "ADR-1", "--write"}, "", "requires --expect"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _, err := parse(test.args, nil)
			if test.fail != "" {
				if err == nil || !strings.Contains(err.Error(), test.fail) {
					t.Fatalf("parse error = %v, want %q", err, test.fail)
				}
				return
			}
			if err != nil || got.operation != test.want {
				t.Fatalf("parse = %#v, err=%v; operation=%q", got, err, test.want)
			}
		})
	}
}

func TestBriefingCLIOverviewDismissalAndManagerContext(t *testing.T) {
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
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "change accepted model")
	revision := gitOutput(t, repo, "rev-parse", "HEAD")
	args := []string{"project", "brief", "--repo", repo, "--since", base, "--revision", revision, "--provenance", "ADR-briefing-1"}
	var previewOut, previewErr bytes.Buffer
	if code := Run(args, &previewOut, &previewErr); code != 0 {
		t.Fatalf("brief preview exit=%d stderr=%s", code, previewErr.String())
	}
	var preview struct {
		Bundle      projectbriefing.Bundle `json:"bundle"`
		StateDigest string                 `json:"stateDigest"`
	}
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Bundle.Events) != 1 || preview.Bundle.Events[0].Severity != "info" {
		t.Fatalf("preview events = %#v", preview.Bundle.Events)
	}
	writeArgs := append(append([]string(nil), args...), "--expect", preview.StateDigest, "--write")
	var writeOut, writeErr bytes.Buffer
	if code := Run(writeArgs, &writeOut, &writeErr); code != 0 {
		t.Fatalf("brief write exit=%d stderr=%s", code, writeErr.String())
	}
	var written struct {
		StateDigest string `json:"stateDigest"`
	}
	if err := json.Unmarshal(writeOut.Bytes(), &written); err != nil {
		t.Fatal(err)
	}
	var overviewOut, overviewErr bytes.Buffer
	if code := Run([]string{"project", "briefings", "--repo", repo}, &overviewOut, &overviewErr); code != 0 {
		t.Fatalf("briefings overview exit=%d stderr=%s", code, overviewErr.String())
	}
	var overview struct {
		Notifications []visibleNotification `json:"notifications"`
	}
	if err := json.Unmarshal(overviewOut.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Notifications) != 1 || overview.Notifications[0].ResolutionStatus != "unresolved" || overview.Notifications[0].Severity != "info" || overview.Notifications[0].CommitTime == "" {
		t.Fatalf("overview notification = %#v", overview.Notifications)
	}
	manager := preview.Bundle.Events[0].AffectedManagers[0]
	var contextOut, contextErr bytes.Buffer
	if code := Run([]string{"project", "briefings", "--repo", repo, "--manager", manager}, &contextOut, &contextErr); code != 0 {
		t.Fatalf("manager briefing exit=%d stderr=%s", code, contextErr.String())
	}
	var managerContext struct {
		Events []managerEvent `json:"events"`
	}
	if err := json.Unmarshal(contextOut.Bytes(), &managerContext); err != nil || len(managerContext.Events) != 1 || managerContext.Events[0].ResolutionStatus != "unresolved" {
		t.Fatalf("manager event context = %#v err=%v", managerContext.Events, err)
	}
	stateDigest := written.StateDigest
	for _, affectedManager := range preview.Bundle.Events[0].AffectedManagers {
		var dismissOut, dismissErr bytes.Buffer
		if code := Run([]string{"project", "dismiss", "--repo", repo, "--event", preview.Bundle.Events[0].ID, "--manager", affectedManager, "--expect", stateDigest, "--write"}, &dismissOut, &dismissErr); code != 0 {
			t.Fatalf("dismiss %s exit=%d stderr=%s", affectedManager, code, dismissErr.String())
		}
		var result struct {
			StateDigest string `json:"stateDigest"`
			Note        string `json:"note"`
		}
		if err := json.Unmarshal(dismissOut.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Note, "does not resolve") && !strings.Contains(result.Note, "remains unresolved") {
			t.Fatalf("dismissal note obscures resolution state: %q", result.Note)
		}
		stateDigest = result.StateDigest
	}
	var dismissedOut, dismissedErr bytes.Buffer
	if code := Run([]string{"project", "briefings", "--repo", repo}, &dismissedOut, &dismissedErr); code != 0 {
		t.Fatalf("dismissed overview exit=%d stderr=%s", code, dismissedErr.String())
	}
	var dismissedOverview struct {
		Notifications        []visibleNotification `json:"notifications"`
		HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
	}
	if err := json.Unmarshal(dismissedOut.Bytes(), &dismissedOverview); err != nil || len(dismissedOverview.Notifications) != 0 || dismissedOverview.HiddenDismissedCount != 1 {
		t.Fatalf("dismissed overview = %#v err=%v", dismissedOverview, err)
	}
	var contextAfterDismiss bytes.Buffer
	if code := Run([]string{"project", "briefings", "--repo", repo, "--manager", manager}, &contextAfterDismiss, new(bytes.Buffer)); code != 0 || !bytes.Contains(contextAfterDismiss.Bytes(), []byte(preview.Bundle.Events[0].ID)) {
		t.Fatalf("dismissal hid manager briefing: code=%d output=%s", code, contextAfterDismiss.String())
	}
}
