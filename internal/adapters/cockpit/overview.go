package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ProjectInfo is the identity of the repository the cockpit was started in,
// as structural preparation observed it: the bound model and source.
type ProjectInfo struct {
	Repository  string    `json:"repository"`
	ModelID     string    `json:"modelID,omitempty"`
	ModelDigest string    `json:"modelDigest,omitempty"`
	Head        string    `json:"head,omitempty"`
	Packages    int       `json:"packages"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	ObservedAt  time.Time `json:"observedAt"`
}

// ProjectObserver prepares the project's identity. Bootstrap supplies the
// product's structural preparation; it reads and writes nothing else.
type ProjectObserver func(context.Context) (ProjectInfo, error)

// projectCache keeps the last observation. Structural preparation of a
// repository takes seconds, so it runs in the background: a request returns
// the last observation and whether a new one is under way.
type projectCache struct {
	mu        sync.Mutex
	value     *ProjectInfo
	observing bool
	observe   ProjectObserver
}

// get returns the last observation and starts one when there is none yet or
// a refresh is asked for.
func (c *projectCache) get(refresh bool) (*ProjectInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.observe == nil {
		return nil, false
	}
	if (c.value == nil || refresh) && !c.observing {
		c.observing = true
		go c.run()
	}
	return c.value, c.observing
}

// observeTimeout bounds one structural preparation of the repository.
const observeTimeout = 2 * time.Minute

func (c *projectCache) run() {
	p := ProjectInfo{}
	defer func() {
		if r := recover(); r != nil {
			p.Status, p.Error = "not_completed", fmt.Sprint("the observation failed: ", r)
		}
		p.ObservedAt = time.Now().UTC()
		c.mu.Lock()
		c.value, c.observing = &p, false
		c.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), observeTimeout)
	defer cancel()
	var err error
	p, err = c.observe(ctx)
	if err != nil {
		p.Error = err.Error()
	}
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	s.writeProject(w, false)
}

// handleObserve runs the repository's structural preparation again. It is a
// POST with the cockpit's own header, so another page cannot start one.
func (s *Server) handleObserve(w http.ResponseWriter, r *http.Request) {
	if !s.ownRequest(w, r, "observe") {
		return
	}
	s.writeProject(w, true)
}

func (s *Server) writeProject(w http.ResponseWriter, refresh bool) {
	p, observing := s.project.get(refresh)
	writeJSON(w, struct {
		Project    *ProjectInfo `json:"project"`
		Observing  bool         `json:"observing"`
		Configured bool         `json:"configured"`
		States     []string     `json:"states"`
		Answers    bool         `json:"answers"`
		Proposals  bool         `json:"proposals"`
	}{p, observing, s.project.observe != nil, s.stateDirs(), s.answer != nil, s.authoring != nil})
}

func (s *Server) stateDirs() []string {
	out := make([]string, len(s.states))
	for i, st := range s.states {
		out[i] = st.dir
	}
	return out
}

// OwnerQuestion is Root's question to the owner as the record shows it, with
// what the owner channel shows now: whether its question file is published
// and an answer file exists. A question can be answered only while its run
// holds the state directory and its file is published.
type OwnerQuestion struct {
	Run        string    `json:"run"`
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	Asked      time.Time `json:"asked"`
	Status     string    `json:"status"`
	Published  bool      `json:"published"`
	AnswerFile bool      `json:"answerFile"`
	RunState   string    `json:"runState"`
	Answerable bool      `json:"answerable"`
	Why        string    `json:"why"`
}

// ownerChannel reports which question files of a run are published and which
// have an answer file. The channel removes the files of earlier runs when a
// run opens it and a question's file when it lapses.
func (st *stateDirectory) ownerChannel(run string) (published, answered map[string]bool) {
	published, answered = map[string]bool{}, map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(st.dir, "questions"))
	if err != nil {
		return published, answered
	}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !e.Type().IsRegular() || !safeName.MatchString(id) {
			continue
		}
		data, err := readBounded(filepath.Join(st.dir, "questions", e.Name()), maxQuestionBytes)
		if err != nil {
			continue
		}
		var q struct {
			ID  string `json:"id"`
			Run string `json:"run"`
		}
		if json.Unmarshal(data, &q) != nil || q.Run != run || q.ID != id {
			continue
		}
		published[id] = true
		if _, err := os.Stat(filepath.Join(st.dir, "answers", id+".txt")); err == nil {
			answered[id] = true
		}
	}
	return published, answered
}

// ownerQuestions lists the owner questions of one run: every question the
// record shows Root asking the user, with the channel's current files. The
// record's questions are projected once per source state, not on every poll.
func (s *Server) ownerQuestions(st *stateDirectory, l *runLog) []OwnerQuestion {
	records, src, report := l.snapshot()
	questions := l.userQuestions(records, src)
	if len(questions) == 0 {
		return nil
	}
	health := s.health(st, l, records, src, report)
	published, answered := st.ownerChannel(l.id)
	var out []OwnerQuestion
	for _, q := range questions {
		oq := OwnerQuestion{Run: l.id, ID: q.ID, Text: q.Text, Asked: q.Asked, Status: q.Status, Published: published[q.ID], AnswerFile: answered[q.ID], RunState: health.Run}
		switch {
		case q.Status != "open":
			oq.Why = "the record shows it " + q.Status
		case health.Run != "running":
			oq.Why = "its run is not running: " + health.Why
		case !oq.Published:
			oq.Why = "its question file is not published in the owner channel"
		case oq.AnswerFile:
			oq.Why = "an answer is written; the run records it when it reads the owner channel"
		default:
			oq.Answerable, oq.Why = true, "its run waits for the owner's answer"
		}
		out = append(out, oq)
	}
	return out
}

func (s *Server) handleQuestions(w http.ResponseWriter, r *http.Request) {
	out := []OwnerQuestion{}
	for _, st := range s.states {
		runs, err := st.list()
		if err != nil {
			continue
		}
		for _, entry := range runs {
			l := st.log(entry.ID)
			if _, err := l.refresh(); err != nil {
				continue
			}
			out = append(out, s.ownerQuestions(st, l)...)
		}
	}
	// Questions that wait for the owner come first, then the newest.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Answerable != out[j].Answerable {
			return out[i].Answerable
		}
		return out[i].Asked.After(out[j].Asked)
	})
	writeJSON(w, out)
}

// userQuestions returns the record's questions to the user, projected again
// only when the consumed records changed.
func (l *runLog) userQuestions(records []Record, src Source) []QuestionView {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := src.EventDigest
	if l.questionsKey == key {
		return l.questions
	}
	var out []QuestionView
	for _, q := range Project(l.id, records, len(records), src, nil).Questions {
		if q.To == "user" {
			out = append(out, q)
		}
	}
	l.questions, l.questionsKey = out, key
	return out
}

// Answerer writes an answer to an owner question through the owner channel.
// Bootstrap supplies the function the answer command uses: it accepts only an
// open question of the run that holds the state directory, once, and returns
// the answer file it wrote.
type Answerer func(stateDirectory, run, question, text string) (string, error)

// answerHeader must accompany every write. A cross-site form cannot send it,
// and a cross-site script cannot send it without a preflight the cockpit
// refuses.
const answerHeader = "X-Markitect-Cockpit"

// ownRequest admits a write only from the cockpit's own page: its header with
// the expected purpose and, when the browser names one, its own origin.
func (s *Server) ownRequest(w http.ResponseWriter, r *http.Request, purpose string) bool {
	if r.Header.Get(answerHeader) != purpose {
		http.Error(w, "a write needs the cockpit's own request", http.StatusForbidden)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && !s.admitHost(strings.TrimPrefix(origin, "http://")) {
		http.Error(w, "foreign origin", http.StatusForbidden)
		return false
	}
	return true
}

// maxAnswerBytes is the owner channel's limit for an answer.
const maxAnswerBytes = 16 << 10

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	if s.answer == nil {
		http.Error(w, "answering is not configured", http.StatusNotImplemented)
		return
	}
	if !s.ownRequest(w, r, "answer") {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "an answer is a JSON request", http.StatusForbidden)
		return
	}
	st, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	id := r.PathValue("question")
	var body struct {
		Text string `json:"text"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || dec.More() {
		http.Error(w, "the answer is not a JSON object with text", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Text) == "" || len(body.Text) > maxAnswerBytes || !utf8.ValidString(body.Text) || strings.ContainsRune(body.Text, 0) {
		http.Error(w, fmt.Sprintf("an answer is non-blank UTF-8 text of at most %d bytes", maxAnswerBytes), http.StatusBadRequest)
		return
	}
	var question *OwnerQuestion
	for _, q := range s.ownerQuestions(st, l) {
		if q.ID == id {
			question = &q
		}
	}
	switch {
	case question == nil:
		http.Error(w, "run "+l.id+" has asked the owner no question "+id, http.StatusNotFound)
		return
	case !question.Answerable:
		http.Error(w, "question "+id+" cannot be answered: "+question.Why, http.StatusConflict)
		return
	}
	path, err := s.answer(st.dir, l.id, id, body.Text)
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, os.ErrPermission) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, struct {
		Question string    `json:"question"`
		Path     string    `json:"path"`
		Written  time.Time `json:"written"`
	}{id, path, time.Now().UTC()})
}
