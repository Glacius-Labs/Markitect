package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestOwnerAskPublishesTheQuestionAndPrintsOneLine(t *testing.T) {
	state := testkit.TempDir(t)
	var stderr bytes.Buffer
	owner, err := newFileOwner(state, "run-1", "/abs/run.json", &stderr)
	if err != nil {
		t.Fatal(err)
	}
	asked := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if err := owner.Ask(context.Background(), runport.Question{ID: "q1", Text: "Which currency\nshould fines use?", Asked: asked}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(state, "questions", "q1.json"))
	var published struct {
		ID, Run, Text string
		Asked         time.Time
	}
	if err != nil || json.Unmarshal(data, &published) != nil || published.ID != "q1" || published.Run != "run-1" || published.Text != "Which currency\nshould fines use?" || !published.Asked.Equal(asked) {
		t.Fatalf("question file: %s %v", data, err)
	}
	line := stderr.String()
	if strings.Count(line, "\n") != 1 || !strings.HasSuffix(line, "\n") || !strings.Contains(line, "q1") || !strings.Contains(line, "Which currency should fines use?") || !strings.Contains(line, "markitect answer q1 --text ANSWER --run-config /abs/run.json") {
		t.Fatalf("stderr is not one useful line: %q", line)
	}
	if err := owner.Ask(context.Background(), runport.Question{ID: "q1", Text: "again"}); err == nil {
		t.Fatal("a question identity was asked twice")
	}
}

func TestOwnerAnswerPollsTheAnswerFileOfTheRunningApply(t *testing.T) {
	state := testkit.TempDir(t)
	owner, err := newFileOwner(state, "run-1", "/abs/run.json", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockState(state) // the running Apply
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := owner.Ask(context.Background(), runport.Question{ID: "q1", Text: "Which currency?"}); err != nil {
		t.Fatal(err)
	}
	if text, answered, err := owner.Answer(context.Background(), "q1"); err != nil || answered || text != "" {
		t.Fatalf("unanswered question reported an answer: %q %v %v", text, answered, err)
	}
	path, err := writeAnswer(state, "q1", "Euro, with a trailing\nnewline.\n")
	if err != nil || path != filepath.Join(state, "answers", "q1.txt") {
		t.Fatalf("%s %v", path, err)
	}
	if text, answered, err := owner.Answer(context.Background(), "q1"); err != nil || !answered || text != "Euro, with a trailing\nnewline.\n" {
		t.Fatalf("answer not returned literally: %q %v %v", text, answered, err)
	}
}

func TestOwnerChannelOpensEmptyAndRefusesUnsafeIdentities(t *testing.T) {
	state := testkit.TempDir(t)
	for _, name := range []string{"questions", "answers"} {
		if err := os.MkdirAll(filepath.Join(state, name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(state, name, "stale.txt"), []byte("earlier Apply"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	owner, err := newFileOwner(state, "run-2", "/abs/run.json", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"questions", "answers"} {
		if entries, err := os.ReadDir(filepath.Join(state, name)); err != nil || len(entries) != 0 {
			t.Fatalf("%s kept earlier Apply files: %v %v", name, entries, err)
		}
	}
	for _, unsafe := range []string{"", "../q", "a/b", `a\b`, ".q"} {
		if err := owner.Ask(context.Background(), runport.Question{ID: unsafe, Text: "x"}); err == nil {
			t.Errorf("Ask accepted %q", unsafe)
		}
		if _, _, err := owner.Answer(context.Background(), unsafe); err == nil {
			t.Errorf("Answer accepted %q", unsafe)
		}
		if _, err := writeAnswer(state, unsafe, "x"); err == nil {
			t.Errorf("writeAnswer accepted %q", unsafe)
		}
	}
}

func TestOwnerAnswerRefusesOversizedOrNonTextFiles(t *testing.T) {
	state := testkit.TempDir(t)
	owner, err := newFileOwner(state, "run-1", "/abs/run.json", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"big": bytes.Repeat([]byte("a"), maxAnswerBytes+1), "binary": {0xff, 0xfe}} {
		if err := os.WriteFile(filepath.Join(state, "answers", name+".txt"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, answered, err := owner.Answer(context.Background(), name); err == nil || answered {
			t.Errorf("%s answer accepted", name)
		}
	}
}

func TestWriteAnswerRequiresARunningApplyAnAskedQuestionAndIsFinal(t *testing.T) {
	state := testkit.TempDir(t)
	if _, err := writeAnswer(state, "q1", "x"); err == nil || !strings.Contains(err.Error(), "no Apply is running") {
		t.Fatalf("answer without a running Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(state, "answers", "q1.txt")); !os.IsNotExist(err) {
		t.Fatal("answer written without a running Apply")
	}
	owner, err := newFileOwner(state, "run-1", "/abs/run.json", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := writeAnswer(state, "q1", "x"); err == nil || !strings.Contains(err.Error(), "has not asked") {
		t.Fatalf("answer to a question that was not asked: %v", err)
	}
	if err := owner.Ask(context.Background(), runport.Question{ID: "q1", Text: "Which?"}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", " \n\t", strings.Repeat("a", maxAnswerBytes+1), "a\x00b", string([]byte{0xff})} {
		if _, err := writeAnswer(state, "q1", bad); err == nil {
			t.Errorf("answer %q accepted", bad)
		}
	}
	if _, err := writeAnswer(state, "q1", "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := writeAnswer(state, "q1", "second"); err == nil || !strings.Contains(err.Error(), "already answered") {
		t.Fatalf("an answer was replaced: %v", err)
	}
	if text, _, _ := owner.Answer(context.Background(), "q1"); text != "first" {
		t.Fatalf("first answer lost: %q", text)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "answers")); len(entries) != 1 {
		t.Fatalf("stray answer files: %v", entries)
	}
}
