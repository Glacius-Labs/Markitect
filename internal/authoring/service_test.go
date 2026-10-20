package authoring

import (
	"context"
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// fakePublisher answers every publication with one receipt.
type fakePublisher struct {
	receipt apply.PublicationReceipt
	err     error
}

func (f fakePublisher) Preflight(_ context.Context, r apply.PublicationRequest, _ apply.ControlCheck) (apply.PublicationPermit, error) {
	return apply.PublicationPermit{Request: r, Token: "t"}, nil
}

func (f fakePublisher) Publish(context.Context, apply.PublicationPermit, apply.ControlCheck) (apply.PublicationReceipt, error) {
	return f.receipt, f.err
}

func (f fakePublisher) Inspect(context.Context, apply.PublicationRequest) (apply.PublicationReceipt, error) {
	return f.receipt, f.err
}

type noControl struct{}

func (noControl) Check(context.Context) error { return nil }

func serviceWith(t *testing.T, publisher apply.Publisher, lock Locker) *Service {
	t.Helper()
	s := New("repo", nil, publisher, lock)
	s.proposals["configuration-1"] = &proposal{Proposal: Proposal{ID: "configuration-1", State: Proposed, Check: Check{Status: "passed"}}}
	return s
}

func receipt(outcome apply.PublicationOutcome, state apply.PathState) apply.PublicationReceipt {
	path, _ := snapshot.NewPath(ExecutionPath)
	return apply.PublicationReceipt{Outcome: outcome, Paths: []apply.PublishedPath{{Path: path, State: state}}, Reason: string(outcome)}
}

func TestAcceptIsAcceptedOnlyForAnAppliedPublication(t *testing.T) {
	free := func(context.Context) (apply.ControlCheck, func(), error) { return noControl{}, func() {}, nil }
	for _, c := range []struct {
		outcome apply.PublicationOutcome
		state   apply.PathState
		err     error
		want    string
	}{
		{apply.PublicationApplied, apply.PathAfter, nil, Accepted},
		{apply.PublicationNoChange, apply.PathAfter, nil, Accepted},
		{apply.PublicationPartial, apply.PathOther, errors.New("write failed"), Failed},
		{apply.PublicationUnknown, apply.PathOther, nil, Failed},
		{apply.PublicationStale, apply.PathBefore, nil, Failed},
	} {
		s := serviceWith(t, fakePublisher{receipt: receipt(c.outcome, c.state), err: c.err}, free)
		p, err := s.Accept(context.Background(), "configuration-1")
		if p.State != c.want || p.Receipt == nil || p.Receipt.Outcome != string(c.outcome) || p.Receipt.Paths[0].State != string(c.state) {
			t.Errorf("%s: proposal %+v", c.outcome, p)
		}
		if (c.want == Accepted) != (err == nil) {
			t.Errorf("%s: err = %v", c.outcome, err)
		}
		if _, err := s.Accept(context.Background(), "configuration-1"); err == nil {
			t.Errorf("%s: a decided proposal was accepted again", c.outcome)
		}
	}
}

func TestAcceptThatCannotTakeTheFenceStaysOpen(t *testing.T) {
	held := func(context.Context) (apply.ControlCheck, func(), error) {
		return nil, nil, errors.New("an Apply holds the state directory")
	}
	s := serviceWith(t, fakePublisher{receipt: receipt(apply.PublicationApplied, apply.PathAfter)}, held)
	p, err := s.Accept(context.Background(), "configuration-1")
	if err == nil || p.State != Proposed || p.Receipt != nil {
		t.Fatalf("a fenced acceptance = %+v, %v", p, err)
	}
	s.Wait()
}

func TestAcceptNeedsAPassedCheck(t *testing.T) {
	s := serviceWith(t, fakePublisher{receipt: receipt(apply.PublicationApplied, apply.PathAfter)}, nil)
	s.proposals["configuration-1"].Check.Status = "not_completed"
	if p, err := s.Accept(context.Background(), "configuration-1"); err == nil || p.State != Proposed {
		t.Fatalf("an unchecked proposal = %+v, %v", p, err)
	}
}
