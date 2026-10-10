package codexappserver

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
	"unicode/utf8"
)

var (
	ErrProtocol         = errors.New("invalid Codex App Server protocol")
	ErrUncertain        = errors.New("Codex App Server execution is uncertain; inspect the owned handle before further work")
	ErrEventLimit       = errors.New("Codex App Server event limit exceeded")
	ErrApprovalRequired = errors.New("Codex App Server requires a user decision")
)

// Event retains exact bounded wire bytes. It is operational evidence, not model
// output. Consumers must persist it privately: events can contain project data.
type Event struct {
	Method string
	Params json.RawMessage
	Wire   json.RawMessage
}

type envelope struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type incoming struct {
	wire []byte
	err  error
}

// Client owns one connection. A Session serializes calls; separate independent
// Managers use separate clients. Close releases blocked reads and writes.
type Client struct {
	conn         io.ReadWriteCloser
	in           chan incoming
	done         chan struct{}
	once         sync.Once
	writeMu      sync.Mutex
	mu           sync.Mutex
	log          hash.Hash
	used         int64
	limit        int64
	nextID       int64
	observe      func(Event) error
	request      func(context.Context, envelope) error
	experimental bool
	readDone     chan struct{}
}

func NewClient(conn io.ReadWriteCloser, maxEventBytes int64, observe func(Event) error) (*Client, error) {
	if conn == nil || maxEventBytes <= 0 || maxEventBytes > 256<<20 {
		return nil, errors.New("connection and bounded event budget required")
	}
	c := &Client{conn: conn, in: make(chan incoming, 16), done: make(chan struct{}), readDone: make(chan struct{}), log: sha256.New(), limit: maxEventBytes, observe: observe}
	go c.read()
	return c, nil
}

func (c *Client) Close() error {
	var err error
	c.once.Do(func() { close(c.done); err = c.conn.Close(); <-c.readDone })
	return err
}

func (c *Client) EventDigest() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return "sha256:" + hex.EncodeToString(c.log.Sum(nil))
}

func (c *Client) read() {
	defer close(c.readDone)
	defer close(c.in)
	r := bufio.NewReaderSize(c.conn, 64<<10)
	for {
		var line []byte
		for {
			part, err := r.ReadSlice('\n')
			c.mu.Lock()
			if int64(len(part)) > c.limit-c.used {
				c.mu.Unlock()
				c.deliver(incoming{err: ErrEventLimit})
				return
			}
			c.used += int64(len(part))
			_, _ = c.log.Write(part)
			c.mu.Unlock()
			line = append(line, part...)
			if errors.Is(err, bufio.ErrBufferFull) {
				continue
			}
			if err != nil {
				c.deliver(incoming{err: err})
				return
			}
			if len(line) < 2 || !utf8.Valid(line) || !json.Valid(line) {
				c.deliver(incoming{err: ErrProtocol})
				return
			}
			if !c.deliver(incoming{wire: line}) {
				return
			}
			break
		}
	}
}
func (c *Client) deliver(v incoming) bool {
	select {
	case c.in <- v:
		return true
	case <-c.done:
		return false
	}
}

func (c *Client) send(ctx context.Context, message any) error {
	wire, err := json.Marshal(message)
	if err != nil {
		return err
	}
	wire = append(wire, '\n')
	var request envelope
	isRequest := json.Unmarshal(wire, &request) == nil && len(request.ID) > 0 && request.Method != "" && len(request.Params) > 0
	if isRequest {
		c.mu.Lock()
		if int64(len(wire)) > c.limit-c.used {
			c.mu.Unlock()
			return ErrEventLimit
		}
		// Reserve before writing so concurrent inbound traffic cannot consume
		// the evidence budget for a request that is about to be dispatched.
		c.used += int64(len(wire))
		c.mu.Unlock()
	}
	finished := make(chan error, 1)
	go func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		n, e := c.conn.Write(wire)
		if e == nil && n != len(wire) {
			e = io.ErrShortWrite
		}
		finished <- e
	}()
	select {
	case err = <-finished:
		if observeErr := c.recordOutgoingRequest(wire, request, isRequest, err); observeErr != nil {
			return observeErr
		}
		return err
	case <-ctx.Done():
		_ = c.Close()
		writeErr := <-finished
		if recordErr := c.recordOutgoingRequest(wire, request, isRequest, writeErr); recordErr != nil {
			return errors.Join(ctx.Err(), recordErr)
		}
		return ctx.Err()
	case <-c.done:
		writeErr := <-finished
		if recordErr := c.recordOutgoingRequest(wire, request, isRequest, writeErr); recordErr != nil {
			return errors.Join(io.ErrClosedPipe, recordErr)
		}
		return io.ErrClosedPipe
	}
}

func (c *Client) recordOutgoingRequest(wire []byte, request envelope, isRequest bool, writeErr error) error {
	if !isRequest {
		return writeErr
	}
	if writeErr != nil {
		c.mu.Lock()
		c.used -= int64(len(wire))
		c.mu.Unlock()
		return writeErr
	}
	c.mu.Lock()
	_, _ = c.log.Write(wire)
	c.mu.Unlock()
	if c.observe == nil {
		return nil
	}
	event := Event{Method: "rpc/request", Params: append(json.RawMessage(nil), request.Params...), Wire: append(json.RawMessage(nil), wire...)}
	return c.observe(event)
}

func (c *Client) next(ctx context.Context) (envelope, error) {
	select {
	case <-ctx.Done():
		return envelope{}, ctx.Err()
	case entry, ok := <-c.in:
		if !ok {
			return envelope{}, io.EOF
		}
		if entry.err != nil {
			return envelope{}, entry.err
		}
		var msg envelope
		if json.Unmarshal(entry.wire, &msg) != nil {
			return envelope{}, ErrProtocol
		}
		if msg.Method != "" {
			if len(msg.Params) == 0 {
				return msg, ErrProtocol
			}
			if c.observe != nil {
				if err := c.observe(Event{msg.Method, msg.Params, entry.wire}); err != nil {
					return msg, err
				}
			}
			if len(msg.ID) > 0 {
				if c.request != nil && (msg.Method == "item/tool/call" || msg.Method == "item/fileChange/requestApproval") {
					return msg, c.request(ctx, msg)
				}
				// Fail closed. Never synthesize approval, input, auth tokens or tool results.
				_ = c.send(ctx, map[string]any{"id": msg.ID, "error": map[string]any{"code": -32000, "message": "Markitect requires an explicit user decision for this server request"}})
				return msg, ErrApprovalRequired
			}
		} else {
			if len(msg.ID) == 0 || (len(msg.Result) == 0) == (len(msg.Error) == 0) {
				return msg, ErrProtocol
			}
			if c.observe != nil {
				params := msg.Result
				if len(params) == 0 {
					params = msg.Error
				}
				if err := c.observe(Event{"rpc/response", params, entry.wire}); err != nil {
					return msg, err
				}
			}
		}
		return msg, nil
	case <-c.done:
		return envelope{}, io.ErrClosedPipe
	}
}

func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	c.nextID++
	id := c.nextID
	if err := c.send(ctx, map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	for {
		msg, err := c.next(ctx)
		if err != nil {
			return err
		}
		if msg.Method != "" {
			continue
		}
		var received int64
		if json.Unmarshal(msg.ID, &received) != nil || received != id {
			return ErrProtocol
		}
		if len(msg.Error) > 0 {
			return fmt.Errorf("App Server %s failed: %s", method, msg.Error)
		}
		if out != nil && json.Unmarshal(msg.Result, out) != nil {
			return ErrProtocol
		}
		return nil
	}
}

func (c *Client) initialize(ctx context.Context) error {
	var response struct {
		UserAgent string `json:"userAgent"`
	}
	params := map[string]any{"clientInfo": map[string]string{"name": "markitect", "title": "Markitect", "version": "0.14.1"}}
	if c.experimental {
		params["capabilities"] = map[string]bool{"experimentalApi": true}
	}
	if err := c.call(ctx, "initialize", params, &response); err != nil {
		return err
	}
	if response.UserAgent == "" {
		return ErrProtocol
	}
	return c.send(ctx, map[string]any{"method": "initialized", "params": map[string]any{}})
}
