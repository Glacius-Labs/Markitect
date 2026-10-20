package apply

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// followWork picks the work Pass after a sealed one: its own open findings repeat
// it or restart at an earlier purpose, but a later purpose they name comes in
// order, so no forward Pass is skipped.
func TestFollowWorkNeverSkipsAForwardPass(t *testing.T) {
	work := applyWork()
	const plan, prepare, implement = 0, 1, 2
	cases := []struct {
		name        string
		current     int
		verdict     run.Verdict
		corrections []run.Purpose
		want        int
	}{
		{"plan names implement: prepare comes next", plan, run.NeedsCorrection, []run.Purpose{run.Implement}, prepare},
		{"plan names prepare", plan, run.NeedsCorrection, []run.Purpose{run.Prepare}, prepare},
		{"plan names plan: repeat", plan, run.NeedsCorrection, []run.Purpose{run.Plan}, plan},
		{"prepare names implement: implement comes next", prepare, run.NeedsCorrection, []run.Purpose{run.Implement}, implement},
		{"prepare names plan: restart", prepare, run.NeedsCorrection, []run.Purpose{run.Plan}, plan},
		{"prepare names prepare: repeat", prepare, run.NeedsCorrection, []run.Purpose{run.Prepare}, prepare},
		{"implement names plan: restart", implement, run.NeedsCorrection, []run.Purpose{run.Plan}, plan},
		{"implement names prepare: restart", implement, run.NeedsCorrection, []run.Purpose{run.Prepare}, prepare},
		{"implement names implement: repeat", implement, run.NeedsCorrection, []run.Purpose{run.Implement}, implement},
		{"implement names several: the earliest wins", implement, run.NeedsCorrection, []run.Purpose{run.Implement, run.Prepare}, prepare},
		{"plan names implement and plan: the earlier wins", plan, run.NeedsCorrection, []run.Purpose{run.Implement, run.Plan}, plan},
		{"a purpose outside the work Passes is ignored", implement, run.NeedsCorrection, []run.Purpose{run.Verify}, implement + 1},
		{"needs_correction without corrections moves on from plan", plan, run.NeedsCorrection, nil, plan + 1},
		{"needs_correction without corrections moves on from implement", implement, run.NeedsCorrection, nil, implement + 1},
		{"satisfied moves on", plan, run.Satisfied, nil, plan + 1},
		{"satisfied moves on from implement", implement, run.Satisfied, nil, implement + 1},
		{"satisfied ignores corrections left by earlier Passes", implement, run.Satisfied, []run.Purpose{run.Plan}, implement + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := followWork(work, c.current, run.PassResult{Verdict: c.verdict, Corrections: c.corrections})
			if got != c.want {
				t.Errorf("followWork(%d, %s, %v) = %d; want %d", c.current, c.verdict, c.corrections, got, c.want)
			}
		})
	}
}

// earliest returns the index of the first work purpose in Pass order that the
// corrections name, whatever order they come in, and the fallback otherwise.
func TestEarliestNamesTheFirstWorkPurposeInPassOrder(t *testing.T) {
	work := applyWork()
	cases := []struct {
		name        string
		corrections []run.Purpose
		fallback    int
		want        int
	}{
		{"implement alone", []run.Purpose{run.Implement}, 7, 2},
		{"order of the corrections is irrelevant", []run.Purpose{run.Implement, run.Prepare}, 7, 1},
		{"plan beats every later purpose", []run.Purpose{run.Implement, run.Prepare, run.Plan}, 7, 0},
		{"none: the fallback", nil, 7, 7},
		{"empty: the fallback", []run.Purpose{}, 3, 3},
		{"only a purpose that is no work Pass: the fallback", []run.Purpose{run.Verify, "elsewhere"}, 5, 5},
		{"a foreign purpose does not hide a work purpose", []run.Purpose{"elsewhere", run.Prepare}, 5, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := earliest(work, c.corrections, c.fallback); got != c.want {
				t.Errorf("earliest(%v, fallback %d) = %d; want %d", c.corrections, c.fallback, got, c.want)
			}
		})
	}
}

// The shared review method and envelope serve two authorities: Apply derives
// its expectations from the bound model, Extract from the fixed artifacts and
// their documented intent, where the proposal is never evidence for itself
// (carried over from draft PR #536 and adapted to the receipt contract).
func TestReviewExpectationsFollowTheWorkflowAuthority(t *testing.T) {
	if mandate := applyVerify().Mandate; !strings.Contains(mandate, "against the bound model") || !strings.Contains(mandate, "probes with the tool check") {
		t.Fatalf("Apply review lacks its bound-model expectations or probes: %s", mandate)
	}
	for _, phase := range []string{ExtractStructurePhase, ExtractModelPhase} {
		_, mandate, _, _ := ExtractPhaseTexts(phase)
		if !strings.Contains(mandate, "fixed artifacts and their documented intent") || !strings.Contains(mandate, "never evidence for itself") || strings.Contains(mandate, "bound model") {
			t.Fatalf("Extract %s has circular or Apply review authority: %s", phase, mandate)
		}
	}
}
