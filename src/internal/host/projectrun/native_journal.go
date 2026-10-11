package projectrun

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
)

const (
	nativeJournalDirectory = "codex-app-server"
	nativeJournalVersion   = 1
	nativeJournalMaxRecord = 32 << 20
	nativeJournalMaxEvents = 384 << 20
)

type nativeJournal struct {
	directory      string
	workspaceID    string
	workspaceCWD   string
	mu             sync.Mutex
	usedRequestIDs map[string]bool
	rootRequestID  string
	recoveryHandle *codexappserver.RecoveryHandle
	eventBytes     int64
	eventSequence  uint64
}

type nativeStartRecord struct {
	Version           int       `json:"version"`
	RequestID         string    `json:"requestId"`
	ProtocolRequestID string    `json:"protocolRequestId"`
	WorkspaceHandleID string    `json:"workspaceHandleId"`
	ParentSessionID   string    `json:"parentSessionId,omitempty"`
	Role              string    `json:"role"`
	Model             string    `json:"model,omitempty"`
	ReasoningEffort   string    `json:"reasoningEffort,omitempty"`
	State             string    `json:"state"`
	RecordedAt        time.Time `json:"recordedAt"`
}

type nativeEventRecord struct {
	Version           int       `json:"version"`
	Sequence          uint64    `json:"sequence"`
	WorkspaceHandleID string    `json:"workspaceHandleId"`
	Method            string    `json:"method"`
	WireBase64        string    `json:"wireBase64,omitempty"`
	ParamsBase64      string    `json:"paramsBase64,omitempty"`
	RecordedAt        time.Time `json:"recordedAt"`
}

type nativeRecoveryRecord struct {
	Version           int                           `json:"version"`
	RecordID          string                        `json:"recordId"`
	WorkspaceHandleID string                        `json:"workspaceHandleId"`
	RecordedAt        time.Time                     `json:"recordedAt"`
	Handle            codexappserver.RecoveryHandle `json:"handle"`
}

// newNativeJournal creates one private journal for an owned workspace handle.
// Its event and recovery files are placed under PrivateLogDirectory and never
// inside the provider's working directory.
func newNativeJournal(privateLogDirectory, workspaceCWD, workspaceID string) (*nativeJournal, error) {
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(privateLogDirectory) == "" || !filepath.IsAbs(workspaceCWD) {
		return nil, errors.New("native journal requires a bound workspace handle and absolute CWD")
	}
	privateAbs, err := filepath.Abs(privateLogDirectory)
	if err != nil {
		return nil, fmt.Errorf("resolve native private log directory: %w", err)
	}
	if _, err := agentexec.PreparePrivateLogDirectory(privateAbs); err != nil {
		return nil, fmt.Errorf("prepare native private log directory: %w", err)
	}
	privateReal, err := filepath.EvalSymlinks(privateAbs)
	if err != nil {
		return nil, fmt.Errorf("resolve native private log directory: %w", err)
	}
	cwdReal, err := filepath.EvalSymlinks(workspaceCWD)
	if err != nil {
		return nil, fmt.Errorf("resolve native workspace CWD: %w", err)
	}
	root := filepath.Join(privateReal, nativeJournalDirectory)
	if err := ensurePrivateDirectory(root); err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve native journal directory: %w", err)
	}
	if pathIsWithin(cwdReal, root) {
		return nil, errors.New("native journal must be outside the provider workspace CWD")
	}

	workspaceDir := filepath.Join(root, workspaceJournalKey(workspaceID))
	if err := ensurePrivateDirectory(workspaceDir); err != nil {
		return nil, err
	}
	var invocationDir string
	for attempt := 0; attempt < 8; attempt++ {
		id, randomErr := nativeRandomID()
		if randomErr != nil {
			return nil, randomErr
		}
		candidate := filepath.Join(workspaceDir, id)
		if err := os.Mkdir(candidate, 0700); err == nil {
			invocationDir = candidate
			break
		} else if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create native invocation journal: %w", err)
		}
	}
	if invocationDir == "" {
		return nil, errors.New("could not allocate a unique native invocation journal")
	}
	if err := os.Chmod(invocationDir, 0700); err != nil {
		return nil, fmt.Errorf("restrict native invocation journal: %w", err)
	}
	if pathIsWithin(cwdReal, invocationDir) {
		return nil, errors.New("native invocation journal must be outside the provider workspace CWD")
	}
	for _, name := range []string{"starts.jsonl", "events.jsonl"} {
		if err := createPrivateFile(filepath.Join(invocationDir, name)); err != nil {
			return nil, err
		}
	}
	return &nativeJournal{directory: invocationDir, workspaceID: workspaceID, workspaceCWD: cwdReal, usedRequestIDs: map[string]bool{}}, nil
}

// bindRecoveryHandle authorizes journal handle updates for the exact already
// dispatched turn. Recovery does not execute BeforeStart and must not create a
// second role-start record; this binding only lets an observed update to the
// same trusted invocation be persisted in the recovery journal.
func (j *nativeJournal) bindRecoveryHandle(handle codexappserver.RecoveryHandle) error {
	if j == nil || handle.Workspace.ID != j.workspaceID || !sameNativeJournalPath(handle.Workspace.CWD, j.workspaceCWD) ||
		handle.Invocation.RunID == "" || handle.Invocation.Nonce == "" || handle.Invocation.InputDigest == "" ||
		handle.ThreadID == "" || handle.SessionID == "" || handle.TurnID == "" || !handle.TurnDispatched ||
		!requestMatch(handle.Invocation.Request, handle.Invocation) {
		return errors.New("recovery journal requires the exact trusted dispatched invocation and workspace")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.rootRequestID != "" {
		if j.recoveryHandle != nil && sameRecoveryJournalBinding(*j.recoveryHandle, handle) {
			return nil
		}
		return errors.New("recovery journal is already bound to another root invocation")
	}
	if j.recoveryHandle != nil {
		return errors.New("recovery journal has an ambiguous root invocation binding")
	}
	bound := handle
	j.rootRequestID = handle.Invocation.RunID
	j.recoveryHandle = &bound
	return nil
}

func sameNativeJournalPath(left, right string) bool {
	left, leftErr := filepath.Abs(left)
	right, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if sameRecoveryPath(left, right) {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && leftInfo.IsDir() && rightInfo.IsDir() && os.SameFile(leftInfo, rightInfo)
}

func sameRecoveryJournalBinding(left, right codexappserver.RecoveryHandle) bool {
	return left.Workspace == right.Workspace && left.Invocation.APIVersion == right.Invocation.APIVersion &&
		left.Invocation.RunID == right.Invocation.RunID &&
		left.Invocation.Nonce == right.Invocation.Nonce && left.Invocation.InputDigest == right.Invocation.InputDigest &&
		left.Fingerprint == right.Fingerprint && left.Protocol == right.Protocol &&
		left.ThreadID == right.ThreadID && left.SessionID == right.SessionID && left.TurnID == right.TurnID &&
		left.TurnDispatched && right.TurnDispatched
}

func (j *nativeJournal) wrapOptions(original codexappserver.Options) codexappserver.Options {
	wrapped := original
	originalBeforeStart := original.BeforeStart
	wrapped.BeforeStart = func(ctx context.Context, request agentexec.RoleStartRequest) error {
		record, err := j.reserveStart(request)
		if err != nil {
			return err
		}
		if originalBeforeStart == nil {
			return nil
		}
		if err := originalBeforeStart(ctx, request); err != nil {
			journalErr := j.finishStart(record, "failed")
			return errors.Join(err, journalErr)
		}
		return nil
	}

	originalOnHandle := original.OnHandle
	wrapped.OnHandle = func(ctx context.Context, handle codexappserver.RecoveryHandle) error {
		if err := j.persistHandle(handle); err != nil {
			return err
		}
		if originalOnHandle != nil {
			return originalOnHandle(ctx, handle)
		}
		return nil
	}

	originalOnEvent := original.OnEvent
	wrapped.OnEvent = func(ctx context.Context, event codexappserver.Event) error {
		if err := j.appendEvent(event); err != nil {
			return err
		}
		if originalOnEvent != nil {
			return originalOnEvent(ctx, event)
		}
		return nil
	}
	return wrapped
}

func (j *nativeJournal) reserveStart(request agentexec.RoleStartRequest) (nativeStartRecord, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if request.RequestID == "" {
		return nativeStartRecord{}, errors.New("native root start request has no protocol request ID")
	}
	requestID, err := j.uniqueRequestIDLocked()
	if err != nil {
		return nativeStartRecord{}, err
	}
	record := nativeStartRecord{
		Version: nativeJournalVersion, RequestID: requestID, ProtocolRequestID: request.RequestID,
		WorkspaceHandleID: j.workspaceID, ParentSessionID: request.ParentSessionID,
		Role: request.Role, Model: request.Model, ReasoningEffort: request.ReasoningEffort,
		State: "requested", RecordedAt: time.Now().UTC()}
	if err := j.appendJSONLine("starts.jsonl", record); err != nil {
		return nativeStartRecord{}, err
	}
	if request.ParentSessionID == "" && j.rootRequestID == "" {
		j.rootRequestID = request.RequestID
	}
	return record, nil
}

func (j *nativeJournal) finishStart(record nativeStartRecord, state string) error {
	if state != "failed" {
		return errors.New("invalid native start journal state")
	}
	update := record
	update.State = state
	update.RecordedAt = time.Now().UTC()
	return j.appendJSONLine("starts.jsonl", update)
}

func (j *nativeJournal) persistHandle(handle codexappserver.RecoveryHandle) error {
	if handle.Workspace.ID != j.workspaceID || handle.Invocation.RunID == "" {
		return errors.New("App Server recovery handle is not bound to the owned workspace")
	}
	j.mu.Lock()
	rootRequestID := j.rootRequestID
	recoveryHandle := j.recoveryHandle
	j.mu.Unlock()
	if rootRequestID == "" || handle.Invocation.RunID != rootRequestID {
		return errors.New("App Server recovery handle does not match the reserved root request")
	}
	if recoveryHandle != nil {
		if !sameNativeJournalPath(handle.Workspace.CWD, j.workspaceCWD) || handle.Invocation.APIVersion != agentexec.APIVersion ||
			!requestMatch(handle.Invocation.Request, handle.Invocation) || !sameRecoveryJournalBinding(*recoveryHandle, handle) {
			return errors.New("App Server recovery handle differs from the exact bound turn")
		}
	}
	recordID, err := nativeRandomID()
	if err != nil {
		return err
	}
	record := nativeRecoveryRecord{Version: nativeJournalVersion, RecordID: recordID,
		WorkspaceHandleID: j.workspaceID, RecordedAt: time.Now().UTC(), Handle: handle}
	wire, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode native recovery handle: %w", err)
	}
	handlesDir := filepath.Join(j.directory, "handles")
	if err := ensurePrivateDirectory(handlesDir); err != nil {
		return err
	}
	return writeAtomicUnique(handlesDir, "handle-"+recordID+".json", wire)
}

func (j *nativeJournal) appendEvent(event codexappserver.Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	wireBytes := len(event.Wire) + len(event.Params)
	if wireBytes == 0 || int64(wireBytes) > nativeJournalMaxEvents-j.eventBytes {
		return errors.New("native private event journal exceeds its configured bound")
	}
	j.eventSequence++
	record := nativeEventRecord{Version: nativeJournalVersion, Sequence: j.eventSequence,
		WorkspaceHandleID: j.workspaceID, Method: event.Method,
		WireBase64:   base64.StdEncoding.EncodeToString(event.Wire),
		ParamsBase64: base64.StdEncoding.EncodeToString(event.Params), RecordedAt: time.Now().UTC()}
	if err := j.appendJSONLine("events.jsonl", record); err != nil {
		return err
	}
	j.eventBytes += int64(wireBytes)
	return nil
}

func (j *nativeJournal) appendJSONLine(name string, value any) error {
	wire, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode native journal record: %w", err)
	}
	if len(wire)+1 > nativeJournalMaxRecord {
		return errors.New("native journal record exceeds its size bound")
	}
	path := filepath.Join(j.directory, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("native journal append target is unavailable or not a regular file")
	}
	if info.Size()+int64(len(wire)+1) > nativeJournalMaxEvents {
		return errors.New("native private journal exceeds its configured byte bound")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open native journal append target: %w", err)
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return fmt.Errorf("restrict native journal append target: %w", err)
	}
	line := append(wire, '\n')
	for len(line) > 0 {
		n, writeErr := f.Write(line)
		if writeErr != nil {
			return fmt.Errorf("append native journal record: %w", writeErr)
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		line = line[n:]
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush native journal record: %w", err)
	}
	return nil
}

func (j *nativeJournal) uniqueRequestIDLocked() (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		id, err := nativeRandomID()
		if err != nil {
			return "", err
		}
		id = "native-" + id
		if !j.usedRequestIDs[id] {
			j.usedRequestIDs[id] = true
			return id, nil
		}
	}
	return "", errors.New("could not allocate a unique native role request ID")
}

// readTrustedRecoveryHandles returns only Host-written recovery handles whose
// embedded workspace identity matches the caller's currently owned handle.
func readTrustedRecoveryHandles(privateLogDirectory, workspaceHandleID string) ([]codexappserver.RecoveryHandle, error) {
	return readTrustedRecoveryHandlesForCache(privateLogDirectory, workspaceHandleID, false)
}

// readTrustedRecoveryHandlesForCache permits an absent workspace path only
// for a caller that already found a Host-cached closed result. Active recovery
// continues to require the original workspace to resolve.
func readTrustedRecoveryHandlesForCache(privateLogDirectory, workspaceHandleID string, allowMissingWorkspace bool) ([]codexappserver.RecoveryHandle, error) {
	if strings.TrimSpace(workspaceHandleID) == "" {
		return nil, errors.New("workspace handle identity is required for recovery lookup")
	}
	privateAbs, err := filepath.Abs(privateLogDirectory)
	if err != nil {
		return nil, err
	}
	privateAbs, err = filepath.EvalSymlinks(privateAbs)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	root := filepath.Join(privateAbs, nativeJournalDirectory, workspaceJournalKey(workspaceHandleID))
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err := requirePrivateDirectory(filepath.Dir(root)); err != nil {
		return nil, err
	}
	if err := requirePrivateDirectory(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, k int) bool { return entries[i].Name() < entries[k].Name() })
	var handles []codexappserver.RecoveryHandle
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		invocationDir := filepath.Join(root, entry.Name())
		if err := requirePrivateDirectory(invocationDir); err != nil {
			return nil, err
		}
		handlesDir := filepath.Join(invocationDir, "handles")
		info, err := os.Lstat(handlesDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("trusted recovery handle directory is not a private directory")
		}
		if err := requirePrivateDirectory(handlesDir); err != nil {
			return nil, err
		}
		handleEntries, err := os.ReadDir(handlesDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sort.Slice(handleEntries, func(i, k int) bool { return handleEntries[i].Name() < handleEntries[k].Name() })
		for _, handleEntry := range handleEntries {
			if handleEntry.IsDir() || !strings.HasPrefix(handleEntry.Name(), "handle-") || !strings.HasSuffix(handleEntry.Name(), ".json") {
				continue
			}
			path := filepath.Join(handlesDir, handleEntry.Name())
			wire, err := readPrivateRegular(path, nativeJournalMaxRecord)
			if err != nil {
				return nil, err
			}
			var record nativeRecoveryRecord
			decoder := json.NewDecoder(strings.NewReader(string(wire)))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&record); err != nil {
				return nil, fmt.Errorf("decode trusted recovery handle: %w", err)
			}
			var trailing any
			if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
				return nil, errors.New("trusted recovery handle has trailing data")
			}
			if record.Version != nativeJournalVersion || record.WorkspaceHandleID != workspaceHandleID || record.Handle.Workspace.ID != workspaceHandleID ||
				record.RecordID == "" || record.Handle.Invocation.RunID == "" || handleEntry.Name() != "handle-"+record.RecordID+".json" {
				return nil, errors.New("trusted recovery handle workspace binding mismatch")
			}
			journalDir, err := filepath.EvalSymlinks(invocationDir)
			if err != nil {
				return nil, errors.New("trusted recovery handle journal is unavailable")
			}
			handleCWD, cwdErr := filepath.EvalSymlinks(record.Handle.Workspace.CWD)
			if cwdErr != nil && !(allowMissingWorkspace && errors.Is(cwdErr, os.ErrNotExist)) {
				return nil, errors.New("trusted recovery handle workspace is unavailable")
			}
			if cwdErr == nil && pathIsWithin(handleCWD, journalDir) {
				return nil, errors.New("trusted recovery handle journal is not outside its workspace")
			}
			handles = append(handles, record.Handle)
		}
	}
	return handles, nil
}

func workspaceJournalKey(workspaceID string) string {
	sum := sha256.Sum256([]byte(workspaceID))
	return hex.EncodeToString(sum[:])
}

func nativeRandomID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate native journal identity: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

// ensurePrivateDirectory creates a subdirectory that only holds Host journal
// files beneath a private log root that agentexec.PreparePrivateLogDirectory
// already prepared; it inherits that root's owner-only access. A directory
// handed to an agent as its own private log directory must be prepared with
// agentexec.PreparePrivateLogDirectory instead.
func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("create native private directory: %w", err)
	}
	return requirePrivateDirectory(path)
}

func requirePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("native private directory is unavailable: %s", path)
	}
	if err := os.Chmod(path, 0700); err != nil {
		return fmt.Errorf("restrict native private directory: %w", err)
	}
	return nil
}

func createPrivateFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create native private file: %w", err)
	}
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return fmt.Errorf("restrict native private file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flush native private file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close native private file: %w", err)
	}
	return nil
}

func writeAtomicUnique(directory, name string, wire []byte) error {
	if len(wire) == 0 || len(wire) > nativeJournalMaxRecord {
		return errors.New("native recovery record is empty or exceeds its bound")
	}
	temp, err := os.CreateTemp(directory, ".pending-recovery-")
	if err != nil {
		return fmt.Errorf("create pending recovery record: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("restrict pending recovery record: %w", err)
	}
	for remaining := wire; len(remaining) > 0; {
		n, writeErr := temp.Write(remaining)
		if writeErr != nil {
			_ = temp.Close()
			return fmt.Errorf("write pending recovery record: %w", writeErr)
		}
		if n == 0 {
			_ = temp.Close()
			return io.ErrShortWrite
		}
		remaining = remaining[n:]
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("flush pending recovery record: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close pending recovery record: %w", err)
	}
	if err := os.Rename(tempPath, filepath.Join(directory, name)); err != nil {
		return fmt.Errorf("publish recovery record atomically: %w", err)
	}
	return nil
}

func readPrivateRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limit {
		return nil, errors.New("trusted private record is unavailable or exceeds its bound")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	wire, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(wire)) > limit {
		return nil, errors.New("trusted private record exceeds its bound")
	}
	return wire, nil
}

func pathIsWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
