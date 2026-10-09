package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

const ProtocolVersion = "2025-11-25"
const MaxMessageBytes = 4 << 20

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type initializeParams struct {
	ProtocolVersion string                     `json:"protocolVersion"`
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}
type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Meta      json.RawMessage `json:"_meta,omitempty"`
}

// Serve processes newline-delimited local stdio. Output contains protocol JSON
// only. The composition root closes input on external shutdown; disconnect
// cancels active calls and waits for their Host contexts to stop. Synchronous
// Host operations retain their existing cancellation limitations.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), MaxMessageBytes)
	var writeMu, activeMu sync.Mutex
	var wg sync.WaitGroup
	active := map[string]context.CancelFunc{}
	var writeErr error
	emit := func(r response) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if writeErr != nil {
			return
		}
		writeErr = json.NewEncoder(out).Encode(r)
		if writeErr != nil {
			cancel()
		}
	}
	fail := func(id json.RawMessage, code int, msg string) {
		if len(id) == 0 {
			id = json.RawMessage("null")
		}
		emit(response{JSONRPC: "2.0", ID: id, Error: &rpcError{code, msg}})
	}
	initialized, ready := false, false
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := append([]byte(nil), scanner.Bytes()...)
		var r request
		// Parse independently from argument validation; no batches are supported by MCP.
		if err := decodeTyped(line, schemaRequest(), &r); err != nil {
			code := -32600
			if !json.Valid(line) {
				code = -32700
			}
			fail(nil, code, "Invalid JSON-RPC message")
			continue
		}
		if r.JSONRPC != "2.0" || r.Method == "" || !validID(r.ID) {
			fail(nil, -32600, "Invalid JSON-RPC request")
			continue
		}
		if len(r.ID) == 0 {
			switch r.Method {
			case "notifications/initialized":
				if initialized {
					ready = true
				}
			case "notifications/cancelled":
				var p struct {
					RequestID json.RawMessage `json:"requestId"`
					Reason    string          `json:"reason,omitempty"`
				}
				if json.Unmarshal(r.Params, &p) == nil {
					activeMu.Lock()
					c := active[string(p.RequestID)]
					activeMu.Unlock()
					if c != nil {
						c()
					}
				}
			}
			continue
		}
		if r.Method == "ping" {
			emit(response{JSONRPC: "2.0", ID: r.ID, Result: struct{}{}})
			continue
		}
		if r.Method == "initialize" {
			if initialized {
				fail(r.ID, -32600, "Already initialized")
				continue
			}
			var p initializeParams
			// Client capability extensions and implementation metadata are extensible MCP objects.
			if json.Unmarshal(r.Params, &p) != nil || p.ProtocolVersion == "" || p.ClientInfo.Name == "" || p.ClientInfo.Version == "" || p.Capabilities == nil {
				fail(r.ID, -32602, "Invalid initialization parameters")
				continue
			}
			version := ProtocolVersion
			switch p.ProtocolVersion {
			case "2024-11-05", "2025-03-26", "2025-06-18", ProtocolVersion:
				version = p.ProtocolVersion
			}
			initialized = true
			emit(response{JSONRPC: "2.0", ID: r.ID, Result: map[string]any{"protocolVersion": version, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "markitect", "version": "source"}, "instructions": "Tools use the explicitly selected repository and existing caller authority. Preserve durable run IDs; protocol request IDs are not product run IDs. Only listed shared operations are available."}})
			continue
		}
		if !ready {
			fail(r.ID, -32600, "Initialize and send notifications/initialized first")
			continue
		}
		switch r.Method {
		case "tools/list":
			var p struct {
				Cursor string          `json:"cursor,omitempty"`
				Meta   json.RawMessage `json:"_meta,omitempty"`
			}
			if err := decodeTyped(r.Params, schemaList(), &p); err != nil || p.Cursor != "" {
				fail(r.ID, -32602, "Invalid list parameters")
				continue
			}
			emit(response{JSONRPC: "2.0", ID: r.ID, Result: map[string]any{"tools": s.Tools()}})
		case "tools/call":
			var p callParams
			if err := decodeTyped(r.Params, schemaCall(), &p); err != nil {
				fail(r.ID, -32602, "Invalid call parameters")
				continue
			}
			if _, ok := s.tools[p.Name]; !ok {
				fail(r.ID, -32602, "Unknown or unavailable tool; inspect tools/list")
				continue
			}
			activeMu.Lock()
			key := string(r.ID)
			_, duplicate := active[key]
			if duplicate || len(active) >= 16 {
				activeMu.Unlock()
				fail(r.ID, -32600, "Request already active or concurrency limit reached")
				continue
			}
			callCtx, callCancel := context.WithCancel(ctx)
			active[key] = callCancel
			activeMu.Unlock()
			wg.Add(1)
			go func(r request, p callParams, key string) {
				defer wg.Done()
				defer callCancel()
				defer func() { activeMu.Lock(); delete(active, key); activeMu.Unlock() }()
				result, err := s.Call(callCtx, p.Name, p.Arguments)
				if err != nil {
					fail(r.ID, -32602, "Arguments do not match closed tool schema")
					return
				}
				emit(response{JSONRPC: "2.0", ID: r.ID, Result: result})
			}(r, p, key)
		default:
			fail(r.ID, -32601, "Method not supported")
		}
	}
	cancel()
	wg.Wait()
	writeMu.Lock()
	defer writeMu.Unlock()
	if writeErr != nil {
		return writeErr
	}
	if err := scanner.Err(); err != nil {
		return errors.New("MCP input exceeds limit or cannot be read")
	}
	return nil
}
func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	var v any
	d := json.Unmarshal(id, &v)
	if d != nil {
		return false
	}
	switch v.(type) {
	case string, float64:
		return true
	}
	return false
}
func schemaRequest() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"jsonrpc": schemaString(), "id": map[string]any{}, "method": schemaString(), "params": map[string]any{}}, "required": []string{"jsonrpc", "method"}, "additionalProperties": false}
}
func schemaString() map[string]any { return map[string]any{"type": "string"} }
func schemaCall() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"name": schemaString(), "arguments": map[string]any{}, "_meta": map[string]any{}}, "required": []string{"name"}, "additionalProperties": false}
}
func schemaList() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"cursor": schemaString(), "_meta": map[string]any{}}, "additionalProperties": false}
}
