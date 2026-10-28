// Package protocol implements the JSON Lines wire format
// "markitect.library-jsonl/v1" on top of the library domain.
package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"example.com/smalllibrary/internal/library"
)

// Response statuses.
const (
	statusOK             = "ok"
	statusRefused        = "refused"
	statusInvalidRequest = "invalidRequest"
)

type response struct {
	ID     any    `json:"id"`
	Status string `json:"status"`
	Result any    `json:"result,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Server answers requests against one library.
type Server struct {
	lib *library.Library
}

// NewServer returns a server holding a new, empty library.
func NewServer() *Server {
	return &Server{lib: library.New()}
}

// Serve reads one request per line from in and writes one response per
// request to out, flushing each response before reading the next line. It
// returns nil when in is exhausted.
func Serve(in io.Reader, out io.Writer) error {
	server := NewServer()
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			reply, err := server.Handle(bytes.TrimRight(line, "\r\n"))
			if err != nil {
				return err
			}
			if _, err := writer.Write(reply); err != nil {
				return err
			}
			if err := writer.Flush(); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// Handle answers one request line (without its line terminator) and returns
// the response line including the final newline. The error is non-nil only
// if the response cannot be encoded.
func (s *Server) Handle(line []byte) ([]byte, error) {
	return encode(s.respond(line))
}

func (s *Server) respond(line []byte) response {
	req, id, err := parseRequest(line)
	if err != nil {
		return response{ID: id, Status: statusInvalidRequest, Reason: err.Error()}
	}
	op, known := operations[req.op]
	if !known {
		return response{ID: id, Status: statusInvalidRequest, Reason: fmt.Sprintf("unknown operation %q", req.op)}
	}
	for name := range req.args {
		if !op.accepts(name) {
			return response{ID: id, Status: statusInvalidRequest, Reason: fmt.Sprintf("unknown argument %q for operation %q", name, req.op)}
		}
	}
	args, err := op.requiredArguments(req.args)
	if err != nil {
		return response{ID: id, Status: statusRefused, Reason: err.Error()}
	}
	result, err := op.run(s.lib, args)
	if err != nil {
		return response{ID: id, Status: statusRefused, Reason: err.Error()}
	}
	return response{ID: id, Status: statusOK, Result: result}
}

func (op operation) accepts(name string) bool {
	for _, field := range op.fields {
		if field == name {
			return true
		}
	}
	return false
}

// requiredArguments checks that every required argument is present and a
// JSON string. Null and values of any other type are refused.
func (op operation) requiredArguments(raw map[string]json.RawMessage) (arguments, error) {
	args := make(arguments, len(op.fields))
	for _, field := range op.fields {
		value, present := raw[field]
		if !present {
			return nil, fmt.Errorf("required argument %q is missing", field)
		}
		text, ok := decodeString(value)
		if !ok {
			return nil, fmt.Errorf("argument %q must be a string", field)
		}
		args[field] = text
	}
	return args, nil
}

func encode(r response) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(r); err != nil { // Encode ends the value with a newline
		return nil, err
	}
	return buffer.Bytes(), nil
}
