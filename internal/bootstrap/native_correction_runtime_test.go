package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	runexec "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// A separately dispatched native component proof, never part of ordinary tests.
// Actual semantic rejection is required: no scripted response, draft or repair is
// injected. This directly invokes production Runs, not the public Workflow selector.
func TestNativeFreshCorrectionRunChain(t *testing.T) {
	if os.Getenv("MARKITECT_TEST_NATIVE_CORRECTION") != "1" {
		t.Skip("requires a separate concrete, finite native correction dispatch")
	}
	const seedCommit = "8271e0c56468651cc2e7db399637ce8c7e0bda26"
	const seedTree = "eb9c339dbc0bf7c625ffd5cf88d869727a3dbdb7"
	const image = "sha256:122a9537aba513ef136f9395239d01499bef489d422ed42a83c21d90489e3a49"
	const digest = "sha256:fc401d6fc671ec94fb4ab28e643fb07eeed525ede66c04263e0c5ff736ac960d"
	envPath := func(name string) string {
		t.Helper()
		path := os.Getenv("MARKITECT_TEST_NATIVE_CORRECTION_" + name)
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			t.Fatal("explicit absolute correction input required:", name)
		}
		return path
	}
	repo, runtimePath, configPath, output := envPath("REPOSITORY"), envPath("RUNTIME"), envPath("CONFIG"), envPath("OUTPUT")
	seedContract, gitPath := envPath("SEED_CONTRACT"), envPath("GIT")
	if inside(repo, output) || inside(output, repo) || inside(output, runtimePath) || inside(output, configPath) {
		t.Fatal("new evidence output must be separate from all frozen inputs")
	}
	if err := os.Mkdir(output, 0700); err != nil {
		t.Fatal("new evidence directory required:", err)
	}
	type stage struct {
		RequestID, TranscriptDirectory string
		Input                          project.CandidateRef
		Findings                       []runexec.Finding
		AllowanceUsedBefore, UsedAfter int
		Returned, Stored               runexec.Result
		ExecuteError, ReadbackError    string
		Readback                       bool
		Transcripts                    map[string]string
	}
	report := struct {
		Kind, Scope, Binding, ModelDigest, SeedCommit, SeedTree string
		CopyFreshness                                           string
		Deadline, WorkDeadline, Store, Image                    string
		InputHashes                                             map[string]string
		Stages                                                  []stage
		Captured                                                project.CandidateRef
		AllowanceUsed, MaxMessages                              int
		InputUnchanged, FinalScopedAbsence, Passed              bool
		ExistingReminderTestsPreserved                          bool
		Failure                                                 string
	}{Kind: "native-correction-run-chain/v1", Scope: "Actual Verify rejection -> fresh Implement -> fresh Verify; not public Workflow selector or publication evidence", CopyFreshness: "Source-backed production LinuxGit.Open allocation per Run plus complete closure; copy/volume IDs are not retained or claimed observed", SeedCommit: seedCommit, SeedTree: seedTree, InputHashes: map[string]string{}}
	defer func() {
		if t.Failed() {
			report.Passed = false
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(output, "correction.json"), append(data, '\n'), 0600)
		}
		if err != nil {
			t.Error("retain actual component observations:", err)
		}
	}()
	fail := func(err error) {
		t.Helper()
		report.Failure = err.Error()
		t.Fatal(err)
	}
	for _, path := range []string{runtimePath, configPath, seedContract, gitPath} {
		hash, err := nativeCorrectionDigest(path)
		if err != nil {
			fail(err)
		}
		report.InputHashes[path] = hash
	}
	if report.InputHashes[seedContract] != "727d9c8223b45dd130a4d72b7e4fb5165c0ac8baa476c335f7d5055823d296df" {
		fail(fmt.Errorf("exact original independent seed contract required"))
	}
	data, err := os.ReadFile(seedContract)
	var seed struct {
		Bundle       string `json:"bundle"`
		BundleSHA256 string `json:"bundle_sha256"`
	}
	if err != nil || json.Unmarshal(data, &seed) != nil || seed.BundleSHA256 != "5fc28f0466565e39c7a2d6c5fd686f9535dec63dba4353ff2f3fddfc72b44d91" {
		fail(fmt.Errorf("frozen whole seeded input bundle required"))
	}
	if actual, err := nativeCorrectionDigest(seed.Bundle); err != nil || actual != seed.BundleSHA256 {
		fail(fmt.Errorf("original seed bundle changed or unavailable"))
	}
	report.InputHashes[seed.Bundle] = seed.BundleSHA256
	cfg, err := loadRunConfig(context.Background(), configPath)
	if err != nil {
		fail(err)
	}
	if cfg.RepositoryIdentity != "small-library" || cfg.MaxManagers != 6 || cfg.MaxMessages > 128 || cfg.MaxMessages < 18 || !inside(output, cfg.StateDirectory) || cfg.StateDirectory == output || time.Until(cfg.deadline()) <= 0 || time.Until(cfg.deadline()) > 45*time.Minute {
		fail(fmt.Errorf("explicit fresh state and original finite six-Manager allowance required"))
	}
	if err = safeStateDestination(repo, cfg.StateDirectory); err != nil {
		fail(err)
	}
	report.Deadline, report.WorkDeadline, report.MaxMessages = cfg.Deadline, cfg.WorkDeadline, cfg.MaxMessages
	ctx, cancel := context.WithDeadline(context.Background(), cfg.deadline())
	defer cancel()
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		fail(err)
	}
	manifest, _ := snapshot.NewPath(".markitect/model.yaml")
	dependencyManifest, _ := snapshot.NewPath("model.yaml")
	runtimeFile, err := snapshot.NewPath(filepath.Base(runtimePath))
	if err != nil {
		fail(err)
	}
	selection := project.InputSelection{
		Repository: repo, Local: project.ModelSelection{Boundary: ".", ManifestPath: manifest},
		Dependencies: []project.ModelSelection{
			{Boundary: ".markitect/dependencies/software", ManifestPath: dependencyManifest},
			{Boundary: ".markitect/dependencies/core", ManifestPath: dependencyManifest},
		},
		RuntimeProfiles: []project.SourceLocation{{Boundary: filepath.Dir(runtimePath), Path: runtimeFile}},
	}
	preparation := project.NewPreparationService(project.NewModelService(repository, repository))
	agents := codexappserver.NewMetadataPreparer(codexappserver.NewEngineImageReader)
	prepared, metadata, status, err := captureApplyInputs(ctx, preparation, repository, selection, agents, checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader))
	if err != nil || status != model.Passed || !prepared.Valid() {
		fail(fmt.Errorf("actual seed preparation failed: %w", err))
	}
	if prepared.Attempt().Model().Inputs().Head != seedCommit || len(prepared.Topology().Managers()) != 6 {
		fail(fmt.Errorf("the independently frozen whole seed and all six Managers are required"))
	}
	for _, m := range prepared.Attempt().Model().Checked().Models {
		if m.Local {
			report.ModelDigest = m.Digest
		}
	}
	if report.ModelDigest != digest {
		fail(fmt.Errorf("unchanged original M2 intent required"))
	}
	report.Binding = prepared.Binding()
	var selected *runtimeconfig.PreparedProfile
	for _, p := range metadata.Profiles {
		matched := false
		inline := p.Input().Inline()
		for i := 0; inline != nil && i+1 < len(inline.Content); i += 2 {
			if inline.Content[i].Value == "engineExecutable" {
				matched = inline.Content[i+1].Value == cfg.EngineExecutable
			}
		}
		if !matched {
			fail(fmt.Errorf("all prepared profiles must use the explicit invocation engine"))
		}
		if p.Input().Namespace() == "agentProfiles" {
			if selected != nil || p.Input().Section() != "codexAppServer" {
				fail(fmt.Errorf("one actual native Codex account profile required"))
			}
			value := p
			selected = &value
		}
	}
	if selected == nil {
		fail(fmt.Errorf("native Codex profile absent"))
	}
	profile, err := codexappserver.ProfileFromPrepared(*selected)
	if err != nil || profile.ImageID != image {
		fail(fmt.Errorf("unchanged qualified runtime image/profile required: %w", err))
	}
	credential := cfg.Credentials[profile.CredentialRef]
	if credential.Provider != "codex-app-server" || credential.Mode != "dedicated-runtime-login" || inside(repo, credential.Path) || inside(output, credential.Path) {
		fail(fmt.Errorf("explicit external dedicated native login required"))
	}
	credentialHash, err := nativeCorrectionDigest(credential.Path)
	if err != nil {
		fail(fmt.Errorf("credential source fingerprint unavailable"))
	}
	engine, err := container.NewEngine(cfg.EngineExecutable)
	if err != nil {
		fail(err)
	}
	report.Store, report.Image = profile.Session.Store, profile.ImageID
	if err = nativeCorrectionAbsence(ctx, engine, cfg.EngineExecutable, report.Store); err != nil {
		fail(err)
	}
	// Final observation never removes resources or converts a failed attempt.
	defer func() {
		final, stop := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
		defer stop()
		if err := nativeCorrectionAbsence(final, engine, cfg.EngineExecutable, report.Store); err != nil {
			report.Passed = false
			t.Error("final scoped absence:", err)
		} else {
			report.FinalScopedAbsence = true
		}
	}()
	resolve, err := newCredentialResolver(cfg.Credentials)
	if err != nil {
		fail(err)
	}
	sessions, err := codexappserver.NewAgents(engine, *selected, resolve)
	if err != nil {
		fail(err)
	}
	checker, err := checks.NewCandidateRunner(metadata.Profiles, engine, map[string]string{"markitect.store": report.Store})
	if err != nil {
		fail(err)
	}
	chain := filepath.Base(output)
	command, err := composeApplyCommand(cfg, chain, repository, engine, codexAgents(sessions, profile), checker)
	if err != nil {
		fail(err)
	}
	input, err := command.Candidates.Capture(ctx, cfg.RepositoryIdentity, prepared)
	if err != nil {
		fail(err)
	}
	report.Captured = input
	if input.Commit != seedCommit || input.Tree != seedTree || input.Binding != prepared.Binding() {
		fail(fmt.Errorf("capture must retain the actual seed, not an earlier correct A2 Prepared census"))
	}
	allowance := runexec.NewAllowance(cfg.deadline(), cfg.MaxMessages)
	transcriptRoot := sessions.TranscriptRoot
	seen := map[string]bool{}
	var feedback []runexec.Finding
	for index, mandate := range []runexec.Mandate{runexec.Verify, runexec.Implement, runexec.Verify} {
		id := fmt.Sprintf("%s-%d-%s", chain, index, mandate)
		transcripts := filepath.Join(transcriptRoot, id)
		if err = os.Mkdir(transcripts, 0700); err != nil {
			fail(err)
		}
		sessions.TranscriptRoot = transcripts
		request := runexec.Request{ID: id, Project: cfg.RepositoryIdentity, Prepared: prepared, Input: input, Mandate: mandate, Findings: slices.Clone(feedback), Allowance: allowance, MaxManagers: cfg.MaxManagers, ProviderIdleTimeout: time.Duration(cfg.ProviderIdleTimeoutSeconds) * time.Second, CleanupTimeout: time.Duration(cfg.CleanupTimeoutSeconds) * time.Second, PersistenceTimeout: time.Duration(cfg.PersistenceTimeoutSeconds) * time.Second}
		observed := stage{RequestID: id, TranscriptDirectory: transcripts, Input: input, Findings: slices.Clone(feedback), AllowanceUsedBefore: allowance.Used(), Transcripts: map[string]string{}}
		returned, runErr := command.Executor.Execute(ctx, request)
		observed.Returned, observed.UsedAfter = returned, allowance.Used()
		report.AllowanceUsed = allowance.Used()
		if runErr != nil {
			observed.ExecuteError = runErr.Error()
		}
		readCtx, readStop := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(cfg.PersistenceTimeoutSeconds)*time.Second)
		stored, readErr := command.Records.Read(readCtx, request.Project, id)
		readStop()
		observed.Stored = stored
		observed.Readback = readErr == nil && reflect.DeepEqual(returned, stored)
		if readErr != nil {
			observed.ReadbackError = readErr.Error()
		}
		// Retain actual failed/partial results before any assertion can stop the proof.
		report.Stages = append(report.Stages, observed)
		saved := &report.Stages[len(report.Stages)-1]
		files, scanErr := filepath.Glob(filepath.Join(transcripts, "markitect-native-*.jsonl"))
		if scanErr != nil {
			fail(scanErr)
		}
		for _, path := range files {
			name := filepath.Base(path)
			if seen[name] {
				fail(fmt.Errorf("a native Session identity was reused"))
			}
			seen[name] = true
			hash, err := nativeCorrectionDigest(path)
			if err != nil {
				fail(err)
			}
			saved.Transcripts[name] = hash
		}
		if runErr != nil || !saved.Readback || stored.Technical != runexec.Completed || stored.Candidate == nil || len(stored.Errors) != 0 {
			fail(fmt.Errorf("actual Run did not complete with exact stored readback; no judgment conversion or retry"))
		}
		if stored.ID != id || stored.Input != input || stored.Binding != report.Binding || len(stored.Sessions) != 6 || len(stored.Handoffs) != 6 || len(files) != 6 || len(stored.Cleanup) != 14 {
			fail(fmt.Errorf("complete bound six-Manager native hierarchy required"))
		}
		closed := map[string]bool{}
		for _, c := range stored.Cleanup {
			if !c.Closed || c.Error != "" || closed[c.Resource] {
				fail(fmt.Errorf("Run cleanup incomplete or duplicated"))
			}
			closed[c.Resource] = true
		}
		for _, manager := range prepared.Topology().Managers() {
			if !closed["session:"+manager.Package] || !closed["copy:"+manager.Package] {
				fail(fmt.Errorf("Manager resources not closed before the next Run"))
			}
		}
		if !closed["manager-workers"] || !closed["git-service"] || request.Allowance != allowance || allowance.Deadline != cfg.deadline() || !reflect.DeepEqual(saved.Findings, feedback) {
			fail(fmt.Errorf("original shared limits, Findings and resource closure required"))
		}
		for _, s := range stored.Sessions {
			if s.Observation.ImageID != image || s.Observation.ProfileIdentity == "" || s.Observation.HarnessDigest == "" {
				fail(fmt.Errorf("actual native image/profile/harness observations required"))
			}
		}
		if index == 0 {
			concrete := false
			for _, f := range stored.Findings {
				concrete = concrete || strings.TrimSpace(f.Text) != ""
			}
			if stored.Decision != runexec.Reject || stored.Candidate.Commit != seedCommit || stored.Candidate.Tree != seedTree || !concrete || !nativeCorrectionFixedCheck(stored, false) {
				fail(fmt.Errorf("actual semantic Verify Reject with real Findings and known nonzero physical check required; approve/blocker/technical failure ends this proof"))
			}
			feedback = slices.Clone(stored.Findings)
		} else if index == 1 {
			if stored.Decision != runexec.Submitted || stored.Candidate.Tree == seedTree {
				fail(fmt.Errorf("fresh Implement must submit its actual changed retained candidate"))
			}
			if err := nativeCorrectionPreservedTests(ctx, gitPath, output, cfg.StateDirectory, *stored.Candidate, prepared); err != nil {
				fail(err)
			}
			report.ExistingReminderTestsPreserved = true
		} else if stored.Decision != runexec.Approve || stored.Candidate.Commit != input.Commit || stored.Candidate.Tree != input.Tree || !nativeCorrectionFixedCheck(stored, true) {
			fail(fmt.Errorf("fresh Verify must approve the exact retained Implement candidate with passing physical checks"))
		}
		// Verify scratch handoff.Result is intentionally never a later input.
		input = *stored.Candidate
	}
	confirmed, err := repository.Capture(ctx, prepared.Attempt().Model().Request())
	if err != nil || confirmed.Guard != prepared.Attempt().Model().Inputs().Guard {
		fail(fmt.Errorf("original frozen seed changed"))
	}
	for path, expected := range report.InputHashes {
		actual, err := nativeCorrectionDigest(path)
		if err != nil || actual != expected {
			fail(fmt.Errorf("frozen invocation input changed"))
		}
	}
	if actual, err := nativeCorrectionDigest(credential.Path); err != nil || actual != credentialHash {
		fail(fmt.Errorf("native credential source changed or could not be observed"))
	}
	report.InputUnchanged = true
	messages := 0
	for _, s := range report.Stages {
		messages += s.Stored.Messages
	}
	if messages != allowance.Used() || len(seen) != 18 {
		fail(fmt.Errorf("actual shared message accounting or fresh Session count mismatch"))
	}
	report.Passed = true
}

func nativeCorrectionDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func nativeCorrectionFixedCheck(r runexec.Result, passing bool) bool {
	for _, c := range r.Checks {
		p := c.Receipt
		if c.Commit == r.Candidate.Commit && c.ID == "project-tests" && c.Error == "" && p.Started && p.Exited && !p.TimedOut && !p.Uncertain && p.CopiesClosed && (p.ExitCode == 0) == passing {
			return true
		}
	}
	return false
}

// Read the actual retained candidate after closure, without materializing its
// source on the host or editing the seed. The predicate is deliberately strict:
// any byte or mode change to an existing reminder test file, including added
// cases in that file, fails this proof. New test files are permitted. This starts
// only local Git, never Engine.
func nativeCorrectionPreservedTests(ctx context.Context, git, output, state string, candidate project.CandidateRef, prepared project.Prepared) error {
	bare := filepath.Join(output, "retained-"+candidate.Key)
	if err := os.Mkdir(bare, 0700); err != nil {
		return err
	}
	hooks := filepath.Join(bare, "empty-hooks")
	if err := os.Mkdir(hooks, 0700); err != nil {
		return err
	}
	var env []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && !strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			env = append(env, value)
		}
	}
	env = append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_ALLOW_PROTOCOL=file", "GIT_TERMINAL_PROMPT=0", "GIT_NO_LAZY_FETCH=1")
	command := func(args ...string) ([]byte, error) {
		limited, stop := context.WithTimeout(ctx, 20*time.Second)
		defer stop()
		argv := []string{"--no-replace-objects", "--literal-pathspecs", "-c", "core.longpaths=true", "-c", "core.fsmonitor=false", "-c", "core.autocrlf=false", "-c", "core.hooksPath=" + hooks, "-C", bare}
		cmd := exec.CommandContext(limited, git, append(argv, args...)...)
		cmd.Env = env
		cmd.WaitDelay = time.Second
		return cmd.Output()
	}
	if _, err := command("init", "--bare", "--quiet", "--object-format="+candidate.Format); err != nil {
		return err
	}
	bundle := filepath.Join(state, candidate.Project, "candidates", candidate.Key, "candidate.bundle")
	if _, err := command("fetch", "--quiet", "--no-tags", bundle, "refs/markitect/retain/"+candidate.Key+":refs/correction/readback"); err != nil {
		return err
	}
	for _, item := range []struct{ object, expected string }{{"refs/correction/readback", candidate.Commit}, {candidate.Commit + "^{tree}", candidate.Tree}} {
		actual, err := command("rev-parse", item.object)
		if err != nil || strings.TrimSpace(string(actual)) != item.expected {
			return fmt.Errorf("actual retained correction commit/tree differs")
		}
	}
	count := 0
	for _, f := range prepared.Attempt().Model().Inputs().Raw.Census.Entries() {
		path := f.Path().String()
		if strings.HasPrefix(path, "realization/lending/reminders/") && strings.HasSuffix(path, "_test.go") {
			data, err := command("show", candidate.Commit+":"+path)
			if err != nil || !bytes.Equal(data, f.Bytes()) {
				return fmt.Errorf("existing reminder test changed: %s", path)
			}
			entry, err := command("ls-tree", candidate.Commit, "--", path)
			mode := "100644"
			if f.Mode() == snapshot.Executable {
				mode = "100755"
			}
			if err != nil || !strings.HasPrefix(string(entry), mode+" blob ") {
				return fmt.Errorf("existing reminder test mode changed: %s", path)
			}
			count++
		}
	}
	if count == 0 {
		return fmt.Errorf("original reminder tests absent from frozen input")
	}
	return nil
}

// Read-only final/baseline evidence, never prune/remove. Fresh private copy/volume
// allocation is source-backed by LinuxGit.Open; their IDs are not invented here.
func nativeCorrectionAbsence(ctx context.Context, engine *container.Engine, executable, store string) error {
	query, stop := context.WithTimeout(ctx, 20*time.Second)
	containers, err := engine.Containers(query, "markitect.store", store)
	stop()
	if err != nil || len(containers) != 0 {
		return fmt.Errorf("scoped container absence unconfirmed (%d rows): %v", len(containers), err)
	}
	query, stop = context.WithTimeout(ctx, 20*time.Second)
	defer stop()
	cmd := exec.CommandContext(query, executable, "volume", "ls", "-q", "--filter", "label=markitect.store="+store)
	cmd.WaitDelay = time.Second
	volumes, err := cmd.Output()
	if err != nil || len(volumes) > 65536 || len(bytes.TrimSpace(volumes)) != 0 {
		return fmt.Errorf("scoped volume absence unconfirmed (%d bytes): %v", len(volumes), err)
	}
	return nil
}
