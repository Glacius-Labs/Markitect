package apply

import (
	"crypto/sha256"
	"fmt"
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
	for _, duty := range []string{"First settle your own approach", "Supply contracts or decisions your children actually need", "Never plan or realize your children's", "this records the dependency"} {
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
	// One Root duty in every strategy: a confirmed model problem ends the run,
	// since nobody answers from outside the tree (closed Runs, PR #560).
	if Direct.rootDuty() != Staged.rootDuty() || !strings.Contains(Direct.rootDuty(), "final assessment problem field") || !strings.Contains(Direct.rootDuty(), "instead of letting a Manager pick a default") ||
		!strings.Contains(Direct.rootDuty(), "a new run starts") || strings.Contains(Direct.rootDuty(), "ask the user") {
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
	if mandate := Direct.verify().Mandate; !strings.Contains(mandate, "against the bound model") || !strings.Contains(mandate, "check with existing: true") || !strings.Contains(mandate, "probes for material gaps") {
		t.Fatalf("Apply review lacks its bound-model expectations or probes: %s", mandate)
	}
	// A public behavior the model leaves open is a model problem, not a default
	// (A4 of the hardening evaluation picked an unstated name maximum of 80),
	// and nobody outside the tree answers it during the run.
	for _, s := range strategies {
		mandate := s.work()[0].Mandate
		if !strings.Contains(mandate, "no implementation choice") || !strings.Contains(mandate, "stays yours to decide and document") || !strings.Contains(mandate, "report it to your parent as a model problem") ||
			strings.Contains(mandate, "the user") || !strings.Contains(s.rootDuty(), "instead of letting a Manager pick a default") {
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

// Apply fulfills the model: the work mandates preserve a conforming stand
// against mere permission, and the review distinguishes deviations,
// suspicions and suggestions (owner refinement of 21 October 2026).
func TestMandatesBoundChangesAndFindingsToConformance(t *testing.T) {
	for _, s := range strategies {
		implement := s.work()[len(s.work())-1].Mandate
		if !strings.Contains(implement, "that the model permits a change is no reason to make it") || !strings.Contains(implement, "which fulfillment it serves") {
			t.Errorf("%s Implement does not bound its changes to fulfillment", s)
		}
	}
	review := Direct.verify()
	for _, want := range []string{"is suspected", "names no correction", "are suggestions", "no new evidence"} {
		if !strings.Contains(review.Guidance, want) {
			t.Errorf("the Verify guidance lacks %q", want)
		}
	}
	if !strings.Contains(review.Mandate, "is a suspected finding, never a suggestion") {
		t.Error("the Verify mandate parks an unsettled question elsewhere")
	}
}

// The Model assessment's mandate and guidance are Session C's method of
// experiments/model-assessment/assessment-pass.json at 7db130d7 byte for byte;
// a change of the method changes these digests deliberately.
func TestTheAssessmentMethodIsPinned(t *testing.T) {
	spec := assessSpec()
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(spec.Mandate))); got != "17353a41dfef65452c925ae552092884de4c4c703654af9e03c090fe1b56dad5" {
		t.Errorf("mandate digest %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(spec.Guidance))); got != "b2f5ba101ea459e7f044caa41c154c7714482c7594db53a7b25d5b3ca517a80d" {
		t.Errorf("guidance digest %s", got)
	}
	if !spec.Review || !spec.Assesses || !slices.Equal(spec.Corrections, []run.Purpose{"model_error", "model_gap", "unclassified"}) {
		t.Errorf("spec %+v", spec)
	}
}
