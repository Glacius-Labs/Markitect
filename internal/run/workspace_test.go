package run_test

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestWorkspaceRequestsNameAWorkspaceAndBoundObservation(t *testing.T) {
	t.Parallel()
	if !(run.WorkspaceSpec{ID: "w"}).Valid() || (run.WorkspaceSpec{}).Valid() {
		t.Fatal("workspace spec validity")
	}
	if !(run.HarvestRequest{ID: "w", MaxEntries: 1, MaxBytes: 1}).Valid() {
		t.Fatal("bounded harvest refused")
	}
	for name, r := range map[string]run.HarvestRequest{
		"no entries":    {ID: "w", MaxBytes: 1},
		"no bytes":      {ID: "w", MaxEntries: 1},
		"huge entries":  {ID: "w", MaxEntries: 1 << 30, MaxBytes: 1},
		"huge bytes":    {ID: "w", MaxEntries: 1, MaxBytes: 1 << 40},
		"no identifier": {MaxEntries: 1, MaxBytes: 1},
	} {
		if r.Valid() {
			t.Fatal(name, "accepted")
		}
	}
	if !(run.WorkspaceHarvest{}).Complete() || (run.WorkspaceHarvest{Unsupported: []string{"link"}}).Complete() || (run.WorkspaceHarvest{Uncertain: true}).Complete() {
		t.Fatal("an incomplete harvest counted as complete")
	}
}
