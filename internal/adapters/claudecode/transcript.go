package claudecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// redactor removes the runtime token, in its literal and its JSON-escaped
// spelling, from everything that leaves the adapter.
type redactor struct{ token string }

func (r redactor) bytes(data []byte) []byte {
	data = bytes.Clone(data)
	if r.token == "" {
		return data
	}
	data = bytes.ReplaceAll(data, []byte(r.token), []byte("[REDACTED]"))
	escaped, _ := json.Marshal(r.token)
	return bytes.ReplaceAll(data, escaped[1:len(escaped)-1], []byte("[REDACTED]"))
}

func (r redactor) text(s string) string { return string(r.bytes([]byte(s))) }

// err keeps the error chain for classification and redacts its text.
func (r redactor) err(err error) error {
	if err == nil || r.text(err.Error()) == err.Error() {
		return err
	}
	return redactedError{text: r.text(err.Error()), cause: err}
}

type redactedError struct {
	text  string
	cause error
}

func (e redactedError) Error() string { return e.text }
func (e redactedError) Unwrap() error { return e.cause }

// transcript writes the provider stream and the Host tool exchange as JSON Lines.
// A nil transcript discards. It is used from one goroutine only.
type transcript struct {
	file *os.File
	r    redactor
}

func openTranscript(path string, r redactor) (*transcript, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%w: transcript path must be absolute", ErrSession)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return &transcript{file: file, r: r}, nil
}

// line records one entry. Provider lines that are valid JSON are embedded as
// JSON, anything else as a string.
func (t *transcript) line(kind string, data []byte) {
	if t == nil {
		return
	}
	data = bytes.TrimRight(t.r.bytes(data), "\r\n")
	var body json.RawMessage
	if json.Valid(data) {
		body = data
	} else {
		body, _ = json.Marshal(string(data))
	}
	entry, err := json.Marshal(struct {
		At   time.Time       `json:"at"`
		Kind string          `json:"kind"`
		Data json.RawMessage `json:"data"`
	}{time.Now().UTC(), kind, body})
	if err == nil {
		_, _ = t.file.Write(append(t.r.bytes(entry), '\n'))
	}
}

func (t *transcript) value(kind string, v any) {
	if t == nil {
		return
	}
	data, _ := json.Marshal(v)
	t.line(kind, data)
}

func (t *transcript) close() error {
	if t == nil {
		return nil
	}
	return errors.Join(t.file.Sync(), t.file.Close())
}
