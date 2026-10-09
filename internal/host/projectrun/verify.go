package projectrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

const maxCheckOutputBytes = 256 << 10

// Verify executes every planned check as a separate direct argv process in a
// fresh materialized candidate. An optional AI verifier is another invocation
// and cannot replace these concrete checks.
func Verify(ctx context.Context, host Host, invoker Invoker, root, runID string) (VerifyReport, error) {
	var out VerifyReport
	if host.Load == nil || host.FromSnapshot == nil || invoker == nil {
		return out, fmt.Errorf("verification requires a project Host frontend")
	}
	s, err := newRunStore(root)
	if err != nil {
		return out, err
	}
	unlock, err := s.lock()
	if err != nil {
		return out, err
	}
	defer unlock()
	plan, err := s.readPlan(runID)
	if err != nil {
		return out, err
	}
	if err := validateExplorationReadiness(root, plan); err != nil {
		return out, err
	}
	run, err := s.readLatestState(runID)
	if err != nil {
		return out, err
	}
	if run.Status != StatusIntegrated {
		return out, fmt.Errorf("run must be integrated before verification (status %s)", run.Status)
	}
	if err := validateReportClosure(run); err != nil {
		return out, err
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		return out, err
	}
	if runtime.Mode != ModeControlledLocal {
		return out, fmt.Errorf("verification mode is unsupported")
	}
	deadline := run.StartedAt.Add(time.Duration(runtime.Limits.MaxDuration))
	if !time.Now().Before(deadline) {
		return out, failVerificationBudget(s, &run, fmt.Errorf("total runtime duration limit exceeded before verification"))
	}
	boundedCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	ctx = boundedCtx
	if runtimeDigest, err := runtimeDigestWithInvoker(invoker, runtime); err != nil || runtimeDigest != plan.RuntimeDigest {
		return out, ErrStale
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return out, err
	}
	if base.Snapshot == nil || base.Snapshot.Digest() != plan.BaseSnapshot || base.Digest != plan.BaseProjectDigest || base.Report.ModelDigest != plan.BaseModelDigest {
		return out, ErrStale
	}
	if err := validateChangeImpact(host, root, base, plan); err != nil {
		return out, err
	}
	if err := repositoryMatches(root, plan); err != nil {
		return out, err
	}
	dir, _ := s.runDir(runID)
	candidate, err := s.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		return out, err
	}
	if candidate.ID != run.Candidate.ID {
		return out, fmt.Errorf("candidate identity mismatch")
	}
	hasPriorAttempt, err := persistedVerificationAttempt(dir, runID, candidate.ID)
	if err != nil {
		return out, err
	}
	if hasPriorAttempt {
		return out, failVerificationBudget(s, &run, fmt.Errorf("verification attempt already exists for candidate %s", candidate.ID))
	}
	compiled, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		return out, err
	}
	if compiled.Report.ModelDigest != plan.ModelDigest || hasErrorFinding(compiled.Report.Findings) {
		return out, fmt.Errorf("integrated candidate has invalid project model")
	}
	if err := validateFinalCandidate(host, root, base.Snapshot, candidate, plan); err != nil {
		return out, err
	}
	if err := requireFreshReviews(host, root, s, dir, base, candidate, plan, runtime, run); err != nil {
		return out, err
	}
	if err := requireArtifacts(compiled.Report); err != nil {
		return out, err
	}
	if len(plan.Checks) == 0 {
		return out, fmt.Errorf("no independent project checks are declared; verification cannot close successfully")
	}
	if err := validateCheckExecutables(plan); err != nil {
		return out, err
	}
	if len(run.Invocations)+len(run.Checks)+len(plan.Checks)+boolInt(runtime.Verifier != nil) > runtime.Limits.MaxStarts {
		return out, failVerificationBudget(s, &run, fmt.Errorf("verification processes would exceed maxStarts"))
	}
	if totalCost(run.Invocations) > runtime.Limits.MaxCostMicros {
		return out, failVerificationBudget(s, &run, fmt.Errorf("run has already exceeded maxCostMicros before verification"))
	}
	verifyDir := filepath.Join(dir, "verification", candidate.ID)
	if err := ensureDirectory(filepath.Dir(verifyDir)); err != nil {
		return out, err
	}
	_ = os.RemoveAll(verifyDir)
	if err := materializeCandidate(verifyDir, base.Snapshot, candidate); err != nil {
		return out, err
	}
	defer os.RemoveAll(verifyDir)
	out = VerifyReport{APIVersion: APIVersion, VerificationScope: "planned", RunID: runID, CandidateID: candidate.ID, CandidateHash: candidate.Digest, Status: "failed", VerifiedAt: time.Now().UTC(), Checks: []CheckResult{}}
	// This durable marker reserves the single verification attempt before any
	// external process starts. A crash after this point cannot replay checks or
	// the verifier and obtain another resource budget.
	run.Status = "verifying"
	if err := persistState(s, &run); err != nil {
		return out, fmt.Errorf("persist verification start: %w", err)
	}
	fail := func(cause error) (VerifyReport, error) {
		out.Status = "failed"
		out.Digest, err = verificationDigest(out)
		run.Status = StatusFailed
		stateErr := persistState(s, &run)
		reportErr := persistVerify(s, dir, out)
		return out, errors.Join(cause, err, stateErr, reportErr)
	}
	for _, check := range plan.Checks {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if err := freshBindings(host, invoker, root, plan, runtime); err != nil {
			return fail(err)
		}
		config, ok := runtime.Agents[check.Owner]
		if !ok {
			return fail(fmt.Errorf("check %s owner %s has no runtime environment policy", check.ID, check.Owner))
		}
		checkStartIndex := -1
		result := runCheck(ctx, verifyDir, check, config, runtime.Limits.MaxDuration, func(started CheckResult) error {
			started.CandidateID = candidate.ID
			started.Outcome = "started"
			run.Checks = append(run.Checks, started)
			checkStartIndex = len(run.Checks) - 1
			return persistState(s, &run)
		})
		result.CandidateID = candidate.ID
		out.Checks = append(out.Checks, result)
		if checkStartIndex < 0 {
			run.Checks = append(run.Checks, result)
		} else {
			run.Checks[checkStartIndex] = result
		}
		if err := persistState(s, &run); err != nil {
			return out, fmt.Errorf("persist check result %s: %w", check.ID, err)
		}
		if check.Required && result.Outcome != "passed" {
			return fail(fmt.Errorf("required check %s did not pass: %s", check.ID, result.Error))
		}
	}
	if err := validateCheckExecutables(plan); err != nil {
		return fail(err)
	}
	if runtime.Verifier != nil {
		if invoker == nil {
			return fail(fmt.Errorf("configured verifier requires an agent invoker"))
		}
		verifierStartIndex := -1
		verifier, invocation, err := runVerifier(ctx, invoker, root, plan, runtime, compiled, candidate, out.Checks, func(started InvocationLog) error {
			started.Outcome = "started"
			run.Invocations = append(run.Invocations, started)
			verifierStartIndex = len(run.Invocations) - 1
			return persistState(s, &run)
		})
		if verifierStartIndex >= 0 {
			run.Invocations[verifierStartIndex] = invocation
			if persistErr := persistState(s, &run); persistErr != nil {
				return out, fmt.Errorf("persist verifier attempt: %w", persistErr)
			}
		}
		out.Verifier = verifier
		if pinErr := validateCheckExecutables(plan); pinErr != nil {
			return fail(pinErr)
		}
		if err != nil {
			return fail(err)
		}
		if totalCost(run.Invocations) > runtime.Limits.MaxCostMicros {
			return fail(fmt.Errorf("estimated cost limit exceeded during verification"))
		}
	}
	if compiled.Config.CoverageMode == "full" {
		full, auditErr := FullVerifyProject(ctx, host, invoker, root, compiled, runtime, FullVerifyBinding{
			ExpectedSnapshot: compiled.Snapshot.Digest(), ExpectedBriefings: plan.BriefingDigests, CheckCandidateID: candidate.ID,
			StartedAt: run.StartedAt, PriorStarts: len(run.Invocations) + len(run.Checks), PriorCostMicros: totalCost(run.Invocations), PreverifiedChecks: out.Checks,
		})
		out.VerificationScope, out.ManagerVerification = "full", &full
		for _, manager := range full.Managers {
			if manager.Receipt != nil {
				run.Invocations = append(run.Invocations, InvocationLog{TaskID: manager.ManagerID, Role: "manager-verifier", Phase: "full-verify", InputDigest: manager.InputDigest, Receipt: *manager.Receipt, ReportID: manager.Receipt.RunID, Outcome: manager.Status, CostMicros: manager.CostMicros})
			}
		}
		if persistErr := persistState(s, &run); persistErr != nil {
			return fail(persistErr)
		}
		if auditErr != nil {
			return fail(auditErr)
		}
		if full.Status != "passed" {
			return fail(fmt.Errorf("full Manager verification did not pass: %s", full.Status))
		}
	}
	if err := freshBindings(host, invoker, root, plan, runtime); err != nil {
		return fail(err)
	}
	if err := validateCheckExecutables(plan); err != nil {
		return fail(err)
	}
	out.Status = "verified"
	out.Digest, err = verificationDigest(out)
	if err != nil {
		return out, err
	}
	if err := persistVerify(s, dir, out); err != nil {
		return fail(err)
	}
	run.Status = StatusVerified
	// Keep checks from earlier failed candidates in the cumulative attempt
	// ledger. The VerifyReport above contains this candidate's checks only.
	if err := persistState(s, &run); err != nil {
		return out, err
	}
	return out, nil
}

func requireArtifacts(report projectmodel.Report) error {
	files := map[string]bool{}
	for _, f := range report.Files {
		if f.Exists {
			files[f.Path] = true
		}
	}
	for _, a := range report.Artifacts {
		if !a.Required {
			continue
		}
		if len(a.Paths) == 0 {
			return fmt.Errorf("required artifact %s has no realization path", a.ID)
		}
		for _, p := range a.Paths {
			if strings.HasSuffix(p, "/") {
				found := false
				for f := range files {
					if strings.HasPrefix(f, p) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("required artifact %s is missing path under %s", a.ID, p)
				}
			} else if !files[p] {
				return fmt.Errorf("required artifact %s is missing %s", a.ID, p)
			}
		}
	}
	return nil
}

func materializeCandidate(dest string, base *Snapshot, candidate candidateData) error {
	if err := ensureDirectory(dest); err != nil {
		return err
	}
	snap, err := snapshotWithCandidate(base, candidate)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(snap.Files))
	for p := range snap.Files {
		if !safeRepoPath(p) {
			return fmt.Errorf("candidate contains unsafe path %s", p)
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		target := filepath.Join(dest, filepath.FromSlash(p))
		if !pathUnder(dest, target) {
			return fmt.Errorf("candidate path escaped verification root: %s", p)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if snap.Modes[p] == "100755" {
			mode = 0o700
		}
		if err := os.WriteFile(target, snap.Files[p], mode); err != nil {
			return fmt.Errorf("materialize candidate %s: %w", p, err)
		}
	}
	return nil
}

func pathUnder(root, target string) bool {
	r, e1 := filepath.Abs(root)
	t, e2 := filepath.Abs(target)
	if e1 != nil || e2 != nil {
		return false
	}
	rel, err := filepath.Rel(r, t)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type boundedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := b.max - b.buf.Len()
	if left > 0 {
		if len(p) > left {
			_, _ = b.buf.Write(p[:left])
			b.truncated = true
		} else {
			_, _ = b.buf.Write(p)
		}
	} else {
		b.truncated = true
	}
	return n, nil
}

func runCheck(parent context.Context, dir string, check CheckPlan, agent Agent, maximum Duration, onStart func(CheckResult) error) CheckResult {
	out := CheckResult{ID: check.ID, Command: append([]string(nil), check.Command...), ExecutablePath: check.ExecutablePath, ExecutableDigest: check.ExecutableDigest, StartedAt: time.Now().UTC(), Outcome: "failed"}
	if check.ExecutablePath == "" || check.ExecutableDigest == "" {
		out.Error = "check executable was not pinned at plan time"
		return out
	}
	if err := checkExecutableUnchanged(check); err != nil {
		out.Error = err.Error()
		return out
	}
	timeout := time.Duration(agent.Timeout)
	if timeout > time.Duration(maximum) {
		timeout = time.Duration(maximum)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, check.ExecutablePath, check.Command[1:]...)
	cmd.Args[0] = check.Command[0]
	cmd.Dir = dir
	cmd.Env = explicitEnvironment(agent.Environment)
	var stdout, stderr boundedBuffer
	stdout.max = maxCheckOutputBytes
	stderr.max = maxCheckOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if onStart != nil {
		if err := onStart(out); err != nil {
			out.Error = "could not persist check start"
			return out
		}
	}
	err := cmd.Run()
	out.Duration = time.Since(out.StartedAt).String()
	out.Stdout = stdout.buf.String()
	out.Stderr = stderr.buf.String()
	if stdout.truncated || stderr.truncated {
		out.Error = "check output exceeded 256 KiB"
		return out
	}
	if pinErr := checkExecutableUnchanged(check); pinErr != nil {
		out.Error = pinErr.Error()
		return out
	}
	if err == nil {
		out.Outcome = "passed"
		out.ExitCode = 0
		return out
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		out.Error = "check timed out"
		return out
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.ExitCode = exitErr.ExitCode()
		out.Error = err.Error()
		return out
	}
	out.Error = err.Error()
	return out
}

func checkExecutableUnchanged(check CheckPlan) error {
	resolved, raw, err := readPinnedExecutable(check.ExecutablePath)
	if err != nil {
		return fmt.Errorf("check executable unavailable: %w", err)
	}
	if resolved != check.ExecutablePath || rawContentDigest(raw) != check.ExecutableDigest {
		return fmt.Errorf("check executable changed since planning")
	}
	return nil
}

func explicitEnvironment(names []string) []string {
	var env []string
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	if runtimeOSNeedsSystemRoot(names) {
		if value, ok := os.LookupEnv("SystemRoot"); ok {
			env = append(env, "SystemRoot="+value)
		}
	}
	return env
}
func runtimeOSNeedsSystemRoot(names []string) bool { return false }

func runVerifier(ctx context.Context, invoker Invoker, root string, plan PlanRecord, runtime Runtime, project *Project, candidate candidateData, checkResults []CheckResult, onStart func(InvocationLog) error) (*VerifierReport, InvocationLog, error) {
	var log InvocationLog
	config, err := runtime.Verifier.AgentConfig()
	if err != nil {
		return nil, log, err
	}
	checks := []string{}
	for _, c := range plan.Checks {
		if c.Required {
			checks = append(checks, c.ID)
		}
	}
	artifacts := []string{}
	for _, a := range project.Report.Artifacts {
		if a.Required {
			artifacts = append(artifacts, a.ID)
		}
	}
	subjects := append(append([]string{}, artifacts...), checks...)
	sort.Strings(subjects)
	refs := make([]string, 0, len(subjects))
	for _, id := range artifacts {
		refs = append(refs, "artifact:"+id)
	}
	for _, id := range checks {
		refs = append(refs, "check:"+id)
	}
	sort.Strings(refs)
	contextJSON, _ := json.Marshal(struct {
		CandidateDigest  string        `json:"candidateDigest"`
		CandidateFiles   []string      `json:"candidateFiles"`
		RequiredSubjects []string      `json:"requiredSubjects"`
		CheckResults     []CheckResult `json:"checkResults"`
	}{candidate.Digest, sortedFileKeys(candidate.Files), subjects, checkResults})
	request := agentexec.Request{Role: agentexec.RoleVerifier, SourceRevision: project.Revision, ModelDigest: project.Report.ModelDigest, ModulePin: project.Report.Digest, ProjectionID: project.Report.Digest, ScopeIDs: refs, PolicyIDs: checks, Context: contextJSON, Artifacts: []agentexec.Artifact{}}
	inputDigest, _ := digest(request)
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, log, context.DeadlineExceeded
		}
		if remaining < config.Timeout {
			config.Timeout = remaining
		}
	}
	log = InvocationLog{TaskID: "verifier", Role: agentexec.RoleVerifier, Phase: "verify", InputDigest: inputDigest, Outcome: "started"}
	if onStart != nil {
		if err := onStart(log); err != nil {
			return nil, log, fmt.Errorf("persist verifier start: %w", err)
		}
	}
	result, err := invoker.Run(ctx, config, request, agentexec.RunOptions{PrivateLogDirectory: filepath.Join(root, ".markitect", "runs", "private")})
	usageCost, costKnown := estimateCost(result.Receipt.Usage, runtime.Verifier.Pricing)
	log = InvocationLog{TaskID: "verifier", Role: agentexec.RoleVerifier, Phase: "verify", InputDigest: inputDigest, Receipt: result.Receipt, ReportID: result.Receipt.RunID, Outcome: result.Receipt.Outcome, CostMicros: usageCost}
	if log.Outcome == "" {
		log.Outcome = agentexec.OutcomeIncomplete
	}
	verifierReport := &VerifierReport{Role: agentexec.RoleVerifier, InputDigest: result.Receipt.InputDigest, Receipt: result.Receipt, Outcome: result.Response.Outcome, Observations: append([]agentexec.Observation(nil), result.Response.VerifierObservations...)}
	if err != nil {
		return verifierReport, log, err
	}
	if result.Response.Role != agentexec.RoleVerifier || result.Response.Outcome != agentexec.OutcomePassed || result.Receipt.Outcome != agentexec.OutcomePassed {
		return verifierReport, log, fmt.Errorf("independent verifier did not pass")
	}
	if err := validateVerifierCoverage(result.Response.VerifierObservations, subjects, refs, result.Response.EvidenceRefs); err != nil {
		return verifierReport, log, err
	}
	if result.Receipt.InputDigest == "" {
		return verifierReport, log, fmt.Errorf("verifier receipt omitted bound input digest")
	}
	if !costKnown {
		return verifierReport, log, fmt.Errorf("verifier usage is missing; bounded cost cannot be asserted")
	}
	return verifierReport, log, nil
}

func validateVerifierCoverage(observations []agentexec.Observation, subjects, refs, evidenceRefs []string) error {
	wanted := map[string]bool{}
	for _, s := range subjects {
		if wanted[s] {
			return fmt.Errorf("verifier contract contains duplicate subject %s", s)
		}
		wanted[s] = true
	}
	seen := map[string]bool{}
	for _, o := range observations {
		if !wanted[o.Subject] {
			return fmt.Errorf("verifier observed unrelated subject %s", o.Subject)
		}
		if seen[o.Subject] {
			return fmt.Errorf("verifier duplicated subject %s", o.Subject)
		}
		if o.Outcome != "passed" {
			return fmt.Errorf("verifier subject %s outcome is %s", o.Subject, o.Outcome)
		}
		seen[o.Subject] = true
	}
	for s := range wanted {
		if !seen[s] {
			return fmt.Errorf("verifier omitted required subject %s", s)
		}
	}
	wantRefs := append([]string(nil), refs...)
	gotRefs := append([]string(nil), evidenceRefs...)
	sort.Strings(wantRefs)
	sort.Strings(gotRefs)
	if len(gotRefs) != len(wantRefs) {
		return fmt.Errorf("verifier evidence reference coverage does not match the contract")
	}
	for i := range wantRefs {
		if gotRefs[i] != wantRefs[i] {
			return fmt.Errorf("verifier evidence reference %q does not match required %q", gotRefs[i], wantRefs[i])
		}
	}
	return nil
}

func freshBindings(host Host, invoker Invoker, root string, plan PlanRecord, runtime Runtime) error {
	if err := validateExplorationReadiness(root, plan); err != nil {
		return err
	}
	fresh, err := host.Load(root, "")
	if err != nil {
		return err
	}
	if fresh.Snapshot.Digest() != plan.WorkingSnapshot || fresh.Digest != plan.WorkingProjectDigest {
		return ErrStale
	}
	for _, manager := range fresh.Report.Managers {
		if plan.ModelEdit != nil {
			if plan.BriefingDigests[manager.ID] != "" {
				return ErrStale
			}
			continue
		}
		// The working snapshot above establishes live source freshness; it is
		// provisional and cannot establish accepted-model ancestry. Briefings
		// remain bound to the plan's exact committed model revision.
		briefing, err := managerBriefing(root, fresh.Report.ModelDigest, manager.ID, plan.BaseRevision)
		if err != nil {
			return err
		}
		if briefing.Digest != plan.BriefingDigests[manager.ID] {
			return ErrStale
		}
	}
	if err := repositoryMatches(root, plan); err != nil {
		return err
	}
	got, err := runtimeDigestWithInvoker(invoker, runtime)
	if err != nil {
		return err
	}
	if got != plan.RuntimeDigest {
		return ErrStale
	}
	return nil
}
func runtimeDigestWithInvoker(invoker Invoker, runtime Runtime) (string, error) {
	if invoker == nil {
		return "", fmt.Errorf("runtime invoker is required for binding")
	}
	fps := map[string]string{}
	for id, a := range runtime.Agents {
		cfg, err := a.AgentConfig()
		if err != nil {
			return "", err
		}
		fp, err := invoker.Fingerprint(cfg)
		if err != nil {
			return "", err
		}
		fps[id] = fp
	}
	if runtime.Verifier != nil {
		cfg, err := runtime.Verifier.AgentConfig()
		if err != nil {
			return "", err
		}
		fp, err := invoker.Fingerprint(cfg)
		if err != nil {
			return "", err
		}
		fps["$verifier"] = fp
	}
	if runtime.Review != nil {
		for id, reviewer := range runtime.Review.Agents {
			cfg, err := reviewer.AgentConfig()
			if err != nil {
				return "", err
			}
			fp, err := invoker.Fingerprint(cfg)
			if err != nil {
				return "", err
			}
			fps["$reviewer:"+id] = fp
		}
	}
	return digest(struct {
		Runtime      Runtime           `json:"runtime"`
		Fingerprints map[string]string `json:"fingerprints"`
	}{runtime, fps})
}
func persistVerify(s *runStore, dir string, report VerifyReport) error {
	if err := ensureDirectory(filepath.Join(dir, "verification")); err != nil {
		return err
	}
	id, err := newID()
	if err != nil {
		return err
	}
	name := filepath.Join(dir, "verification", id+".json")
	return writeImmutableJSON(name, report)
}

func persistedVerificationAttempt(dir, runID, candidateID string) (bool, error) {
	verificationDir := filepath.Join(dir, "verification")
	entries, err := os.ReadDir(verificationDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect prior verification records: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			return false, fmt.Errorf("unknown verification record %q", entry.Name())
		}
		path := filepath.Join(verificationDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return false, fmt.Errorf("verification record %q is not a regular file", entry.Name())
		}
		var report VerifyReport
		if err := readJSON(path, &report); err != nil {
			return false, fmt.Errorf("read prior verification record %q: %w", entry.Name(), err)
		}
		if report.APIVersion != APIVersion || report.RunID != runID || report.CandidateID == "" {
			return false, fmt.Errorf("prior verification record %q has invalid identity", entry.Name())
		}
		if report.Digest != "" {
			computed, err := verificationDigest(report)
			if err != nil || computed != report.Digest {
				return false, fmt.Errorf("prior verification record %q has an invalid digest", entry.Name())
			}
		}
		if report.CandidateID == candidateID {
			return true, nil
		}
	}
	return false, nil
}

func failVerificationBudget(store *runStore, run *RunReport, cause error) error {
	run.Status = StatusFailed
	run.Findings = append(run.Findings, cause.Error())
	return errors.Join(cause, persistState(store, run))
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

var _ io.Writer = (*boundedBuffer)(nil)
