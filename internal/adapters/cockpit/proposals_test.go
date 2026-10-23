package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/authoring"
)

// fakeAuthoring records what reached the proposal service.
type fakeAuthoring struct {
	proposed   []authoring.Settings
	decided    []string
	settings   error
	accept     func(id string) (authoring.Proposal, error)
	sessionsOn bool
	sessionErr error
	prompts    []string
}

func (f *fakeAuthoring) Settings(context.Context) (authoring.Settings, error) {
	return authoring.Settings{Head: "abc"}, f.settings
}

func (f *fakeAuthoring) ProposeSettings(_ context.Context, s authoring.Settings) (authoring.Proposal, error) {
	f.proposed = append(f.proposed, s)
	return authoring.Proposal{ID: "configuration-1", State: authoring.Proposed}, nil
}

func (f *fakeAuthoring) Proposals() []authoring.Proposal {
	return []authoring.Proposal{{ID: "configuration-1", State: authoring.Proposed}}
}

func (f *fakeAuthoring) Proposal(id string) (authoring.Proposal, bool) {
	return authoring.Proposal{ID: id, State: authoring.Proposed}, id == "configuration-1"
}

func (f *fakeAuthoring) Accept(_ context.Context, id string) (authoring.Proposal, error) {
	f.decided = append(f.decided, "accept "+id)
	return f.accept(id)
}

func (f *fakeAuthoring) Discard(id string) (authoring.Proposal, error) {
	f.decided = append(f.decided, "discard "+id)
	return authoring.Proposal{ID: id, State: authoring.Discarded}, nil
}

const local = "127.0.0.1:7341"

func authoringServer(t *testing.T, a Authoring) http.Handler {
	t.Helper()
	srv, err := New(Config{States: []string{t.TempDir()}, Authoring: a})
	if err != nil {
		t.Fatal(err)
	}
	srv.host = local
	return srv.Handler()
}

func getLocal(h http.Handler, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = local
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestProposalsNeedARepository(t *testing.T) {
	h := authoringServer(t, nil)
	for _, target := range []string{"/api/settings", "/api/proposals", "/api/proposals/configuration-1"} {
		if rec := getLocal(h, target); rec.Code != http.StatusNotImplemented {
			t.Errorf("%s without a repository = %d", target, rec.Code)
		}
	}
	if rec := post(h, local, "", "propose", "/api/proposals/settings", `{}`); rec.Code != http.StatusNotImplemented {
		t.Errorf("propose without a repository = %d", rec.Code)
	}
}

func TestProposalWritesNeedTheCockpitsOwnJSONRequest(t *testing.T) {
	fake := &fakeAuthoring{accept: func(id string) (authoring.Proposal, error) {
		return authoring.Proposal{ID: id, State: authoring.Accepted}, nil
	}}
	h := authoringServer(t, fake)
	refused := map[string]*httptest.ResponseRecorder{
		"no header":         post(h, local, "", "", "/api/proposals/settings", `{}`),
		"wrong purpose":     post(h, local, "", "answer", "/api/proposals/settings", `{}`),
		"foreign origin":    post(h, local, "http://evil.example", "propose", "/api/proposals/settings", `{}`),
		"accept no header":  post(h, local, "", "", "/api/proposals/configuration-1/accept", ``),
		"accept as discard": post(h, local, "", "discard", "/api/proposals/configuration-1/accept", ``),
	}
	for name, rec := range refused {
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, rec.Code)
		}
	}
	plain := httptest.NewRequest(http.MethodPost, "/api/proposals/settings", nil)
	plain.Host = local
	plain.Header.Set("Content-Type", "text/plain")
	plain.Header.Set(requestHeader, "propose")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, plain)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a text/plain proposal = %d", rec.Code)
	}
	for body, want := range map[string]int{
		`{"budget":{"agentTurns":"many"}}`: http.StatusBadRequest,
		`{"unknown":1}`:                    http.StatusBadRequest,
		`{} {}`:                            http.StatusBadRequest,
	} {
		if rec := post(h, local, "", "propose", "/api/proposals/settings", body); rec.Code != want {
			t.Errorf("body %s = %d, want %d", body, rec.Code, want)
		}
	}
	if len(fake.proposed) != 0 || len(fake.decided) != 0 {
		t.Fatalf("refused requests reached the service: %+v %+v", fake.proposed, fake.decided)
	}
	if rec := post(h, local, "", "propose", "/api/proposals/settings", `{"budget":{"agentTurns":7}}`); rec.Code != http.StatusOK || len(fake.proposed) != 1 || fake.proposed[0].Budget.AgentTurns != 7 {
		t.Fatalf("propose = %d %s, %+v", rec.Code, rec.Body, fake.proposed)
	}
	if rec := post(h, local, "", "publish", "/api/proposals/configuration-1/publish", ``); rec.Code != http.StatusNotFound {
		t.Errorf("an unknown decision = %d", rec.Code)
	}
	if rec := post(h, local, "", "accept", "/api/proposals/configuration-9/accept", ``); rec.Code != http.StatusNotFound {
		t.Errorf("an unknown proposal = %d", rec.Code)
	}
	if rec := post(h, local, "", "accept", "/api/proposals/configuration-1/accept", ``); rec.Code != http.StatusOK {
		t.Errorf("accept = %d %s", rec.Code, rec.Body)
	}
}

// A refused acceptance answers with its reason; one the publication did not
// apply answers with the failed proposal and its receipt, never as accepted.
func TestAcceptReportsRefusalsAndFailedPublications(t *testing.T) {
	fake := &fakeAuthoring{}
	h := authoringServer(t, fake)
	fake.accept = func(id string) (authoring.Proposal, error) {
		return authoring.Proposal{ID: id, State: authoring.Proposed}, errors.New("an Apply holds the state directory s; accept after it ends")
	}
	if rec := post(h, local, "", "accept", "/api/proposals/configuration-1/accept", ``); rec.Code != http.StatusConflict {
		t.Fatalf("a fenced acceptance = %d %s", rec.Code, rec.Body)
	}
	fake.accept = func(id string) (authoring.Proposal, error) {
		return authoring.Proposal{ID: id, State: authoring.Failed, Error: "the publication was not applied: partial",
			Receipt: &authoring.Receipt{Outcome: "partial", Paths: []authoring.PathState{{Path: ".markitect/execution.yaml", State: "other"}}}}, fmt.Errorf("partial")
	}
	rec := post(h, local, "", "accept", "/api/proposals/configuration-1/accept", ``)
	var p authoring.Proposal
	if err := json.Unmarshal(rec.Body.Bytes(), &p); rec.Code != http.StatusOK || err != nil || p.State != authoring.Failed || p.Receipt.Paths[0].State != "other" {
		t.Fatalf("a partial publication = %d %s", rec.Code, rec.Body)
	}
}

func TestSettingsRefusalsKeepTheirReason(t *testing.T) {
	fake := &fakeAuthoring{settings: fmt.Errorf("%w (dirty)", authoring.ErrNotClean)}
	h := authoringServer(t, fake)
	if rec := getLocal(h, "/api/settings"); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "uncommitted changes") {
		t.Fatalf("settings on a dirty tree = %d %s", rec.Code, rec.Body)
	}
}

func (f *fakeAuthoring) SessionsAvailable() bool       { return f.sessionsOn }
func (f *fakeAuthoring) Sessions() []authoring.Session { return nil }
func (f *fakeAuthoring) Session(id string) (authoring.Session, bool) {
	return authoring.Session{ID: id, State: authoring.Waiting}, id == "model-1"
}
func (f *fakeAuthoring) StartSession(_ context.Context, prompt string) (authoring.Session, error) {
	if !f.sessionsOn {
		return authoring.Session{}, authoring.ErrNoSessions
	}
	f.prompts = append(f.prompts, prompt)
	return authoring.Session{ID: "model-1", State: authoring.Working}, f.sessionErr
}
func (f *fakeAuthoring) PromptSession(id, prompt string) (authoring.Session, error) {
	f.prompts = append(f.prompts, id+": "+prompt)
	return authoring.Session{ID: id, State: authoring.Working}, f.sessionErr
}
func (f *fakeAuthoring) CloseSession(_ context.Context, id string) (authoring.Session, error) {
	f.decided = append(f.decided, "close "+id)
	return authoring.Session{ID: id, State: authoring.Closed}, nil
}

func TestSessionRequestsNeedTheCockpitsOwnJSONRequest(t *testing.T) {
	fake := &fakeAuthoring{sessionsOn: true}
	h := authoringServer(t, fake)
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"start without header": post(h, local, "", "", "/api/sessions", `{"prompt":"x"}`),
		"start as prompt":      post(h, local, "", "prompt", "/api/sessions", `{"prompt":"x"}`),
		"foreign origin":       post(h, local, "http://evil.example", "start", "/api/sessions", `{"prompt":"x"}`),
		"prompt as start":      post(h, local, "", "start", "/api/sessions/model-1/prompt", `{"prompt":"x"}`),
		"close without header": post(h, local, "", "", "/api/sessions/model-1/close", ``),
	} {
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, rec.Code)
		}
	}
	for body, want := range map[string]int{`{"prompt":"x","extra":1}`: http.StatusBadRequest, `{"text":"x"}`: http.StatusBadRequest, `{} {}`: http.StatusBadRequest} {
		if rec := post(h, local, "", "start", "/api/sessions", body); rec.Code != want {
			t.Errorf("body %s = %d, want %d", body, rec.Code, want)
		}
	}
	if len(fake.prompts) != 0 || len(fake.decided) != 0 {
		t.Fatalf("refused requests reached the service: %v %v", fake.prompts, fake.decided)
	}
	if rec := post(h, local, "", "start", "/api/sessions", `{"prompt":"Rename the Rule."}`); rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, local, "", "prompt", "/api/sessions/model-1/prompt", `{"prompt":"And its title."}`); rec.Code != http.StatusOK {
		t.Fatalf("prompt = %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, local, "", "close", "/api/sessions/model-9/close", ``); rec.Code != http.StatusNotFound {
		t.Fatalf("closing an unknown session = %d", rec.Code)
	}
	if rec := post(h, local, "", "close", "/api/sessions/model-1/close", ``); rec.Code != http.StatusOK {
		t.Fatalf("close = %d %s", rec.Code, rec.Body)
	}
	if strings.Join(fake.prompts, "|") != "Rename the Rule.|model-1: And its title." || len(fake.decided) != 1 {
		t.Fatalf("service saw %v %v", fake.prompts, fake.decided)
	}
}

func TestSessionRefusalsMapToTheirStatus(t *testing.T) {
	if rec := post(authoringServer(t, &fakeAuthoring{}), local, "", "start", "/api/sessions", `{"prompt":"x"}`); rec.Code != http.StatusNotImplemented {
		t.Errorf("start without a Runtime profile = %d", rec.Code)
	}
	for err, want := range map[error]int{authoring.ErrBusy: http.StatusConflict, authoring.ErrLimit: http.StatusConflict, authoring.ErrSessionOpen: http.StatusConflict, authoring.ErrPrompt: http.StatusBadRequest} {
		h := authoringServer(t, &fakeAuthoring{sessionsOn: true, sessionErr: err})
		if rec := post(h, local, "", "prompt", "/api/sessions/model-1/prompt", `{"prompt":"x"}`); rec.Code != want {
			t.Errorf("%v = %d, want %d", err, rec.Code, want)
		}
	}
}
