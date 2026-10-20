package cockpit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const ownerRun = "20261020T120000Z-owner"

// ownerState writes the library run up to Root's open owner question Q3,
// names it in current-run and publishes Q3's question file, as the owner
// channel does while the run waits.
func ownerState(t *testing.T) (string, int) {
	t.Helper()
	s, at := library(t)
	asked := 0
	for i, e := range s.events {
		if e.Kind == "question.asked" && i+1 > at["closed"]-2 {
			asked = i + 1
		}
	}
	if asked == 0 {
		t.Fatal("the fixture asks the owner no question")
	}
	s.events = s.events[:asked]
	state := s.writeRun(ownerRun, nil)
	files := map[string]string{
		"current-run":       ownerRun + "\n",
		"questions/Q3.json": `{"id":"Q3","run":"` + ownerRun + `","text":"May the receipt mention the limit?","asked":"2026-10-20T12:00:00Z"}`,
	}
	for name, content := range files {
		path := filepath.Join(state, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return state, asked
}

type answerCall struct{ state, run, question, text string }

func answering(t *testing.T, state string, held bool, calls *[]answerCall) (*Server, http.Handler) {
	t.Helper()
	srv, err := New(Config{
		States:   []string{state},
		Liveness: func(string) (bool, error) { return held, nil },
		Answer: func(st, run, question, text string) (string, error) {
			*calls = append(*calls, answerCall{st, run, question, text})
			return filepath.Join(st, "answers", question+".txt"), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv, srv.Handler()
}

func post(h http.Handler, host, origin, header, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set(answerHeader, header)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAnswerReachesTheOwnerChannelOnlyForAWaitingQuestion(t *testing.T) {
	state, _ := ownerState(t)
	var calls []answerCall
	srv, h := answering(t, state, true, &calls)
	srv.host = "127.0.0.1:7341"
	target := "/api/runs/" + ownerRun + "/questions/Q3/answer"
	body := `{"text":"Yes, mention the limit."}`
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"without the cockpit header": post(h, "127.0.0.1:7341", "", "", target, body),
		"from a foreign origin":      post(h, "127.0.0.1:7341", "http://evil.example", "answer", target, body),
		"to a foreign host":          post(h, "evil.example:7341", "", "answer", target, body),
		"for an unknown question":    post(h, "127.0.0.1:7341", "", "answer", "/api/runs/"+ownerRun+"/questions/Q9/answer", body),
	} {
		if rec.Code == http.StatusOK {
			t.Errorf("an answer %s was accepted", name)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("refused answers reached the owner channel: %+v", calls)
	}
	rec := post(h, "127.0.0.1:7341", "http://127.0.0.1:7341", "answer", target, body)
	if rec.Code != http.StatusOK || len(calls) != 1 || calls[0] != (answerCall{srv.states[0].dir, ownerRun, "Q3", "Yes, mention the limit."}) {
		t.Fatalf("answer = %d %s, calls %+v", rec.Code, rec.Body.String(), calls)
	}

	if err := os.MkdirAll(filepath.Join(state, "answers"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "answers", "Q3.txt"), []byte("Yes."), 0o600); err != nil {
		t.Fatal(err)
	}
	if rec := post(h, "127.0.0.1:7341", "", "answer", target, body); rec.Code != http.StatusConflict || len(calls) != 1 {
		t.Fatalf("a second answer = %d, calls %d", rec.Code, len(calls))
	}
}

func TestAnswerIsRefusedWhenTheRunIsNotRunningOrTheFileIsWithdrawn(t *testing.T) {
	state, _ := ownerState(t)
	var calls []answerCall
	_, h := answering(t, state, false, &calls)
	target := "/api/runs/" + ownerRun + "/questions/Q3/answer"
	if rec := post(h, "", "", "answer", target, `{"text":"Yes."}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "not running") {
		t.Fatalf("answer to a run that does not run = %d %s", rec.Code, rec.Body.String())
	}
	_, h = answering(t, state, true, &calls)
	if err := os.Remove(filepath.Join(state, "questions", "Q3.json")); err != nil {
		t.Fatal(err)
	}
	if rec := post(h, "", "", "answer", target, `{"text":"Yes."}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "not published") {
		t.Fatalf("answer to a withdrawn question = %d %s", rec.Code, rec.Body.String())
	}
	if len(calls) != 0 {
		t.Fatalf("refused answers reached the owner channel: %+v", calls)
	}
}

func TestQuestionsListTheOwnerChannelState(t *testing.T) {
	state, _ := ownerState(t)
	var calls []answerCall
	_, h := answering(t, state, true, &calls)
	var questions []OwnerQuestion
	get(t, h, "", "/api/questions", &questions)
	if len(questions) != 1 || !questions[0].Answerable || !questions[0].Published || questions[0].RunState != "running" {
		t.Fatalf("questions = %+v", questions)
	}
	s, _ := library(t)
	lapsed := s.writeRun(ownerRun, nil)
	_, h = answering(t, lapsed, true, &calls)
	get(t, h, "", "/api/questions", &questions)
	if len(questions) != 1 || questions[0].Answerable || questions[0].Status != "lapsed" {
		t.Fatalf("a lapsed question is listed as not answerable: %+v", questions)
	}
}

func TestProjectIsObservedOnceAndAgainOnRequest(t *testing.T) {
	calls := 0
	srv, err := New(Config{States: []string{t.TempDir()}, Project: func(context.Context) (ProjectInfo, error) {
		calls++
		return ProjectInfo{Repository: "repo", ModelID: "markitect", Status: "passed", Packages: 3}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	var got struct {
		Project   *ProjectInfo `json:"project"`
		Observing bool         `json:"observing"`
		Answers   bool         `json:"answers"`
	}
	read := func(target string) {
		rec := get(t, h, "", target, nil)
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
	}
	// The observation runs in the background; poll until it is there.
	settle := func(target string) {
		read(target)
		for i := 0; got.Observing || got.Project == nil; i++ {
			if i == 200 {
				t.Fatal("the project observation never finished")
			}
			time.Sleep(5 * time.Millisecond)
			read("/api/project")
		}
	}
	settle("/api/project")
	settle("/api/project")
	settle("/api/project?refresh=1")
	if calls != 2 || got.Project.ModelID != "markitect" || got.Answers {
		t.Fatalf("project observed %d times: %+v", calls, got)
	}
}
