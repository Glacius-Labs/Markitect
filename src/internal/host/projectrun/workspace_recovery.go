package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const maxWorkspaceJournalBytes = 32 << 20

type recoverableWorkspaceService interface {
	candidateWorkspaceService
	ReopenCandidate(context.Context, projectworkspace.Request, projectworkspace.Handle, []projectworkspace.Change, string, bool) error
}

type nativeTurnRecoverer interface {
	Recover(context.Context, agentexec.Config, codexappserver.RecoveryHandle, agentexec.RunOptions) (agentexec.RunResult, error)
}

type safeRecoveryCause struct {
	message string
	cause   error
}

func (e safeRecoveryCause) Error() string { return e.message }
func (e safeRecoveryCause) Unwrap() error { return e.cause }

func safeRecoveryError(err error, message string) error {
	if err == nil {
		return nil
	}
	return safeRecoveryCause{message: message, cause: err}
}

// RecoverProjectAgent resumes inspection of the exact saved native turn for a
// task. It never starts a role or replays a turn. found is true once an exact
// task journal exists, including when its workspace must be preserved on error.
func RecoverProjectAgent(ctx context.Context, host Host, invoker Invoker, root, taskID string, config agentexec.Config, limits Limits, expected nativeRecoveryBinding, expectedRequest agentexec.Request) (result agentexec.RunResult, found bool, err error) {
	if ctx == nil || strings.TrimSpace(taskID) == "" || config.Transport != TransportCodexAppServer {
		return result, false, errors.New("native recovery requires a context, task ID, and codex-app-server transport")
	}
	if expected.OwnerRunID == "" || expected.InputDigest == "" {
		return result, false, errors.New("native recovery requires the current owner run ID, input digest, and request")
	}
	expectedInvocation, _, err := agentexec.PrepareInvocation(expectedRequest)
	if err != nil {
		return result, false, errors.New("native recovery request cannot be prepared")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return result, false, errors.New("could not resolve selected repository root")
	}
	root = filepath.Clean(root)
	privateDir := filepath.Join(root, ".markitect", "runs", "private")
	journals, err := readTaskWorkspaceJournals(privateDir, taskID)
	if err != nil {
		return result, false, err
	}
	if len(journals) == 0 {
		return result, false, nil
	}
	found = true
	if expectedInvocation.InputDigest != expected.InputDigest {
		return result, true, errors.New("native recovery request does not match the durable original input digest")
	}
	if invoker == nil {
		return result, true, errors.New("native recovery invoker is unavailable")
	}
	fingerprint, fpErr := fingerprintForRequest(invoker, config, expectedRequest)
	if fpErr != nil {
		return result, true, errors.New("native recovery configuration cannot be verified")
	}
	type exactJournal struct {
		candidate taskWorkspaceJournal
		handle    codexappserver.RecoveryHandle
	}
	var exact []exactJournal
	for _, candidate := range journals {
		journal := candidate.journal
		if journal.Request.TaskID != taskID {
			continue
		}
		ownerMatch := journal.OwnerRunID != "" && journal.OwnerRunID == expected.OwnerRunID
		if expected.RunID == "" {
			if !ownerMatch {
				continue
			}
			if journal.Receipt.InputDigest != "" && journal.Receipt.InputDigest != expected.InputDigest {
				return result, true, errors.New("current owner workspace receipt conflicts with the original input digest")
			}
		} else {
			if journal.Receipt.RunID != "" && journal.Receipt.RunID != expected.RunID {
				if ownerMatch {
					return result, true, errors.New("current owner workspace receipt conflicts with the durable original run ID")
				}
				continue
			}
			if journal.Receipt.InputDigest != "" && journal.Receipt.InputDigest != expected.InputDigest {
				if ownerMatch || journal.Receipt.RunID == expected.RunID {
					return result, true, errors.New("workspace receipt conflicts with the durable original input digest")
				}
				continue
			}
			if journal.Receipt.RunID != expected.RunID && !ownerMatch && journal.Receipt.RunID != "" {
				continue
			}
		}
		closedCache := journal.State == "closed" && journal.CachedResult != nil
		handles, handleErr := readTrustedRecoveryHandlesForCache(privateDir, journal.Handle.ID, closedCache)
		if handleErr != nil {
			return result, true, errors.New("private native recovery handles are unavailable or invalid")
		}
		matchingHandles := make([]codexappserver.RecoveryHandle, 0, len(handles))
		for _, handle := range handles {
			if handle.Invocation.InputDigest != expected.InputDigest || !requestMatch(expectedRequest, handle.Invocation) {
				continue
			}
			if expected.RunID != "" && handle.Invocation.RunID != expected.RunID {
				continue
			}
			matchingHandles = append(matchingHandles, handle)
		}
		if len(matchingHandles) == 0 {
			if ownerMatch || journal.Receipt.RunID == expected.RunID && expected.RunID != "" {
				return result, true, errors.New("current owner workspace has no trusted handle for the exact original request")
			}
			continue
		}
		selected, selectErr := selectRecoveryHandle(matchingHandles, journal.Handle, fingerprint)
		if selectErr != nil {
			return result, true, selectErr
		}
		if selected.Invocation.InputDigest != expected.InputDigest || expected.RunID != "" && selected.Invocation.RunID != expected.RunID {
			return result, true, errors.New("trusted native handle does not match the durable original run ID and input digest")
		}
		if journal.Receipt.RunID != "" && (journal.Receipt.RunID != selected.Invocation.RunID || journal.Receipt.InputDigest != selected.Invocation.InputDigest) {
			return result, true, errors.New("private receipt does not match the original native invocation")
		}
		if expected.RunID == "" && (!ownerMatch || !selected.TurnDispatched || selected.ThreadID == "" || selected.SessionID == "" || selected.TurnID == "") {
			return result, true, errors.New("current owner workspace has no trusted dispatched original turn")
		}
		exact = append(exact, exactJournal{candidate: candidate, handle: selected})
	}
	if len(exact) == 0 {
		if expected.RunID != "" {
			return result, true, errors.New("no private workspace journal matches the durable original run ID and input digest")
		}
		return result, true, errors.New("no current owner workspace journal matches the exact dispatched request")
	}
	if len(exact) != 1 {
		return result, true, errors.New("multiple private workspace journals match the durable owner, run ID, and input digest")
	}
	journal, journalPath := exact[0].candidate.journal, exact[0].candidate.path
	selected := exact[0].handle
	if !sameRecoveryPath(journal.Request.RepositoryRoot, root) || journal.Request.TaskID != taskID || journal.Handle.TaskID != taskID || journal.Handle.ID == "" {
		return result, true, errors.New("private workspace journal does not match the selected task and repository")
	}
	if err := journal.Handle.ValidateFor(journal.Request); err != nil {
		return result, true, errors.New("private workspace ownership binding is invalid")
	}
	overlayDigest, digestErr := projectworkspace.CandidateOverlayDigest(journal.Overlay)
	if digestErr != nil || overlayDigest != journal.OverlayDigest {
		return result, true, errors.New("private candidate overlay binding is invalid")
	}
	identity, identityErr := source.IdentifyGit(root)
	if identityErr != nil || identity.Digest != journal.Request.RepositoryIdentity {
		return result, true, errors.New("selected repository identity changed since the native invocation")
	}
	binding, bindingErr := projectworkspace.InspectRepository(ctx, root, journal.Request.BaseSHA)
	if bindingErr != nil || binding.OverlayDigest != journal.Request.OverlayDigest {
		return result, true, errors.New("selected source snapshot changed since the native invocation")
	}
	closedCache := journal.State == "closed" && journal.CachedResult != nil
	harvestedCache := journal.State == "harvested" && journal.CachedResult != nil
	if closedCache && !validCachedWorkspacePath(root, privateDir, journal.Handle) {
		return result, true, errors.New("cached workspace handle path does not match the original private Git workspace")
	}
	if journal.State == "prepared" && (!selected.TurnDispatched || selected.ThreadID == "" || selected.SessionID == "" || selected.TurnID == "") {
		return result, true, errors.New("prepared workspace has no trusted dispatched original turn to recover")
	}
	if expected.RunID != "" && selected.Invocation.RunID != expected.RunID || selected.Invocation.InputDigest != expected.InputDigest {
		return result, true, errors.New("trusted native handle does not match the durable original run ID and input digest")
	}
	if !requestMatch(expectedRequest, selected.Invocation) {
		return result, true, errors.New("native recovery request does not match the original invocation")
	}
	if journal.Receipt.RunID != "" && journal.Receipt.RunID != selected.Invocation.RunID {
		return result, true, errors.New("private receipt does not match the original native invocation")
	}
	if closedCache {
		cached, cacheErr := validateCachedWorkspaceResult(journal, selected, fingerprint, limits)
		if cacheErr != nil {
			return result, true, cacheErr
		}
		return cached, true, nil
	}
	if harvestedCache {
		cached, cacheErr := validateCachedWorkspaceResult(journal, selected, fingerprint, limits)
		if cacheErr != nil {
			return result, true, cacheErr
		}
		if !validCachedWorkspacePath(root, privateDir, journal.Handle) || !workspaceInvocationTerminal(cached.Receipt.Lifecycle) {
			return result, true, errors.New("harvested cache has no confirmed terminal owned workspace to close")
		}
		service, ok := host.Workspaces.(recoverableWorkspaceService)
		if !ok {
			return result, true, errors.New("Host Git candidate workspace recovery service is unavailable")
		}
		cwd, cwdErr := filepath.EvalSymlinks(journal.Handle.CWD)
		if cwdErr != nil || !sameRecoveryPath(cwd, journal.Handle.CWD) {
			return result, true, errors.New("harvested cached workspace is missing; cleanup state remains unknown")
		}
		if err := service.ReopenCandidate(ctx, journal.Request, journal.Handle, journal.Overlay, journal.OverlayDigest, true); err != nil {
			return result, true, safeRecoveryError(err, "harvested workspace ownership could not be revalidated")
		}
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		closeErr := service.Close(closeCtx, journal.Handle)
		closeCancel()
		if closeErr != nil {
			return result, true, safeRecoveryError(closeErr, "harvested workspace cleanup retry failed")
		}
		journal.State = "closed"
		if err := persistWorkspaceJournal(journalPath, journal); err != nil {
			return result, true, safeRecoveryError(err, "closed workspace state could not be persisted")
		}
		return cached, true, nil
	}
	cwd, cwdErr := filepath.EvalSymlinks(journal.Handle.CWD)
	if cwdErr != nil || !sameRecoveryPath(cwd, journal.Handle.CWD) {
		return result, true, errors.New("owned native workspace is unavailable for exact-turn recovery")
	}
	service, ok := host.Workspaces.(recoverableWorkspaceService)
	if !ok {
		return result, true, errors.New("Host Git candidate workspace recovery service is unavailable")
	}
	recoverer, ok := invoker.(nativeTurnRecoverer)
	if !ok {
		return result, true, errors.New("configured native transport does not support exact-turn recovery")
	}
	if limits.MaxCandidateFileBytes <= 0 || limits.MaxCandidateBytes < limits.MaxCandidateFileBytes {
		return result, true, errors.New("candidate recovery limits are invalid")
	}
	options := agentexec.RunOptions{Workspace: &journal.Handle, PrivateLogDirectory: privateDir}
	result, recoveryErr := recoverer.Recover(ctx, config, selected, options)
	if !workspaceInvocationTerminal(result.Receipt.Lifecycle) {
		journal.State = "preserved"
		journal.Receipt = result.Receipt
		persistErr := persistWorkspaceJournal(journalPath, journal)
		return result, true, errors.Join(errors.New("native execution or one of its children is not observed terminal; workspace retained"), safeRecoveryError(recoveryErr, "native turn inspection returned an error"), safeRecoveryError(persistErr, "private recovery receipt could not be persisted"))
	}
	// This trusted application edge restores P03's state only after the exact
	// original root and all tracked child starts are observed terminal.
	if err := service.ReopenCandidate(ctx, journal.Request, journal.Handle, journal.Overlay, journal.OverlayDigest, true); err != nil {
		journal.State = "preserved"
		journal.Receipt = result.Receipt
		persistErr := persistWorkspaceJournal(journalPath, journal)
		return result, true, errors.Join(errors.New("owned candidate could not be safely reopened"), safeRecoveryError(recoveryErr, "native turn inspection returned an error"), safeRecoveryError(err, "candidate ownership validation failed"), safeRecoveryError(persistErr, "private recovery receipt could not be persisted"))
	}
	journal.Receipt = result.Receipt
	journal.State = "preserved"
	if result.Receipt.RunID != selected.Invocation.RunID || result.Receipt.InputDigest != selected.Invocation.InputDigest || result.Receipt.ConfigDigest != fingerprint {
		recoveryErr = errors.Join(recoveryErr, errors.New("recovered receipt does not match the original invocation"))
	}
	if recoveryErr == nil {
		wire, marshalErr := json.Marshal(result.Response)
		if marshalErr != nil {
			recoveryErr = marshalErr
		} else if response, decodeErr := agentexec.DecodeResponse(wire, selected.Invocation, ""); decodeErr != nil {
			recoveryErr = decodeErr
		} else {
			result.Response = response
		}
	}
	harvestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	delta, harvestErr := service.Harvest(harvestCtx, journal.Handle)
	cancel()
	if harvestErr == nil {
		var normalized projectworkspace.Delta
		normalized, harvestErr = projectworkspace.NormalizeDelta(journal.Request, journal.Handle, delta.Changes, projectworkspace.Limits{
			MaxFiles: 10000, MaxFileBytes: int(limits.MaxCandidateFileBytes), MaxTotalBytes: int(limits.MaxCandidateBytes),
		})
		if harvestErr == nil && (normalized.Digest != delta.Digest || delta.RepositoryIdentity != journal.Request.RepositoryIdentity || delta.BaseSHA != journal.Request.BaseSHA || delta.OverlayDigest != journal.Request.OverlayDigest || delta.TaskID != taskID || delta.BaseDigest != journal.Handle.BaseDigest) {
			harvestErr = errors.New("recovered workspace delta binding or digest differs")
		}
		if harvestErr == nil {
			result.Delta = &normalized
			journal.Delta = &normalized
		}
	}
	if harvestErr != nil {
		journal.Receipt = result.Receipt
		persistErr := persistWorkspaceJournal(journalPath, journal)
		return result, true, errors.Join(errors.New("recovered workspace changes could not be validated"), safeRecoveryError(recoveryErr, "native turn inspection returned an error"), safeRecoveryError(harvestErr, "workspace delta validation failed"), safeRecoveryError(persistErr, "private recovery receipt could not be persisted"))
	}
	if recoveryErr == nil {
		journal.State = "harvested"
		cached := result
		journal.CachedResult = &cached
	}
	if err := persistWorkspaceJournal(journalPath, journal); err != nil {
		return result, true, errors.Join(errors.New("could not persist recovered native result"), safeRecoveryError(recoveryErr, "native turn inspection returned an error"), safeRecoveryError(err, "private recovery result could not be persisted"))
	}
	if recoveryErr != nil {
		return result, true, safeRecoveryError(recoveryErr, "native turn inspection did not produce a successful result")
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	closeErr := service.Close(closeCtx, journal.Handle)
	closeCancel()
	if closeErr != nil {
		return result, true, errors.Join(errors.New("recovered workspace remains open"), safeRecoveryError(closeErr, "workspace cleanup failed"))
	}
	journal.State = "closed"
	if err := persistWorkspaceJournal(journalPath, journal); err != nil {
		return result, true, errors.Join(errors.New("recovered result is cached but close state could not be recorded"), safeRecoveryError(err, "private close state could not be persisted"))
	}
	return result, true, nil
}

func validCachedWorkspacePath(repositoryRoot, privateDir string, handle projectworkspace.Handle) bool {
	cwd, err := filepath.Abs(handle.CWD)
	if err != nil || !sameRecoveryPath(cwd, handle.CWD) || filepath.Base(cwd) != "repo" ||
		!strings.HasPrefix(filepath.Base(filepath.Dir(cwd)), "workspace-"+handle.ID+"-") {
		return false
	}
	return !pathIsWithin(repositoryRoot, cwd) && !pathIsWithin(privateDir, cwd)
}

func sameRecoveryPath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}

type taskWorkspaceJournal struct {
	journal workspaceJournal
	path    string
}

func readTaskWorkspaceJournals(privateDir, taskID string) ([]taskWorkspaceJournal, error) {
	for _, path := range []string{filepath.Dir(filepath.Dir(privateDir)), filepath.Dir(privateDir), privateDir} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("private workspace journal parent is unavailable")
		}
	}
	workspaces := filepath.Join(privateDir, "workspaces")
	info, err := os.Lstat(workspaces)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("private workspace journal directory is unavailable")
	}
	entries, err := os.ReadDir(workspaces)
	if err != nil {
		return nil, errors.New("private workspace journals cannot be listed")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var matching []taskWorkspaceJournal
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(workspaces, entry.Name())
		wire, readErr := readPrivateRegular(path, maxWorkspaceJournalBytes)
		if readErr != nil {
			return nil, errors.New("private workspace journal is invalid")
		}
		var journal workspaceJournal
		decoder := json.NewDecoder(bytes.NewReader(wire))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&journal); decodeErr != nil {
			return nil, errors.New("private workspace journal cannot be decoded")
		}
		var trailing any
		if decodeErr := decoder.Decode(&trailing); !errors.Is(decodeErr, io.EOF) {
			return nil, errors.New("private workspace journal has trailing data")
		}
		if journal.Request.TaskID == taskID {
			if entry.Name() != journal.Handle.ID+".json" {
				return nil, errors.New("private workspace journal filename binding is invalid")
			}
			matching = append(matching, taskWorkspaceJournal{journal: journal, path: path})
		}
	}
	return matching, nil
}

func selectRecoveryHandle(handles []codexappserver.RecoveryHandle, workspace projectworkspace.Handle, fingerprint string) (codexappserver.RecoveryHandle, error) {
	var candidates []codexappserver.RecoveryHandle
	turns := map[string]bool{}
	for _, handle := range handles {
		if handle.Workspace != workspace || handle.Fingerprint != fingerprint || handle.Invocation.APIVersion != agentexec.APIVersion || handle.Invocation.RunID == "" || handle.Invocation.Nonce == "" || !requestMatch(handle.Invocation.Request, handle.Invocation) {
			continue
		}
		if handle.Invocation.Request.SourceRevision != workspace.BaseSHA {
			continue
		}
		if handle.ThreadID != "" && handle.SessionID != "" && handle.TurnDispatched && handle.TurnID != "" {
			turns[handle.ThreadID+"\x00"+handle.SessionID+"\x00"+handle.TurnID] = true
		}
		candidates = append(candidates, handle)
	}
	if len(candidates) == 0 {
		return codexappserver.RecoveryHandle{}, errors.New("no trusted recovery handle matches the original task, source, and configuration")
	}
	if len(turns) > 1 {
		return codexappserver.RecoveryHandle{}, errors.New("conflicting dispatched turn identities exist for the original task")
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return recoveryHandleProgress(candidates[i]) > recoveryHandleProgress(candidates[j])
	})
	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Invocation.RunID != selected.Invocation.RunID || candidate.Invocation.Nonce != selected.Invocation.Nonce || candidate.ThreadID != selected.ThreadID || candidate.SessionID != selected.SessionID {
			return codexappserver.RecoveryHandle{}, errors.New("multiple original native invocations match this task; recovery is ambiguous")
		}
		if candidate.TurnDispatched && candidate.TurnID != "" && selected.TurnDispatched && selected.TurnID != "" && candidate.TurnID != selected.TurnID {
			return codexappserver.RecoveryHandle{}, errors.New("conflicting dispatched turn identities exist for the original task")
		}
		if recoveryHandleProgress(candidate) > recoveryHandleProgress(selected) {
			selected = candidate
		}
	}
	return selected, nil
}

func recoveryHandleProgress(h codexappserver.RecoveryHandle) int {
	if h.TurnDispatched && h.TurnID != "" {
		return 3
	}
	if h.TurnDispatched {
		return 2
	}
	if h.ThreadID != "" && h.SessionID != "" {
		return 1
	}
	return 0
}

func requestMatch(request agentexec.Request, invocation agentexec.Invocation) bool {
	expected, _, err := agentexec.PrepareInvocation(request)
	return err == nil && expected.InputDigest == invocation.InputDigest && reflect.DeepEqual(expected.Request, invocation.Request)
}

func validateCachedWorkspaceResult(journal workspaceJournal, handle codexappserver.RecoveryHandle, fingerprint string, limits Limits) (agentexec.RunResult, error) {
	if journal.CachedResult == nil || journal.State != "closed" && journal.State != "harvested" {
		return agentexec.RunResult{}, errors.New("closed native workspace has no trusted cached result")
	}
	result := *journal.CachedResult
	if result.Receipt.RunID != handle.Invocation.RunID || result.Receipt.InputDigest != handle.Invocation.InputDigest || result.Receipt.ConfigDigest != fingerprint || result.Delta == nil || journal.Delta == nil || result.Delta.Digest != journal.Delta.Digest {
		return agentexec.RunResult{}, errors.New("cached native result does not match its original invocation")
	}
	wire, err := json.Marshal(result.Response)
	if err != nil {
		return agentexec.RunResult{}, errors.New("cached native response cannot be encoded")
	}
	response, err := agentexec.DecodeResponse(wire, handle.Invocation, "")
	if err != nil {
		return agentexec.RunResult{}, errors.New("cached native response binding is invalid")
	}
	delta, err := projectworkspace.NormalizeDelta(journal.Request, journal.Handle, result.Delta.Changes, projectworkspace.Limits{
		MaxFiles: 10000, MaxFileBytes: int(limits.MaxCandidateFileBytes), MaxTotalBytes: int(limits.MaxCandidateBytes),
	})
	if err != nil || delta.Digest != result.Delta.Digest || delta.Digest != journal.Delta.Digest {
		return agentexec.RunResult{}, errors.New("cached native workspace delta binding is invalid")
	}
	result.Response = response
	result.Delta = &delta
	return result, nil
}
