package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrganizationResolutionIsExplicitSharedAndShowsProgress(t *testing.T) {
	started, finish := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	srv, err := New(Config{States: []string{t.TempDir()}, Project: func(context.Context) (ProjectInfo, error) {
		return ProjectInfo{Organization: &OrganizationView{Mode: "declared", Head: "draft"}}, nil
	}, Organization: func(ctx context.Context) (*OrganizationView, error) {
		calls.Add(1)
		close(started)
		select {
		case <-finish:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &OrganizationView{Mode: "resolved", Head: "committed"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(finish) })
	var view struct {
		Declared, Resolved               *OrganizationView
		Observing, Resolving, Resolvable bool
	}
	read := func(target string) {
		t.Helper()
		r := get(t, srv.Handler(), "", target, nil)
		if r.Code != http.StatusOK {
			t.Fatal(r.Code, r.Body)
		}
		if err := json.Unmarshal(r.Body.Bytes(), &view); err != nil {
			t.Fatal(err)
		}
	}
	read("/api/organization")
	if calls.Load() != 0 || view.Resolving || !view.Resolvable {
		t.Fatalf("implicit resolution: %+v", view)
	}
	read("/api/organization?resolve=1")
	<-started
	read("/api/organization?resolve=1")
	if !view.Resolving || calls.Load() != 1 {
		t.Fatalf("concurrent resolution = %+v, calls %d", view, calls.Load())
	}
}

func TestOrganizationResolutionRetainsRefusalsAndNoRuntimeStaysDeclared(t *testing.T) {
	c := organizationCache{resolve: func(context.Context) (*OrganizationView, error) { return nil, errors.New("uncommitted source") }}
	c.get(true)
	for deadline := time.Now().Add(time.Second); ; {
		v, problem, working := c.get(false)
		if !working {
			if v != nil || problem != "uncommitted source" {
				t.Fatal(v, problem)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("resolution did not end")
		}
		time.Sleep(time.Millisecond)
	}
	srv, _ := New(Config{States: []string{t.TempDir()}})
	r := get(t, srv.Handler(), "", "/api/organization?resolve=1", nil)
	var view struct {
		Resolvable, Resolving bool
		Resolved              *OrganizationView
	}
	if json.Unmarshal(r.Body.Bytes(), &view); view.Resolvable || view.Resolving || view.Resolved != nil {
		t.Fatalf("no Runtime: %s", r.Body)
	}
	if rec := post(srv.Handler(), "", "", "", "/api/organization", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("organization write = %d", rec.Code)
	}
}
