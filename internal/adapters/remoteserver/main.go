package remoteserver

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// DefaultRoot is the remote's directory in its container, and Port its port.
const (
	DefaultRoot = "/srv/markitect"
	Port        = 8080
)

const usage = `usage: markitect-remote COMMAND [--root DIR]

  seed     create the bare repository from a bundle on stdin
  serve    serve the repository on port 8080
  hook     the pre-receive hook (run by Git)
  ready    wait until the server answers
  check    run the required checks on --commit from a bundle on stdin;
           print the results as JSON (the host's landing check)
  start    move --branch, still at the base, to --commit of its parent's
           branch before its Manager starts
  integrate push --commit from a bundle on stdin to --branch as the host,
           through the hook; print the decision as JSON
  export   write a bundle of every branch to stdout
  log      write the decision log to stdout
`

// Main runs one command of markitect-remote, the program of the Run
// container, and returns its exit status.
func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", DefaultRoot, "the remote's directory")
	listen := fs.String("listen", fmt.Sprintf(":%d", Port), "serve: listen address")
	wait := fs.Duration("timeout", 30*time.Second, "ready: how long to wait")
	commit := fs.String("commit", "", "check, start, integrate: the commit")
	branch := fs.String("branch", "", "start, integrate: the Manager's branch")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	l := Layout{Root: *root}
	err := func() error {
		switch args[0] {
		case "hook":
			if Hook(l, stdin, stderr) != 0 {
				return errRefused
			}
			return nil
		case "seed":
			git, err := exec.LookPath("git")
			if err != nil {
				return err
			}
			self, err := os.Executable()
			if err != nil {
				return err
			}
			return Seed(l, git, stdin, []string{self})
		case "serve":
			return serve(ctx, l, *listen)
		case "check":
			return landing(l, *commit, stdin, stdout, stderr)
		case "start":
			git, err := exec.LookPath("git")
			if err != nil {
				return err
			}
			return StartFrom(l, git, *branch, *commit)
		case "integrate":
			return integrate(l, *branch, *commit, stdin, stdout)
		case "ready":
			return ready(ctx, *listen, *wait)
		case "export":
			git, err := exec.LookPath("git")
			if err != nil {
				return err
			}
			return Export(l, git, stdout)
		case "log":
			f, err := os.Open(l.Log())
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(stdout, f)
			return err
		}
		fmt.Fprint(stderr, usage)
		return errUsage
	}()
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errUsage):
		return 2
	case !errors.Is(err, errRefused):
		fmt.Fprintf(stderr, "markitect-remote %s: %v\n", args[0], err)
	}
	return 1
}

var (
	errUsage   = errors.New("usage")
	errRefused = errors.New("refused")
)

func serve(ctx context.Context, l Layout, listen string) error {
	plan, err := LoadPlan(l)
	if err != nil {
		return err
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(l.Tmp(), 0o700); err != nil {
		return err
	}
	if isolated() {
		if err := protect(); err != nil {
			return fmt.Errorf("protect the remote from the memory limit: %w", err)
		}
	}
	srv := &http.Server{Addr: listen, Handler: &Server{Layout: l, Plan: plan, Git: git}, ReadHeaderTimeout: 30 * time.Second}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func ready(ctx context.Context, listen string, wait time.Duration) error {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	url := "http://127.0.0.1:" + port + "/healthz"
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the remote did not answer at %s within %s", url, wait)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
