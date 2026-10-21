package checks

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type commandPeer struct {
	call func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error)
}

func (p commandPeer) RunCommand(ctx context.Context, s container.ContainerSpec, a, e []string, max int64) (container.CommandResult, error) {
	return p.call(ctx, s, a, e, max)
}

type workspacePeer struct {
	request     run.WorkspaceSpec
	requests    []run.WorkspaceSpec
	materialize func(run.WorkspaceSpec) (run.WorkspaceObservation, error)
	close       func(context.Context, run.WorkspaceID) (run.WorkspaceObservation, error)
	closed      int
}

func (p *workspacePeer) Materialize(_ context.Context, s run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	if p.request.ID == "" {
		p.request = s
	}
	p.requests = append(p.requests, s)
	if p.materialize != nil {
		return p.materialize(s)
	}
	volume := "private-copy"
	if len(p.requests) > 1 {
		volume = "private-home"
	}
	return run.WorkspaceObservation{ID: s.ID, Volume: volume, Tree: s.Tree.Digest()}, nil
}
func (p *workspacePeer) Harvest(context.Context, run.HarvestRequest) (run.WorkspaceHarvest, error) {
	panic("check does not harvest its mutated copy")
}
func (p *workspacePeer) Peek(context.Context, run.HarvestRequest) (run.WorkspaceHarvest, error) {
	panic("check does not peek its mutated copy")
}
func (p *workspacePeer) Close(ctx context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	p.closed++
	if p.close != nil {
		return p.close(ctx, id)
	}
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}
func checkRequest(t *testing.T) run.CheckRequest {
	t.Helper()
	p, err := snapshot.NewPath("pkg/tool")
	if err != nil {
		t.Fatal(err)
	}
	e, err := snapshot.NewEntry(p, snapshot.Executable, []byte("exact\x00bytes\n"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshot.NewTree([]snapshot.Entry{e})
	if err != nil {
		t.Fatal(err)
	}
	d, err := snapshot.DirectoryAt(mustCheckPath(t, "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	return run.CheckRequest{ID: "test-check", Tree: tree, ImageID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Executable: "/usr/bin/check", Args: []string{"prefix", "space arg", "$(literal); | >"}, Env: []string{"A=literal=$value", "EMPTY="}, Directory: d, Limits: run.CheckLimits{MilliCPUs: 1250, MemoryBytes: 123456, Processes: 17}, Timeout: time.Second, MaxOutputBytes: 64}
}
func mustCheckPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, e := snapshot.NewPath(s)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func passedCommand(req run.CheckRequest) container.CommandResult {
	return container.CommandResult{ImageID: req.ImageID, Started: true, Exited: true, ExitCode: 0, Output: []byte("stdout\nstderr\n")}
}
func TestRunnerBindsExactCopyLiteralCommandAndResources(t *testing.T) {
	req := checkRequest(t)
	w := &workspacePeer{}
	labels := map[string]string{"markitect.store": "owned-attempt", "markitect.check": "caller-cannot-override"}
	called := 0
	r, err := NewRunnerWithRuntime(commandPeer{func(ctx context.Context, s container.ContainerSpec, a, e []string, max int64) (container.CommandResult, error) {
		called++
		if s.Labels["markitect.store"] != "owned-attempt" || s.Labels["markitect.check"] != string(req.ID) {
			t.Fatalf("owned cleanup labels lost or redirected: %+v", s.Labels)
		}
		if !w.request.Tree.Equal(req.Tree) || w.request.ID == "" || s.Name != string(w.request.ID) {
			t.Fatal("copy identity not fixed")
		}
		if s.Image != req.ImageID || s.Workdir != "/work/pkg" || s.CPUs != 1.25 || s.MemoryBytes != req.Limits.MemoryBytes || s.Pids != 17 || s.Network != "none" || s.User != "1000:1000" {
			t.Fatalf("spec: %+v", s)
		}
		if len(w.requests) != 2 || w.requests[1].Tree.EntryCount() != 0 || w.requests[0].ID == w.requests[1].ID {
			t.Fatal("home is not independently empty")
		}
		if !reflect.DeepEqual(s.Mounts, []container.Mount{{Volume: "private-copy", Target: "/work", NoCopy: true}, {Volume: "private-home", Target: "/home/agent", NoCopy: true}}) {
			t.Fatalf("mounts: %+v", s.Mounts)
		}
		if !reflect.DeepEqual(a, append([]string{req.Executable}, req.Args...)) || !reflect.DeepEqual(e, req.Env) || max != req.MaxOutputBytes {
			t.Fatal("literal command changed")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		return passedCommand(req), nil
	}}, w, labels)
	if err != nil {
		t.Fatal(err)
	}
	labels["markitect.store"] = "foreign-after-construction"
	got, err := r.Run(context.Background(), req)
	if err != nil || !got.Passed(req) || called != 1 || w.closed != 2 {
		t.Fatalf("receipt=%+v err=%v calls=%d close=%d", got, err, called, w.closed)
	}
}

func TestRunnerKeepsVerifiedEmptyVolumesFreeOfImageCopyUp(t *testing.T) {
	req := checkRequest(t)
	req.Tree = snapshot.Tree{}
	req.Directory = snapshot.RootDirectory()
	w := &workspacePeer{}
	r, _ := NewRunnerWithRuntime(commandPeer{func(_ context.Context, s container.ContainerSpec, _ []string, _ []string, _ int64) (container.CommandResult, error) {
		if s.Workdir != "/work" || len(s.Mounts) != 2 || !s.Mounts[0].NoCopy || !s.Mounts[1].NoCopy {
			t.Fatalf("empty volume copy-up not disabled: %+v", s)
		}
		if w.requests[0].Tree.EntryCount() != 0 || w.requests[1].Tree.EntryCount() != 0 {
			t.Fatal("unexpected input tree")
		}
		return passedCommand(req), nil
	}}, w, nil)
	got, err := r.Run(context.Background(), req)
	if err != nil || !got.Passed(req) || w.closed != 2 {
		t.Fatalf("receipt=%+v err=%v", got, err)
	}
}
func TestRunnerRefusesUnverifiedMaterializationAndClosesPartialCopy(t *testing.T) {
	for _, kind := range []string{"identity", "tree", "uncertain", "closed", "volume", "error"} {
		t.Run(kind, func(t *testing.T) {
			req := checkRequest(t)
			w := &workspacePeer{materialize: func(s run.WorkspaceSpec) (run.WorkspaceObservation, error) {
				o := run.WorkspaceObservation{ID: s.ID, Volume: "private", Tree: s.Tree.Digest()}
				switch kind {
				case "identity":
					o.ID = "different"
				case "tree":
					o.Tree = snapshot.Tree{}.Digest()
				case "uncertain":
					o.Uncertain = true
				case "closed":
					o.Closed = true
				case "volume":
					o.Volume = ""
				case "error":
					return o, errors.New("failed readback")
				}
				return o, nil
			}}
			r, _ := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
				t.Fatal("unverified copy executed")
				return container.CommandResult{}, nil
			}}, w, nil)
			got, err := r.Run(context.Background(), req)
			if err == nil || got.Passed(req) || got.Started || w.closed != 1 {
				t.Fatalf("receipt=%+v err=%v closed=%d", got, err, w.closed)
			}
		})
	}
}
func TestRunnerPreservesFailedExitAndBoundsCombinedOutput(t *testing.T) {
	req := checkRequest(t)
	req.MaxOutputBytes = 5
	w := &workspacePeer{}
	r, _ := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
		o := passedCommand(req)
		o.ExitCode = 23
		return o, nil
	}}, w, nil)
	got, err := r.Run(context.Background(), req)
	if err != nil || got.Passed(req) || got.ExitCode != 23 || !got.Exited || got.Uncertain || !got.Truncated || string(got.Output) != "stdou" {
		t.Fatalf("receipt=%+v err=%v", got, err)
	}
}
func TestRunnerCannotPassUncertainOrMismatchedRuntimeFacts(t *testing.T) {
	for _, kind := range []string{"image", "missing exit", "contradiction", "uncertain", "error"} {
		t.Run(kind, func(t *testing.T) {
			req := checkRequest(t)
			w := &workspacePeer{}
			r, _ := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
				o := passedCommand(req)
				switch kind {
				case "image":
					o.ImageID = "other"
				case "missing exit":
					o.Exited = false
				case "contradiction":
					o.Started = false
				case "uncertain":
					o.Uncertain = true
				case "error":
					return o, errors.New("lost observation")
				}
				return o, nil
			}}, w, nil)
			got, _ := r.Run(context.Background(), req)
			if got.Passed(req) || !got.Uncertain || w.closed != 2 {
				t.Fatalf("receipt=%+v", got)
			}
		})
	}
}
func TestRunnerCancellationAndTimeoutCannotBecomePass(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "deadline"}[deadline], func(t *testing.T) {
			req := checkRequest(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if deadline {
				req.Timeout = time.Millisecond
			}
			w := &workspacePeer{close: func(ctx context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
				if ctx.Err() != nil {
					t.Fatal("cleanup inherited cancellation")
				}
				return run.WorkspaceObservation{ID: id, Closed: true}, nil
			}}
			r, _ := NewRunnerWithRuntime(commandPeer{func(ctx context.Context, _ container.ContainerSpec, _ []string, _ []string, _ int64) (container.CommandResult, error) {
				if !deadline {
					cancel()
				}
				<-ctx.Done()
				return passedCommand(req), ctx.Err()
			}}, w, nil)
			got, err := r.Run(ctx, req)
			if err == nil || got.Passed(req) || !got.Uncertain || got.TimedOut != deadline || w.closed != 2 {
				t.Fatalf("receipt=%+v err=%v", got, err)
			}
		})
	}
}
func TestRunnerCleanupFailureCannotBecomePass(t *testing.T) {
	for _, kind := range []string{"error", "identity", "not closed", "uncertain"} {
		t.Run(kind, func(t *testing.T) {
			req := checkRequest(t)
			w := &workspacePeer{close: func(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
				o := run.WorkspaceObservation{ID: id, Closed: true}
				switch kind {
				case "error":
					return o, errors.New("busy volume")
				case "identity":
					o.ID = "other"
				case "not closed":
					o.Closed = false
				case "uncertain":
					o.Uncertain = true
				}
				return o, nil
			}}
			r, _ := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
				return passedCommand(req), nil
			}}, w, nil)
			got, err := r.Run(context.Background(), req)
			if err == nil || got.Passed(req) || !got.Uncertain || w.closed != 2 {
				t.Fatalf("receipt=%+v err=%v", got, err)
			}
		})
	}
}

func TestRunnerRefusesUnverifiedHomeAndClosesBothCopies(t *testing.T) {
	for _, kind := range []string{"tree", "same volume", "error"} {
		t.Run(kind, func(t *testing.T) {
			req := checkRequest(t)
			w := &workspacePeer{}
			w.materialize = func(s run.WorkspaceSpec) (run.WorkspaceObservation, error) {
				o := run.WorkspaceObservation{ID: s.ID, Volume: "private-copy", Tree: s.Tree.Digest()}
				if len(w.requests) == 2 {
					o.Volume = "private-home"
					switch kind {
					case "tree":
						o.Tree = req.Tree.Digest()
					case "same volume":
						o.Volume = "private-copy"
					case "error":
						return o, errors.New("home readback failed")
					}
				}
				return o, nil
			}
			var first context.Context
			w.close = func(ctx context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
				if first == nil {
					first = ctx
				} else if first == ctx || ctx.Err() != nil {
					t.Fatal("closures share an expired context")
				}
				return run.WorkspaceObservation{ID: id, Closed: true}, nil
			}
			r, _ := NewRunnerWithRuntime(commandPeer{func(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error) {
				t.Fatal("unverified home executed")
				return container.CommandResult{}, nil
			}}, w, nil)
			got, err := r.Run(context.Background(), req)
			if err == nil || got.Passed(req) || got.Started || w.closed != 2 {
				t.Fatalf("receipt=%+v err=%v closed=%d", got, err, w.closed)
			}
		})
	}
}
func TestRunnerRejectsInvalidRequestBeforeCreatingCopy(t *testing.T) {
	req := checkRequest(t)
	req.Args = []string{"bad\x00arg"}
	w := &workspacePeer{}
	r, _ := NewRunnerWithRuntime(commandPeer{}, w, nil)
	got, err := r.Run(context.Background(), req)
	if err == nil || got.Passed(req) || w.request.ID != "" || w.closed != 0 {
		t.Fatalf("receipt=%+v err=%v", got, err)
	}
}
