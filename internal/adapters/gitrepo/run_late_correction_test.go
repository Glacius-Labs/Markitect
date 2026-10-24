package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestLateRejectRepairsAcceptedChildAndRenewsExactIntegration(t *testing.T) {
	_, p, g, ref := runFixture(t)
	store, _ := runstore.New(g.StateRoot)
	sessions := &controlledManagers{crossed: make(chan struct{}), lateReject: true}
	executor := run.Executor{Workspaces: g, Sessions: sessions, Records: store}
	r, err := executor.Execute(context.Background(), run.Request{MaxManagers: 5, ID: "late-repair", Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(time.Now().Add(60*time.Second), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || r.Technical != run.Completed || r.Decision != run.Submitted || !sessions.didLateReject || sessions.corrections != 2 {
		t.Fatal(r, err)
	}
	destination := t.TempDir()
	if err = g.Reopen(context.Background(), *r.Candidate, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "out/deep.txt"))
	if err != nil || string(data) != "late corrected\n" {
		t.Fatal("late accepted correction missing", string(data), err)
	}
	for _, h := range r.Handoffs {
		if h.Decision == run.Reject || h.Review != nil && h.Review.Decision != run.Approve {
			t.Fatal("resolved late Reject remained latched", h)
		}
		for _, i := range h.Children {
			if _, err = localCommand(context.Background(), destination, nil, "merge-base", "--is-ancestor", i.Child.ID, r.Candidate.Commit); err != nil {
				t.Fatal("accepted history missing", err)
			}
		}
	}
}

func TestKnownInvalidManifestCandidateReturnsToItsManagerForRepair(t *testing.T) {
	_, p, g, ref := runFixture(t)
	store, _ := runstore.New(g.StateRoot)
	sessions := &controlledManagers{crossed: make(chan struct{}), badManifest: true}
	executor := run.Executor{Workspaces: g, Sessions: sessions, Records: store}
	r, err := executor.Execute(context.Background(), run.Request{MaxManagers: 5, ID: "manifest-repair", Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
	if err != nil || r.Technical != run.Completed || r.Decision != run.Submitted {
		t.Fatal(r, err)
	}
	// Root has no parent: its completion check returns the diagnostic to Root,
	// which repairs its own state and judges again.
	if len(sessions.refusals) != 1 || !strings.Contains(sessions.refusals[0], "invalid candidate manifest") {
		t.Fatal("invalid manifest did not reach Root as its completion diagnostic", sessions.refusals)
	}
	for _, s := range sessions.sessions {
		if s.spec.Manager.Package == "." && s.judgments != 1 {
			t.Fatal("Root judgment count changed", s.judgments)
		}
	}
}
