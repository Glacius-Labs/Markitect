package projectrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// RunReadOnlyRepository invokes the selected native transport in an owned Git
// copy of sourceRoot. The source repository may differ from the repository
// whose model described the request. This function never loads or interprets
// project-control files from sourceRoot; it supplies the source commit and its
// complete current working inventory as read context only.
func RunReadOnlyRepository(
	ctx context.Context,
	service projectworkspace.Service,
	invoker Invoker,
	sourceRoot, privateLogDirectory string,
	config agentexec.Config,
	request agentexec.Request,
	limits Limits,
) (agentexec.RunResult, error) {
	if ctx == nil || service == nil || invoker == nil {
		return agentexec.RunResult{}, errors.New("read-only repository invocation requires context, workspace service, and invoker")
	}
	if config.Transport != TransportCodexAppServer {
		return agentexec.RunResult{}, fmt.Errorf("read-only repository invocation requires transport %q", TransportCodexAppServer)
	}
	if strings.TrimSpace(privateLogDirectory) == "" || limits.MaxCandidateFileBytes <= 0 || limits.MaxCandidateFileBytes > 8<<20 ||
		limits.MaxCandidateBytes < limits.MaxCandidateFileBytes || limits.MaxCandidateBytes > 32<<20 {
		return agentexec.RunResult{}, errors.New("read-only repository invocation requires a private log directory and positive candidate size limits")
	}
	if err := ctx.Err(); err != nil {
		return agentexec.RunResult{}, err
	}

	identity, err := source.IdentifyGit(sourceRoot)
	if err != nil {
		return agentexec.RunResult{}, fmt.Errorf("identify selected read-only source repository: %w", err)
	}
	baseSHA, err := readRepositoryHead(ctx, identity.Root)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	if request.SourceRevision != baseSHA {
		return agentexec.RunResult{}, fmt.Errorf("read-only source request revision %q does not match selected repository HEAD %q", request.SourceRevision, baseSHA)
	}
	binding, err := projectworkspace.InspectRepository(ctx, identity.Root, baseSHA)
	if err != nil {
		return agentexec.RunResult{}, fmt.Errorf("inspect selected read-only source repository: %w", err)
	}
	taskID, err := readOnlyRepositoryTaskID(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	workspaceRequest := projectworkspace.Request{
		RepositoryRoot: identity.Root, RepositoryIdentity: identity.Digest, BaseSHA: baseSHA,
		OverlayDigest: binding.OverlayDigest, TaskID: taskID,
		// Empty scopes are intentional: no harvested write is authorized.
	}
	handle, err := service.Prepare(ctx, workspaceRequest)
	if err != nil {
		return agentexec.RunResult{}, fmt.Errorf("prepare read-only source workspace: %w", err)
	}
	if err := handle.ValidateFor(workspaceRequest); err != nil {
		closeErr := service.Close(context.Background(), handle)
		return agentexec.RunResult{}, errors.Join(fmt.Errorf("workspace service returned an unbound read-only handle: %w", err), closeErr)
	}

	privateDir, err := filepath.Abs(privateLogDirectory)
	if err != nil {
		closeErr := service.Close(context.Background(), handle)
		return agentexec.RunResult{}, errors.Join(fmt.Errorf("resolve private workspace journal directory: %w", err), closeErr)
	}
	if _, err := agentexec.PreparePrivateLogDirectory(privateDir); err != nil {
		closeErr := service.Close(context.Background(), handle)
		return agentexec.RunResult{}, errors.Join(err, closeErr)
	}
	journalPath := filepath.Join(privateDir, "workspaces", handle.ID+".json")
	journal := workspaceJournal{Request: workspaceRequest, Handle: handle, State: "prepared"}
	if err := persistWorkspaceJournalOutsideCWD(journalPath, handle.CWD, journal); err != nil {
		closeErr := service.Close(context.Background(), handle)
		return agentexec.RunResult{}, errors.Join(fmt.Errorf("persist read-only workspace journal before invocation: %w", err), closeErr)
	}

	options := agentexec.RunOptions{Workspace: &handle, PrivateLogDirectory: privateDir}
	result, invokeErr := invoker.Run(ctx, config, request, options)
	journal.Receipt = result.Receipt
	journal.State = "preserved"
	if result.Delta != nil {
		invokeErr = errors.Join(invokeErr, errors.New("native invoker returned a delta outside Host workspace harvest"))
	}

	terminal := workspaceInvocationTerminal(result.Receipt.Lifecycle)
	if terminal {
		harvestCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		delta, harvestErr := service.Harvest(harvestCtx, handle)
		cancel()
		if harvestErr == nil {
			validationRequest := workspaceRequest
			if len(delta.Changes) > 0 {
				// Temporarily enumerate observed paths so NormalizeDelta can verify
				// their canonical shape and digest. These scopes are never used to
				// authorize or accept a read-only change.
				validationRequest.AllowedPaths = observedDeltaPaths(delta.Changes)
			}
			normalized, normalizeErr := projectworkspace.NormalizeDelta(validationRequest, handle, delta.Changes, projectworkspace.Limits{
				MaxFiles: localWorkspaceMaxFiles, MaxFileBytes: int(limits.MaxCandidateFileBytes), MaxTotalBytes: int(limits.MaxCandidateBytes),
			})
			if normalizeErr != nil || normalized.Digest != delta.Digest || delta.RepositoryIdentity != workspaceRequest.RepositoryIdentity ||
				delta.BaseSHA != workspaceRequest.BaseSHA || delta.OverlayDigest != workspaceRequest.OverlayDigest ||
				delta.TaskID != workspaceRequest.TaskID || delta.BaseDigest != handle.BaseDigest {
				harvestErr = errors.Join(normalizeErr, errors.New("read-only harvested delta binding or digest differs"))
			} else if len(normalized.Changes) != 0 {
				harvestErr = errors.New("read-only repository invocation changed source files; workspace preserved")
			} else {
				result.Delta = &normalized
				journal.Delta = &normalized
				journal.State = "harvested"
			}
		} else if errors.Is(harvestErr, projectworkspace.ErrInvalidDelta) {
			harvestErr = errors.Join(errors.New("read-only repository invocation changed source files or produced an unsupported delta; workspace preserved"), harvestErr)
		}
		invokeErr = errors.Join(invokeErr, harvestErr)
	} else {
		invokeErr = errors.Join(invokeErr, errors.New("native invocation termination is unknown; read-only source workspace preserved for recovery"))
	}
	if err := persistWorkspaceJournal(journalPath, journal); err != nil {
		return result, errors.Join(invokeErr, fmt.Errorf("persist read-only workspace result journal: %w", err))
	}
	if terminal && journal.State == "harvested" {
		closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		closeErr := service.Close(closeCtx, handle)
		cancel()
		if closeErr == nil {
			journal.State = "closed"
			closeErr = persistWorkspaceJournal(journalPath, journal)
		}
		invokeErr = errors.Join(invokeErr, closeErr)
	}
	return result, invokeErr
}

func observedDeltaPaths(changes []projectworkspace.Change) []string {
	paths := make([]string, 0, len(changes)*2)
	seen := make(map[string]bool, len(changes)*2)
	for _, change := range changes {
		for _, path := range []string{change.Path, change.OldPath} {
			if path == "" {
				continue
			}
			key := strings.ToLower(path)
			if !seen[key] {
				seen[key] = true
				paths = append(paths, path)
			}
		}
	}
	return paths
}

func readOnlyRepositoryTaskID(request agentexec.Request) (string, error) {
	wire, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode read-only invocation request identity: %w", err)
	}
	requestHash := sha256.Sum256(wire)
	randomID, err := nativeRandomID()
	if err != nil {
		return "", err
	}
	return "readonly-" + randomID[:16] + "-" + hex.EncodeToString(requestHash[:8]), nil
}

func readRepositoryHead(ctx context.Context, root string) (string, error) {
	command := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "HEAD^{commit}")
	command.Env = append(source.CleanGitEnv(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("read selected source repository HEAD: %w", err)
	}
	head := strings.TrimSpace(string(output))
	if len(head) != 40 && len(head) != 64 {
		return "", errors.New("selected source repository HEAD is not a full commit SHA")
	}
	for _, char := range head {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return "", errors.New("selected source repository HEAD is not a lowercase full commit SHA")
		}
	}
	return head, nil
}

func persistWorkspaceJournalOutsideCWD(path, workspaceCWD string, journal workspaceJournal) error {
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	cwd, err := filepath.EvalSymlinks(workspaceCWD)
	if err != nil {
		return fmt.Errorf("resolve owned workspace CWD: %w", err)
	}
	journalDir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("resolve private workspace journal directory: %w", err)
	}
	if pathIsWithin(cwd, journalDir) || pathIsWithin(journalDir, cwd) {
		return errors.New("private workspace journal must be outside the owned workspace CWD")
	}
	return persistWorkspaceJournal(path, journal)
}
