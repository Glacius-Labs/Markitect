package run_test

import (
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const testImage = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func checkRequest(t *testing.T) run.CheckRequest {
	t.Helper()
	p, err := snapshot.NewPath("realization")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := snapshot.DirectoryAt(p)
	if err != nil {
		t.Fatal(err)
	}
	return run.CheckRequest{ID: "c", ImageID: testImage, Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."}, Env: []string{"GOFLAGS=-mod=readonly", "CGO_ENABLED=0"},
		Directory: dir, Limits: run.CheckLimits{MilliCPUs: 2000, MemoryBytes: 1 << 30, Processes: 256}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}
}

func TestCheckRequestIsCompleteAndBounded(t *testing.T) {
	if !checkRequest(t).Valid() {
		t.Fatal("complete request refused")
	}
	for name, edit := range map[string]func(*run.CheckRequest){
		"no image":            func(r *run.CheckRequest) { r.ImageID = "latest" },
		"relative executable": func(r *run.CheckRequest) { r.Executable = "go" },
		"no directory":        func(r *run.CheckRequest) { r.Directory = snapshot.DirectoryBoundary{} },
		"no cpu limit":        func(r *run.CheckRequest) { r.Limits.MilliCPUs = 0 },
		"no memory limit":     func(r *run.CheckRequest) { r.Limits.MemoryBytes = 0 },
		"no process limit":    func(r *run.CheckRequest) { r.Limits.Processes = 0 },
		"no timeout":          func(r *run.CheckRequest) { r.Timeout = 0 },
		"no output cap":       func(r *run.CheckRequest) { r.MaxOutputBytes = 0 },
		"huge output":         func(r *run.CheckRequest) { r.MaxOutputBytes = 1 << 40 },
		"env without value":   func(r *run.CheckRequest) { r.Env = []string{"GOFLAGS"} },
		"duplicate env key":   func(r *run.CheckRequest) { r.Env = []string{"A=1", "A=2"} },
		"no identifier":       func(r *run.CheckRequest) { r.ID = "" },
	} {
		bad := checkRequest(t)
		edit(&bad)
		if bad.Valid() {
			t.Fatal(name, "accepted")
		}
	}
}

func TestOnlyAnObservedZeroExitOfTheSameRequestPasses(t *testing.T) {
	req := checkRequest(t)
	pass := run.CheckReceipt{ID: req.ID, ImageID: req.ImageID, Tree: req.Tree.Digest(), Started: true, Exited: true}
	if !pass.Passed(req) {
		t.Fatal("observed zero exit is not a pass")
	}
	for name, edit := range map[string]func(*run.CheckReceipt){
		"nonzero":     func(r *run.CheckReceipt) { r.ExitCode = 1 },
		"timed out":   func(r *run.CheckReceipt) { r.TimedOut = true },
		"uncertain":   func(r *run.CheckReceipt) { r.Uncertain = true },
		"no exit":     func(r *run.CheckReceipt) { r.Exited = false },
		"no start":    func(r *run.CheckReceipt) { r.Started = false },
		"other check": func(r *run.CheckReceipt) { r.ID = "other" },
		"other image": func(r *run.CheckReceipt) { r.ImageID = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" },
		"other tree":  func(r *run.CheckReceipt) { r.Tree = snapshot.TreeID{1} },
	} {
		r := pass
		edit(&r)
		if r.Passed(req) {
			t.Fatal(name, "passed")
		}
	}
}
