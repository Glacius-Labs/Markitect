// Command smoke checks Markitect's packaged distribution end to end. It
// packages the source (or, for a release, builds and installs the bundle),
// tests the standalone bootstrap, builds the command-line tools from the
// packaged source and drives them through fixed scenarios with exact
// assertions. The same checks run on every operating system, so CI and the
// release workflow need no shell-specific copies (backlog CI-03).
//
//	go run ./tools/smoke [source] [-repo DIR] [-work DIR]
//	go run ./tools/smoke bundle [-repo DIR] [-work DIR] -revision SHA [-expect-sha256 HEX]
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func main() {
	mode, args := "source", os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		mode, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("smoke "+mode, flag.ExitOnError)
	repo := flags.String("repo", ".", "Markitect source checkout")
	work := flags.String("work", "", "empty working directory (default: a new temporary directory)")
	revision := flags.String("revision", "HEAD", "bundle: the commit to bundle")
	expectSHA := flags.String("expect-sha256", "", "bundle: the bundle's SHA-256 from the release preflight")
	flags.Parse(args)
	s, err := newSmoke(*repo, *work, os.Stdout)
	if err == nil {
		switch mode {
		case "source":
			err = s.source()
		case "bundle":
			err = s.bundle(*revision, *expectSHA)
		default:
			err = fmt.Errorf("unknown mode %q; want source or bundle", mode)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stdout, "smoke: FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "smoke: passed")
}

type smoke struct {
	repo, work string
	bin        string // the tools built from the packaged source
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

// runChecks runs checks in order, reporting each outcome and duration, and
// stops at the first failure because later checks build on earlier ones.
func (s *smoke) runChecks(checks []check) error {
	fmt.Fprintf(s.out, "smoke: repo %s\nsmoke: work %s\n", s.repo, s.work)
	head, err := s.run(s.repo, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	status, err := s.run(s.repo, nil, "git", "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return err
	}
	fmt.Fprintf(s.out, "smoke: source HEAD %s; working changes %t\n", strings.TrimSpace(head), strings.TrimSpace(status) != "")
	for _, c := range checks {
		start := time.Now()
		if err := c.run(); err != nil {
			fmt.Fprintf(s.out, "smoke: %s ... FAIL (%.1fs)\n", c.name, time.Since(start).Seconds())
			return fmt.Errorf("%s: %w", c.name, err)
		}
		fmt.Fprintf(s.out, "smoke: %s ... ok (%.1fs)\n", c.name, time.Since(start).Seconds())
	}
	return nil
}

// source checks the package made from the checkout.
func (s *smoke) source() error {
	pkg := filepath.Join(s.work, "package")
	checks := []check{
		{"package the source", func() error { return s.packageSource(pkg) }},
		{"test the standalone bootstrap", func() error { return s.testBootstrap(pkg) }},
		{"build the tools from the packaged source", func() error {
			return s.buildTools(filepath.Join(pkg, ".markitect", "tool", "source.zip"))
		}},
	}
	return s.runChecks(append(checks, s.distributionChecks(pkg)...))
}

// bundle checks the release bundle of revision as a consumer installs it. A
// non-empty expectSHA must match, which shows the bundle is reproducible
// across release jobs.
func (s *smoke) bundle(revision, expectSHA string) error {
	bundle := filepath.Join(s.work, "markitect-bundle.zip")
	consumer := filepath.Join(s.work, "consumer")
	sum := ""
	checks := []check{
		{"bundle the revision", func() error {
			if _, err := s.run(s.repo, nil, "go", "run", "./cmd/markitect-release", "bundle", "--repo", s.repo, "--revision", revision, "--output", bundle); err != nil {
				return err
			}
			var err error
			if sum, err = sha256File(bundle); err != nil {
				return err
			}
			if expectSHA != "" && sum != expectSHA {
				return fmt.Errorf("bundle SHA-256 %s differs from the preflight's %s; the bundle is not reproducible", sum, expectSHA)
			}
			return nil
		}},
		{"install the bundle into a consumer repository", func() error { return s.installBundle(consumer, bundle, sum) }},
		{"test the installed bootstrap", func() error { return s.testBootstrap(consumer) }},
		{"build the tools from the installed source", func() error {
			return s.buildTools(filepath.Join(consumer, ".markitect", "tool", "source.zip"))
		}},
	}
	return s.runChecks(append(checks, s.distributionChecks(consumer)...))
}

// distributionChecks checks the tools built into s.bin and the bootstrap in
// bootstrapDir: an unpacked package or a repository with an installed bundle.
func (s *smoke) distributionChecks(bootstrapDir string) []check {
	return []check{
		{"bootstrap prints third-party notices", func() error { return s.notices(bootstrapDir) }},
		{"packaged CLI reports a build version", s.version},
		{"real Library M1/M2 read-only preparation and refusals", s.libraryPreparation},
		{"actual self-model structural accounting", func() error {
			out, err := s.run(s.repo, nil, s.tool("markitect"), "check", "--repo", s.repo)
			if err != nil {
				return err
			}
			var report struct{ Status string }
			if err := decode(out, &report); err != nil {
				return err
			}
			if report.Status != "passed" {
				return fmt.Errorf("self-model preparation did not pass: %s", tail(out))
			}
			return nil
		}},
	}
}

func (s *smoke) testBootstrap(dir string) error {
	_, err := s.run(dir, nil, "go", "test", ".markitect/bootstrap/run.go", ".markitect/bootstrap/run_test.go")
	return err
}

// installBundle installs the bundle into a committed consumer repository.
func (s *smoke) installBundle(consumer, bundle, sum string) error {
	if _, err := s.gitRepo(filepath.Base(consumer), "feature/release-smoke"); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(consumer, "README.md"), []byte("# Release smoke consumer\n"), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "--quiet", "-m", "Create isolated Markitect release smoke fixture"}} {
		if _, err := s.run(consumer, nil, "git", args...); err != nil {
			return err
		}
	}
	_, err := s.run(s.repo, nil, "go", "run", "./cmd/markitect-release", "install", "--repo", consumer, "--bundle", bundle, "--sha256", sum, "--write")
	return err
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s *smoke) packageSource(pkg string) error {
	if _, err := s.run(s.repo, nil, "go", "run", "./cmd/markitect-release", "package", "--repo", s.repo, "--output", pkg); err != nil {
		return err
	}
	digest, err := sha256File(filepath.Join(pkg, ".markitect", "tool", "source.zip"))
	if err != nil {
		return err
	}
	fmt.Fprintf(s.out, "smoke: source archive sha256:%s\n", digest)
	bootstrap := filepath.Join(pkg, ".markitect", "bootstrap")
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		return err
	}
	for source, target := range map[string]string{"run-markitect.go": "run.go", "run-markitect_test.go": "run_test.go"} {
		data, err := os.ReadFile(filepath.Join(s.repo, "integration", source))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(bootstrap, target), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// buildTools builds the command-line tools from a packaged source archive, so
// every later step exercises exactly the packaged bytes.
func (s *smoke) buildTools(archive string) error {
	source := filepath.Join(s.work, "packaged-source")
	if err := unzip(archive, source); err != nil {
		return err
	}
	for _, name := range []string{"markitect", "markitect-release", "markitect-check-architecture", "markitect-check-modules", "markitect-agentd"} {
		if _, err := s.run(source, nil, "go", "build", "-trimpath", "-buildvcs=false", "-o", s.tool(name), "./cmd/"+name); err != nil {
			return err
		}
	}
	return nil
}

func (s *smoke) notices(pkg string) error {
	out, err := s.run(pkg, nil, "go", "run", ".markitect/bootstrap/run.go", "licenses")
	if err != nil {
		return err
	}
	return requireContains(out, "Copyright 2009 The Go Authors.", "END OF TERMS AND CONDITIONS")
}

// gitRepo creates an empty repository on a feature branch, where Markitect
// writes are allowed.
func (s *smoke) gitRepo(name, branch string) (string, error) {
	repo := filepath.Join(s.work, name)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return "", err
	}
	for _, args := range [][]string{
		{"init", "--quiet", "--initial-branch=" + branch},
		{"config", "user.name", "Markitect Smoke"},
		{"config", "user.email", "markitect-smoke@example.invalid"},
		{"config", "core.autocrlf", "false"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := s.run(repo, nil, "git", args...); err != nil {
			return "", err
		}
	}
	return repo, nil
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

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func isDigest(value string) bool { return digestPattern.MatchString(value) }

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

func requireAbsent(root string, names ...string) error {
	for _, name := range names {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name))); err == nil {
			return fmt.Errorf("%s exists", name)
		}
	}
	return nil
}

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		path := filepath.Join(target, filepath.FromSlash(file.Name))
		if rel, err := filepath.Rel(target, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive entry %q leaves the target", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := extract(file, path); err != nil {
			return err
		}
	}
	return nil
}

func extract(file *zip.File, path string) error {
	in, err := file.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	return errors.Join(copyErr, out.Close())
}
