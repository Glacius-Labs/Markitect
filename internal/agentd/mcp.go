package agentd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// mcpProtocolVersion is answered when a client names no version.
const mcpProtocolVersion = "2025-06-18"

// maxMCPMessage bounds one newline-delimited MCP message from a bridge.
const maxMCPMessage = frame.MaxPayload

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func (s *server) acceptBridges(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go s.serveBridge(conn)
	}
}

// serveBridge speaks MCP to one CLI-started bridge. Discovery is answered
// locally from the Host's catalogue; every tools/call goes to the Host.
func (s *server) serveBridge(conn net.Conn) {
	done := make(chan struct{})
	var once sync.Once
	closeConn := func() { once.Do(func() { close(done); _ = conn.Close() }) }
	var wmu sync.Mutex
	reply := func(r rpcResponse) {
		r.JSONRPC = "2.0"
		b, err := json.Marshal(r)
		if err != nil {
			return
		}
		wmu.Lock()
		defer wmu.Unlock()
		if _, err := conn.Write(append(b, '\n')); err != nil {
			closeConn()
		}
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64<<10), maxMCPMessage)
	var calls sync.WaitGroup
	defer func() {
		// The bridge left: release its waiting calls before waiting for them.
		closeConn()
		calls.Wait()
	}()
	for scanner.Scan() {
		var m rpcMessage
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			reply(rpcResponse{ID: json.RawMessage("null"), Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if m.Method == "" || len(m.ID) == 0 {
			continue // responses and notifications need no answer
		}
		switch m.Method {
		case "initialize":
			reply(rpcResponse{ID: m.ID, Result: initializeResult(m.Params)})
		case "ping":
			reply(rpcResponse{ID: m.ID, Result: json.RawMessage("{}")})
		case "tools/list":
			list, _ := s.catalogue()
			result, _ := json.Marshal(struct {
				Tools []frame.ToolSpec `json:"tools"`
			}{Tools: nonNil(list)})
			reply(rpcResponse{ID: m.ID, Result: result})
		case "tools/call":
			calls.Add(1)
			go func(m rpcMessage) {
				defer calls.Done()
				reply(s.toolCall(done, m))
			}(m)
		default:
			reply(rpcResponse{ID: m.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
		}
	}
}

func (s *server) toolCall(done <-chan struct{}, m rpcMessage) rpcResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Meta      json.RawMessage `json:"_meta"`
	}
	if err := json.Unmarshal(m.Params, &params); err != nil || params.Name == "" {
		return rpcResponse{ID: m.ID, Error: &rpcError{Code: -32602, Message: "tools/call needs a tool name"}}
	}
	if _, known := s.lookup(params.Name); !known {
		return rpcResponse{ID: m.ID, Error: &rpcError{Code: -32602, Message: "unknown tool " + params.Name}}
	}
	if len(params.Arguments) == 0 || string(params.Arguments) == "null" {
		params.Arguments = json.RawMessage("{}")
	}
	r, err := s.call(done, params.Name, params.Arguments, params.Meta)
	if err != nil {
		return rpcResponse{ID: m.ID, Error: &rpcError{Code: -32603, Message: err.Error()}}
	}
	if r.Error != nil {
		return rpcResponse{ID: m.ID, Error: &rpcError{Code: r.Error.Code, Message: r.Error.Message}}
	}
	return rpcResponse{ID: m.ID, Result: r.Result}
}

func (s *server) lookup(name string) (frame.ToolSpec, bool) {
	_, tools := s.catalogue()
	t, ok := tools[name]
	return t, ok
}

func initializeResult(params json.RawMessage) json.RawMessage {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := p.ProtocolVersion
	if version == "" {
		version = mcpProtocolVersion
	}
	b, _ := json.Marshal(map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]any{"name": "markitect-host", "version": "1"},
	})
	return b
}

func nonNil(t []frame.ToolSpec) []frame.ToolSpec {
	if t == nil {
		return []frame.ToolSpec{}
	}
	return t
}

// Bridge connects a CLI's stdio MCP client to the agentd serving this
// session. The CLI starts `markitect-agentd mcp` as its MCP server; Bridge
// copies bytes in both directions until either side closes.
func Bridge(ctx context.Context, socketPath string, in io.Reader, out io.Writer) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, in)
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		errc <- err
	}()
	go func() {
		_, err := io.Copy(out, conn)
		errc <- err
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
		// The CLI closed its side or agentd ended the session; finish the
		// remaining direction without waiting for the other peer forever.
		return nil
	}
}
