// Command smoke builds markitect from the repository and drives its commands
// over a small project with exact assertions. It checks the version, init,
// check, tidy, doc, list, vet, status and mark, and the refusal of a spec that
// does not compile. It starts no agent and needs no Git. The same checks run on
// every operating system, so CI needs no shell-specific copies.
//
//	go run ./tools/smoke [-repo DIR] [-work DIR]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	flags := flag.NewFlagSet("smoke", flag.ExitOnError)
	repo := flags.String("repo", ".", "Markitect source checkout")
	work := flags.String("work", "", "empty working directory (default: a new temporary directory)")
	flags.Parse(os.Args[1:])
	s, err := newSmoke(*repo, *work, os.Stdout)
	if err == nil {
		err = s.checks()
	}
	if err != nil {
		fmt.Fprintf(os.Stdout, "smoke: FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "smoke: passed")
}

type smoke struct {
	repo, work string
	bin        string // the markitect built from the repository
	out        io.Writer
}

func newSmoke(repo, work string, out io.Writer) (*smoke, error) {
	repo, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	if work == "" {
		if work, err = os.MkdirTemp("", "markitect-smoke-"); err != nil {
			return nil, err
		}
	} else if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, err
	}
	if work, err = filepath.Abs(work); err != nil {
		return nil, err
	}
	return &smoke{repo: repo, work: work, bin: filepath.Join(work, "bin"), out: out}, nil
}

type check struct {
	name string
	run  func() error
}

// checks runs the checks in order, reporting each outcome and duration, and
// stops at the first failure because later checks build on earlier ones.
func (s *smoke) checks() error {
	fmt.Fprintf(s.out, "smoke: repo %s\nsmoke: work %s\n", s.repo, s.work)
	for _, c := range []check{
		{"build markitect from the repository", s.build},
		{"markitect reports a build version", s.version},
		{"spec commands over a small project, and refusals", s.specCommands},
	} {
		start := time.Now()
		if err := c.run(); err != nil {
			fmt.Fprintf(s.out, "smoke: %s ... FAIL (%.1fs)\n", c.name, time.Since(start).Seconds())
			return fmt.Errorf("%s: %w", c.name, err)
		}
		fmt.Fprintf(s.out, "smoke: %s ... ok (%.1fs)\n", c.name, time.Since(start).Seconds())
	}
	return nil
}

// build builds markitect from the repository into the work folder, so that
// every later check drives the built binary.
func (s *smoke) build() error {
	_, err := s.run(s.repo, nil, "go", "build", "-trimpath", "-buildvcs=false", "-o", s.tool("markitect"), "./cmd/markitect")
	return err
}

func (s *smoke) tool(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(s.bin, name)
}

// run executes a command and returns its standard output. A failure reports
// the exit status and the tail of both output streams.
func (s *smoke) run(dir string, env []string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), "PATH="+s.bin+string(os.PathListSeparator)+os.Getenv("PATH")), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s %s: %v\nstdout: %s\nstderr: %s", filepath.Base(name), strings.Join(args, " "), err, tail(stdout.String()), tail(stderr.String()))
	}
	return stdout.String(), nil
}

func tail(text string) string {
	const limit = 2000
	text = strings.TrimSpace(text)
	if len(text) > limit {
		return "..." + text[len(text)-limit:]
	}
	return text
}

func decode(out string, value any) error {
	if err := json.Unmarshal([]byte(out), value); err != nil {
		return fmt.Errorf("output is not the expected JSON: %v\n%s", err, tail(out))
	}
	return nil
}

func requireContains(out string, wanted ...string) error {
	for _, want := range wanted {
		if !strings.Contains(out, want) {
			return fmt.Errorf("output lacks %q:\n%s", want, tail(out))
		}
	}
	return nil
}

func requirePresent(root string, names ...string) error {
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s is missing", name)
		}
	}
	return nil
}
