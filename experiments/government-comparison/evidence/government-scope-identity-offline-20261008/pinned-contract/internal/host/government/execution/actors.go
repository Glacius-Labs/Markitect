package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

// actorSession bounds actual fresh invocations across all branches and repairs.
// Its lock protects bookkeeping only; no process runs while holding the lock.
type actorSession struct {
	ctx               context.Context
	runtime           Runtime
	report            *Report
	runDir, temporary string
	constitution      string
	frozen            map[string]runnerFingerprint
	mu                sync.Mutex
	calls             int
	semaphore         chan struct{}
	control           Control
}

func (s *actorSession) invoke(phase string, spec RunnerSpec, scopes []string, candidate *snapshot.Snapshot, workspace string, inputPaths []string, contextValue map[string]any) (result agentexec.RunResult, runErr error) {
	if s.semaphore != nil {
		select {
		case s.semaphore <- struct{}{}:
			defer func() { <-s.semaphore }()
		case <-s.ctx.Done():
			return result, s.ctx.Err()
		}
	}
	s.mu.Lock()
	if s.runtime.Recursion != nil && s.calls >= s.runtime.Recursion.Limits.MaxCalls {
		s.mu.Unlock()
		return result, errors.New("global actor invocation limit exhausted")
	}
	s.calls++
	index := s.calls
	s.mu.Unlock()
	pin, err := fingerprintRunner(spec)
	if err != nil || pin != s.frozen[spec.SlotID] {
		return result, errors.Join(errors.New("runner differs from frozen runtime before invocation"), err)
	}
	contextBytes, err := json.Marshal(contextValue)
	if err != nil {
		return result, err
	}
	role := agentexec.RoleVerifier
	if phase == "execute" {
		role = agentexec.RoleExecutor
	}
	request := agentexec.Request{Role: role, SourceRevision: s.runtime.ExpectedBase, ModelDigest: s.report.PriorConstitution, ModulePin: s.report.ToolPins, ProjectionID: "government/" + phase + "/" + spec.SlotID, ScopeIDs: scopes, PolicyIDs: []string{s.constitution}, Context: contextBytes, Artifacts: scopedArtifacts(candidate, inputPaths)}
	if s.control != nil {
		if err := s.control.ReserveActor(s.report.RunID, index, phase, spec); err != nil {
			return result, err
		}
		if err := s.control.Fence(); err != nil {
			return result, err
		}
	}
	started := time.Now().UTC()
	result, runErr = Invoke(s.ctx, spec, request, workspace, s.runDir, s.temporary)
	if runErr == nil && (result.Receipt.ConfigDigest != pin.ConfigDigest || result.Receipt.ExecutableDigest != pin.ExecutableDigest) {
		runErr = errors.New("actual runner receipt differs from frozen runtime")
	}
	record := ActorRecord{Phase: phase, SlotID: spec.SlotID, Scopes: scopes, Result: result, StartedAt: started, FinishedAt: time.Now().UTC(), Sequence: index}
	if runErr != nil {
		record.Error = runErr.Error()
	}
	s.mu.Lock()
	s.report.Actors = append(s.report.Actors, record)
	s.mu.Unlock()
	persistErr := persistJSON(filepath.Join(s.runDir, fmt.Sprintf("actor-%04d.json", index)), record)
	if s.control != nil {
		persistErr = errors.Join(persistErr, s.control.CompleteActor(s.report.RunID, record))
	}
	return result, errors.Join(runErr, persistErr)
}

func scopedArtifacts(candidate *snapshot.Snapshot, paths []string) []agentexec.Artifact {
	out := []agentexec.Artifact{}
	selected := map[string]bool{}
	for _, path := range paths {
		selected[path] = true
	}
	for _, path := range sortedPaths(candidate.Files) {
		if !selected[path] {
			continue
		}
		mode := "0644"
		if candidate.Modes[path] == snapshot.ExecutableMode {
			mode = "0755"
		}
		out = append(out, agentexec.Artifact{Path: path, Mode: mode, Digest: government.BytesDigest(candidate.Files[path]), Content: candidate.Files[path]})
	}
	return out
}
