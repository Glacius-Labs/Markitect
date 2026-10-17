package apply_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestControlRecaptureDriftRefusesBeforeRestoreAndRetainsObservation(t *testing.T) {
	for _, kind := range []string{"HEAD", "original external bytes"} {
		t.Run(kind, func(t *testing.T) {
			prepared, facts, s, j := controlFixture(t)
			a, _ := prepared.Prepared()
			m := a.Structural().Model()
			original, _ := prepared.BindRun(facts)
			observed := m.Inputs()
			if kind == "HEAD" {
				observed.Head = "1111111111111111111111111111111111111111"
			} else {
				file := &observed.Raw.Instructions[0]
				file.File, _ = snapshot.NewEntry(file.File.Path(), file.File.Mode(), []byte("Changed external instruction under lock."))
			}
			source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) { return observed, nil }}
			r, err := apply.NewControlService(source, controlAcquire(t, prepared, facts, s)).Open(context.Background(), prepared, facts, controlOptions)
			if _, ok := r.Controlled(); !errors.Is(err, apply.ErrCaptureChanged) || ok || s.closes != 1 || j.reads != 0 {
				t.Fatal("drift became controlled history", err)
			}
			got, ok := r.Recapture()
			if !ok || got.Head != observed.Head || got.Raw.Census.Digest() != observed.Raw.Census.Digest() || r.Binding() != s.binding {
				t.Fatal("actual failed observation was lost")
			}
			again, _ := prepared.BindRun(facts)
			if again.Ref() != original.Ref() {
				t.Fatal("failed recapture replaced original successful input")
			}
		})
	}
}

func TestControlPartialAcquisitionIsClosedWithoutUsingIncompleteSession(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	acquireErr, closeErr := errors.New("partial control acquisition"), errors.New("partial cleanup failed")
	s.closeErr = closeErr
	ctx, cancel := context.WithCancel(context.Background())
	port := controlPort{acquire: func(context.Context, apply.ControlRequest) (apply.ControlSession, error) {
		cancel()
		return s, acquireErr
	}}
	r, err := apply.NewControlService(nil, port).Open(ctx, prepared, facts, controlOptions)
	// An unavailable Source is refused before any control effect.
	if !errors.Is(err, apply.ErrControlUnavailable) || s.closes != 0 {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	r, err = apply.NewControlService(controlSource(t, prepared, s), port).Open(ctx, prepared, facts, controlOptions)
	if _, ok := r.Controlled(); !errors.Is(err, acquireErr) || !errors.Is(err, closeErr) || ok || s.closes != 1 || s.checks != 0 || s.bindings != 0 {
		t.Fatal("partial session was used or cleanup error lost", err)
	}
}

func TestControlRecaptureFailureAndCancellationCloseWithoutReplacingBasis(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial failure", true: "cancellation"}[canceled], func(t *testing.T) {
			prepared, facts, s, j := controlFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("partial source unavailable")
			source := controlledSource{capture: func(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) {
				if canceled {
					cancel()
					return apply.CapturedInputs{Head: "partial-head"}, nil
				}
				return apply.CapturedInputs{Head: "partial-head"}, failure
			}}
			r, err := apply.NewControlService(source, controlAcquire(t, prepared, facts, s)).Open(ctx, prepared, facts, controlOptions)
			want := failure
			if canceled {
				want = context.Canceled
			}
			if _, ok := r.Controlled(); !errors.Is(err, want) || ok || s.closes != 1 || j.reads != 0 {
				t.Fatal("failed capture became controlled", err)
			}
			observed, ok := r.Recapture()
			if !ok || observed.Head != "partial-head" || observed.Guard != "" {
				t.Fatal("partial source observation lost")
			}
		})
	}
}

func TestControlRejectsMismatchedOrUnboundStoreAndEpoch(t *testing.T) {
	for _, kind := range []string{"repository", "store", "epoch"} {
		t.Run(kind, func(t *testing.T) {
			prepared, facts, s, j := controlFixture(t)
			switch kind {
			case "repository":
				s.binding.Repository = "different-repository"
			case "store":
				s.binding.StoreID = ""
			case "epoch":
				s.binding.Epoch = 0
			}
			r, err := apply.NewControlService(controlSource(t, prepared, s), controlAcquire(t, prepared, facts, s)).Open(context.Background(), prepared, facts, controlOptions)
			if _, ok := r.Controlled(); !errors.Is(err, apply.ErrControlBinding) || ok || s.closes != 1 || j.reads != 0 {
				t.Fatal("invalid identity became live", err)
			}
		})
	}
	for _, kind := range []string{"store", "epoch"} {
		t.Run("changed "+kind, func(t *testing.T) {
			prepared, facts, s, j := controlFixture(t)
			c := openControlled(t, prepared, facts, s)
			if kind == "store" {
				s.binding.StoreID = "another-registered-store"
			} else {
				s.binding.Epoch++
			}
			if _, err := c.Journal().Put(context.Background(), c.Basis().Payload()); !errors.Is(err, apply.ErrControlBinding) || j.puts != 0 {
				t.Fatal("changed binding reached journal", err)
			}
		})
	}
}

func TestControlRestoreFailureNeverFallsThroughToEmptyHistory(t *testing.T) {
	prepared, facts, s, j := controlFixture(t)
	failure, cleanup := errors.New("unreachable registered store"), errors.New("release unconfirmed")
	j.readErr, s.closeErr = failure, cleanup
	r, err := apply.NewControlService(controlSource(t, prepared, s), controlAcquire(t, prepared, facts, s)).Open(context.Background(), prepared, facts, controlOptions)
	if _, ok := r.Controlled(); !errors.Is(err, run.ErrUncertain) || !errors.Is(err, cleanup) || ok || s.closes != 1 || j.reads != 1 || j.puts != 0 || j.appends != 0 {
		t.Fatal("failed restore invented state or hid cleanup", err)
	}
	if !strings.Contains(err.Error(), failure.Error()) {
		t.Fatal("original restore failure diagnostic was lost", err)
	}
}

func TestControlCloseRetainsFailureAndInvalidatesJournal(t *testing.T) {
	prepared, facts, s, j := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	failure := errors.New("controller release unconfirmed")
	s.closeErr = failure
	if !errors.Is(c.Close(), failure) || !errors.Is(c.Close(), failure) || s.closes != 1 {
		t.Fatal("cleanup failure was hidden or repeated")
	}
	if _, err := c.Journal().Read(context.Background(), run.EventQuery{Limit: 1}); !errors.Is(err, apply.ErrControlClosed) || j.reads != 1 {
		t.Fatal("failed cleanup left a usable controller", err)
	}
}

func TestControlRejectsInvalidOpeningInputsBeforeAcquisition(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	options := controlOptions
	options.RestoreLimits.Bytes = 0
	acquired := 0
	port := controlPort{acquire: func(context.Context, apply.ControlRequest) (apply.ControlSession, error) { acquired++; return s, nil }}
	service := apply.NewControlService(controlSource(t, prepared, s), port)
	if _, err := service.Open(context.Background(), prepared, facts, options); err == nil || acquired != 0 {
		t.Fatal("invalid read limits acquired control", err)
	}
	if _, err := service.Open(context.Background(), apply.CapturedSettingsResult{}, facts, controlOptions); err == nil || acquired != 0 {
		t.Fatal("incomplete preparation acquired control", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Open(ctx, prepared, facts, controlOptions); !errors.Is(err, context.Canceled) || acquired != 0 {
		t.Fatal("canceled open acquired control", err)
	}
}
