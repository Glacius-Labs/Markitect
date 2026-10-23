package gitrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type gitSessionRuntime interface {
	StartOwnedExec(context.Context, container.ContainerSpec, []string) (container.ExecSession, error)
}

// StartGit reserves all possible assignment volumes before starting one Git
// container for this Apply. A failed start retains its volumes; it is not retried.
func (w *Workspaces) StartGit(ctx context.Context, ids []run.WorkspaceID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	runtime, ok := w.runtime.(gitSessionRuntime)
	if !ok || len(ids) == 0 || w.reserved != nil || len(w.owned) != 0 {
		return fmt.Errorf("Apply Git needs an exec runtime and a fresh, nonempty workspace reservation")
	}
	seen := make(map[run.WorkspaceID]bool)
	for _, id := range ids {
		if !(run.WorkspaceSpec{ID: id}).Valid() || !container.ValidSubpath(string(id)) || strings.Contains(string(id), "/") || seen[id] {
			return fmt.Errorf("Apply Git workspace IDs must be valid, flat and unique")
		}
		seen[id] = true
	}
	w.reserved = make(map[run.WorkspaceID]bool)
	w.gitPaths = make(map[string]string)
	spec := w.helper
	spec.Name += "-git"
	spec.Labels = cloneWorkspaceLabels(spec.Labels)
	spec.Workdir = "/tmp"
	for _, id := range slices.Sorted(slices.Values(ids)) {
		if err := w.createOwnedWorkspace(ctx, id); err != nil {
			return fmt.Errorf("reserve Git workspace %s (%s): %w", id, w.volume(id), err)
		}
		w.reserved[id] = false
		volume, target := w.volume(id), "/assignments/"+string(id)
		w.gitPaths[volume] = target
		spec.Mounts = append(spec.Mounts, container.Mount{Volume: volume, Target: target, NoCopy: true})
	}
	session, err := runtime.StartOwnedExec(ctx, spec, []string{"/usr/local/bin/node", "-e", "setInterval(() => {}, 2147483647)"})
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("Apply Git container was not established")
	}
	decoded, identityErr := hex.DecodeString(session.ID())
	if identityErr != nil || len(decoded) != 32 {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		return errors.Join(fmt.Errorf("Apply Git container identity is not a full ID"), session.Close(cleanup))
	}
	w.git = session
	return nil
}

// CloseGit ends only the Git container. Assignment volumes remain available
// for evidence and explicit later cleanup. Cleanup failure remains an error.
func (w *Workspaces) CloseGit(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.git == nil {
		return nil
	}
	w.gitClosed = true
	w.gitCloseErr = w.git.Close(ctx)
	return w.gitCloseErr
}

// GitContainerID retains the observed full ID after CloseGit.
func (w *Workspaces) GitContainerID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.git == nil {
		return ""
	}
	return w.git.ID()
}

// GitExecCount counts started engine CLI processes, not remote Git success.
func (w *Workspaces) GitExecCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.git == nil {
		return 0
	}
	return w.git.GitExecCount()
}

func (w *Workspaces) createOwnedWorkspace(ctx context.Context, id run.WorkspaceID) error {
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return err
	}
	token := hex.EncodeToString(secret[:])
	labels := cloneWorkspaceLabels(w.helper.Labels)
	labels[workspaceOwnerLabel] = token
	volume := w.volume(id)
	observed, err := w.runtime.CreateVolumeExclusive(ctx, volume, labels)
	if observed.Exists && observed.Name == volume && observed.Labels[workspaceOwnerLabel] == token {
		w.owned[id] = token
	}
	if err != nil {
		return err
	}
	if w.owned[id] != token {
		return fmt.Errorf("runtime did not establish ownership of new workspace volume")
	}
	return nil
}

// runGit translates only absolute path arguments belonging to the supplied
// workspace mounts. Messages, revisions and relative pathspecs remain literal.
// The service's mounts and path map are immutable after StartGit.
func (w *Workspaces) runGit(ctx context.Context, spec container.ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	if w.git == nil || len(argv) == 0 || argv[0] != "/usr/bin/git" {
		return fmt.Errorf("assignment Git needs the Apply Git container started first")
	}
	translated := slices.Clone(argv)
	for _, mount := range spec.Mounts {
		if w.gitPaths[mount.Volume] == "" {
			return fmt.Errorf("Git workspace volume was not reserved for this Apply")
		}
	}
	for i, arg := range translated[1:] {
		if argv[i] == "-m" || argv[i] == "--message" {
			continue
		}
		for _, mount := range spec.Mounts {
			target := w.gitPaths[mount.Volume]
			if arg == mount.Target || strings.HasPrefix(arg, mount.Target+"/") {
				translated[i+1] = target + strings.TrimPrefix(arg, mount.Target)
				break
			}
		}
	}
	return w.git.Exec(ctx, translated, stdin, stdout)
}
