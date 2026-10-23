package checks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CommandRuntime observes a command in its own hardened container. Production
// implementations must preserve Engine.RunCommand's physical observations.
type CommandRuntime interface {
	RunCommand(context.Context, container.ContainerSpec, []string, []string, int64) (container.CommandResult, error)
}

// closeBound bounds the closure of each disposable volume after a check. A
// container engine under the load of a whole Manager team took longer than ten
// seconds to remove a volume, which made otherwise observed probes uncertain.
const closeBound = 45 * time.Second

// Runner executes allowed checks on fresh, verified workspace copies. It keeps
// no task or review state; run decides what the resulting receipt establishes.
type Runner struct {
	runtime    CommandRuntime
	workspaces DisposableCopies
	labels     map[string]string
}

var _ run.Checks = (*Runner)(nil)

// NewRunner composes the actual container runtime with an independently
// configured workspace adapter. Workspace helper images and check images have
// separate identities; every check runs only in the request's exact image.
// Ownership labels are copied into each command's container specification.
func NewRunner(engine *container.Engine, workspaces DisposableCopies, labels map[string]string) (*Runner, error) {
	if engine == nil {
		return nil, fmt.Errorf("check runner needs a container engine")
	}
	return NewRunnerWithRuntime(engine, workspaces, labels)
}

// NewRunnerWithRuntime supplies an observation-compatible runtime, including
// deterministic peers for source tests. A workspace must be private to Run
// until its command has ended and Close has completed.
// Labels are cloned at construction and per command; markitect.check is reserved.
func NewRunnerWithRuntime(runtime CommandRuntime, workspaces DisposableCopies, labels map[string]string) (*Runner, error) {
	if runtime == nil || workspaces == nil {
		return nil, fmt.Errorf("check runner needs runtime and workspaces")
	}
	return &Runner{runtime: runtime, workspaces: workspaces, labels: maps.Clone(labels)}, nil
}

func (r *Runner) Run(ctx context.Context, req run.CheckRequest) (receipt run.CheckReceipt, err error) {
	receipt.ID = req.ID
	defer func() {
		if err != nil && receipt.Reason == "" {
			receipt.Reason = err.Error()
		}
	}()
	if r == nil || r.runtime == nil || r.workspaces == nil || !req.Valid() {
		return receipt, fmt.Errorf("invalid check request or runner")
	}
	if path.Clean(req.Executable) != req.Executable || strings.ContainsRune(req.Executable, 0) {
		return receipt, fmt.Errorf("check executable must be a clean absolute container path")
	}
	for _, arg := range req.Args {
		if strings.ContainsRune(arg, 0) {
			return receipt, fmt.Errorf("check argument contains NUL")
		}
	}
	if req.Limits.Processes > int64(int(^uint(0)>>1)) {
		return receipt, fmt.Errorf("check process limit cannot be represented")
	}
	if err := ctx.Err(); err != nil {
		receipt.Uncertain = true
		receipt.Reason = err.Error()
		return receipt, err
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return receipt, fmt.Errorf("allocate disposable check identity: %w", err)
	}
	name := "check-" + hex.EncodeToString(token)
	id := CopyID(name)
	identities := []CopyID{id}
	limited, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	copy, materializeErr := r.workspaces.Materialize(limited, CopySpec{ID: id, Tree: req.Tree, Directories: req.Directories})
	// Even a failed materialization may have created a volume. Close its exact
	// private identity using a bounded context that survives caller cancellation.
	defer func() {
		operationErr := limited.Err()
		cancel()
		// The volumes close concurrently, so cleanup takes at most one bound.
		type closure struct {
			observed CopyObservation
			err      error
		}
		closures := make([]closure, len(identities))
		var wg sync.WaitGroup
		for i, identity := range identities {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), closeBound)
				defer stop()
				closures[i].observed, closures[i].err = r.workspaces.Close(cleanup, identity)
			}()
		}
		wg.Wait()
		receipt.CopiesClosed = true
		for i, identity := range identities {
			closed, closeErr := closures[i].observed, closures[i].err
			if closeErr != nil || closed.ID != identity || !closed.Closed || closed.Uncertain {
				receipt.Uncertain = true
				receipt.CopiesClosed = false
				failure := fmt.Errorf("disposable check workspace %s closure unconfirmed: %s", identity, closed.Reason)
				if closeErr != nil {
					failure = fmt.Errorf("close disposable check workspace %s: %w", identity, closeErr)
				}
				err = errors.Join(err, failure)
				receipt.Reason = err.Error()
			}
		}
		if operationErr != nil {
			receipt.Uncertain = true
			receipt.TimedOut = errors.Is(operationErr, context.DeadlineExceeded)
			err = errors.Join(err, operationErr)
			receipt.Reason = err.Error()
		}
	}()
	if materializeErr != nil || copy.ID != id || copy.Closed || copy.Uncertain || !copy.usable() || copy.Tree != req.Tree.Digest() || !directoryPathsEqual(copy.Directories, req.Directories) {
		err = fmt.Errorf("disposable check tree was not confirmed: %s", copy.Reason)
		if materializeErr != nil {
			err = fmt.Errorf("materialize disposable check tree: %w", materializeErr)
		}
		receipt.Uncertain = copy.Uncertain
		receipt.Reason = err.Error()
		return receipt, err
	}
	receipt.Tree = copy.Tree
	for _, d := range copy.Directories {
		receipt.Directories = append(receipt.Directories, d.String())
	}
	// The runtime image keeps tool caches in /home/agent. Give the check an
	// independently verified empty disposable home without rewriting fixed Env.
	homeID := CopyID(name + "-home")
	identities = append(identities, homeID)
	empty := snapshot.Tree{}
	home, homeErr := r.workspaces.Materialize(limited, CopySpec{ID: homeID, Tree: empty})
	if homeErr != nil || home.ID != homeID || home.Closed || home.Uncertain || !home.usable() || home.sameLocation(copy) || home.Tree != empty.Digest() {
		err = fmt.Errorf("disposable check home was not confirmed: %s", home.Reason)
		if homeErr != nil {
			err = fmt.Errorf("materialize disposable check home: %w", homeErr)
		}
		receipt.Uncertain = home.Uncertain
		return receipt, err
	}
	workdir := "/work"
	if !req.Directory.IsRoot() {
		workdir += "/" + req.Directory.String()
	}
	labels := make(map[string]string, len(r.labels)+1)
	maps.Copy(labels, r.labels)
	labels["markitect.check"] = string(req.ID)
	spec := container.ContainerSpec{
		Name: name, Image: req.ImageID, Labels: labels, User: "1000:1000",
		Mounts: []container.Mount{copy.mount("/work"), home.mount("/home/agent")}, Workdir: workdir,
		CPUs: float64(req.Limits.MilliCPUs) / 1000, MemoryBytes: req.Limits.MemoryBytes, Pids: int(req.Limits.Processes),
		TmpBytes: 64 << 20, Network: "none",
	}
	argv := append([]string{req.Executable}, req.Args...)
	observed, commandErr := r.runtime.RunCommand(limited, spec, argv, append([]string(nil), req.Env...), req.MaxOutputBytes)
	receipt.ImageID = observed.ImageID
	receipt.Started, receipt.Exited, receipt.ExitCode = observed.Started, observed.Exited, observed.ExitCode
	receipt.Output = append([]byte(nil), observed.Output...)
	receipt.Truncated, receipt.Uncertain, receipt.Reason = observed.Truncated, observed.Uncertain, observed.Reason
	if int64(len(receipt.Output)) > req.MaxOutputBytes {
		receipt.Output = receipt.Output[:req.MaxOutputBytes]
		receipt.Truncated = true
	}
	if observed.Started && observed.ImageID != req.ImageID || observed.Started && !observed.Exited || observed.Exited && !observed.Started {
		receipt.Uncertain = true
		commandErr = errors.Join(commandErr, fmt.Errorf("check image, start or exit was not confirmed"))
	}
	if !observed.Started && commandErr == nil {
		commandErr = fmt.Errorf("check did not start")
	}
	if commandErr != nil {
		// An error accompanying an otherwise passing exit cannot be dropped by
		// Passed. Known startup refusal retains the runtime's certainty.
		if observed.Started && observed.Exited {
			receipt.Uncertain = true
		}
		receipt.Reason = commandErr.Error()
	}
	return receipt, commandErr
}

func directoryPathsEqual(a, b []snapshot.Path) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
