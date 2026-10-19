package run

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// turnState is what the handlers of one turn share with the engine. Its
// fields are guarded by engine.mu.
type turnState struct {
	m        *mgr
	kind     TurnKind
	id       string
	round    int
	status   string
	finished bool
}

// runTurn runs one Agent turn through the Agents port and records it. A turn
// whose container end was not observed stops the run: quiescence is then
// uncertain.
func (e *engine) runTurn(ts *turnState, turn Turn, handler ToolHandler) (TurnResult, error) {
	turn.ID, turn.Manager, turn.Kind, turn.Timeout = ts.id, ts.m.Path, ts.kind, e.cfg.Budget.TurnTimeout
	turn.Transcript = e.cfg.Evidence.Transcript(ts.id)
	started := e.now()
	_ = e.record(Event{Kind: "turn.start", Manager: ts.m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"kind": ts.kind, "profile": turn.Profile, "workspace": turn.Workspace, "readOnly": turn.ReadOnly, "home": turn.Home, "resume": turn.Resume}})
	if e.ctx.Err() != nil {
		e.mu.Lock()
		e.turns = append(e.turns, TurnRecord{ID: ts.id, Manager: ts.m.Path, Kind: ts.kind, Round: ts.round, Started: started, Ended: e.now(), Stopped: true, Reason: "the run stopped before the turn started"})
		e.mu.Unlock()
		return TurnResult{Stopped: true}, e.ctx.Err()
	}
	res, err := e.cfg.Agents.Run(e.ctx, turn, handler)
	e.mu.Lock()
	rec := TurnRecord{ID: ts.id, Manager: ts.m.Path, Kind: ts.kind, Round: ts.round, Started: res.Started, Ended: res.Ended, Usage: res.Usage, Stopped: res.Stopped, Status: ts.status, Reason: res.Reason}
	if rec.Started.IsZero() {
		rec.Started = started
	}
	if rec.Ended.IsZero() {
		rec.Ended = e.now()
	}
	if err != nil {
		rec.Reason = strings.TrimSpace(rec.Reason + " " + err.Error())
	}
	e.turns = append(e.turns, rec)
	e.usage = e.usage.Add(res.Usage)
	data := map[string]any{"kind": rec.Kind, "usage": rec.Usage, "stopped": rec.Stopped, "status": rec.Status, "reason": rec.Reason, "conversation": res.Conversation}
	_ = e.record(Event{Kind: "turn.end", Manager: ts.m.Path, Turn: ts.id, Round: ts.round, Data: data})
	e.mu.Unlock()
	if !res.Stopped {
		e.halt(fmt.Sprintf("turn %s ended without an observed container stop; quiescence is uncertain", ts.id))
	}
	return res, err
}

// decodeStrict reads one tool call's arguments and refuses unknown fields and
// trailing data.
func decodeStrict(raw json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("the arguments are not valid: %v", err)
	}
	if dec.More() {
		return fmt.Errorf("the arguments have trailing data")
	}
	return nil
}

func refuse(format string, args ...any) ToolAnswer {
	return ToolAnswer{Text: fmt.Sprintf(format, args...), Refused: true}
}

// toolText checks one free-text argument.
func toolText(name, s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "" || !utf8.ValidString(s) || strings.ContainsRune(s, 0):
		return "", fmt.Errorf("%s must be non-empty text", name)
	case len(s) > maxToolText:
		return "", fmt.Errorf("%s is longer than %d bytes", name, maxToolText)
	}
	return s, nil
}

// tail returns the last n bytes of output as valid text.
func tail(b []byte, n int) string {
	cut := false
	if len(b) > n {
		b, cut = b[len(b)-n:], true
		for len(b) > 0 && !utf8.RuneStart(b[0]) {
			b = b[1:]
		}
	}
	s := strings.ToValidUTF8(string(b), "�")
	if cut {
		s = "[output cut]\n" + s
	}
	return s
}
