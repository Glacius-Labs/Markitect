package projectworkspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

const ownershipRecordName = "workspace-owner.json"
const maxOwnershipRecordBytes = 1 << 20

type ownershipRecord struct {
	Schema        string  `json:"schema"`
	Handle        Handle  `json:"handle"`
	Request       Request `json:"request"`
	SourceDigest  string  `json:"sourceDigest"`
	OverlayDigest string  `json:"candidateOverlayDigest"`
}

func writeOwnershipRecord(st *workspaceState) error {
	record := ownershipRecord{"markitect-owned-workspace/v1", st.handle, st.request, st.sourceDigest, st.overlayDigest}
	content, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if len(content) > maxOwnershipRecordBytes {
		return fmt.Errorf("%w: ownership record exceeds limit", ErrInvalidRequest)
	}
	path := filepath.Join(st.storage, ownershipRecordName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// ValidateOwnedWorkspace verifies the live physical workspace path and its
// stable Host ownership record without registering, harvesting, or closing it.
func ValidateOwnedWorkspace(r Request, h Handle) error {
	if err := h.ValidateFor(r); err != nil || len(h.ID) != 32 || filepath.Base(h.CWD) != "repo" {
		return ErrInvalidHandle
	}
	for _, c := range h.ID {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ErrInvalidHandle
		}
	}
	storage := filepath.Dir(h.CWD)
	if !strings.HasPrefix(filepath.Base(storage), "workspace-"+h.ID+"-") {
		return ErrInvalidHandle
	}
	for _, path := range []string{storage, h.CWD, filepath.Join(h.CWD, ".git")} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrInvalidHandle
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || !samePath(resolved, path) {
			return ErrInvalidHandle
		}
	}
	sourceRoot, err := filepath.EvalSymlinks(r.RepositoryRoot)
	if err != nil {
		return err
	}
	inside, err := withinRepository(sourceRoot, storage)
	if err != nil {
		return err
	}
	if inside {
		return ErrInvalidHandle
	}
	_, err = readOwnedWorkspaceRecord(storage, r, h)
	return err
}

func readOwnedWorkspaceRecord(storage string, r Request, h Handle) (ownershipRecord, error) {
	// Confine the marker read and require a stable regular file. It is not a
	// content snapshot of agent output; it binds original Prepare ownership.
	root, err := os.OpenRoot(storage)
	if err != nil {
		return ownershipRecord{}, err
	}
	defer root.Close()
	info, err := root.Lstat(ownershipRecordName)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxOwnershipRecordBytes {
		return ownershipRecord{}, ErrInvalidHandle
	}
	file, err := root.Open(ownershipRecordName)
	if err != nil {
		return ownershipRecord{}, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		file.Close()
		return ownershipRecord{}, ErrInvalidHandle
	}
	content, err := io.ReadAll(io.LimitReader(file, maxOwnershipRecordBytes+1))
	after, statErr := file.Stat()
	file.Close()
	current, currentErr := root.Lstat(ownershipRecordName)
	if err != nil {
		return ownershipRecord{}, err
	}
	if statErr != nil || currentErr != nil || len(content) > maxOwnershipRecordBytes || !os.SameFile(info, after) || !os.SameFile(info, current) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return ownershipRecord{}, ErrInvalidHandle
	}
	var record ownershipRecord
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil || decoder.Decode(new(any)) != io.EOF || record.Schema != "markitect-owned-workspace/v1" ||
		record.Handle != h || !sameRequest(record.Request, r) || !sha256Digest.MatchString(record.SourceDigest) || (record.OverlayDigest != "" && !sha256Digest.MatchString(record.OverlayDigest)) {
		return ownershipRecord{}, ErrInvalidHandle
	}
	return record, nil
}

// ReopenCandidate is an explicit trusted Host-journal seam, never discovery or
// automatic adoption. terminalConfirmed must attest the original execution is
// observed terminal and all writers are stopped; unknown/active execution fails.
// Quiescence cannot be proven from provider state by this filesystem service.
// Legacy/foreign candidates lacking the original ownership record fail intact.
func (s *GitService) ReopenCandidate(ctx context.Context, r Request, h Handle, overlay []Change, digest string, terminalConfirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !terminalConfirmed {
		return fmt.Errorf("%w: original execution terminal/quiescent state is not confirmed", ErrInvalidHandle)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := h.ValidateFor(r); err != nil {
		return err
	}
	if len(h.ID) != 32 {
		return ErrInvalidHandle
	}
	for _, c := range h.ID {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ErrInvalidHandle
		}
	}
	if _, exists := s.states[h.ID]; exists {
		return fmt.Errorf("%w: workspace already open", ErrInvalidHandle)
	}
	storage := filepath.Dir(h.CWD)
	if filepath.Base(h.CWD) != "repo" || !samePath(filepath.Dir(storage), s.root) || !strings.HasPrefix(filepath.Base(storage), "workspace-"+h.ID+"-") {
		return ErrInvalidHandle
	}
	for _, path := range []string{storage, h.CWD, filepath.Join(h.CWD, ".git")} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrInvalidHandle
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || !samePath(resolved, path) {
			return ErrInvalidHandle
		}
	}
	sourceRoot, err := filepath.EvalSymlinks(r.RepositoryRoot)
	if err != nil {
		return err
	}
	inside, err := withinRepository(sourceRoot, storage)
	if err != nil {
		return err
	}
	if inside {
		return ErrInvalidHandle
	}
	record, err := readOwnedWorkspaceRecord(storage, r, h)
	if err != nil {
		return err
	}
	if record.OverlayDigest != digest {
		return ErrInvalidHandle
	}
	binding, err := InspectRepository(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return err
	}
	if binding.OverlayDigest != r.OverlayDigest || binding.InventoryDigest != record.SourceDigest {
		return fmt.Errorf("%w: source changed before recovery", ErrInvalidRequest)
	}
	initial, err := sourceInventory(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return err
	}
	if inventoryDigest(initial) != record.SourceDigest {
		return ErrInvalidRequest
	}
	if len(overlay) > 0 || digest != "" {
		canonical, err := canonicalOverlayWithLimits(overlay, s.limits)
		if err != nil {
			return err
		}
		if canonical.Digest != digest {
			return fmt.Errorf("%w: recovery overlay differs", ErrInvalidRequest)
		}
		if err := materializeOverlay(initial, canonical.Changes); err != nil {
			return err
		}
	}
	if inventoryDigest(initial) != h.BaseDigest {
		return fmt.Errorf("%w: reconstructed candidate baseline differs", ErrInvalidHandle)
	}
	if err := validateCandidateGit(ctx, h); err != nil {
		return err
	}
	if _, err := git(ctx, h.CWD, "merge-base", "--is-ancestor", r.BaseSHA, "HEAD"); err != nil {
		return ErrInvalidHandle
	}
	first, err := readInventoryBounded(ctx, h.CWD, initial, s.limits)
	if err != nil {
		return err
	}
	second, err := readInventoryBounded(ctx, h.CWD, initial, s.limits)
	if err != nil {
		return err
	}
	if inventoryDigest(first) != inventoryDigest(second) {
		return fmt.Errorf("%w: candidate changed during recovery", ErrInvalidDelta)
	}
	// Recovery does not accept writes. Harvest still checks actual scope/delta
	// semantics before any preserved output can become a candidate.
	fresh, err := InspectRepository(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return err
	}
	if fresh != binding {
		return fmt.Errorf("%w: source changed during recovery", ErrInvalidRequest)
	}
	r.AllowedPaths = append([]string(nil), r.AllowedPaths...)
	r.ExcludedPaths = append([]string(nil), r.ExcludedPaths...)
	s.states[h.ID] = &workspaceState{handle: h, request: r, initial: initial, storage: storage, sourceDigest: record.SourceDigest, overlayDigest: digest}
	return nil
}
func sameRequest(a, b Request) bool {
	// Empty and nil scopes have identical contract semantics.
	a.AllowedPaths = append([]string(nil), a.AllowedPaths...)
	a.ExcludedPaths = append([]string(nil), a.ExcludedPaths...)
	b.AllowedPaths = append([]string(nil), b.AllowedPaths...)
	b.ExcludedPaths = append([]string(nil), b.ExcludedPaths...)
	return reflect.DeepEqual(a, b)
}

func validateCandidateGit(ctx context.Context, h Handle) error {
	metadata, err := os.Lstat(filepath.Join(h.CWD, ".git"))
	if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 {
		return ErrInvalidHandle
	}
	output, err := git(ctx, h.CWD, "rev-parse", "--absolute-git-dir", "--show-toplevel", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return ErrInvalidHandle
	}
	paths := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(paths) != 3 {
		return ErrInvalidHandle
	}
	for i, expected := range []string{filepath.Join(h.CWD, ".git"), h.CWD, filepath.Join(h.CWD, ".git")} {
		if !sameGitReportedPath(strings.TrimSpace(paths[i]), expected) {
			return ErrInvalidHandle
		}
	}
	return nil
}

// Git may report a Windows package-virtualized path spelling for the same
// directory that the Host named. Keep ordinary path and containment checks
// lexical; only a Git-reported directory identity may use the underlying
// existing directory identity as a fallback.
func sameGitReportedPath(reported, expected string) bool {
	if samePath(reported, expected) {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}
	reportedInfo, reportedErr := os.Stat(reported)
	expectedInfo, expectedErr := os.Stat(expected)
	return reportedErr == nil && expectedErr == nil && reportedInfo.IsDir() && expectedInfo.IsDir() && os.SameFile(reportedInfo, expectedInfo)
}
