package codexappserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type fileUpdateChange struct {
	Path string         `json:"path"`
	Kind fileChangeKind `json:"kind"`
}

type fileChangeKind struct {
	Type          string  `json:"type"`
	MovePath      *string `json:"movePath"`
	MovePathSnake *string `json:"move_path"`
}

func (k fileChangeKind) MarshalJSON() ([]byte, error) {
	value := map[string]any{"type": k.Type}
	if k.Type == "update" || k.MovePathSnake != nil {
		value["move_path"] = k.MovePathSnake
	}
	return json.Marshal(value)
}

type trackedFileChange struct {
	ThreadID string
	TurnID   string
	Item     item
}

func fileChangeKey(threadID, turnID, itemID string) string {
	return strings.Join([]string{threadID, turnID, itemID}, "\x00")
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func readPinned(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("runtime instruction source is not a bounded regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("runtime instruction source exceeds limit")
	}
	return b, nil
}

func (s *session) observe(e Event) error {
	if s.a.options.OnEvent != nil {
		copy := Event{Method: e.Method, Params: append(json.RawMessage(nil), e.Params...), Wire: append(json.RawMessage(nil), e.Wire...)}
		if err := s.a.options.OnEvent(s.ctx, copy); err != nil {
			return err
		}
	}
	var p struct {
		ThreadID       string `json:"threadId"`
		TurnID         string `json:"turnId"`
		Turn           turn   `json:"turn"`
		Item           item   `json:"item"`
		Thread         thread `json:"thread"`
		ThreadSettings struct {
			Model                   string `json:"model"`
			Effort                  string `json:"effort"`
			CWD                     string `json:"cwd"`
			ActivePermissionProfile *struct {
				ID string `json:"id"`
			} `json:"activePermissionProfile"`
		} `json:"threadSettings"`
		TokenUsage struct {
			Total struct {
				InputTokens       *int64 `json:"inputTokens"`
				OutputTokens      *int64 `json:"outputTokens"`
				CachedInputTokens *int64 `json:"cachedInputTokens"`
			} `json:"total"`
		} `json:"tokenUsage"`
	}
	if json.Unmarshal(e.Params, &p) != nil {
		return ErrProtocol
	}
	switch e.Method {
	case "thread/started":
		if p.Thread.ParentThreadID == "" {
			return nil
		}
		parentDepth, known := s.children[p.Thread.ParentThreadID]
		if p.Thread.ParentThreadID == s.h.ThreadID {
			known = true
		}
		if !known {
			return nil
		} // retain raw evidence; unknown ancestry is not invented
		if p.Thread.ID == "" || p.Thread.SessionID == "" {
			return ErrProtocol
		}
		depth := parentDepth + 1
		s.children[p.Thread.ID] = depth
		s.sessions[p.Thread.ID] = p.Thread.SessionID
		if index, ok := s.receivers[p.Thread.ID]; ok {
			s.life.StartRequests[index].SessionID = p.Thread.SessionID
		}
		if !s.a.config.Helpers.Enabled || depth > s.a.config.Helpers.MaxDepth {
			return errors.New("observed native helper depth/policy exceeded bound")
		}
	case "turn/started", "turn/completed":
		if p.ThreadID != s.h.ThreadID {
			return nil
		}
		if p.Turn.ID == "" || (s.h.TurnID != "" && s.h.TurnID != p.Turn.ID) {
			return ErrProtocol
		}
		s.h.TurnID = p.Turn.ID
		s.life.TurnID = p.Turn.ID
		if e.Method == "turn/started" {
			s.life.State = "running"
			return s.save()
		}
		for _, v := range p.Turn.Items {
			if err := s.takeItem(p.ThreadID, v, true); err != nil {
				return err
			}
		}
		return s.finish(p.Turn)
	case "item/started", "item/completed":
		if p.ThreadID != s.h.ThreadID {
			if _, ok := s.children[p.ThreadID]; !ok {
				return nil
			}
		}
		if p.ThreadID == s.h.ThreadID && s.h.TurnID != "" && p.TurnID != s.h.TurnID {
			return ErrProtocol
		}
		if p.Item.Type == "fileChange" && p.ThreadID == s.h.ThreadID {
			if s.pendingFileChanges == nil {
				s.pendingFileChanges = make(map[string]trackedFileChange)
			}
			key := fileChangeKey(p.ThreadID, p.TurnID, p.Item.ID)
			if e.Method == "item/started" {
				if p.TurnID == "" || p.TurnID != s.h.TurnID || p.Item.Status != "inProgress" || len(p.Item.Changes) == 0 || len(p.Item.Changes) > 128 || len(s.pendingFileChanges) >= 1024 {
					return ErrProtocol
				}
				s.pendingFileChanges[key] = trackedFileChange{ThreadID: p.ThreadID, TurnID: p.TurnID, Item: p.Item}
			} else {
				delete(s.pendingFileChanges, key)
			}
		}
		return s.takeItem(p.ThreadID, p.Item, e.Method == "item/completed")
	case "item/fileChange/patchUpdated":
		var update struct {
			ThreadID string             `json:"threadId"`
			TurnID   string             `json:"turnId"`
			ItemID   string             `json:"itemId"`
			Changes  []fileUpdateChange `json:"changes"`
		}
		if json.Unmarshal(e.Params, &update) != nil {
			return ErrProtocol
		}
		if update.ThreadID != s.h.ThreadID {
			// Child-thread patch telemetry is retained in the raw event journal,
			// but is not evidence for root-thread approval and cannot fail the
			// parent invocation merely by being observed.
			return nil
		}
		if update.TurnID == "" || update.TurnID != s.h.TurnID || update.ItemID == "" || len(update.Changes) == 0 || len(update.Changes) > 128 {
			return ErrProtocol
		}
		key := fileChangeKey(update.ThreadID, update.TurnID, update.ItemID)
		if s.pendingFileChanges == nil {
			s.pendingFileChanges = make(map[string]trackedFileChange)
		}
		if _, exists := s.pendingFileChanges[key]; !exists {
			return ErrProtocol
		}
		s.pendingFileChanges[key] = trackedFileChange{ThreadID: update.ThreadID, TurnID: update.TurnID, Item: item{ID: update.ItemID, Type: "fileChange", Status: "inProgress", Changes: update.Changes}}
	case "thread/tokenUsage/updated":
		if p.ThreadID != s.h.ThreadID || (s.h.TurnID != "" && p.TurnID != s.h.TurnID) {
			return nil
		}
		v := p.TokenUsage.Total
		for _, n := range []*int64{v.InputTokens, v.OutputTokens, v.CachedInputTokens} {
			if n != nil && *n < 0 {
				return ErrProtocol
			}
		}
		s.usage = &agentexec.Usage{Source: "codex-app-server/thread-total", InputTokens: v.InputTokens, OutputTokens: v.OutputTokens, CachedTokens: v.CachedInputTokens}
	case "thread/settings/updated":
		if p.ThreadID != s.h.ThreadID {
			return nil
		}
		v := p.ThreadSettings
		if v.Model != s.a.config.Model || v.Effort != s.a.config.ReasoningEffort || filepath.Clean(v.CWD) != filepath.Clean(s.h.Workspace.CWD) {
			return errors.New("effective thread settings changed")
		}
		if s.life.Effective != nil && v.ActivePermissionProfile != nil {
			s.life.Effective.PermissionProfile = v.ActivePermissionProfile.ID
		}
		if s.a.config.PermissionProfile != "" && (v.ActivePermissionProfile == nil || v.ActivePermissionProfile.ID != s.a.config.PermissionProfile) {
			return errors.New("effective permission profile mismatch")
		}
	}
	return nil
}

func (s *session) takeItem(sender string, v item, complete bool) error {
	if v.ID == "" || v.Type == "" {
		return ErrProtocol
	}
	if v.Type == "agentMessage" && sender == s.h.ThreadID && complete && (v.Phase == "final_answer" || v.Phase == "") {
		if len(v.Text) > s.outputLimit {
			return agentexec.ErrOutputTooLarge
		}
		s.output[v.ID] = v.Text
		// final_answer is authoritative; commentary is never a Response.
		s.final = v.Text
	}
	if v.Type != "collabAgentToolCall" {
		return nil
	}
	if v.Tool == "spawnAgent" || v.Tool == "resumeAgent" {
		key := sender + "/" + v.ID
		index, exists := s.spawns[key]
		if !exists {
			index = len(s.life.StartRequests)
			s.spawns[key] = index
			s.life.StartRequests = append(s.life.StartRequests, agentexec.RoleStartRequest{RequestID: v.ID, ParentSessionID: s.sessions[sender], Role: "helper", Model: v.Model, ReasoningEffort: v.ReasoningEffort, State: "requested"})
		}
		r := &s.life.StartRequests[index]
		if v.Status == "failed" {
			r.State = "failed"
		} else if v.Status == "interrupted" {
			r.State = "interrupted"
		}
		if len(v.ReceiverThreadIDs) > 1 {
			return fmt.Errorf("%w: helper item has multiple receiving threads", ErrProtocol)
		}
		if len(v.ReceiverThreadIDs) == 1 {
			child := v.ReceiverThreadIDs[0]
			s.receivers[child] = index
			r.SessionID = s.sessions[child] // receiverThreadIds alone does not establish session identity
			r.State = "started"
			depth := s.children[sender] + 1
			s.children[child] = depth
			if depth > s.a.config.Helpers.MaxDepth {
				return errors.New("observed native helper depth exceeded bound")
			}
		}
		if !s.a.config.Helpers.Enabled || len(s.spawns) > s.a.config.Helpers.MaxStartRequests {
			return errors.New("observed native helper start requests exceeded bound; pre-dispatch accounting unavailable")
		}
	}
	for child, state := range v.AgentsStates {
		if i, ok := s.receivers[child]; ok {
			r := &s.life.StartRequests[i]
			switch state.Status {
			case "completed", "errored", "shutdown", "notFound", "interrupted":
				r.State = map[string]string{"completed": "completed", "errored": "failed", "shutdown": "interrupted", "notFound": "unknown", "interrupted": "interrupted"}[state.Status]
			}
		}
	}
	return nil
}

func (s *session) finish(t turn) error {
	if t.ID != s.h.TurnID {
		return ErrProtocol
	}
	switch t.Status {
	case "completed", "failed", "interrupted":
	default:
		return ErrProtocol
	}
	s.terminal = true
	s.life.State = t.Status
	if len(s.life.StartRequests) > 0 {
		s.life.StartRequests[0].State = t.Status
	}
	if t.Status == "completed" && len(t.Error) > 0 && string(t.Error) != "null" {
		return ErrProtocol
	}
	return nil
}
