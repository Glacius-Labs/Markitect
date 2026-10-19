package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

// fileOwner is the human's channel of one Apply. Ask publishes Root's question
// as <state>/questions/<id>.json and prints one line to stderr; the human
// answers with `markitect answer`, which writes <state>/answers/<id>.txt, and
// Answer polls that file. Questions and answers of earlier Applies are removed
// when the channel opens, so a question file exists only while its Apply asked
// it and an answer can only reach the Apply that is waiting for it.
type fileOwner struct {
	state, runID, config string
	stderr               io.Writer
}

var _ runport.Owner = (*fileOwner)(nil)

func newFileOwner(stateDirectory, runID, configPath string, stderr io.Writer) (*fileOwner, error) {
	for _, name := range []string{questionsDir, answersDir} {
		dir := filepath.Join(stateDirectory, name)
		if err := os.RemoveAll(dir); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}
	return &fileOwner{state: stateDirectory, runID: runID, config: configPath, stderr: stderr}, nil
}

func (o *fileOwner) Ask(_ context.Context, q runport.Question) error {
	if !stateName.MatchString(q.ID) {
		return fmt.Errorf("question identity %q is not a safe path element", q.ID)
	}
	data, err := json.MarshalIndent(map[string]any{"id": q.ID, "run": o.runID, "text": q.Text, "asked": q.Asked}, "", "  ")
	if err != nil {
		return err
	}
	if err := writeStateFile(filepath.Join(o.state, questionsDir, q.ID+".json"), append(data, '\n'), false); err != nil {
		return err
	}
	fmt.Fprintf(o.stderr, "markitect: question %s: %s -- answer with: markitect answer %s --text ANSWER --run-config %s\n", q.ID, strings.Join(strings.Fields(q.Text), " "), q.ID, o.config)
	return nil
}

func (o *fileOwner) Answer(_ context.Context, id string) (string, bool, error) {
	if !stateName.MatchString(id) {
		return "", false, fmt.Errorf("question identity %q is not a safe path element", id)
	}
	data, err := os.ReadFile(filepath.Join(o.state, answersDir, id+".txt"))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if len(data) > maxAnswerBytes || !utf8.Valid(data) {
		return "", false, fmt.Errorf("answer to question %s is not UTF-8 text of at most %d bytes", id, maxAnswerBytes)
	}
	return string(data), true, nil
}

// writeAnswer records the owner's answer to a question of the Apply that holds
// the state directory. An answer is final: it is written once, atomically.
func writeAnswer(stateDirectory, id, text string) (string, error) {
	if !stateName.MatchString(id) {
		return "", fmt.Errorf("question identity %q is not a safe path element", id)
	}
	if strings.TrimSpace(text) == "" || len(text) > maxAnswerBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return "", fmt.Errorf("an answer is non-blank UTF-8 text of at most %d bytes", maxAnswerBytes)
	}
	lock, err := lockState(stateDirectory)
	if err == nil {
		lock.Release()
		return "", errors.New("no Apply is running on this state directory")
	}
	if !errors.Is(err, errStateLocked) {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(stateDirectory, questionsDir, id+".json")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("the running Apply has not asked question %s", id)
		}
		return "", err
	}
	path := filepath.Join(stateDirectory, answersDir, id+".txt")
	if err := writeStateFile(path, []byte(text), false); err != nil {
		if errors.Is(err, errExists) {
			return "", fmt.Errorf("question %s is already answered", id)
		}
		return "", err
	}
	return path, nil
}
