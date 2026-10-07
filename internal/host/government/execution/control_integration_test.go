package execution

import (
	"context"
	"errors"
	"testing"
)

type refusingControl struct {
	runID                                  string
	denyActor, denyPromotion, denyComplete bool
	reservations                           int
	checkpoints                            []Report
}

func (c *refusingControl) StartRun(id, path string) error { c.runID = id; return nil }
func (c *refusingControl) Fence() error                   { return nil }
func (c *refusingControl) ReserveActor(id string, sequence int, phase string, spec RunnerSpec) error {
	c.reservations++
	if c.denyActor {
		return errors.New("durable actor reservation refused")
	}
	return nil
}
func (c *refusingControl) CompleteActor(id string, record ActorRecord) error { return nil }
func (c *refusingControl) ReserveRepair(id, area string, attempt int) error  { return nil }
func (c *refusingControl) Checkpoint(report Report) error {
	c.checkpoints = append(c.checkpoints, report)
	if c.denyPromotion && report.Stage == "promote" {
		return errors.New("durable promotion checkpoint refused")
	}
	if c.denyComplete && report.Stage == "complete" {
		return errors.New("durable terminal checkpoint refused")
	}
	return nil
}

func TestRunTerminalCheckpointFailureRetainsIncompleteReport(t *testing.T) {
	opts := fixtureOptions(t)
	opts.Control = &refusingControl{denyComplete: true}
	report, err := Run(context.Background(), opts)
	if err == nil || report.Status != "incomplete" || report.Promotion == nil || report.Promotion.Status != "promoted" {
		t.Fatalf("completion checkpoint failure was hidden: %+v: %v", report, err)
	}
	assertRetainedReport(t, report)
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != report.CandidateCommit {
		t.Fatal("terminal record failure rolled back the already performed CAS")
	}
}

func TestRunRequiresDurableControlBeforeExternalEffects(t *testing.T) {
	for _, phase := range []string{"actor", "promotion"} {
		t.Run(phase, func(t *testing.T) {
			opts := fixtureOptions(t)
			control := &refusingControl{denyActor: phase == "actor", denyPromotion: phase == "promotion"}
			opts.Control = control
			report, err := Run(context.Background(), opts)
			if err == nil || report.Promotion != nil || report.RunID != control.runID {
				t.Fatalf("control refusal failed open: %+v: %v", report, err)
			}
			if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != opts.Runtime.ExpectedBase {
				t.Fatal("control refusal changed Active")
			}
			if phase == "actor" && (len(report.Actors) != 0 || control.reservations != 1) {
				t.Fatal("actor ran without durable reservation")
			}
			if phase == "promotion" {
				if len(report.Actors) != 4 || report.Decision == nil || len(control.checkpoints) < 2 {
					t.Fatal("refusal was not at the fully bound promotion boundary")
				}
				bound := control.checkpoints[len(control.checkpoints)-1]
				if bound.Candidate == nil || bound.Evidence == nil || bound.Decision == nil || bound.CandidateCommit == "" {
					t.Fatal("pre-promotion checkpoint omitted recovery bindings")
				}
			}
		})
	}
}
