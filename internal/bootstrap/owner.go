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

// fileOwner is the channel of one run to the AI that invoked Markitect, which
// can consult the user. Ask publishes Root's question as
// <state>/questions/<id>.json and prints one line with the run identity to
// stderr; the answer arrives with `markitect answer`, which writes
// <state>/answers/<id>.txt, and Answer polls that file. Withdraw removes the
// question file of a question that lapsed, so no answer can reach it. The channel opens right after the run took the
// state lock: it removes the question and answer files of earlier runs and
// records its run in <state>/current-run, so a question file exists only while
// its run asked it and an answer can only reach the run that is waiting for
// it. Only files the channel owns are removed: .json files in questions and .txt
// files in answers.
type fileOwner struct {
	state, runID, config string
	stderr               io.Writer
}

var _ runport.Owner = (*fileOwner)(nil)

func newFileOwner(stateDirectory, runID, configPath string, stderr io.Writer) (*fileOwner, error) {
	for _, channel := range []struct{ dir, ext string }{{questionsDir, ".json"}, {answersDir, ".txt"}} {
		if err := clearChannel(filepath.Join(stateDirectory, channel.dir), channel.ext); err != nil {
			return nil, err
		}
	}
	if err := writeStateFile(filepath.Join(stateDirectory, currentRunFile), []byte(runID+"\n"), true); err != nil {
		return nil, err
	}
	return &fileOwner{state: stateDirectory, runID: runID, config: configPath, stderr: stderr}, nil
}

// clearChannel creates dir if needed and removes the regular files with the
// channel's extension directly inside it. Nothing else is touched.
func clearChannel(dir, ext string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ext {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
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
	fmt.Fprintf(o.stderr, "markitect: question %s of run %s: %s -- answer with: markitect answer %s --run %s --text ANSWER --run-config %s\n", q.ID, o.runID, strings.Join(strings.Fields(q.Text), " "), q.ID, o.runID, o.config)
	return nil
}

func (o *fileOwner) Withdraw(_ context.Context, id string) error {
	if !stateName.MatchString(id) {
		return fmt.Errorf("question identity %q is not a safe path element", id)
	}
	if err := os.Remove(filepath.Join(o.state, questionsDir, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
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

// writeAnswer records the answer to a question of the run that holds the
// state directory. The run is observed without creating anything and without
// taking the lock; the question must carry the identity the run recorded in
// <state>/current-run, and that identity must be the run the answer names. An
// answer is final: it is written once, atomically.
func writeAnswer(stateDirectory, run, id, text string) (string, error) {
	if !stateName.MatchString(id) {
		return "", fmt.Errorf("question identity %q is not a safe path element", id)
	}
	if strings.TrimSpace(text) == "" || len(text) > maxAnswerBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return "", fmt.Errorf("an answer is non-blank UTF-8 text of at most %d bytes", maxAnswerBytes)
	}
	held, err := stateHeld(stateDirectory)
	if err != nil {
		return "", err
	}
	if !held {
		return "", errors.New("no run is running on this state directory")
	}
	question, err := os.ReadFile(filepath.Join(stateDirectory, questionsDir, id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("the current run has not asked question %s", id)
	}
	if err != nil {
		return "", err
	}
	current, err := os.ReadFile(filepath.Join(stateDirectory, currentRunFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	var asked struct {
		Run string `json:"run"`
	}
	runID := strings.TrimSpace(string(current))
	if json.Unmarshal(question, &asked) != nil || runID == "" || asked.Run != runID {
		return "", fmt.Errorf("question %s does not belong to the current run", id)
	}
	if asked.Run != run {
		return "", fmt.Errorf("question %s belongs to run %s, not to run %s", id, asked.Run, run)
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
