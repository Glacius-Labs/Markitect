package ms

import (
	"bytes"
	"encoding/json"
)

// Server holds one makerspace and answers request lines.
type Server struct {
	st *state
}

// NewServer returns a server in the initial state.
func NewServer() *Server { return &Server{st: newState()} }

type response struct {
	id     any // string or nil
	status string
	result rec
	code   string
	reason string
}

func (r response) marshal() []byte {
	var m map[string]any
	switch r.status {
	case "ok":
		m = map[string]any{"id": r.id, "status": "ok", "result": r.result}
	case "refused":
		m = map[string]any{"id": r.id, "status": "refused", "code": r.code, "reason": r.reason}
	default:
		m = map[string]any{"id": r.id, "status": "invalidRequest", "reason": r.reason}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(m)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

func invalid(id any, reason string) response {
	return response{id: id, status: "invalidRequest", reason: reason}
}

// Handle answers one request line (with or without its line feed) with one
// response line (without a line feed).
func (sv *Server) Handle(line []byte) []byte {
	return sv.handle(line).marshal()
}

func (sv *Server) handle(line []byte) response {
	line = bytes.TrimRight(line, "\r\n")
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return invalid(nil, "The line is empty.")
	}
	if !json.Valid(line) {
		return invalid(nil, "The line is not valid JSON.")
	}
	var top map[string]json.RawMessage
	if trimmed[0] != '{' || json.Unmarshal(line, &top) != nil {
		return invalid(nil, "The line is not a JSON object.")
	}
	var id any
	if raw, ok := top["id"]; ok && len(raw) > 0 && raw[0] == '"' {
		var text string
		if json.Unmarshal(raw, &text) == nil && text != "" {
			id = text
		}
	}
	if id == nil {
		return invalid(nil, "The request needs a nonempty string id.")
	}
	rawOp, ok := top["op"]
	var opName string
	if !ok || len(rawOp) == 0 || rawOp[0] != '"' || json.Unmarshal(rawOp, &opName) != nil {
		return invalid(id, "The request needs a string op.")
	}
	rawArgs, ok := top["args"]
	if !ok || len(rawArgs) == 0 || rawArgs[0] != '{' {
		return invalid(id, "The request needs an args object.")
	}
	for k := range top {
		if k != "id" && k != "op" && k != "args" {
			return invalid(id, "The request has an unknown member "+k+".")
		}
	}
	op, ok := lookupOp(opName)
	if !ok {
		return invalid(id, "Unknown operation "+opName+".")
	}
	var args map[string]json.RawMessage
	if json.Unmarshal(rawArgs, &args) != nil {
		return invalid(id, "The args are not a JSON object.")
	}
	known := map[string]bool{"actor": true, "at": true}
	for _, a := range op.args {
		known[a.name] = true
	}
	for k := range args {
		if !known[k] {
			return invalid(id, "Unknown argument "+k+" for "+opName+".")
		}
	}
	r := sv.st.execute(op, args)
	r.id = id
	return r
}
