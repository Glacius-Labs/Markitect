package claudecode

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// transcript appends one JSON line per observed exchange of a turn: provider
// output, Orchestrator writes and Orchestrator tool calls and answers, with known secrets
// redacted. A nil transcript records nothing.
type transcript struct {
	file   *os.File
	redact func([]byte) []byte
	err    error
}

func openTranscript(path string, redact func([]byte) []byte) (*transcript, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		return nil, ErrSession
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.Join(ErrSession, err)
	}
	return &transcript{file: f, redact: redact}, nil
}

func (w *transcript) record(direction string, data []byte) {
	if w == nil || w.err != nil {
		return
	}
	line, err := json.Marshal(struct {
		At        string `json:"at"`
		Direction string `json:"direction"`
		Data      string `json:"data"`
	}{time.Now().UTC().Format(time.RFC3339Nano), direction, string(w.redact(data))})
	if err == nil {
		_, err = w.file.Write(append(line, '\n'))
	}
	w.err = err
}

// close reports the first recording failure: a transcript that lost lines is
// not the evidence the caller asked for.
func (w *transcript) close() error {
	if w == nil {
		return nil
	}
	return errors.Join(w.err, w.file.Close())
}
