package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	routerCodex  = "application/vnd.markitect.codex-session-profile+json"
	routerClaude = "application/vnd.markitect.claude-session-profile+json"
)

type routerCall struct {
	method string
	ctx    context.Context
	input  any
	io     runport.SessionIO
}

type routerPeer struct {
	mu      sync.Mutex
	calls   []routerCall
	opened  runport.SessionObservation
	stepped runport.StepObservation
	stopped runport.StopObservation
	err     error
	onOpen  func()
}

func (p *routerPeer) record(method string, ctx context.Context, input any, io runport.SessionIO) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, routerCall{method, ctx, input, io})
}

func (p *routerPeer) Open(ctx context.Context, a runport.SessionAdmission, io runport.SessionIO) (runport.SessionObservation, error) {
	p.record("open", ctx, a, io)
	if p.onOpen != nil {
		p.onOpen()
	}
	return p.opened, p.err
}

func (p *routerPeer) Step(ctx context.Context, a runport.StepAdmission, io runport.SessionIO) (runport.StepObservation, error) {
	p.record("step", ctx, a, io)
	return p.stepped, p.err
}

func (p *routerPeer) Inspect(ctx context.Context, a runport.InspectionRequest) (runport.StepObservation, error) {
	p.record("inspect", ctx, a, nil)
	return p.stepped, p.err
}

func (p *routerPeer) Stop(ctx context.Context, a runport.StopRequest, io runport.SessionIO) (runport.StopObservation, error) {
	p.record("stop", ctx, a, io)
	return p.stopped, p.err
}

func routerAdmission(t *testing.T, id runport.SessionID, media string) runport.SessionAdmission {
	t.Helper()
	payload, err := runport.NewPayload(media, "utf-8", []byte(`{"version":1}`))
	if err != nil {
		t.Fatal(err)
	}
	return runport.SessionAdmission{Plan: runport.SessionPlan{ID: id, Profile: payload.Ref()}, Profile: payload, Invocation: "invocation"}
}

func routerStep(a runport.SessionAdmission) runport.StepAdmission {
	return runport.StepAdmission{Session: a.Plan, Plan: runport.StepPlan{Session: a.Plan.ID, ID: "step", SessionBinding: a.Plan.Binding()}}
}

func TestSessionsRouterKeepsMixedProviderSessionsAndResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	channel := &struct{ runport.SessionIO }{}
	providerError := errors.New("partial provider observation")
	peers := []*routerPeer{
		{opened: runport.SessionObservation{ProviderSession: "codex-provider", Uncertain: true}, stepped: runport.StepObservation{ProviderStep: "codex-step", Uncertain: true}, stopped: runport.StopObservation{Reason: "codex stop uncertain"}, err: providerError},
		{opened: runport.SessionObservation{Ready: true}, stepped: runport.StepObservation{ProviderStep: "claude-step", Terminal: true}, stopped: runport.StopObservation{Terminated: true}},
	}
	router := NewSessionsRouter(peers[0], peers[1])
	admissions := []runport.SessionAdmission{routerAdmission(t, "codex-host", routerCodex), routerAdmission(t, "claude-host", routerClaude)}
	for index, a := range admissions {
		peer := peers[index]
		observed, err := router.Open(ctx, a, channel)
		if !reflect.DeepEqual(observed, peer.opened) || err != peer.err {
			t.Fatalf("Open lost provider result: %#v, %v", observed, err)
		}
	}
	// Interleave providers, using distinct Host IDs rather than provider IDs.
	for _, index := range []int{1, 0} {
		a, peer := admissions[index], peers[index]
		step := routerStep(a)
		observed, err := router.Step(ctx, step, channel)
		if !reflect.DeepEqual(observed, peer.stepped) || err != peer.err {
			t.Fatalf("Step lost provider result: %#v, %v", observed, err)
		}
		inspect := runport.InspectionRequest{Session: a.Plan.ID, ProviderSession: peer.opened.ProviderSession, Step: "step"}
		observed, err = router.Inspect(ctx, inspect)
		if !reflect.DeepEqual(observed, peer.stepped) || err != peer.err {
			t.Fatalf("Inspect lost provider result: %#v, %v", observed, err)
		}
		stop := runport.StopRequest{Session: a.Plan.ID, ProviderSession: peer.opened.ProviderSession, Reason: "requested"}
		stopped, err := router.Stop(ctx, stop, channel)
		if !reflect.DeepEqual(stopped, peer.stopped) || err != peer.err {
			t.Fatalf("Stop lost provider result: %#v, %v", stopped, err)
		}
		if _, err := router.Inspect(ctx, inspect); err != peer.err {
			t.Fatalf("Stop discarded route: %v", err)
		}
		want := []routerCall{{"open", ctx, a, channel}, {"step", ctx, step, channel}, {"inspect", ctx, inspect, nil}, {"stop", ctx, stop, channel}, {"inspect", ctx, inspect, nil}}
		if !reflect.DeepEqual(peer.calls, want) {
			t.Fatalf("provider %d received changed or misrouted calls: %#v", index, peer.calls)
		}
	}
}

func TestSessionsRouterRefusesUnknownChangedAndDuplicateSessions(t *testing.T) {
	ctx := context.Background()
	peer := &routerPeer{}
	router := NewSessionsRouter(peer, peer)
	a := routerAdmission(t, "known", routerCodex)
	if _, err := router.Open(ctx, a, nil); err != nil {
		t.Fatal(err)
	}
	for _, media := range []string{routerCodex, routerClaude} {
		duplicate := routerAdmission(t, "known", media)
		if _, err := router.Open(ctx, duplicate, nil); !errors.Is(err, runport.ErrConflict) {
			t.Fatalf("duplicate Open was not refused: %v", err)
		}
	}
	step := routerStep(a)
	step.Session.CredentialRef = "changed"
	if _, err := router.Step(ctx, step, nil); !errors.Is(err, runport.ErrConflict) {
		t.Fatalf("changed SessionPlan accepted: %v", err)
	}
	step = routerStep(a)
	step.Plan.SessionBinding = runport.SessionPlan{}.Binding()
	if _, err := router.Step(ctx, step, nil); !errors.Is(err, runport.ErrConflict) {
		t.Fatalf("changed binding accepted: %v", err)
	}
	step.Plan.Session = "unknown"
	if _, err := router.Step(ctx, step, nil); !errors.Is(err, runport.ErrInvalid) {
		t.Fatalf("unknown Step accepted: %v", err)
	}
	if _, err := router.Inspect(ctx, runport.InspectionRequest{Session: "unknown"}); !errors.Is(err, runport.ErrInvalid) {
		t.Fatalf("unknown Inspect accepted: %v", err)
	}
	if _, err := router.Stop(ctx, runport.StopRequest{Session: "unknown"}, nil); !errors.Is(err, runport.ErrInvalid) {
		t.Fatalf("unknown Stop accepted: %v", err)
	}
	if len(peer.calls) != 1 {
		t.Fatalf("refused requests reached provider: %#v", peer.calls)
	}
}

func TestSessionsRouterRefusesUnselectedAndUnconfiguredProfiles(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*runport.SessionAdmission)
	}{
		{"unknown media", func(a *runport.SessionAdmission) {
			a.Profile, _ = runport.NewPayload("application/json", "utf-8", []byte(`{}`))
			a.Plan.Profile = a.Profile.Ref()
		}},
		{"invalid payload", func(a *runport.SessionAdmission) { a.Profile = runport.Payload{} }},
		{"changed reference", func(a *runport.SessionAdmission) { a.Plan.Profile.Size++ }},
		{"empty ID", func(a *runport.SessionAdmission) { a.Plan.ID = "" }},
		{"wrong encoding", func(a *runport.SessionAdmission) {
			a.Profile, _ = runport.NewPayload(routerCodex, "UTF-8", []byte(`{}`))
			a.Plan.Profile = a.Profile.Ref()
		}},
		{"invalid UTF-8", func(a *runport.SessionAdmission) {
			a.Profile, _ = runport.NewPayload(routerCodex, "utf-8", []byte{0xff})
			a.Plan.Profile = a.Profile.Ref()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			peer := &routerPeer{}
			router := NewSessionsRouter(peer, peer)
			a := routerAdmission(t, "host", routerCodex)
			test.change(&a)
			if _, err := router.Open(context.Background(), a, nil); !errors.Is(err, runport.ErrInvalid) || len(peer.calls) != 0 {
				t.Fatalf("profile refusal: %v, calls=%#v", err, peer.calls)
			}
		})
	}
	for _, media := range []string{routerCodex, routerClaude} {
		peer := &routerPeer{}
		var router runport.Sessions
		if media == routerCodex {
			router = NewSessionsRouter(nil, peer)
		} else {
			router = NewSessionsRouter(peer, nil)
		}
		if _, err := router.Open(context.Background(), routerAdmission(t, "host", media), nil); !errors.Is(err, runport.ErrInvalid) || len(peer.calls) != 0 {
			t.Fatalf("unconfigured provider fell back: %v, calls=%#v", err, peer.calls)
		}
	}
}

func TestSessionsRouterBindsBeforeOpenReturns(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	peer := &routerPeer{err: errors.New("failed after dispatch"), onOpen: func() { close(entered); <-release }}
	router := NewSessionsRouter(peer, nil)
	a := routerAdmission(t, "host", routerCodex)
	done := make(chan error, 1)
	go func() { _, err := router.Open(context.Background(), a, nil); done <- err }()
	<-entered
	if _, err := router.Open(context.Background(), a, nil); !errors.Is(err, runport.ErrConflict) {
		t.Errorf("concurrent duplicate Open was not refused: %v", err)
	}
	if _, err := router.Stop(context.Background(), runport.StopRequest{Session: a.Plan.ID}, nil); err != peer.err {
		t.Errorf("route absent before failed Open returned: %v", err)
	}
	close(release)
	if err := <-done; err != peer.err {
		t.Fatalf("Open error changed: %v", err)
	}
	if _, err := router.Open(context.Background(), a, nil); !errors.Is(err, runport.ErrConflict) {
		t.Fatalf("failed Open was replayed: %v", err)
	}
	if len(peer.calls) != 2 || peer.calls[0].method != "open" || peer.calls[1].method != "stop" {
		t.Fatalf("unexpected dispatch: %#v", peer.calls)
	}
}
