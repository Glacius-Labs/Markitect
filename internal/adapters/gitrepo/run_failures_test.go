package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type failingRecords struct {
	run.Records
	failSave, mismatch bool
}

func (f failingRecords) Save(ctx context.Context, r run.Result) error {
	if f.failSave {
		return errors.New("injected storage failure")
	}
	return f.Records.Save(ctx, r)
}
func (f failingRecords) Read(ctx context.Context, p, id string) (run.Result, error) {
	r, err := f.Records.Read(ctx, p, id)
	if f.mismatch {
		r.Binding = "substituted"
	}
	return r, err
}

type inspectFailure struct{ run.Workspaces }

func (f inspectFailure) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := f.Workspaces.Open(ctx, r)
	return failedService{s}, err
}

type failedService struct{ run.WorkspaceService }

func (f failedService) Copy(ctx context.Context, p string, c run.Commit) (run.Copy, error) {
	copy, err := f.WorkspaceService.Copy(ctx, p, c)
	if p == "." {
		return failedInspection{copy}, err
	}
	return copy, err
}

type failedInspection struct{ run.Copy }

func (f failedInspection) Inspect(context.Context, run.Commit) (project.PreviewPreparationAttempt, error) {
	return project.PreviewPreparationAttempt{}, errors.New("injected acquisition failure")
}

func TestRunFailuresPersistHonestTerminalStateAndCloseRealGit(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, kind := range []string{"partial startup", "cleanup failure", "stuck close", "stuck exchange", "storage failure", "readback substitution", "reject technical acquisition"} {
		t.Run(kind, func(t *testing.T) {
			store, _ := runstore.New(g.StateRoot)
			managers := &controlledManagers{crossed: make(chan struct{})}
			var records run.Records = store
			var git run.Workspaces = g
			release := make(chan struct{})
			defer close(release)
			deadline := time.Now().Add(45 * time.Second)
			switch kind {
			case "partial startup":
				managers.failOpen = true
			case "cleanup failure":
				managers.failClose = true
			case "stuck close":
				managers.closeBlock = release
			case "stuck exchange":
				managers.exchangeBlock = release
				deadline = time.Now().Add(5 * time.Second)
			case "storage failure":
				records = failingRecords{Records: store, failSave: true}
			case "readback substitution":
				records = failingRecords{Records: store, mismatch: true}
			case "reject technical acquisition":
				managers.forceDecision = run.Reject
				git = inspectFailure{g}
			}
			id := "failure-" + fmt.Sprintf("%x", []byte(kind))
			e := run.Executor{Sessions: managers, Workspaces: git, Records: records}
			started := time.Now()
			r, err := e.Execute(context.Background(), run.Request{MaxManagers: 5, ID: id, Project: "fixture", Prepared: p, Input: ref, Mandate: run.Assessment, Allowance: run.NewAllowance(deadline, 80), CleanupTimeout: time.Second, PersistenceTimeout: time.Second})
			if err == nil || r.Technical == run.Completed {
				t.Fatalf("technical failure became completed: %+v err=%v", r, err)
			}
			if kind == "stuck exchange" && time.Since(started) > 8*time.Second {
				t.Fatal("Exchange ignored finite work/cleanup bounds")
			}
			if kind == "reject technical acquisition" && (r.Technical != run.Failed || !strings.Contains(strings.Join(r.Errors, " "), "injected acquisition failure")) {
				t.Fatalf("semantic Reject swallowed technical acquisition: %+v", r)
			}
			if kind != "storage failure" && kind != "readback substitution" {
				fresh, _ := runstore.New(g.StateRoot)
				saved, readErr := fresh.Read(context.Background(), "fixture", id)
				if readErr != nil || saved.Technical == run.Completed || saved.Input != ref {
					t.Fatal("failed terminal record unavailable", saved, readErr)
				}
			}
			for _, s := range managers.sessions {
				if _, err := os.Stat(s.spec.Workspace.Location().Directory); !os.IsNotExist(err) {
					t.Fatalf("private Git copy survived failure %s: %v", kind, err)
				}
			}
			if kind == "stuck close" || kind == "stuck exchange" {
				found := false
				for _, c := range r.Cleanup {
					if !c.Closed {
						found = true
					}
				}
				if !found {
					t.Fatal("unresponsive Session falsely reported closed")
				}
			}
		})
	}
}

func TestRunPreflightRefusesInsufficientLimitsBeforeAnyEnvironment(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, kind := range []string{"managers", "messages"} {
		sessions := &controlledManagers{}
		store, _ := runstore.New(g.StateRoot)
		r := run.Request{MaxManagers: 5, ID: "preflight", Project: "fixture", Prepared: p, Input: ref, Mandate: run.Assessment, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 80), CleanupTimeout: time.Second, PersistenceTimeout: time.Second}
		if kind == "managers" {
			r.MaxManagers = 4
		} else {
			r.Allowance = run.NewAllowance(time.Now().Add(time.Minute), 4)
		}
		e := run.Executor{Sessions: sessions, Workspaces: g, Records: store}
		_, err := e.Execute(context.Background(), r)
		if err == nil || len(sessions.sessions) != 0 {
			t.Fatal("insufficient limits started Manager environments")
		}
		entries, err := os.ReadDir(g.TempRoot)
		if err != nil || len(entries) != 0 {
			t.Fatal("preflight allocated Git resources", entries, err)
		}
	}
}
