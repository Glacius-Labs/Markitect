package cockpit

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func serve(t *testing.T, state string) (*Server, http.Handler) {
	t.Helper()
	srv, err := New(Config{States: []string{state}})
	if err != nil {
		t.Fatal(err)
	}
	return srv, srv.Handler()
}

func get(t *testing.T, h http.Handler, host, target string, out any) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if out != nil && rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	return rec
}

func TestServerServesBoundedViews(t *testing.T) {
	s, at := library(t)
	state := s.writeRun("20261020T120000Z-aaaa", applyReport("success", "applied"))
	_, h := serve(t, state)
	var runs []RunSummary
	if rec := get(t, h, "", "/api/runs", &runs); rec.Code != http.StatusOK || len(runs) != 1 || runs[0].RunOutcome != "success" || runs[0].Managers != 3 {
		t.Fatalf("runs = %d %+v", rec.Code, runs)
	}
	var v View
	get(t, h, "", "/api/runs/20261020T120000Z-aaaa/view?at="+strconv.Itoa(at["asked"]), &v)
	if v.Boundary.Seq != at["asked"] || v.Boundary.Final || v.Status.Report != nil {
		t.Fatalf("view boundary = %+v", v.Boundary)
	}
	get(t, h, "", "/api/runs/20261020T120000Z-aaaa/view", &v)
	if !v.Boundary.Final || v.Status.Report == nil {
		t.Fatalf("the default view is the latest: %+v", v.Boundary)
	}
	var entries struct{ Entries []Entry }
	get(t, h, "", "/api/runs/20261020T120000Z-aaaa/entries?after=5", &entries)
	if len(entries.Entries) == 0 || entries.Entries[0].Seq != 6 {
		t.Fatalf("entries after 5 start at 6: %+v", entries.Entries[:1])
	}
	for _, target := range []string{"/api/runs/..%2F..%2Fetc/view", "/api/runs/missing/view", "/api/runs/20261020T120000Z-aaaa/view?at=x"} {
		if rec := get(t, h, "", target, nil); rec.Code == http.StatusOK {
			t.Fatalf("%s answered 200", target)
		}
	}
}

func TestServerServesTranscriptsOnlyOfStartedTurns(t *testing.T) {
	s, at := library(t)
	state := s.writeRun("20261020T120000Z-aaaa", nil)
	transcript := strings.Join([]string{
		`{"at":"2026-10-20T12:00:01Z","kind":"sent","data":{"type":"user","message":{"role":"user","content":"Plan your work."}}}`,
		`{"at":"2026-10-20T12:00:02Z","kind":"tool_call","data":{"id":1,"name":"report","arguments":{"kind":"basis"}}}`,
		`{"at":"2026-10-20T23:00:00Z","kind":"end","data":{"reason":"provider result","stopped":true}}`,
	}, "\n") + "\n"
	path := filepath.Join(state, "runs", "20261020T120000Z-aaaa", "transcripts", "r-p1-root-m1.jsonl")
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	_, h := serve(t, state)
	var tr Transcript
	rec := get(t, h, "", "/api/runs/20261020T120000Z-aaaa/transcript?turn=r-p1-root-m1&at="+strconv.Itoa(at["asked"]), &tr)
	if rec.Code != http.StatusOK || len(tr.Entries) != 2 || tr.Later != 1 || tr.Entries[0].From != "input" || tr.Entries[1].Name != "report" {
		t.Fatalf("transcript = %d %+v", rec.Code, tr)
	}
	if rec := get(t, h, "", "/api/runs/20261020T120000Z-aaaa/transcript?turn=r-p3-lending-r1&at=5", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("a turn not started at the boundary = %d", rec.Code)
	}
	if rec := get(t, h, "", "/api/runs/20261020T120000Z-aaaa/transcript?turn=..%2Fevents&at=0", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("a path-like turn = %d", rec.Code)
	}
}

func TestServerRefusesForeignHostsAndSetsAPolicy(t *testing.T) {
	s, _ := library(t)
	srv, h := serve(t, s.writeRun("20261020T120000Z-aaaa", nil))
	srv.host = "127.0.0.1:7341"
	if rec := get(t, h, "evil.example:7341", "/api/runs", nil); rec.Code != http.StatusMisdirectedRequest {
		t.Fatalf("foreign host = %d", rec.Code)
	}
	rec := get(t, h, "localhost:7341", "/", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "script-src 'self'") {
		t.Fatalf("index = %d %v", rec.Code, rec.Header())
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/runs", nil)
		req.Host = "127.0.0.1:7341"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /api/runs = %d: the read interface accepts only GET", method, rec.Code)
		}
	}
}

func TestNewRefusesMissingStateAndServeRefusesNonLoopback(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("no state directory accepted")
	}
	if _, err := New(Config{States: []string{filepath.Join(t.TempDir(), "absent")}}); err == nil {
		t.Fatal("a missing state directory accepted")
	}
	srv, err := New(Config{States: []string{t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Serve(t.Context(), "0.0.0.0:0", nil); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("Serve on all interfaces = %v", err)
	}
}

// The page loads its modules by relative path; a renamed or missing module
// would only fail in the browser.
func TestEmbeddedPageReferencesExistingFiles(t *testing.T) {
	refs := regexp.MustCompile(`(?:from |import\()"(\./[^"]+)"|(?:src|href)="([^":]+\.(?:js|css))"`)
	err := fs.WalkDir(webFiles, "web", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(webFiles, name)
		if err != nil {
			return err
		}
		for _, m := range refs.FindAllStringSubmatch(string(data), -1) {
			ref := m[1] + m[2]
			target := path.Join(path.Dir(name), ref)
			if _, err := fs.Stat(webFiles, target); err != nil {
				t.Errorf("%s references %s, which is not embedded", name, ref)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
