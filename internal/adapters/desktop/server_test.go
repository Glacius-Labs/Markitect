package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// memRepo is a repository's working tree in memory.
type memRepo struct {
	root  string
	files map[string]string
}

func (m memRepo) Root() string { return m.root }

func (m memRepo) Files() ([]string, error) {
	var names []string
	for f := range m.files {
		names = append(names, f)
	}
	sort.Strings(names)
	return names, nil
}

func (m memRepo) Read(file string) ([]byte, error) {
	data, ok := m.files[file]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

func (memRepo) GitIgnored(string) bool { return false }

// fakeRepos knows the repositories by root; Init and Clone add one.
type fakeRepos struct {
	mu    sync.Mutex
	repos map[string]memRepo
}

func newFakeRepos(roots ...string) *fakeRepos {
	f := &fakeRepos{repos: map[string]memRepo{}}
	for _, r := range roots {
		f.repos[r] = memRepo{root: r, files: map[string]string{"README.md": "# x\n"}}
	}
	return f
}

func (f *fakeRepos) Open(_ context.Context, dir string) (Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for root, r := range f.repos {
		if dir == root || strings.HasPrefix(dir, root+string(filepath.Separator)) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("%s is not inside a Git repository", dir)
}

func (f *fakeRepos) Init(_ context.Context, dir string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[dir] = memRepo{root: dir, files: map[string]string{}}
	return nil
}

func (f *fakeRepos) Clone(_ context.Context, url, dir string) error {
	if url == "https://example.com/missing.git" {
		return errors.New("git clone: repository not found")
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}
	return f.Init(context.Background(), dir)
}

// fakeCommands' init writes only the manifest, enough to count as set up.
type fakeCommands struct{ repos *fakeRepos }

func (c fakeCommands) Init(_ context.Context, dir string, out io.Writer) (bool, error) {
	c.repos.mu.Lock()
	defer c.repos.mu.Unlock()
	r, ok := c.repos.repos[dir]
	if !ok {
		return false, fmt.Errorf("no repository at %s", dir)
	}
	r.files[".markitect/manifest.yaml"] = "libraries:\n  std: 1.0.0\n"
	fmt.Fprintln(out, "created .markitect/manifest.yaml")
	return true, nil
}

const (
	testSecret = "0123456789abcdef"
	testPort   = "7340"
)

func testServer(t *testing.T, roots ...string) (*server, *fakeRepos) {
	t.Helper()
	repos := newFakeRepos(roots...)
	store := &Store{File: filepath.Join(t.TempDir(), projectsFile)}
	s, err := newServer(testSecret, testPort, store, repos, fakeCommands{repos})
	if err != nil {
		t.Fatal(err)
	}
	return s, repos
}

// call sends a request as the app's own window does: to the loopback host,
// with the session cookie and, for a body, as JSON.
func call(s *server, method, path, body string, edit ...func(*http.Request)) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Host = "127.0.0.1:" + testPort
	req.AddCookie(&http.Cookie{Name: cookieName, Value: s.session})
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, e := range edit {
		e(req)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("%d %s: %v", w.Code, w.Body, err)
	}
	return v
}

func TestServerAnswersOnlyItsLoopbackNames(t *testing.T) {
	s, _ := testServer(t)
	for _, host := range []string{"127.0.0.1:" + testPort, "localhost:" + testPort, "[::1]:" + testPort} {
		if w := call(s, "GET", "/api/app", "", func(r *http.Request) { r.Host = host }); w.Code != http.StatusOK {
			t.Errorf("%s: %d", host, w.Code)
		}
	}
	// A page of another site whose name resolves to 127.0.0.1 sends its name.
	for _, host := range []string{"evil.example:" + testPort, "127.0.0.1:9999", "127.0.0.1"} {
		if w := call(s, "GET", "/api/app", "", func(r *http.Request) { r.Host = host }); w.Code != http.StatusMisdirectedRequest {
			t.Errorf("%s: %d", host, w.Code)
		}
	}
}

func TestALaunchCodeWorksOnceAndBriefly(t *testing.T) {
	s, _ := testServer(t)
	noCookie := func(r *http.Request) { r.Header.Del("Cookie") }
	code := s.launch()
	w := call(s, "GET", "/?t="+code, "", noCookie)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Fatalf("exchange: %d %q", w.Code, w.Header().Get("Location"))
	}
	c := w.Result().Cookies()
	if len(c) != 1 || c[0].Value != s.session || !c[0].HttpOnly || c[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie: %+v", c)
	}
	if s.session == testSecret || s.session == code {
		t.Fatal("the session reuses another secret")
	}
	if w := call(s, "GET", "/?t="+code, "", noCookie); w.Code != http.StatusForbidden {
		t.Errorf("second use: %d", w.Code)
	}
	late := s.launch()
	s.now = func() time.Time { return time.Now().Add(launchLife + time.Second) }
	if w := call(s, "GET", "/?t="+late, "", noCookie); w.Code != http.StatusForbidden {
		t.Errorf("expired code: %d", w.Code)
	}
	if w := call(s, "GET", "/?t=wrong", "", noCookie); w.Code != http.StatusForbidden {
		t.Errorf("wrong code: %d", w.Code)
	}
	for _, path := range []string{"/", "/api/projects"} {
		if w := call(s, "GET", path, "", noCookie); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without cookie: %d", path, w.Code)
		}
		for _, value := range []string{"wrong", testSecret, code} {
			if w := call(s, "GET", path, "", noCookie, func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: value}) }); w.Code != http.StatusUnauthorized {
				t.Errorf("%s with cookie %q: %d", path, value, w.Code)
			}
		}
	}
}

func TestALaterStartProvesItselfWithTheRecordsSecret(t *testing.T) {
	s, _ := testServer(t)
	noCookie := func(r *http.Request) { r.Header.Del("Cookie") }
	w := call(s, "GET", "/api/hello?c=challenge", "", noCookie)
	if w.Code != http.StatusOK || decode(t, w)["proof"] != proof(testSecret, "challenge") {
		t.Fatalf("hello: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), testSecret) {
		t.Fatal("hello reveals the secret")
	}
	bearer := func(secret string) func(*http.Request) {
		return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+secret) }
	}
	w = call(s, "POST", "/api/launch", "{}", noCookie, bearer(testSecret))
	if w.Code != http.StatusOK {
		t.Fatalf("launch: %d %s", w.Code, w.Body)
	}
	if code, _ := decode(t, w)["code"].(string); !s.redeem(code) {
		t.Error("the launch code does not work")
	}
	for _, edit := range []func(*http.Request){bearer("wrong"), func(*http.Request) {}} {
		if w := call(s, "POST", "/api/launch", "{}", noCookie, edit); w.Code != http.StatusUnauthorized {
			t.Errorf("launch without the secret: %d", w.Code)
		}
	}
	if w := call(s, "POST", "/api/launch", "{}", noCookie, bearer(testSecret), func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }); w.Code != http.StatusForbidden {
		t.Errorf("launch from another page: %d", w.Code)
	}
}

func TestANewProjectsParentIsSuggestedNext(t *testing.T) {
	s, _ := testServer(t)
	home, _ := os.UserHomeDir()
	if got := decode(t, call(s, "GET", "/api/app", ""))["parent"]; got != home {
		t.Errorf("first suggestion %v, want the home folder %s", got, home)
	}
	parent := t.TempDir()
	data, _ := json.Marshal(map[string]string{"action": "create", "parent": parent, "name": "library"})
	if w := call(s, "POST", "/api/projects", string(data)); w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	if got := decode(t, call(s, "GET", "/api/app", ""))["parent"]; got != parent {
		t.Errorf("suggestion %v, want %s", got, parent)
	}
}

func TestThePageLoadsOnlyItsOwnFiles(t *testing.T) {
	s, _ := testServer(t)
	w := call(s, "GET", "/", "")
	if w.Code != http.StatusOK || !strings.Contains(strings.ToLower(w.Body.String()), "<!doctype html>") {
		t.Fatalf("page: %d %.80s", w.Code, w.Body)
	}
	if got := w.Header().Get("Content-Security-Policy"); got != csp {
		t.Errorf("CSP %q", got)
	}
	files, err := fs.Sub(ui, "ui")
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".css")) {
			return err
		}
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		for _, ref := range []string{`src="http`, `href="http`, `url(http`, `url("http`, `@import`} {
			if strings.Contains(string(data), ref) {
				t.Errorf("%s loads an external resource: %s", path, ref)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWritesNeedJSONFromTheOwnOrigin(t *testing.T) {
	s, _ := testServer(t)
	w := call(s, "POST", "/api/projects", `{"action":"open","path":"/x"}`, func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") })
	if w.Code != http.StatusForbidden {
		t.Errorf("foreign origin: %d", w.Code)
	}
	w = call(s, "POST", "/api/projects", `{"action":"open","path":"/x"}`, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") })
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("plain text: %d", w.Code)
	}
	w = call(s, "POST", "/api/projects", `{"action":"open","path":"/x","extra":1}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown field: %d", w.Code)
	}
}

func TestOpeningAFolderAddsItsRepositoryOnce(t *testing.T) {
	root := filepath.Join(t.TempDir(), "library")
	s, _ := testServer(t, root)
	body := func(path string) string {
		data, _ := json.Marshal(map[string]string{"action": "open", "path": path})
		return string(data)
	}
	for range 2 {
		w := call(s, "POST", "/api/projects", body(filepath.Join(root, "docs")))
		if w.Code != http.StatusOK {
			t.Fatalf("open: %d %s", w.Code, w.Body)
		}
		p := decode(t, w)["project"].(map[string]any)
		if p["path"] != root || p["name"] != "library" {
			t.Fatalf("project %v", p)
		}
		if st := p["status"].(map[string]any); st["initialized"] != false {
			t.Errorf("status %v", st)
		}
	}
	if list := decode(t, call(s, "GET", "/api/projects", ""))["projects"].([]any); len(list) != 1 {
		t.Errorf("listed %d times", len(list))
	}
	if w := call(s, "POST", "/api/projects", body("relative")); w.Code != http.StatusBadRequest {
		t.Errorf("relative path: %d", w.Code)
	}
	if w := call(s, "POST", "/api/projects", body(t.TempDir())); w.Code != http.StatusBadRequest {
		t.Errorf("no repository: %d", w.Code)
	}
}

func TestCreatingAProjectMakesARepositoryAndRunsInit(t *testing.T) {
	parent := t.TempDir()
	s, _ := testServer(t)
	create := func(name string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(map[string]string{"action": "create", "parent": parent, "name": name})
		return call(s, "POST", "/api/projects", string(data))
	}
	w := create("library")
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	v := decode(t, w)
	if v["ok"] != true || !strings.Contains(v["output"].(string), "manifest.yaml") {
		t.Errorf("init result %v", v)
	}
	p := v["project"].(map[string]any)
	if p["path"] != filepath.Join(parent, "library") || p["status"].(map[string]any)["initialized"] != true {
		t.Errorf("project %v", p)
	}
	if info, err := os.Stat(filepath.Join(parent, "library")); err != nil || !info.IsDir() {
		t.Errorf("folder: %v", err)
	}
	for _, name := range []string{"library", "", ".", "..", "a/b", `a\b`, "x:", " lead", "trail.", ".git", ".markitect", "CON", "con.txt", "nul", "Lpt9", "a\x01b"} {
		if w := create(name); w.Code != http.StatusBadRequest {
			t.Errorf("name %q: %d", name, w.Code)
		}
	}
}

func TestCloningNamesTheFolderAfterTheRepository(t *testing.T) {
	parent := t.TempDir()
	s, _ := testServer(t)
	clone := func(url, name string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(map[string]string{"action": "clone", "url": url, "parent": parent, "name": name})
		return call(s, "POST", "/api/projects", string(data))
	}
	w := clone("https://example.com/org/library.git", "")
	if w.Code != http.StatusOK {
		t.Fatalf("clone: %d %s", w.Code, w.Body)
	}
	if p := decode(t, w)["project"].(map[string]any); p["path"] != filepath.Join(parent, "library") {
		t.Errorf("project %v", p)
	}
	for _, url := range []string{"", "-u", "https://example.com/missing.git"} {
		if w := clone(url, ""); w.Code != http.StatusBadRequest {
			t.Errorf("url %q: %d", url, w.Code)
		}
	}
	for url, want := range map[string]string{
		"git@github.com:org/repo.git": "repo",
		"https://host/a/repo/":        "repo",
		`C:\src\repo`:                 "repo",
		"../repo.git":                 "repo",
	} {
		if got := nameFromURL(url); got != want {
			t.Errorf("nameFromURL(%q) = %q", url, got)
		}
	}
}

func TestInitAndRemoveWorkOnAListedProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "library")
	s, _ := testServer(t, root)
	data, _ := json.Marshal(map[string]string{"action": "open", "path": root})
	id := decode(t, call(s, "POST", "/api/projects", string(data)))["project"].(map[string]any)["id"].(string)

	w := call(s, "POST", "/api/projects/"+id+"/init", "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("init: %d %s", w.Code, w.Body)
	}
	if st := decode(t, w)["project"].(map[string]any)["status"].(map[string]any); st["initialized"] != true || st["check"] == nil {
		t.Errorf("status after init %v", st)
	}
	if w := call(s, "DELETE", "/api/projects/"+id, ""); w.Code != http.StatusNoContent {
		t.Fatalf("remove: %d", w.Code)
	}
	if w := call(s, "GET", "/api/projects/"+id, ""); w.Code != http.StatusNotFound {
		t.Errorf("removed project: %d", w.Code)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/init", "{}"); w.Code != http.StatusNotFound {
		t.Errorf("init of a removed project: %d", w.Code)
	}
}

func TestAProjectThatCannotBeReadSaysWhy(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	s, _ := testServer(t)
	if _, err := s.store.Add(gone); err != nil {
		t.Fatal(err)
	}
	list := decode(t, call(s, "GET", "/api/projects", ""))["projects"].([]any)
	st := list[0].(map[string]any)["status"].(map[string]any)
	if !strings.Contains(fmt.Sprint(st["problem"]), "not inside a Git repository") {
		t.Errorf("status %v", st)
	}
}
