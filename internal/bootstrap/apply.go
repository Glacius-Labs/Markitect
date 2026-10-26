package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	runexec "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/tooling/guardedwrite"
)

// The optional work window is an additional cap on productive Runs, never a
// renewed allowance or deadline after Verify rejection.
type workWindow struct {
	executor *runexec.Executor
	deadline time.Time
}

func (w workWindow) Execute(ctx context.Context, request runexec.Request) (runexec.Result, error) {
	if request.Mandate != runexec.Verify && !w.deadline.IsZero() {
		limited, cancel := context.WithDeadline(ctx, w.deadline)
		defer cancel()
		return w.executor.Execute(limited, request)
	}
	return w.executor.Execute(ctx, request)
}

func inside(root, selected string) bool {
	rel, err := filepath.Rel(root, selected)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func safeStateDestination(repository, state string) error {
	existing := state
	for {
		info, err := os.Lstat(existing)
		if err == nil {
			if !info.IsDir() || guardedwrite.IsReparsePoint(info) {
				return fmt.Errorf("state ancestor must be a real directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return fmt.Errorf("state has no existing directory anchor")
		}
		existing = parent
	}
	if err := guardedwrite.RejectReparseAncestors(existing); err != nil {
		return err
	}
	root, err := guardedwrite.CanonicalPathSpelling(repository)
	if err != nil {
		return err
	}
	anchor, err := guardedwrite.CanonicalPathSpelling(existing)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(existing, state)
	if err != nil {
		return err
	}
	if inside(root, filepath.Join(anchor, rel)) {
		return fmt.Errorf("operator state must be outside the product repository")
	}
	return nil
}

func applyExecution(preparation *project.PreparationService, repository *gitrepo.Repository, agents runtimeconfig.AgentProfilePreparer, checkMetadata runtimeconfig.CheckProfilePreparer) cli.ApplyFunc {
	return func(ctx context.Context, request cli.ApplyRequest) (out cli.ApplyReport, err error) {
		out = cli.ApplyReport{FormatVersion: 1, Operation: "apply", Status: model.NotCompleted}
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil {
			return out, err
		}
		if cfg.ProductBuild != request.Build || !stateName.MatchString(cfg.RepositoryIdentity) || len(cfg.RepositoryIdentity) > 100 || !time.Now().Before(cfg.deadline()) || !cfg.workDeadline().IsZero() && !time.Now().Before(cfg.workDeadline()) {
			return out, fmt.Errorf("invocation build, project identity or original deadline refused")
		}
		if err = safeStateDestination(request.Selection.Repository, cfg.StateDirectory); err != nil {
			return out, err
		}
		for _, credential := range cfg.Credentials {
			if inside(request.Selection.Repository, credential.Path) {
				return out, fmt.Errorf("dedicated account input must be outside candidate content")
			}
		}
		ctx, cancel := context.WithDeadline(ctx, cfg.deadline())
		defer cancel()
		prepared, metadata, status, err := captureApplyInputs(ctx, preparation, repository, request.Selection, agents, checkMetadata)
		if err != nil {
			out.Status = status
			return out, err
		}
		count := len(prepared.Topology().Managers())
		if count > cfg.MaxManagers || count*3 > cfg.MaxMessages {
			return out, fmt.Errorf("complete Assessment, Implement and Verify hierarchy exceeds invocation capacity")
		}
		var selected *runtimeconfig.PreparedProfile
		for _, profile := range metadata.Profiles {
			inline := profile.Input().Inline()
			matched := false
			for i := 0; inline != nil && i+1 < len(inline.Content); i += 2 {
				if inline.Content[i].Value == "engineExecutable" {
					matched = inline.Content[i+1].Value == cfg.EngineExecutable
				}
			}
			if !matched {
				return out, fmt.Errorf("every captured runtime profile must use the explicit invocation engine")
			}
			if profile.Input().Namespace() == "agentProfiles" {
				if selected != nil || profile.Input().Section() != "codexAppServer" {
					return out, fmt.Errorf("V1 Apply requires exactly one executable Codex account profile")
				}
				value := profile
				selected = &value
			}
		}
		if selected == nil {
			return out, fmt.Errorf("one executable Agent profile required")
		}
		profile, err := codexappserver.ProfileFromPrepared(*selected)
		if err != nil {
			return out, err
		}
		credential := cfg.Credentials[profile.CredentialRef]
		if credential.Provider != "codex-app-server" || credential.Mode != "dedicated-runtime-login" {
			return out, errRuntimeCredential
		}
		engine, err := container.NewEngine(cfg.EngineExecutable)
		if err != nil {
			return out, err
		}
		resolve, err := newCredentialResolver(cfg.Credentials)
		if err != nil {
			return out, err
		}
		sessions, err := codexappserver.NewAgents(engine, *selected, resolve)
		if err != nil {
			return out, err
		}
		labels := map[string]string{"markitect.store": profile.Session.Store}
		checker, err := checks.NewCandidateRunner(metadata.Profiles, engine, labels)
		if err != nil {
			return out, err
		}
		var nonce [16]byte
		if _, err = rand.Read(nonce[:]); err != nil {
			return out, err
		}
		id := hex.EncodeToString(nonce[:])
		if err = safeStateDestination(request.Selection.Repository, cfg.StateDirectory); err != nil {
			return out, err
		}
		lock, err := lockStateWithin(ctx, cfg.StateDirectory, startLockWait)
		if err != nil {
			return out, err
		}
		defer func() { err = errors.Join(err, lock.Release()) }()
		command, err := composeApplyCommand(cfg, id, repository, engine, sessions, checker, profile)
		if err != nil {
			return out, err
		}
		result, executionErr := command.Execute(ctx, runexec.Request{ID: id, Project: cfg.RepositoryIdentity, Prepared: prepared, Allowance: runexec.NewAllowance(cfg.deadline(), cfg.MaxMessages), MaxManagers: cfg.MaxManagers, MessageTimeout: time.Duration(cfg.MessageTimeoutSeconds) * time.Second, CleanupTimeout: time.Duration(cfg.CleanupTimeoutSeconds) * time.Second, PersistenceTimeout: time.Duration(cfg.PersistenceTimeoutSeconds) * time.Second})
		if lockErr := lock.Release(); lockErr != nil {
			executionErr = errors.Join(executionErr, lockErr)
			result.Status = model.NotCompleted
		}
		out.Status, out.Apply, out.ID, out.StateDirectory = result.Status, &result, id, cfg.StateDirectory
		if executionErr != nil {
			out.Error = executionErr.Error()
		}
		out.EvidencePath = filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "applies", id+".json")
		out.ProviderRetries = sessions.ProviderRetries()
		data, marshalErr := json.MarshalIndent(cli.ReportValue(out), "", "  ")
		if marshalErr == nil {
			marshalErr = writeStateFile(out.EvidencePath, append(data, '\n'), false)
		}
		if marshalErr == nil {
			readback, readErr := os.ReadFile(out.EvidencePath)
			marshalErr = readErr
			if readErr == nil && !bytes.Equal(readback, append(data, '\n')) {
				marshalErr = fmt.Errorf("Apply report readback mismatch")
			}
		}
		if marshalErr != nil {
			out.Status = model.NotCompleted
			out.Error = "Apply report persistence failed"
			return out, errors.Join(executionErr, marshalErr)
		}
		return out, executionErr
	}
}

// composeApplyCommand prepares host evidence before any Run can acquire resources.
// The concrete native adapters retain their ordinary startup and closure contracts.
func composeApplyCommand(cfg runConfig, id string, repository *gitrepo.Repository, engine *container.Engine, sessions *codexappserver.Agents, checker *checks.CandidateRunner, profile codexappserver.Profile) (apply.Command, error) {
	sessions.TranscriptRoot = filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "transcripts", id)
	if err := os.MkdirAll(sessions.TranscriptRoot, 0700); err != nil {
		return apply.Command{}, fmt.Errorf("prepare Apply transcripts: %w", err)
	}
	temporary := filepath.Join(cfg.StateDirectory, cfg.RepositoryIdentity, "temporary")
	if err := os.MkdirAll(temporary, 0700); err != nil {
		return apply.Command{}, err
	}
	store := &gitrepo.LocalGit{StateRoot: cfg.StateDirectory, TempRoot: temporary}
	records, err := runstore.New(cfg.StateDirectory)
	if err != nil {
		return apply.Command{}, err
	}
	spec := container.ContainerSpec{Image: profile.ImageID, Labels: map[string]string{"markitect.store": profile.Session.Store}, User: "1000:1000", CPUs: profile.Session.CPUs, MemoryBytes: profile.Session.MemoryBytes, Pids: profile.Session.Pids, TmpBytes: profile.Session.TmpBytes, Network: "none"}
	workspaces := &gitrepo.LinuxGit{Engine: engine, Spec: spec, Store: store, MaxTransferBytes: 512 << 20}
	executor := &runexec.Executor{Sessions: sessions, Workspaces: workspaces, Records: records, Checks: checker}
	publisher := &gitrepo.Publisher{Repository: repository, Store: store, Checks: checker, CheckCopies: &gitrepo.LinuxCheckCopies{Engine: engine, Spec: spec, MaxTransferBytes: 512 << 20}}
	return apply.Command{Executor: workWindow{executor: executor, deadline: cfg.workDeadline()}, Candidates: store, Records: records, Publisher: publisher}, nil
}
