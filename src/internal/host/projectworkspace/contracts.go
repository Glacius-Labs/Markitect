// Package projectworkspace defines the Host seam for bounded project workspaces.
// It contains contracts and delta validation only; it does not create a workspace.
package projectworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type Service interface {
	Prepare(context.Context, Request) (Handle, error)
	Harvest(context.Context, Handle) (Delta, error)
	Close(context.Context, Handle) error
}

// Request binds a future workspace to one repository snapshot and task. The
// overlay digest identifies caller-owned WIP bytes; it does not invent Git history.
type Request struct {
	RepositoryRoot     string
	RepositoryIdentity string
	BaseSHA            string
	OverlayDigest      string
	TaskID             string
	AllowedPaths       []string
	ExcludedPaths      []string
}

// Handle is the opaque identity and immutable binding returned by Prepare.
// CWD is the process working directory a future implementation would provide.
type Handle struct {
	ID                 string
	CWD                string
	RepositoryRoot     string
	RepositoryIdentity string
	BaseSHA            string
	OverlayDigest      string
	TaskID             string
	BaseDigest         string
}

type ChangeKind string

const (
	ChangeAdd    ChangeKind = "add"
	ChangeModify ChangeKind = "modify"
	ChangeDelete ChangeKind = "delete"
	ChangeRename ChangeKind = "rename"
)

// Change is a typed byte-level delta. Rename carries the destination bytes and
// mode plus its source path; it is not represented as a pretend filesystem move.
// Delete has Path set and must have empty OldPath, Mode, and nil Content.
type Change struct {
	Kind    ChangeKind `json:"kind"`
	Path    string     `json:"path"`
	OldPath string     `json:"oldPath,omitempty"`
	Mode    string     `json:"mode,omitempty"`
	Content []byte     `json:"content"`
}

type Delta struct {
	RepositoryIdentity string   `json:"repositoryIdentity"`
	BaseSHA            string   `json:"baseSha"`
	OverlayDigest      string   `json:"overlayDigest"`
	TaskID             string   `json:"taskId"`
	BaseDigest         string   `json:"baseDigest"`
	Changes            []Change `json:"changes"`
	Digest             string   `json:"digest"`
}

type Limits struct {
	MaxFiles      int
	MaxFileBytes  int
	MaxTotalBytes int
}

var (
	ErrInvalidRequest = errors.New("invalid project workspace request")
	ErrInvalidHandle  = errors.New("project workspace handle does not match request")
	ErrInvalidDelta   = errors.New("invalid project workspace delta")
	sha1OrSHA256      = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	sha256Digest      = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	drivePath         = regexp.MustCompile(`^[A-Za-z]:`)
	reservedWindows   = regexp.MustCompile(`(?i)^(?:con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\..*)?$`)
)

func (r Request) Validate() error {
	if !utf8.ValidString(r.RepositoryRoot) || !utf8.ValidString(r.RepositoryIdentity) || !utf8.ValidString(r.TaskID) ||
		!filepath.IsAbs(r.RepositoryRoot) || strings.TrimSpace(r.RepositoryIdentity) == "" ||
		!sha1OrSHA256.MatchString(r.BaseSHA) || !sha256Digest.MatchString(r.OverlayDigest) || strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: repository, identity, full base SHA, overlay digest, and task ID are required", ErrInvalidRequest)
	}
	if err := validateScopes(r.AllowedPaths, "allowed paths"); err != nil {
		return err
	}
	if err := validateScopes(r.ExcludedPaths, "excluded paths"); err != nil {
		return err
	}
	return nil
}

func (h Handle) ValidateFor(r Request) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(h.ID) || !utf8.ValidString(h.CWD) || strings.TrimSpace(h.ID) == "" || !filepath.IsAbs(h.CWD) ||
		h.RepositoryRoot != r.RepositoryRoot || h.RepositoryIdentity != r.RepositoryIdentity ||
		h.BaseSHA != r.BaseSHA || h.OverlayDigest != r.OverlayDigest || h.TaskID != r.TaskID ||
		!sha256Digest.MatchString(h.BaseDigest) {
		return ErrInvalidHandle
	}
	return nil
}

// NormalizeDelta validates scope and operation semantics, binds the delta to
// the exact request/handle snapshot, clones caller-owned bytes, sorts changes,
// and computes a deterministic digest. It does not assert source-tree existence.
func NormalizeDelta(r Request, h Handle, changes []Change, limits Limits) (Delta, error) {
	if err := h.ValidateFor(r); err != nil {
		return Delta{}, err
	}
	if limits.MaxFiles <= 0 || limits.MaxFileBytes <= 0 || limits.MaxTotalBytes <= 0 {
		return Delta{}, fmt.Errorf("%w: all delta limits must be positive", ErrInvalidDelta)
	}
	if len(changes) > limits.MaxFiles {
		return Delta{}, fmt.Errorf("%w: change count exceeds limit", ErrInvalidDelta)
	}
	allowed, err := normalizeScopes(r.AllowedPaths, "allowed paths")
	if err != nil {
		return Delta{}, err
	}
	excluded, err := normalizeScopes(r.ExcludedPaths, "excluded paths")
	if err != nil {
		return Delta{}, err
	}

	normalized := make([]Change, 0, len(changes))
	touched := make(map[string]reservedPath, len(changes)*2)
	totalBytes := 0
	for _, original := range changes {
		change := original
		path, err := portablePath(change.Path)
		if err != nil {
			return Delta{}, fmt.Errorf("%w: invalid changed path", ErrInvalidDelta)
		}
		change.Path = path
		if err := authorize(path, allowed, excluded); err != nil {
			return Delta{}, err
		}
		deletesPath := false
		switch change.Kind {
		case ChangeAdd, ChangeModify:
			if change.OldPath != "" || !validMode(change.Mode) || change.Content == nil {
				return Delta{}, fmt.Errorf("%w: add/modify requires mode and content, and no old path", ErrInvalidDelta)
			}
		case ChangeDelete:
			if change.OldPath != "" || change.Mode != "" || change.Content != nil {
				return Delta{}, fmt.Errorf("%w: delete cannot carry mode or content", ErrInvalidDelta)
			}
			deletesPath = true
		case ChangeRename:
			oldPath, pathErr := portablePath(change.OldPath)
			if pathErr != nil || oldPath == path {
				return Delta{}, fmt.Errorf("%w: rename requires a distinct safe old path", ErrInvalidDelta)
			}
			if err := authorize(oldPath, allowed, excluded); err != nil {
				return Delta{}, err
			}
			change.OldPath = oldPath
			if !validMode(change.Mode) || change.Content == nil {
				return Delta{}, fmt.Errorf("%w: rename requires destination mode and content", ErrInvalidDelta)
			}
		default:
			return Delta{}, fmt.Errorf("%w: unsupported change kind", ErrInvalidDelta)
		}
		if err := reservePath(touched, path, deletesPath); err != nil {
			return Delta{}, err
		}
		if change.Kind == ChangeRename {
			if err := reservePath(touched, change.OldPath, true); err != nil {
				return Delta{}, err
			}
		}
		if change.Content != nil {
			if len(change.Content) > limits.MaxFileBytes {
				return Delta{}, fmt.Errorf("%w: file content exceeds limit", ErrInvalidDelta)
			}
			if len(change.Content) > limits.MaxTotalBytes-totalBytes {
				return Delta{}, fmt.Errorf("%w: total content exceeds limit", ErrInvalidDelta)
			}
			totalBytes += len(change.Content)
			cloned := make([]byte, len(change.Content))
			copy(cloned, change.Content)
			change.Content = cloned
		}
		normalized = append(normalized, change)
	}
	sort.Slice(normalized, func(i, j int) bool {
		left, right := strings.ToLower(normalized[i].Path), strings.ToLower(normalized[j].Path)
		if left != right {
			return left < right
		}
		return normalized[i].Path < normalized[j].Path
	})
	delta := Delta{
		RepositoryIdentity: r.RepositoryIdentity,
		BaseSHA:            r.BaseSHA,
		OverlayDigest:      r.OverlayDigest,
		TaskID:             r.TaskID,
		BaseDigest:         h.BaseDigest,
		Changes:            normalized,
	}
	digestBytes, err := json.Marshal(struct {
		RepositoryIdentity string   `json:"repositoryIdentity"`
		BaseSHA            string   `json:"baseSha"`
		OverlayDigest      string   `json:"overlayDigest"`
		TaskID             string   `json:"taskId"`
		BaseDigest         string   `json:"baseDigest"`
		Changes            []Change `json:"changes"`
	}{delta.RepositoryIdentity, delta.BaseSHA, delta.OverlayDigest, delta.TaskID, delta.BaseDigest, delta.Changes})
	if err != nil {
		return Delta{}, fmt.Errorf("%w: could not encode canonical delta", ErrInvalidDelta)
	}
	sum := sha256.Sum256(digestBytes)
	delta.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return delta, nil
}

func validMode(mode string) bool { return mode == "100644" || mode == "100755" }

func portablePath(value string) (string, error) {
	if value == "" || !utf8.ValidString(value) || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") || drivePath.MatchString(value) {
		return "", ErrInvalidDelta
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", ErrInvalidDelta
		}
		if strings.ContainsAny(part, `<>:"|?*`) || strings.HasSuffix(part, " ") || strings.HasSuffix(part, ".") || reservedWindows.MatchString(part) {
			return "", ErrInvalidDelta
		}
		for _, r := range part {
			if r < 32 || r == 127 {
				return "", ErrInvalidDelta
			}
		}
	}
	for _, part := range parts {
		// git~1 is the usual Windows 8.3 short name of .git; Git's
		// core.protectNTFS refuses it on every platform.
		if strings.EqualFold(part, ".git") || strings.EqualFold(part, "git~1") {
			return "", fmt.Errorf("%w: Git metadata paths are not workspace data", ErrInvalidDelta)
		}
	}
	if strings.EqualFold(parts[0], ".markitect") {
		return "", fmt.Errorf("%w: control-plane paths are not workspace data", ErrInvalidDelta)
	}
	return strings.Join(parts, "/"), nil
}

func validateScopes(scopes []string, label string) error {
	_, err := normalizeScopes(scopes, label)
	return err
}

func normalizeScopes(scopes []string, label string) ([]string, error) {
	result := make([]string, 0, len(scopes))
	seen := make(map[string]string, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimRight(scope, "/")
		path, err := portablePath(scope)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid %s scope", ErrInvalidRequest, label)
		}
		key := strings.ToLower(path)
		if prior, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: duplicate or case-alias %s scopes %q and %q", ErrInvalidRequest, label, prior, path)
		}
		seen[key] = path
		result = append(result, path)
	}
	return result, nil
}

func authorize(path string, allowed, excluded []string) error {
	for _, scope := range excluded {
		if within(path, scope) {
			return fmt.Errorf("%w: path is excluded: %s", ErrInvalidDelta, path)
		}
	}
	for _, scope := range allowed {
		if within(path, scope) {
			return nil
		}
	}
	return fmt.Errorf("%w: path is outside allowed ownership scopes: %s", ErrInvalidDelta, path)
}

func within(path, scope string) bool {
	path, scope = strings.ToLower(path), strings.ToLower(scope)
	return path == scope || strings.HasPrefix(path, scope+"/")
}

type reservedPath struct {
	path   string
	delete bool
}

func reservePath(touched map[string]reservedPath, path string, deletes bool) error {
	key := strings.ToLower(path)
	if prior, ok := touched[key]; ok {
		if prior.path != path {
			return fmt.Errorf("%w: case-alias paths %q and %q", ErrInvalidDelta, prior.path, path)
		}
		return fmt.Errorf("%w: path is used by multiple changes: %s", ErrInvalidDelta, path)
	}
	keys := make([]string, 0, len(touched))
	for priorKey := range touched {
		keys = append(keys, priorKey)
	}
	sort.Strings(keys)
	for _, priorKey := range keys {
		prior := touched[priorKey]
		if priorKey == key {
			continue
		}
		if strings.HasPrefix(priorKey, key+"/") || strings.HasPrefix(key, priorKey+"/") {
			// Replacing a file with a directory tree (or the inverse) is
			// valid when the old path is explicitly deleted. Two writes still
			// cannot name a file and one of its descendants at once.
			if !prior.delete && !deletes {
				return fmt.Errorf("%w: file/directory path collision between %q and %q", ErrInvalidDelta, prior.path, path)
			}
		}
	}
	touched[key] = reservedPath{path: path, delete: deletes}
	return nil
}
