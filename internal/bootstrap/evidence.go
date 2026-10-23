package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

// fileEvidence keeps one run's evidence under <state>/runs/<runID>. Every event
// is appended to events.jsonl and synced before Record returns, so evidence
// that was reported recorded survives a crash. Provider streams go to
// transcripts/<turnID>.jsonl, written by the Agents adapter. The final report
// is written once at the end.
type fileEvidence struct {
	dir string

	mu     sync.Mutex
	events *os.File
}

var _ runport.Evidence = (*fileEvidence)(nil)

// newFileEvidence creates the run's evidence directory. A run identity that
// already has one is refused: evidence is never overwritten.
func newFileEvidence(stateDirectory, runID string) (*fileEvidence, error) {
	if !stateName.MatchString(runID) {
		return nil, fmt.Errorf("run identity %q is not a safe path element", runID)
	}
	runs := filepath.Join(stateDirectory, runsDirName)
	if err := os.MkdirAll(runs, 0o700); err != nil {
		return nil, err
	}
	dir := filepath.Join(runs, runID)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(dir, "transcripts"), 0o700); err != nil {
		return nil, err
	}
	events, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &fileEvidence{dir: dir, events: events}, nil
}

// Dir is the run's evidence directory.
func (e *fileEvidence) Dir() string { return e.dir }

func (e *fileEvidence) Record(event runport.Event) error {
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.events == nil {
		return errors.New("evidence is closed")
	}
	if _, err := e.events.Write(line); err != nil {
		return err
	}
	return e.events.Sync()
}

func (e *fileEvidence) Transcript(turn string) string {
	if !stateName.MatchString(turn) {
		return ""
	}
	return filepath.Join(e.dir, "transcripts", turn+".jsonl")
}

// WriteReport writes report.json atomically. It is the run's last evidence.
func (e *fileEvidence) WriteReport(report any) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return writeStateFile(filepath.Join(e.dir, "report.json"), append(data, '\n'), true)
}

// Close ends the event log. Later events are refused.
func (e *fileEvidence) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.events == nil {
		return nil
	}
	err := e.events.Close()
	e.events = nil
	return err
}
