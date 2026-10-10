package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

const maxApprovalChanges = 128

type fileChangeApprovalParams struct {
	ThreadID    string  `json:"threadId"`
	TurnID      string  `json:"turnId"`
	ItemID      string  `json:"itemId"`
	StartedAtMs *int64  `json:"startedAtMs"`
	Reason      *string `json:"reason"`
	GrantRoot   *string `json:"grantRoot"`
}

type workspaceApprovalContext struct {
	Kind               string          `json:"kind"`
	ManagerID          string          `json:"managerId"`
	Phase              string          `json:"phase"`
	AllowedWritePaths  []string        `json:"allowedWritePaths"`
	ExcludedWritePaths []string        `json:"excludedWritePaths"`
	HelperDepth        int             `json:"helperDepth"`
	HelperAccounting   string          `json:"helperAccounting"`
	Task               string          `json:"task"`
	ParentContext      json.RawMessage `json:"parentContext"`
}

func (s *session) serverRequest(ctx context.Context, msg envelope) error {
	switch msg.Method {
	case "item/tool/call":
		return s.toolRequest(ctx, msg)
	case "item/fileChange/requestApproval":
		return s.fileChangeApproval(ctx, msg)
	default:
		return ErrApprovalRequired
	}
}

func (s *session) fileChangeApproval(ctx context.Context, msg envelope) error {
	var params fileChangeApprovalParams
	parseErr := decodeFileChangeApprovalParams(msg.Params, &params)
	decision := "decline"
	var changes []projectworkspace.Change
	key := fileChangeKey(params.ThreadID, params.TurnID, params.ItemID)
	if parseErr == nil && params.ThreadID == s.h.ThreadID && params.TurnID != "" && params.TurnID == s.h.TurnID && params.ItemID != "" &&
		params.StartedAtMs != nil && *params.StartedAtMs >= 0 && (params.Reason == nil || len(*params.Reason) <= 8192) && params.GrantRoot == nil && !s.handledApprovals[key] {
		if tracked, ok := s.pendingFileChanges[key]; ok && tracked.ThreadID == params.ThreadID && tracked.TurnID == params.TurnID && tracked.Item.ID == params.ItemID {
			changes, parseErr = s.scopedFileChanges(ctx, tracked.Item.Changes)
			if parseErr == nil {
				decision = "accept"
			}
		} else {
			parseErr = ErrProtocol
		}
	} else if parseErr == nil {
		parseErr = ErrProtocol
	}
	if err := s.recordApprovalDecision(ctx, params, decision, changes); err != nil {
		return err
	}
	if err := s.c.send(ctx, map[string]any{"id": msg.ID, "result": map[string]string{"decision": decision}}); err != nil {
		return err
	}
	if decision != "accept" {
		return ErrApprovalRequired
	}
	s.handledApprovals[key] = true
	return nil
}

func decodeFileChangeApprovalParams(raw []byte, params *fileChangeApprovalParams) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return ErrProtocol
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return ErrProtocol
		}
		switch key {
		case "threadId", "turnId", "itemId", "startedAtMs", "reason", "grantRoot":
		default:
			return ErrProtocol
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return ErrProtocol
		}
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return ErrProtocol
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrProtocol
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(params); err != nil {
		return ErrProtocol
	}
	return nil
}

func (s *session) recordApprovalDecision(ctx context.Context, params fileChangeApprovalParams, decision string, changes []projectworkspace.Change) error {
	if s.a.options.OnEvent == nil {
		return nil
	}
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.Path)
	}
	wire, err := json.Marshal(struct {
		ThreadID string   `json:"threadId"`
		TurnID   string   `json:"turnId"`
		ItemID   string   `json:"itemId"`
		Decision string   `json:"decision"`
		Scope    string   `json:"scope"`
		Paths    []string `json:"paths"`
	}{params.ThreadID, params.TurnID, params.ItemID, decision, "delegated-write-scope", paths})
	if err != nil {
		return err
	}
	return s.a.options.OnEvent(ctx, Event{Method: "markitect/fileChangeApproval/decision", Params: wire})
}

func (s *session) scopedFileChanges(ctx context.Context, proposed []fileUpdateChange) ([]projectworkspace.Change, error) {
	if len(proposed) == 0 || len(proposed) > maxApprovalChanges || s.inv.Request.Role != agentexec.RoleExecutor || s.inv.Request.SourceRevision != s.h.Workspace.BaseSHA ||
		s.effectiveSandboxType != "workspaceWrite" || s.a.config.PermissionProfile == ":read-only" {
		return nil, ErrApprovalRequired
	}
	var scope workspaceApprovalContext
	if len(s.inv.Request.Context) == 0 || len(s.inv.Request.Context) > 1<<20 || json.Unmarshal(s.inv.Request.Context, &scope) != nil || strings.TrimSpace(scope.ManagerID) == "" || len(scope.AllowedWritePaths) == 0 {
		return nil, ErrApprovalRequired
	}
	parentScope := scope
	switch scope.Kind {
	case "projectrun-task/v1":
		if scope.Phase != "work" && scope.Phase != "integrate" && scope.Phase != "repair" {
			return nil, ErrApprovalRequired
		}
	case "projectrun-helper/v1":
		if scope.HelperDepth != 1 || scope.HelperAccounting != "partial" || strings.TrimSpace(scope.Task) == "" || len(scope.ParentContext) == 0 || s.a.options.BeforeStart == nil {
			return nil, ErrApprovalRequired
		}
		if json.Unmarshal(scope.ParentContext, &parentScope) != nil || parentScope.Kind != "projectrun-task/v1" || parentScope.ManagerID != scope.ManagerID ||
			(parentScope.Phase != "work" && parentScope.Phase != "integrate" && parentScope.Phase != "repair") || len(parentScope.AllowedWritePaths) == 0 {
			return nil, ErrApprovalRequired
		}
	default:
		return nil, ErrApprovalRequired
	}

	h := s.h.Workspace
	r := projectworkspace.Request{RepositoryRoot: h.RepositoryRoot, RepositoryIdentity: h.RepositoryIdentity, BaseSHA: h.BaseSHA,
		OverlayDigest: h.OverlayDigest, TaskID: h.TaskID, AllowedPaths: append([]string(nil), scope.AllowedWritePaths...), ExcludedPaths: append([]string(nil), scope.ExcludedWritePaths...)}
	if h.ValidateFor(r) != nil || projectworkspace.ValidateOwnedWorkspace(r, h) != nil {
		return nil, ErrApprovalRequired
	}
	changes := make([]projectworkspace.Change, 0, len(proposed))
	for _, proposedChange := range proposed {
		path, err := workspaceRelativeChangePath(h.CWD, proposedChange.Path, proposedChange.Kind.Type)
		if err != nil {
			return nil, ErrApprovalRequired
		}
		change := projectworkspace.Change{Path: path, Mode: "100644"}
		if proposedChange.Kind.MovePath != nil || proposedChange.Kind.MovePathSnake != nil {
			return nil, ErrApprovalRequired
		}
		switch proposedChange.Kind.Type {
		case "add":
			change.Kind, change.Content = projectworkspace.ChangeAdd, []byte{}
		case "update":
			change.Kind, change.Content = projectworkspace.ChangeModify, []byte{}
		case "delete":
			change.Kind, change.Mode = projectworkspace.ChangeDelete, ""
		default:
			return nil, ErrApprovalRequired
		}
		changes = append(changes, change)
	}
	limits := projectworkspace.Limits{MaxFiles: maxApprovalChanges, MaxFileBytes: 1, MaxTotalBytes: maxApprovalChanges}
	if _, err := projectworkspace.NormalizeDelta(r, h, changes, limits); err != nil {
		return nil, ErrApprovalRequired
	}
	if scope.Kind == "projectrun-helper/v1" {
		parentRequest := r
		parentRequest.AllowedPaths = append([]string(nil), parentScope.AllowedWritePaths...)
		parentRequest.ExcludedPaths = append([]string(nil), parentScope.ExcludedWritePaths...)
		if _, err := projectworkspace.NormalizeDelta(parentRequest, h, changes, limits); err != nil {
			return nil, ErrApprovalRequired
		}
	}
	return changes, nil
}

func sameFilesystemPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func workspaceRelativeChangePath(root, candidate, kind string) (string, error) {
	if !filepath.IsAbs(candidate) || candidate == "" || hasPathDotSegment(candidate) {
		return "", projectworkspace.ErrInvalidDelta
	}
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", projectworkspace.ErrInvalidHandle
	}
	if relative, relErr := filepath.Rel(root, candidate); relErr == nil && relative != "." && !filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		if err := verifyApprovalPath(root, relative, kind); err != nil {
			return "", err
		}
		return filepath.ToSlash(relative), nil
	}
	if runtime.GOOS != "windows" {
		return "", projectworkspace.ErrInvalidDelta
	}
	if !localWindowsAbsolutePath(root) || !localWindowsAbsolutePath(candidate) {
		return "", projectworkspace.ErrInvalidDelta
	}
	for ancestor := filepath.Dir(candidate); ; ancestor = filepath.Dir(ancestor) {
		ancestorInfo, statErr := os.Stat(ancestor)
		if statErr == nil && ancestorInfo.IsDir() && os.SameFile(info, ancestorInfo) {
			relative, relErr := filepath.Rel(ancestor, candidate)
			if relErr == nil && relative != "." && !filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				if err := verifyPhysicalAliasRoot(ancestor); err != nil {
					return "", err
				}
				if err := verifyApprovalPath(ancestor, relative, kind); err != nil {
					return "", err
				}
				return filepath.ToSlash(relative), nil
			}
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
	}
	return "", projectworkspace.ErrInvalidDelta
}

func localWindowsAbsolutePath(path string) bool {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, `\\?\`) {
		path = strings.TrimPrefix(path, `\\?\`)
	}
	if strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `\\.\`) || len(path) < 3 {
		return false
	}
	letter := path[0]
	return ((letter >= 'a' && letter <= 'z') || (letter >= 'A' && letter <= 'Z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}

func verifyPhysicalAliasRoot(root string) error {
	for current := root; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return projectworkspace.ErrInvalidHandle
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return projectworkspace.ErrInvalidHandle
		}
		resolvedInfo, err := os.Stat(resolved)
		if err != nil || !resolvedInfo.IsDir() || !os.SameFile(info, resolvedInfo) {
			return projectworkspace.ErrInvalidHandle
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return nil
}

func hasPathDotSegment(value string) bool {
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

func verifyApprovalPath(root, relative, kind string) error {
	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	if len(parts) == 0 || relative == "." || filepath.IsAbs(relative) {
		return projectworkspace.ErrInvalidDelta
	}
	current := root
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			return projectworkspace.ErrInvalidDelta
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		last := index == len(parts)-1
		if errors.Is(err, os.ErrNotExist) && kind == "add" {
			// Missing descendants are safe for an add once every existing
			// ancestor has already been proven to be a real in-root directory.
			return nil
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !storedName(current) {
			return projectworkspace.ErrInvalidDelta
		}
		if last {
			if kind == "add" || !info.Mode().IsRegular() {
				return projectworkspace.ErrInvalidDelta
			}
		} else if !info.IsDir() {
			return projectworkspace.ErrInvalidDelta
		}
	}
	return nil
}

// storedName reports whether an existing path's final component is spelled
// as its directory entry. Windows also resolves an 8.3 short name or another
// case to the entry, and scope checks only see the requested text, so an
// alias such as docs/GENERA~1 could otherwise approve a write to an excluded
// docs/generated-protos. Only stored names appear in a directory listing.
func storedName(path string) bool {
	if runtime.GOOS != "windows" {
		return true
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	name := filepath.Base(path)
	for _, entry := range entries {
		if entry.Name() == name {
			return true
		}
	}
	return false
}
