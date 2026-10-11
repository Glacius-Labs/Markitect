package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// verifyLimit bounds the fixed-revision verification, so a rejected scripted
// response fails the smoke instead of leaving it waiting.
const verifyLimit = 5 * time.Minute

const fullVerifyKind = "projectrun-full-verify/v1"

var projectWorldChecks = []string{cancellationCheck, `["project.markitect.example.org/v1alpha1","Check","engineering","readme-guide"]`}

// verifyFixedRevision replays full verification of a committed revision of
// the project-world example. No model runs: config points every role at
// markitect-exchange-executor, and the smoke answers each Manager's audit
// request as a scripted auditor that passes every required subject. The Host
// still runs the declared checks, validates every response and requires an
// assessment from every Manager.
func (s *smoke) verifyFixedRevision() error {
	repo, err := s.gitRepo("verify-fixed-revision", "feature/verify")
	if err != nil {
		return err
	}
	if err := os.CopyFS(repo, os.DirFS(filepath.Join(s.repo, "examples", "project-world"))); err != nil {
		return err
	}
	if _, err := s.commit(repo, "Freeze the project-world fixture"); err != nil {
		return err
	}
	exchange := filepath.Join(s.work, "exchange")
	if err := os.MkdirAll(exchange, 0o755); err != nil {
		return err
	}
	cli := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect"), args...) }
	config := []string{"config", "--repo", repo, "--provider", "process", "--model", "scripted-smoke",
		"--provider-executable", s.tool("markitect-exchange-executor"), "--provider-arg", "--dir", "--provider-arg", exchange,
		"--provider-version", "exchange/smoke", "--cost-mode", "unmetered", "--max-cost-micros", "1000000"}
	for _, command := range [][]string{config, {"docs", "--repo", repo}} {
		out, err := cli(command...)
		if err != nil {
			return err
		}
		var plan struct {
			Digest   string
			EditPlan struct{ Digest string }
		}
		if err := decode(out, &plan); err != nil {
			return err
		}
		digest := plan.Digest
		if digest == "" {
			digest = plan.EditPlan.Digest
		}
		if !isDigest(strings.TrimPrefix(digest, "sha256:")) {
			return fmt.Errorf("%s preview has no digest: %s", command[0], tail(out))
		}
		if _, err := cli(append(command, "--expect", digest, "--write")...); err != nil {
			return err
		}
	}
	revision, err := s.commit(repo, "Configure the scripted runtime")
	if err != nil {
		return err
	}

	auditor := &scriptedAuditor{dir: exchange, stop: make(chan struct{})}
	done := make(chan struct{})
	go func() { auditor.serve(); close(done) }()
	out, err := s.runLimited(repo, verifyLimit, s.tool("markitect"), "verify", "--repo", repo, "--revision", revision, "--execute")
	close(auditor.stop)
	<-done
	if problems := auditor.problems(); len(problems) > 0 {
		return fmt.Errorf("scripted auditor: %s", strings.Join(problems, "; "))
	}
	if err != nil {
		return err
	}
	var report struct {
		Status   string
		Checks   []struct{ ID, Outcome, Status string }
		Managers []struct{ Status string }
	}
	if err := decode(out, &report); err != nil {
		return err
	}
	passedChecks := []string{}
	for _, check := range report.Checks {
		if check.Outcome == "passed" || check.Status == "passed" {
			passedChecks = append(passedChecks, check.ID)
		}
	}
	slices.Sort(passedChecks)
	if report.Status != "passed" || !slices.Equal(passedChecks, projectWorldChecks) {
		return fmt.Errorf("verify did not pass both declared checks: %s", tail(out))
	}
	if len(report.Managers) != 6 || auditor.answered() != 6 {
		return fmt.Errorf("verify assessed %d Managers with %d scripted audits, want 6", len(report.Managers), auditor.answered())
	}
	for _, manager := range report.Managers {
		if manager.Status != "passed" {
			return fmt.Errorf("a Manager assessment did not pass: %s", tail(out))
		}
	}
	return nil
}

// scriptedAuditor answers markitect-exchange-executor requests in dir.
type scriptedAuditor struct {
	dir  string
	stop chan struct{}

	mu    sync.Mutex
	seen  map[string]bool
	count int
	issue []string
}

func (a *scriptedAuditor) serve() {
	a.seen = map[string]bool{}
	for {
		a.poll()
		select {
		case <-a.stop:
			a.poll()
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (a *scriptedAuditor) poll() {
	requests, _ := filepath.Glob(filepath.Join(a.dir, "*", "request.json"))
	for _, request := range requests {
		invocation := filepath.Dir(request)
		if rejected, err := os.ReadFile(filepath.Join(invocation, "response-error.txt")); err == nil {
			a.note(fmt.Sprintf("response for %s rejected: %s", filepath.Base(invocation), strings.TrimSpace(string(rejected))))
		}
		a.mu.Lock()
		seen := a.seen[invocation]
		a.seen[invocation] = true
		a.mu.Unlock()
		if seen {
			continue
		}
		if err := a.answer(request); err != nil {
			a.note(err.Error())
		}
	}
}

func (a *scriptedAuditor) answer(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var invocation struct {
		Request struct {
			Role    string
			Context struct {
				Kind             string
				RequiredSubjects []string
			}
		}
	}
	if err := json.Unmarshal(data, &invocation); err != nil {
		return fmt.Errorf("unreadable request %s: %v", path, err)
	}
	role, kind := invocation.Request.Role, invocation.Request.Context.Kind
	response := map[string]any{"candidateFiles": []any{}, "evidenceRefs": []any{}, "verifierObservations": []any{}, "uncertainty": []any{}}
	if role == "executor" && kind == fullVerifyKind {
		assessments := []map[string]string{}
		for _, subject := range invocation.Request.Context.RequiredSubjects {
			assessments = append(assessments, map[string]string{"subject": subject, "outcome": "pass", "detail": "Scripted smoke audit."})
		}
		response["outcome"] = "proposed"
		response["reportJson"] = map[string]any{"status": "pass", "summary": "Scripted smoke audit.", "assessments": assessments, "findings": []any{}, "counterexamples": []any{}}
	} else {
		// Fail the invocation instead of leaving it waiting.
		response["outcome"] = "failed"
		a.note(fmt.Sprintf("unexpected %s request of kind %q", role, kind))
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return err
	}
	// The executor polls for response.json; rename it into place whole.
	staged := filepath.Join(filepath.Dir(path), "response.staged")
	if err := os.WriteFile(staged, encoded, 0o644); err != nil {
		return err
	}
	if err := os.Rename(staged, filepath.Join(filepath.Dir(path), "response.json")); err != nil {
		return err
	}
	if response["outcome"] == "proposed" {
		a.mu.Lock()
		a.count++
		a.mu.Unlock()
	}
	return nil
}

func (a *scriptedAuditor) note(problem string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !slices.Contains(a.issue, problem) {
		a.issue = append(a.issue, problem)
	}
}

func (a *scriptedAuditor) problems() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.issue)
}

func (a *scriptedAuditor) answered() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.count
}

// runLimited is run with a time limit.
func (s *smoke) runLimited(dir string, limit time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+s.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("no result within %s: %w", limit, ctx.Err())
		}
		return stdout.String(), fmt.Errorf("%s %s: %v\nstdout: %s\nstderr: %s", filepath.Base(name), strings.Join(args, " "), err, tail(stdout.String()), tail(stderr.String()))
	}
	return stdout.String(), nil
}
