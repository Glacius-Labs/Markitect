package remoteserver

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/cgi"
	"os"
	"runtime"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// maxRequest bounds one request body; receive.maxInputSize bounds the pack.
const maxRequest = 1 << 30

// Server serves the Run's bare repository over Git's smart HTTP protocol.
// Every request needs an agent's credentials; pushes are taken one at a time,
// so the hook decides each against a repository no other push changes.
type Server struct {
	Layout Layout
	Plan   *runremote.Plan
	Git    string
	// Env is added to the environment of git http-backend and the hook.
	Env []string

	push sync.Mutex
}

// routes are the only paths the server answers besides /healthz.
var routes = map[string]string{
	"/" + repoName + "/info/refs":        http.MethodGet,
	"/" + repoName + "/git-upload-pack":  http.MethodPost,
	"/" + repoName + "/git-receive-pack": http.MethodPost,
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		if !Seeded(s.Layout) {
			http.Error(w, "not seeded", http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "ok\n")
		return
	}
	method, ok := routes[r.URL.Path]
	if !ok || r.Method != method {
		http.NotFound(w, r)
		return
	}
	user, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Markitect Run"`)
		http.Error(w, "credentials required", http.StatusUnauthorized)
		return
	}
	// The backend and the hook learn the user from REMOTE_USER, never the token.
	r.Header.Del("Authorization")
	if r.Method == http.MethodPost {
		body, err := s.buffer(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		defer func() {
			body.Close()
			os.Remove(body.Name())
		}()
		if r.URL.Path == "/"+repoName+"/git-receive-pack" {
			s.push.Lock()
			defer s.push.Unlock()
		}
	}
	h := &cgi.Handler{
		Path: s.Git,
		Args: []string{"http-backend"},
		Root: "/",
		Env: append(append([]string{
			"GIT_PROJECT_ROOT=" + s.Layout.Root,
			"GIT_HTTP_EXPORT_ALL=1",
			"REMOTE_USER=" + user,
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + s.Layout.Root,
		}, gitConfigEnv()...), s.Env...),
	}
	if runtime.GOOS == "windows" {
		h.InheritEnv = []string{"TEMP", "TMP"}
	}
	h.ServeHTTP(w, r)
}

// authenticate returns the agent whose token the request carries.
func (s *Server) authenticate(r *http.Request) (string, bool) {
	user, token, ok := r.BasicAuth()
	if !ok {
		return "", false
	}
	a, known := s.Plan.Agent(user)
	sum := sha256.Sum256([]byte(token))
	want, err := hex.DecodeString(a.TokenSHA256)
	if err != nil || len(want) != len(sum) {
		want = make([]byte, len(sum))
	}
	return user, subtle.ConstantTimeCompare(sum[:], want) == 1 && known
}

// buffer stores a request body in a file, so the backend gets a body of known
// length whatever transfer encoding the client used.
func (s *Server) buffer(r *http.Request) (*os.File, error) {
	f, err := os.CreateTemp(s.Layout.Tmp(), "request-")
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(f, io.LimitReader(r.Body, maxRequest+1))
	if err == nil && n > maxRequest {
		err = errors.New("request body too large")
	}
	if err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	r.Body, r.ContentLength = f, n
	r.TransferEncoding = nil
	r.Header.Del("Transfer-Encoding")
	return f, nil
}
