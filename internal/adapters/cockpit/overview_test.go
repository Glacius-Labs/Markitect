package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func post(h http.Handler, host, origin, header, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set(requestHeader, header)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// A Run is closed: the cockpit offers no route to list or answer a question of
// a run, whatever the evidence of an old run records.
func TestNoRouteListsOrAnswersAQuestion(t *testing.T) {
	s, _ := library(t)
	state := s.writeRun("20261020T120000Z-closed", nil)
	srv, err := New(Config{States: []string{state}, Liveness: func(string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	body := `{"text":"Yes, mention the limit."}`
	for _, target := range []string{"/api/runs/20261020T120000Z-closed/questions/Q3/answer", "/api/questions"} {
		if rec := post(h, "127.0.0.1:7341", "", "answer", target, body); rec.Code == http.StatusOK {
			t.Errorf("POST %s was accepted", target)
		}
	}
	if rec := get(t, h, "", "/api/questions", nil); rec.Code == http.StatusOK && strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("GET /api/questions answers JSON: %s", rec.Body.String())
	}
}

func TestProjectIsObservedOnceAndAgainOnRequest(t *testing.T) {
	calls := 0
	srv, err := New(Config{States: []string{t.TempDir()}, Project: func(context.Context) (ProjectInfo, error) {
		calls++
		return ProjectInfo{Repository: "repo", ModelID: "markitect", Status: "passed", Packages: 3, Organization: &OrganizationView{Status: "passed", Managers: []OrganizationManager{{Path: ".", Label: "Root"}, {Path: "stock", Parent: ".", Depth: 1, Label: "stock"}, {Path: "stock/deep", Parent: "stock", Depth: 2, Label: "deep"}}}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	var got struct {
		Project      *ProjectInfo           `json:"project"`
		Observing    bool                   `json:"observing"`
		Organization *OrganizationThumbnail `json:"organization"`
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
	if got.Organization == nil || got.Organization.Total != 3 || got.Organization.Hidden != 1 || len(got.Organization.Managers) != 2 || calls != 1 {
		t.Fatalf("thumbnail caused another observation or invented structure: %+v, calls %d", got.Organization, calls)
	}
	if rec := post(h, "", "", "", "/api/project/observe", ""); rec.Code != http.StatusForbidden || calls != 1 {
		t.Fatalf("an observation without the cockpit's request = %d, %d calls", rec.Code, calls)
	}
	if rec := post(h, "", "", "observe", "/api/project/observe", ""); rec.Code != http.StatusOK {
		t.Fatalf("observe = %d", rec.Code)
	}
	settle("/api/project")
	if calls != 2 || got.Project.ModelID != "markitect" {
		t.Fatalf("project observed %d times: %+v", calls, got)
	}
}

func TestProjectObservationFailuresAreShown(t *testing.T) {
	for name, observe := range map[string]ProjectObserver{
		"error": func(context.Context) (ProjectInfo, error) {
			return ProjectInfo{Status: "failed"}, errors.New("no model here")
		},
		"panic": func(context.Context) (ProjectInfo, error) { panic("broken preparation") },
	} {
		c := projectCache{observe: observe}
		c.get(false)
		var p *ProjectInfo
		for i := 0; ; i++ {
			var observing bool
			if p, observing = c.get(false); p != nil && !observing {
				break
			}
			if i == 200 {
				t.Fatalf("%s: the observation never finished", name)
			}
			time.Sleep(5 * time.Millisecond)
		}
		if p.Error == "" {
			t.Errorf("%s: the failure is not shown: %+v", name, p)
		}
	}
}
