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
	"slices"
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
	return s.runChecks(append(checks, s.distributionChecks(pkg, filepath.Join(s.repo, "examples", "minimal"))...))
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
			if _, err := s.run(s.repo, nil, "go", "run", "./src/cmd/markitect-legacy", "bundle", "--repo", s.repo, "--revision", revision, "--output", bundle); err != nil {
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
	return s.runChecks(append(checks, s.distributionChecks(consumer, consumer)...))
}

// distributionChecks checks the tools built into s.bin and the bootstrap in
// bootstrapDir: an unpacked package or a repository with an installed bundle.
func (s *smoke) distributionChecks(bootstrapDir, minimalRepo string) []check {
	checks := []check{
		{"bootstrap prints third-party notices", func() error { return s.notices(bootstrapDir) }},
		{"version names the build platform", s.version},
		{"init previews, refuses a stale digest and writes", func() error { return s.initProject(bootstrapDir) }},
		{"onboard previews, writes and is idempotent", s.onboard},
		{"a policy edit has the expected fixed-revision impact", s.policyEditImpact},
		{"a fixed revision passes full verification with a scripted auditor", s.verifyFixedRevision},
		{"selective adoption replays with the packaged CLI", s.selectiveAdoption},
		{"artifact accounting with the packaged checker", func() error {
			_, err := s.run(s.repo, nil, s.tool("markitect-check-artifacts"), "--repo", s.repo, "--config", "markitect-artifacts.yaml")
			return err
		}},
	}
	return append(checks, s.legacySteps(minimalRepo)...)
}

func (s *smoke) testBootstrap(dir string) error {
	_, err := s.run(dir, nil, "go", "test", ".markitect/bootstrap/run.go", ".markitect/bootstrap/run_test.go")
	return err
}

// installBundle installs the bundle into a committed copy of the minimal
// example, as an adopting repository does.
func (s *smoke) installBundle(consumer, bundle, sum string) error {
	if _, err := s.gitRepo(filepath.Base(consumer), "feature/release-smoke"); err != nil {
		return err
	}
	if err := os.CopyFS(consumer, os.DirFS(filepath.Join(s.repo, "examples", "minimal"))); err != nil {
		return err
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "--quiet", "-m", "Create isolated Markitect release smoke fixture"}} {
		if _, err := s.run(consumer, nil, "git", args...); err != nil {
			return err
		}
	}
	_, err := s.run(s.repo, nil, "go", "run", "./src/cmd/markitect-legacy", "install", "--repo", consumer, "--bundle", bundle, "--sha256", sum, "--write")
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
	if _, err := s.run(s.repo, nil, "go", "run", "./src/cmd/markitect-legacy", "package", "--repo", s.repo, "--output", pkg); err != nil {
		return err
	}
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
	for _, name := range []string{"markitect", "markitect-legacy", "markitect-adapter-dotnet", "markitect-check-artifacts", "markitect-exchange-executor"} {
		if _, err := s.run(source, nil, "go", "build", "-trimpath", "-buildvcs=false", "-o", s.tool(name), "./src/cmd/"+name); err != nil {
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

var initPaths = []string{".markitect/.gitignore", ".markitect/model/manager.yaml", ".markitect/project.yaml", ".markitect/runtime.yaml", "docs/markitect/project.md"}

const projectAPI = "project.markitect.example.org/v1alpha1"

// initProject drives init through the bootstrap, as an adopting repository
// does: a read-only preview, a refused write with a stale digest, the write
// with the previewed digest, and a passing check.
func (s *smoke) initProject(pkg string) error {
	repo, err := s.gitRepo("init-project", "feature/authoring")
	if err != nil {
		return err
	}
	bootstrap := func(args ...string) (string, error) {
		return s.run(pkg, nil, "go", append([]string{"run", ".markitect/bootstrap/run.go"}, args...)...)
	}
	out, err := bootstrap("init", "--repo", repo, "--name", "started-project")
	if err != nil {
		return err
	}
	var preview struct {
		APIVersion, Name, Digest string
		Files                    []struct{ Path string }
	}
	if err := decode(out, &preview); err != nil {
		return err
	}
	if preview.APIVersion != projectAPI || preview.Name != "started-project" || !isDigest(preview.Digest) || !slices.Equal(paths(preview.Files), initPaths) {
		return fmt.Errorf("unexpected init preview: %s", out)
	}
	if err := requireAbsent(repo, initPaths...); err != nil {
		return fmt.Errorf("the preview wrote files: %w", err)
	}
	if _, err := bootstrap("init", "--repo", repo, "--name", "started-project", "--expect", strings.Repeat("0", 64), "--write"); err == nil {
		return errors.New("init accepted a stale digest")
	}
	if err := requireAbsent(repo, initPaths...); err != nil {
		return fmt.Errorf("a refused write changed the repository: %w", err)
	}
	if out, err = bootstrap("init", "--repo", repo, "--name", "started-project", "--expect", preview.Digest, "--write"); err != nil {
		return err
	}
	var written struct {
		APIVersion, Name string
		Written          []string
	}
	if err := decode(out, &written); err != nil {
		return err
	}
	if written.APIVersion != projectAPI || written.Name != "started-project" || !slices.Equal(written.Written, initPaths) {
		return fmt.Errorf("unexpected init write: %s", out)
	}
	if err := requirePresent(repo, initPaths...); err != nil {
		return err
	}
	if err := requireAbsent(repo, "markitect.yaml"); err != nil {
		return fmt.Errorf("init created the retired top-level model: %w", err)
	}
	return s.checkConforms(func() (string, error) { return bootstrap("check", "--repo", repo) })
}

const onboardingAPI = "markitect.example.org/project-onboarding/v1alpha1"

// onboard installs the agent guidance into a new project with the built CLI:
// preview, digest-checked write, an unchanged second preview and a check.
func (s *smoke) onboard() error {
	repo, err := s.gitRepo("onboard-project", "feature/onboarding")
	if err != nil {
		return err
	}
	cli := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect"), args...) }
	out, err := cli("init", "--repo", repo, "--name", "onboarded-project")
	if err != nil {
		return err
	}
	var plan struct {
		APIVersion, Digest string
		Files              []struct{ Path, Action string }
	}
	if err := decode(out, &plan); err != nil {
		return err
	}
	if _, err := cli("init", "--repo", repo, "--name", "onboarded-project", "--expect", plan.Digest, "--write"); err != nil {
		return err
	}
	if out, err = cli("onboard", "--repo", repo, "--provider", "codex"); err != nil {
		return err
	}
	plan.Files = nil
	if err := decode(out, &plan); err != nil {
		return err
	}
	planned := []string{}
	for _, file := range plan.Files {
		planned = append(planned, file.Path)
	}
	if plan.APIVersion != onboardingAPI || !isDigest(plan.Digest) || !slices.Contains(planned, "AGENTS.md") || !slices.Contains(planned, ".markitect/workflows/model-first.md") {
		return fmt.Errorf("unexpected onboarding preview: %s", out)
	}
	if err := requireAbsent(repo, planned...); err != nil {
		return fmt.Errorf("the preview wrote files: %w", err)
	}
	if out, err = cli("onboard", "--repo", repo, "--provider", "codex", "--expect", plan.Digest, "--write"); err != nil {
		return err
	}
	var written struct{ Written []string }
	if err := decode(out, &written); err != nil {
		return err
	}
	if !slices.Equal(written.Written, planned) {
		return fmt.Errorf("onboarding wrote %v, planned %v", written.Written, planned)
	}
	if err := requirePresent(repo, planned...); err != nil {
		return err
	}
	if out, err = cli("onboard", "--repo", repo, "--provider", "codex"); err != nil {
		return err
	}
	plan.Files = nil
	if err := decode(out, &plan); err != nil {
		return err
	}
	for _, file := range plan.Files {
		if file.Action != "unchanged" {
			return fmt.Errorf("a second onboarding preview plans %s for %s", file.Action, file.Path)
		}
	}
	return s.checkConforms(func() (string, error) { return cli("check", "--repo", repo) })
}

func (s *smoke) checkSucceeds(check func() (string, error)) error {
	out, err := check()
	if err != nil {
		return err
	}
	var report struct{ Status string }
	if err := decode(out, &report); err != nil {
		return err
	}
	if report.Status != "succeeded" {
		return fmt.Errorf("check did not succeed: %s", tail(out))
	}
	return nil
}

func (s *smoke) checkConforms(check func() (string, error)) error {
	out, err := check()
	if err != nil {
		return err
	}
	var report struct {
		Status   string
		Coverage struct{ Conforming bool }
	}
	if err := decode(out, &report); err != nil {
		return err
	}
	if report.Status != "succeeded" || !report.Coverage.Conforming {
		return fmt.Errorf("check did not succeed with conforming coverage: %s", out)
	}
	return nil
}

func (s *smoke) selectiveAdoption() error {
	legacy := s.tool("markitect-legacy")
	if _, err := s.run(s.repo, []string{"MARKITECT_ADOPTION_BINARY=" + legacy}, "go", "test", "./src/harness/examples", "-run", "^TestSelectiveAdoptionCLI$", "-count=1"); err != nil {
		return err
	}
	_, err := s.run(s.repo, []string{"MARKITECT_POLICY_ANALYSIS_BINARY=" + legacy}, "go", "test", "./src/internal/host/cli", "-count=1",
		"-run", "^(TestPolicyFailureAnalysisCLIIsExplicitAndKeepsAcceptanceStrict|TestPolicyFailingWorkingTreeRejectsReconciliationAndWritersWithoutMutation|TestPolicyFailureAnalysisBlocksStructuralDiagnostics)$")
	return err
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
	cmd := exec.Command(name, args...)
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

func paths(files []struct{ Path string }) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
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
