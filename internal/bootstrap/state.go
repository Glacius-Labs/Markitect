package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

// The state directory holds everything one operator-visible Apply leaves on the
// Host: the lock, the evidence of every run, and the owner channel files.
//
//	<state>/apply.lock
//	<state>/runs/<runID>/events.jsonl, transcripts/<turnID>.jsonl, report.json
//	<state>/questions/<questionID>.json
//	<state>/answers/<questionID>.txt
const (
	lockFileName   = "apply.lock"
	runsDirName    = "runs"
	questionsDir   = "questions"
	answersDir     = "answers"
	maxAnswerBytes = 16 << 10
)

// stateName admits identifiers that are safe as one path element on every
// supported Host. Run, turn and question identities come from the engine and
// the operator; none of them may steer a path.
var stateName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

var errExists = errors.New("file already exists")

// writeStateFile writes data atomically: a reader sees the whole file or none.
// With replace the file is replaced; otherwise an existing file is kept and
// errExists is returned. Data is synced before it becomes visible.
func writeStateFile(path string, data []byte, replace bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if replace {
		return os.Rename(tmp.Name(), path)
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errExists
		}
		return err
	}
	return nil
}
