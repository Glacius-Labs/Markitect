package apply

import (
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Direct runs one Implement, in which every Manager plans and prepares its own
// part, before independent Verify; every correction is a new Implement. Staged
// runs Plan, Prepare and Implement. Both share Verify and its guidance.
func TestStrategiesComposeTheOnePassProcess(t *testing.T) {
	work, review := Direct.work(), Direct.verify()
	if len(work) != 1 || work[0].Purpose != run.Implement || !work[0].Writes || work[0].Plans || !work[0].BindWaits {
		t.Fatalf("direct work Passes %+v", work)
	}
	for _, spec := range []run.PassSpec{work[0], review} {
		if len(spec.Corrections) != 1 || spec.Corrections[0] != run.Implement {
			t.Errorf("direct %s corrections %v", spec.Purpose, spec.Corrections)
		}
	}
	for _, duty := range []string{"First settle your own approach", "Your basis is what your children need from you before they start", "Never plan or realize your children's", "this records the dependency"} {
		if !strings.Contains(work[0].Mandate, duty) {
			t.Errorf("the direct Implement mandate lacks %q", duty)
		}
	}
	staged := Staged.work()
	if len(staged) != 3 || staged[0].Purpose != run.Plan || !staged[0].Plans || staged[1].Purpose != run.Prepare || staged[2].Purpose != run.Implement {
		t.Fatalf("staged work Passes %+v", staged)
	}
	for _, spec := range append(staged, Staged.verify()) {
		if spec.BindWaits || !slices.Equal(spec.Corrections, []run.Purpose{run.Plan, run.Prepare, run.Implement}) {
			t.Errorf("staged %s binds waits or has corrections %v", spec.Purpose, spec.Corrections)
		}
	}
	if d, s := Direct.verify(), Staged.verify(); d.Purpose != run.Verify || !d.Review || d.Mandate != s.Mandate || d.Guidance != s.Guidance {
		t.Errorf("the strategies' Verify differ: %+v / %+v", d, s)
	}
	// Staged keeps Main's Root duty: it accepts a plan, which direct has not.
	if !strings.Contains(Staged.rootDuty(), "accept a plan that depends on it") || !strings.Contains(Direct.rootDuty(), "accept a submission that depends on it") ||
		strings.Replace(Direct.rootDuty(), "a submission", "a plan", 1) != Staged.rootDuty() {
		t.Errorf("Root duties: staged %q, direct %q", Staged.rootDuty(), Direct.rootDuty())
	}
}

// followWork picks the work Pass after a sealed one: its own open findings repeat
// it or restart at an earlier purpose, but a later purpose they name comes in
// order, so no forward Pass is skipped.
func TestFollowWorkNeverSkipsAForwardPass(t *testing.T) {
	work := Staged.work()
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
	work := Staged.work()
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
	if mandate := Direct.verify().Mandate; !strings.Contains(mandate, "against the bound model") || !strings.Contains(mandate, "probes with the tool check") {
		t.Fatalf("Apply review lacks its bound-model expectations or probes: %s", mandate)
	}
	// A public behavior the model leaves open is asked, not defaulted (A4 of
	// the hardening evaluation picked an unstated name maximum of 80).
	for _, s := range strategies {
		if !strings.Contains(s.work()[0].Mandate, "no implementation choice") || !strings.Contains(s.work()[0].Mandate, "stays yours to decide and document") || !strings.Contains(s.rootDuty(), "ask the user with the concrete alternatives") {
			t.Fatalf("%s lets a Manager settle a public behavior the model leaves open", s)
		}
	}
	for _, phase := range []string{ExtractStructurePhase, ExtractModelPhase} {
		_, mandate, _, _ := ExtractPhaseTexts(phase)
		if !strings.Contains(mandate, "fixed artifacts and their documented intent") || !strings.Contains(mandate, "never evidence for itself") || strings.Contains(mandate, "bound model") {
			t.Fatalf("Extract %s has circular or Apply review authority: %s", phase, mandate)
		}
	}
}
