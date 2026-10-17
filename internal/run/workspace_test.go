package run_test

import (
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

const testImage = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestWorkspaceRequestsNameAWorkspaceAndBoundObservation(t *testing.T) {
	if !(run.WorkspaceSpec{ID: "w"}).Valid() || (run.WorkspaceSpec{}).Valid() {
		t.Fatal("workspace spec validity")
	}
	if !(run.HarvestRequest{ID: "w", MaxEntries: 1, MaxBytes: 1}).Valid() || (run.HarvestRequest{ID: "w", MaxBytes: 1}).Valid() || (run.HarvestRequest{ID: "w", MaxEntries: 1}).Valid() {
		t.Fatal("harvest request validity")
	}
	if !(run.WorkspaceHarvest{}).Complete() || (run.WorkspaceHarvest{Unsupported: []string{"link"}}).Complete() || (run.WorkspaceHarvest{Uncertain: true}).Complete() {
		t.Fatal("an incomplete harvest counted as complete")
	}
}

func TestCheckRequestIsBoundedAndOnlyAnObservedZeroExitPasses(t *testing.T) {
	ok := run.CheckRequest{ID: "c", ImageID: testImage, Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}
	if !ok.Valid() {
		t.Fatal("complete request refused")
	}
	for name, edit := range map[string]func(*run.CheckRequest){
		"no image":      func(r *run.CheckRequest) { r.ImageID = "latest" },
		"no executable": func(r *run.CheckRequest) { r.Executable = "" },
		"no timeout":    func(r *run.CheckRequest) { r.Timeout = 0 },
		"no output cap": func(r *run.CheckRequest) { r.MaxOutputBytes = 0 },
		"huge output":   func(r *run.CheckRequest) { r.MaxOutputBytes = 1 << 40 },
		"no identifier": func(r *run.CheckRequest) { r.ID = "" },
	} {
		bad := ok
		edit(&bad)
		if bad.Valid() {
			t.Fatal(name, "accepted")
		}
	}
	pass := run.CheckReceipt{Started: true, Exited: true}
	if !pass.Passed() {
		t.Fatal("observed zero exit is not a pass")
	}
	for name, receipt := range map[string]run.CheckReceipt{
		"nonzero":   {Started: true, Exited: true, ExitCode: 1},
		"timed out": {Started: true, Exited: true, TimedOut: true},
		"uncertain": {Started: true, Exited: true, Uncertain: true},
		"no exit":   {Started: true},
		"no start":  {Exited: true},
	} {
		if receipt.Passed() {
			t.Fatal(name, "passed")
		}
	}
}
