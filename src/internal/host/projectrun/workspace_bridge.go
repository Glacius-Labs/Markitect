package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

type candidateWorkspaceService interface {
	projectworkspace.Service
	PrepareCandidate(context.Context, projectworkspace.Request, []projectworkspace.Change, string) (projectworkspace.Handle, error)
}

type workspaceJournal struct {
	OwnerRunID    string                    `json:"ownerRunId,omitempty"`
	Request       projectworkspace.Request  `json:"request"`
	Handle        projectworkspace.Handle   `json:"handle"`
	Overlay       []projectworkspace.Change `json:"overlay"`
	OverlayDigest string                    `json:"overlayDigest"`
	State         string                    `json:"state"`
	Receipt       agentexec.Receipt         `json:"receipt"`
	Delta         *projectworkspace.Delta   `json:"delta,omitempty"`
	CachedResult  *agentexec.RunResult      `json:"cachedResult,omitempty"`
}

// invokeProjectAgent owns workspace preparation and observation. A model cannot
// supply a delta. Unknown transport/child termination preserves the private
// workspace and journal for recovery; it never permits a blind cleanup/replay.
func invokeProjectAgent(ctx context.Context, host Host, invoker Invoker, root string, project *Project, agent Agent, ownerRunID, taskID string, allowed, excluded []string, limits Limits, cfg agentexec.Config, req agentexec.Request) (agentexec.RunResult, error) {
	opts := agentexec.RunOptions{PrivateLogDirectory: filepath.Join(root, ".markitect", "runs", "private")}
	if agent.Transport != TransportCodexAppServer {
		return invokeAgent(ctx, invoker, cfg, req, opts)
	}
	if strings.TrimSpace(ownerRunID) == "" {
		return agentexec.RunResult{}, errors.New("native invocation requires its owning run identity")
	}
	absRoot, rootErr := filepath.Abs(root)
	if rootErr != nil {
		return agentexec.RunResult{}, rootErr
	}
	root = filepath.Clean(absRoot)
	opts.PrivateLogDirectory = filepath.Join(root, ".markitect", "runs", "private")
	if agent.WorkspaceMode != "git" || project == nil || project.Snapshot == nil || host.Load == nil {
		return agentexec.RunResult{}, errors.New("native App Server invocation requires an explicit owned Git workspace")
	}
	if _, err := buildNativeWorkspace(root, project.Revision, project, agent); err != nil {
		return agentexec.RunResult{}, err
	}
	service, ok := host.Workspaces.(candidateWorkspaceService)
	if !ok {
		return agentexec.RunResult{}, errors.New("Host Git candidate workspace service is unavailable")
	}
	fixed, err := host.Load(root, project.Revision)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	binding, err := projectworkspace.InspectRepository(ctx, root, project.Revision)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	overlay, err := workspaceOverlay(fixed.Snapshot, project.Snapshot)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(overlay)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	wr := projectworkspace.Request{RepositoryRoot: root, RepositoryIdentity: identity.Digest, BaseSHA: project.Revision, OverlayDigest: binding.OverlayDigest, TaskID: taskID, AllowedPaths: allowed, ExcludedPaths: excluded}
	handle, err := service.PrepareCandidate(ctx, wr, overlay, overlayDigest)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	journal := workspaceJournal{OwnerRunID: ownerRunID, Request: wr, Handle: handle, Overlay: overlay, OverlayDigest: overlayDigest, State: "prepared"}
	journalPath := filepath.Join(opts.PrivateLogDirectory, "workspaces", handle.ID+".json")
	if err := persistWorkspaceJournal(journalPath, journal); err != nil {
		return agentexec.RunResult{}, errors.Join(err, service.Close(context.Background(), handle))
	}
	opts.Workspace = &handle
	result, invokeErr := invoker.Run(ctx, cfg, req, opts)
	journal.Receipt = result.Receipt
	journal.State = "preserved"
	// A nil/error result is never evidence of zero live provider work. The
	// observed terminal predicate is cooperative evidence, not OS isolation or
	// proof that generic helper accounting is complete.
	terminal := workspaceInvocationTerminal(result.Receipt.Lifecycle)
	if terminal {
		harvestCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		delta, harvestErr := service.Harvest(harvestCtx, handle)
		cancel()
		if harvestErr == nil {
			normalized, normalizeErr := projectworkspace.NormalizeDelta(wr, handle, delta.Changes, projectworkspace.Limits{MaxFiles: 10000, MaxFileBytes: int(limits.MaxCandidateFileBytes), MaxTotalBytes: int(limits.MaxCandidateBytes)})
			if normalizeErr != nil || normalized.Digest != delta.Digest || delta.RepositoryIdentity != wr.RepositoryIdentity || delta.BaseSHA != wr.BaseSHA || delta.OverlayDigest != wr.OverlayDigest || delta.TaskID != wr.TaskID || delta.BaseDigest != handle.BaseDigest {
				harvestErr = errors.Join(normalizeErr, errors.New("harvested delta binding or digest differs"))
			} else {
				result.Delta = &normalized
				journal.Delta = &normalized
				journal.State = "harvested"
				if invokeErr == nil {
					cached := result
					journal.CachedResult = &cached
				}
			}
		}
		invokeErr = errors.Join(invokeErr, harvestErr)
	} else {
		invokeErr = errors.Join(invokeErr, errors.New("native invocation termination is unknown; workspace preserved for recovery"))
	}
	if err := persistWorkspaceJournal(journalPath, journal); err != nil {
		return result, errors.Join(invokeErr, err)
	}
	if terminal && journal.State == "harvested" {
		closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		closeErr := service.Close(closeCtx, handle)
		cancel()
		if closeErr == nil {
			journal.State = "closed"
			closeErr = persistWorkspaceJournal(journalPath, journal)
		}
		invokeErr = errors.Join(invokeErr, closeErr)
	}
	return result, invokeErr
}

func workspaceInvocationTerminal(life *agentexec.Lifecycle) bool {
	if life == nil || life.Provider != TransportCodexAppServer || life.SessionID == "" || life.TurnID == "" || !terminalWorkspaceState(life.State) || len(life.StartRequests) == 0 {
		return false
	}
	for _, start := range life.StartRequests {
		if !terminalWorkspaceState(start.State) {
			return false
		}
	}
	return true
}

func terminalWorkspaceState(state string) bool {
	return state == "completed" || state == "failed" || state == "interrupted"
}

func persistWorkspaceJournal(path string, journal workspaceJournal) error {
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".journal-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func workspaceOverlay(before, after *Snapshot) ([]projectworkspace.Change, error) {
	if before == nil || after == nil {
		return nil, fmt.Errorf("workspace overlay requires source and candidate snapshots")
	}
	paths := map[string]bool{}
	for path := range before.Files {
		paths[path] = true
	}
	for path := range after.Files {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	changes := []projectworkspace.Change{}
	for _, path := range ordered {
		old, had := before.Files[path]
		data, has := after.Files[path]
		if had && has && bytes.Equal(old, data) && before.Modes[path] == after.Modes[path] {
			continue
		}
		change := projectworkspace.Change{Path: path, Kind: projectworkspace.ChangeDelete}
		if has {
			change.Kind = projectworkspace.ChangeAdd
			if had {
				change.Kind = projectworkspace.ChangeModify
			}
			change.Mode = after.Modes[path]
			change.Content = append([]byte{}, data...)
		}
		changes = append(changes, change)
	}
	return changes, nil
}
