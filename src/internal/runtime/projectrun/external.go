package projectrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
)

const externalPacketVersion = "markitect.example.org/external-work-packet/v1alpha1"

var (
	// ErrAwaitingExternal signals a durable packet wait, not a failed or
	// uncertain provider invocation. The run scheduler handles this sentinel.
	ErrAwaitingExternal = errors.New("run is awaiting an external packet response")
)

type externalExecutionContextKey struct{}

// ExternalPacket is the immutable, Host-authored request exported to an
// external executor or reviewer. InvocationBytes is the exact protocol input;
// repeated exports return the same bytes and packet digest.
type ExternalPacket struct {
	APIVersion            string               `json:"apiVersion"`
	ID                    string               `json:"id"`
	RunID                 string               `json:"runId"`
	TaskID                string               `json:"taskId"`
	AttemptID             string               `json:"attemptId"`
	ManagerID             string               `json:"managerId"`
	Kind                  string               `json:"kind"`
	Phase                 string               `json:"phase"`
	Attempt               int                  `json:"attempt"`
	CandidateID           string               `json:"candidateId,omitempty"`
	CandidateDigest       string               `json:"candidateDigest,omitempty"`
	CandidateRecordDigest string               `json:"candidateRecordDigest,omitempty"`
	BaseRevision          string               `json:"baseRevision"`
	BaseSnapshot          string               `json:"baseSnapshot"`
	PlanDigest            string               `json:"planDigest"`
	RequestDigest         string               `json:"requestDigest"`
	Invocation            agentexec.Invocation `json:"invocation"`
	InvocationBytes       []byte               `json:"invocationBytes"`
	WorkspaceMode         string               `json:"workspaceMode,omitempty"`
	Digest                string               `json:"digest"`
}

type externalSubmission struct {
	APIVersion   string                  `json:"apiVersion"`
	PacketID     string                  `json:"packetId"`
	PacketDigest string                  `json:"packetDigest"`
	ResponseHash string                  `json:"responseDigest"`
	Response     []byte                  `json:"response"`
	GitCandidate *ExternalGitCandidate   `json:"gitCandidate,omitempty"`
	GitDelta     *projectworkspace.Delta `json:"gitDelta,omitempty"`
}

// ExternalGitCandidate identifies a committed source-tree delta supplied for
// one executor packet. Its paths and bytes are observed from Git objects, not
// trusted from the response JSON.
type ExternalGitCandidate struct {
	Root         string `json:"root"`
	BaseRevision string `json:"baseRevision"`
	Revision     string `json:"revision"`
}

type externalRequestContext struct {
	Kind            string `json:"kind"`
	Phase           string `json:"phase"`
	ManagerID       string `json:"managerId"`
	Round           int    `json:"round"`
	CandidateID     string `json:"candidateId"`
	CandidateDigest string `json:"candidateDigest"`
}

// ExternalExecutionContext marks a Run or Resume as an external-packet run.
// The marker is carried only in context; it does not alter the project plan or
// executor configuration.
func ExternalExecutionContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, externalExecutionContextKey{}, true)
}

// IsExternalExecution reports whether a caller explicitly selected external
// packet execution for the current Run or Resume operation.
func IsExternalExecution(ctx context.Context) bool {
	return externalExecutionEnabled(ctx)
}

func externalExecutionEnabled(ctx context.Context) bool {
	enabled, _ := ctx.Value(externalExecutionContextKey{}).(bool)
	return enabled
}

// RunExternal starts a plan through the normal run scheduler in external
// packet mode. A pending packet is persisted by the scheduler before this
// returns ErrAwaitingExternal.
func RunExternal(ctx context.Context, host Host, invoker Invoker, root, planID string) (RunReport, error) {
	return Run(ExternalExecutionContext(ctx), host, invoker, root, planID)
}

// ResumeExternal continues the normal run scheduler in external packet mode.
func ResumeExternal(ctx context.Context, host Host, invoker Invoker, root, runID string) (RunReport, error) {
	return Resume(ExternalExecutionContext(ctx), host, invoker, root, runID)
}

// externalInvocation is called by the scheduler while it owns the run lock.
// It is intentionally before provider start accounting: a pending packet is
// durable but is not an uncertain native/process invocation.
func externalInvocation(ctx context.Context, host Host, root, planID, attemptID string, config agentexec.Config, request agentexec.Request) (agentexec.RunResult, bool, error) {
	if !externalExecutionEnabled(ctx) {
		return agentexec.RunResult{}, false, nil
	}
	meta, err := decodeExternalRequestContext(request.Context)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	if meta.Kind != "projectrun-task/v1" && meta.Kind != "projectrun-review/v1" {
		return agentexec.RunResult{}, true, fmt.Errorf("request kind %q cannot be exported as an external packet", meta.Kind)
	}
	if meta.ManagerID == "" || meta.Phase == "" {
		return agentexec.RunResult{}, true, errors.New("external request is missing its Manager or phase binding")
	}
	store, err := newRunStore(root)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	dir, err := store.runDir(planID)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	plan, err := store.readPlan(planID)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	report, err := store.readLatestState(planID)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	if !report.ExternalExecution {
		return agentexec.RunResult{}, true, ErrStale
	}
	task := findTask(report.Tasks, meta.ManagerID)
	if task == nil {
		return agentexec.RunResult{}, true, ErrStale
	}
	packet, err := createOrReadExternalPacket(store, dir, plan, report, *task, attemptID, request, meta)
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	submission, err := readExternalSubmission(dir, packet.ID, store)
	if os.IsNotExist(err) {
		return agentexec.RunResult{}, true, fmt.Errorf("%w: %s", ErrAwaitingExternal, packet.ID)
	}
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	if err := validateExternalPacketFresh(store, dir, plan, report, *task, packet); err != nil {
		return agentexec.RunResult{}, true, err
	}
	if submission.PacketDigest != packet.Digest || submission.ResponseHash != externalResponseDigest(submission.Response, submission.GitCandidate, submission.GitDelta) {
		return agentexec.RunResult{}, true, errors.New("external submission binding is invalid")
	}
	if submission.GitCandidate != nil {
		stored := submission.GitDelta
		if stored == nil || stored.BaseSHA != submission.GitCandidate.BaseRevision || stored.OverlayDigest != packet.RequestDigest ||
			stored.TaskID != packet.AttemptID || stored.BaseDigest != "sha256:"+packet.CandidateDigest {
			return agentexec.RunResult{}, true, errors.New("external Git delta does not match its packet binding")
		}
		observed, err := importExternalGitCandidate(host, root, packet, *submission.GitCandidate)
		if err != nil {
			return agentexec.RunResult{}, true, err
		}
		if observed.Digest != stored.Digest {
			return agentexec.RunResult{}, true, errors.New("external Git candidate changed after response submission")
		}
	}
	response, err := agentexec.DecodeResponse(submission.Response, packet.Invocation, "")
	if err != nil {
		return agentexec.RunResult{}, true, err
	}
	var delta *projectworkspace.Delta
	if submission.GitDelta != nil {
		copy := *submission.GitDelta
		delta = &copy
	}
	receipt := externalReceipt(packet, response)
	return agentexec.RunResult{Response: response, Receipt: receipt, Delta: delta}, true, nil
}

// ExternalPackets returns the pending packets for a run in stable ID order.
func ExternalPackets(host Host, root, runID string) ([]ExternalPacket, error) {
	store, err := newRunStore(root)
	if err != nil {
		return nil, err
	}
	dir, err := store.runDir(runID)
	if err != nil {
		return nil, err
	}
	plan, err := store.readPlan(runID)
	if err != nil {
		return nil, err
	}
	report, err := store.readLatestState(runID)
	if err != nil {
		return nil, err
	}
	if !report.ExternalExecution {
		return nil, ErrStale
	}
	if err := validateExternalSubmissionBinding(host, root, plan); err != nil {
		return nil, err
	}
	packets, err := readExternalPackets(dir, store)
	if err != nil {
		return nil, err
	}
	result := make([]ExternalPacket, 0, len(packets))
	for _, packet := range packets {
		if _, err := readExternalSubmission(dir, packet.ID, store); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if err := validateExternalPacketFresh(store, dir, plan, report, taskForPacket(report, packet), packet); err != nil {
			if errors.Is(err, ErrStale) {
				continue
			}
			return nil, err
		}
		result = append(result, packet)
	}
	return result, nil
}

// ExportExternalPacket returns one exact pending packet. If packetID is empty,
// the call succeeds only when the run has exactly one current pending packet.
func ExportExternalPacket(host Host, root, runID, packetID string) (ExternalPacket, error) {
	packets, err := ExternalPackets(host, root, runID)
	if err != nil {
		return ExternalPacket{}, err
	}
	if packetID == "" {
		if len(packets) != 1 {
			return ExternalPacket{}, fmt.Errorf("expected exactly one pending external packet; found %d", len(packets))
		}
		return packets[0], nil
	}
	for _, packet := range packets {
		if packet.ID == packetID {
			return packet, nil
		}
	}
	return ExternalPacket{}, ErrNotFound
}

// SubmitExternalResponse stores a strictly bound protocol response once. The
// returned bool is false for an exact duplicate; changed duplicate bytes are
// rejected. The caller must separately resume the run with explicit execute
// authority.
func SubmitExternalResponse(host Host, root, runID, packetID, expectedDigest string, responseBytes []byte, gitCandidate *ExternalGitCandidate) (returnedPacket ExternalPacket, returnedStored bool, resultErr error) {
	store, err := newRunStore(root)
	if err != nil {
		return ExternalPacket{}, false, err
	}
	unlock, err := store.lock()
	if err != nil {
		return ExternalPacket{}, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, unlock()) }()
	dir, err := store.runDir(runID)
	if err != nil {
		return ExternalPacket{}, false, err
	}
	plan, err := store.readPlan(runID)
	if err != nil {
		return ExternalPacket{}, false, err
	}
	report, err := store.readLatestState(runID)
	if err != nil {
		return ExternalPacket{}, false, err
	}
	packet, err := readExternalPacket(dir, packetID, store)
	if err != nil {
		return ExternalPacket{}, false, err
	}
	if expectedDigest == "" || expectedDigest != packet.Digest {
		return ExternalPacket{}, false, errors.New("external packet digest does not match --expect")
	}
	responseDigest := externalResponseDigest(responseBytes, gitCandidate, nil)
	if existing, readErr := readExternalSubmission(dir, packet.ID, store); readErr == nil {
		if existing.PacketDigest != packet.Digest {
			return ExternalPacket{}, false, errors.New("stored external submission does not match packet digest")
		}
		if externalResponseDigest(responseBytes, gitCandidate, existing.GitDelta) != existing.ResponseHash {
			return ExternalPacket{}, false, errors.New("external packet already has a different submitted response")
		}
		return packet, false, nil
	} else if !os.IsNotExist(readErr) {
		return ExternalPacket{}, false, readErr
	}
	if err := validateExternalSubmissionBinding(host, root, plan); err != nil {
		return ExternalPacket{}, false, err
	}
	task := taskForPacket(report, packet)
	if err := validateExternalPacketFresh(store, dir, plan, report, task, packet); err != nil {
		return ExternalPacket{}, false, err
	}
	response, err := agentexec.DecodeResponse(responseBytes, packet.Invocation, "")
	if err != nil {
		return ExternalPacket{}, false, err
	}
	var importedDelta *projectworkspace.Delta
	if gitCandidate != nil {
		delta, err := importExternalGitCandidate(host, root, packet, *gitCandidate)
		if err != nil {
			return ExternalPacket{}, false, err
		}
		if err := validateGitCandidateProposals(response.CandidateFiles, delta); err != nil {
			return ExternalPacket{}, false, err
		}
		importedDelta = &delta
		responseDigest = externalResponseDigest(responseBytes, gitCandidate, importedDelta)
	}
	submission := externalSubmission{APIVersion: externalPacketVersion, PacketID: packet.ID,
		PacketDigest: packet.Digest, ResponseHash: responseDigest, Response: append([]byte(nil), responseBytes...),
		GitCandidate: cloneExternalGitCandidate(gitCandidate), GitDelta: importedDelta}
	path := externalSubmissionPath(dir, packet.ID)
	if err := ensureDirectory(filepath.Join(dir, "external"), store); err != nil {
		return ExternalPacket{}, false, err
	}
	if err := ensureDirectory(filepath.Join(dir, "external", "responses"), store); err != nil {
		return ExternalPacket{}, false, err
	}
	if err := writeImmutableJSON(path, submission, store); err != nil {
		if recordstore.WasPublished(err) || recordstore.OutcomeUnknown(err) {
			return ExternalPacket{}, false, err
		}
		if _, readErr := readExternalSubmission(dir, packet.ID, store); readErr == nil {
			existing, readErr := readExternalSubmission(dir, packet.ID, store)
			if readErr == nil && existing.ResponseHash == responseDigest {
				return packet, false, nil
			}
		}
		return ExternalPacket{}, false, err
	}
	return packet, true, nil
}

func importExternalGitCandidate(host Host, root string, packet ExternalPacket, candidate ExternalGitCandidate) (projectworkspace.Delta, error) {
	if packet.Kind != "projectrun-task/v1" || (packet.Phase != "work" && packet.Phase != "integrate") {
		return projectworkspace.Delta{}, errors.New("Git candidates are supported only for executor work and integration packets")
	}
	if !filepath.IsAbs(candidate.Root) || candidate.Root == "" {
		return projectworkspace.Delta{}, errors.New("external Git candidate root must be absolute")
	}
	candidateRoot, err := filepath.Abs(candidate.Root)
	if err != nil {
		return projectworkspace.Delta{}, errors.New("external Git candidate root is invalid")
	}
	candidateRoot = filepath.Clean(candidateRoot)
	identity, err := source.IdentifyGit(candidateRoot)
	if err != nil {
		return projectworkspace.Delta{}, err
	}
	var scopes struct {
		AllowedPaths  []string `json:"allowedWritePaths"`
		ExcludedPaths []string `json:"excludedWritePaths"`
	}
	if err := json.Unmarshal(packet.Invocation.Request.Context, &scopes); err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("decode packet write scopes: %w", err)
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		return projectworkspace.Delta{}, err
	}
	request := projectworkspace.Request{RepositoryRoot: candidateRoot, RepositoryIdentity: identity.Digest,
		BaseSHA: candidate.BaseRevision, OverlayDigest: packet.RequestDigest, TaskID: packet.AttemptID,
		AllowedPaths: append([]string(nil), scopes.AllowedPaths...), ExcludedPaths: append([]string(nil), scopes.ExcludedPaths...)}
	handle := projectworkspace.Handle{ID: packet.ID, CWD: candidateRoot, RepositoryRoot: candidateRoot,
		RepositoryIdentity: identity.Digest, BaseSHA: candidate.BaseRevision, OverlayDigest: packet.RequestDigest,
		TaskID: packet.AttemptID, BaseDigest: "sha256:" + packet.CandidateDigest}
	limits := projectworkspace.Limits{MaxFiles: 10000, MaxFileBytes: int(runtime.Limits.MaxCandidateFileBytes), MaxTotalBytes: int(runtime.Limits.MaxCandidateBytes)}
	return ImportExternalGitCandidate(host, candidateRoot, candidate.BaseRevision, candidate.Revision, packet.CandidateDigest, request, handle, limits)
}

func validateGitCandidateProposals(proposals []agentexec.CandidateFile, delta projectworkspace.Delta) error {
	writes := make(map[string]projectworkspace.Change, len(delta.Changes))
	for _, change := range delta.Changes {
		if change.Kind == projectworkspace.ChangeAdd || change.Kind == projectworkspace.ChangeModify ||
			change.Kind == projectworkspace.ChangeRename || change.Kind == projectworkspace.ChangeDelete {
			writes[change.Path] = change
		}
	}
	seen := make(map[string]bool, len(proposals))
	for _, proposal := range proposals {
		change, ok := writes[proposal.Path]
		if !ok || seen[proposal.Path] {
			return fmt.Errorf("model proposal %s contradicts the observed Git candidate", proposal.Path)
		}
		if proposal.Delete {
			if change.Kind != projectworkspace.ChangeDelete || proposal.Mode != "" || proposal.Content != "" {
				return fmt.Errorf("model deletion %s contradicts the observed Git candidate", proposal.Path)
			}
		} else if change.Kind == projectworkspace.ChangeDelete || snapshotMode(proposal.Mode) != change.Mode || string(change.Content) != proposal.Content {
			return fmt.Errorf("model proposal %s contradicts the observed Git candidate", proposal.Path)
		}
		seen[proposal.Path] = true
	}
	return nil
}

func cloneExternalGitCandidate(candidate *ExternalGitCandidate) *ExternalGitCandidate {
	if candidate == nil {
		return nil
	}
	copy := *candidate
	return &copy
}

func externalResponseDigest(response []byte, candidate *ExternalGitCandidate, delta *projectworkspace.Delta) string {
	var deltaDigest string
	if delta != nil {
		deltaDigest = delta.Digest
	}
	return mustDigest(struct {
		ResponseDigest string                `json:"responseDigest"`
		GitCandidate   *ExternalGitCandidate `json:"gitCandidate,omitempty"`
		DeltaDigest    string                `json:"deltaDigest,omitempty"`
	}{sha256Digest(response), candidate, deltaDigest})
}

func externalGitDeltaDigest(delta projectworkspace.Delta) string {
	return mustDigest(struct {
		RepositoryIdentity string                    `json:"repositoryIdentity"`
		BaseSHA            string                    `json:"baseSha"`
		OverlayDigest      string                    `json:"overlayDigest"`
		TaskID             string                    `json:"taskId"`
		BaseDigest         string                    `json:"baseDigest"`
		Changes            []projectworkspace.Change `json:"changes"`
	}{delta.RepositoryIdentity, delta.BaseSHA, delta.OverlayDigest, delta.TaskID, delta.BaseDigest, delta.Changes})
}

func validateExternalSubmissionBinding(host Host, root string, plan PlanRecord) error {
	if host.Load == nil || host.FromSnapshot == nil {
		return errors.New("project runtime requires Host load/snapshot")
	}
	if err := requireCurrentRuntimeBinding(plan); err != nil {
		return err
	}
	if err := validateExplorationReadiness(host, root, plan); err != nil {
		return err
	}
	base, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return err
	}
	if base.Snapshot == nil || base.Snapshot.Digest() != plan.BaseSnapshot || base.Digest != plan.BaseProjectDigest || base.Report.ModelDigest != plan.BaseModelDigest {
		return ErrStale
	}
	if err := validateChangeImpact(host, root, base, plan); err != nil {
		return err
	}
	working, err := host.Load(root, "")
	if err != nil {
		return err
	}
	if working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot || working.Digest != plan.WorkingProjectDigest {
		return ErrStale
	}
	if err := repositoryMatches(root, plan); err != nil {
		return err
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		return err
	}
	policyDigest, err := methodPolicyDigest(runtime)
	if err != nil {
		return err
	}
	if policyDigest != plan.RuntimeDigest {
		return ErrStale
	}
	return nil
}

func createOrReadExternalPacket(store *runStore, dir string, plan PlanRecord, report RunReport, task ManagerTask, attemptID string, request agentexec.Request, meta externalRequestContext) (ExternalPacket, error) {
	requestDigest, err := digest(request)
	if err != nil {
		return ExternalPacket{}, err
	}
	attempt := externalAttempt(task, meta)
	candidateID, candidateDigest := meta.CandidateID, meta.CandidateDigest
	if candidateID == "" {
		candidateID = report.Candidate.ID
	}
	candidateRecordDigest := ""
	if candidateID != "" {
		candidate, err := store.readCandidate(dir, candidateID)
		if err != nil {
			return ExternalPacket{}, err
		}
		candidateRecordDigest = candidate.Digest
	}
	packetID := externalPacketID(report.ID, task.ID, attemptID, meta.Kind, meta.Phase, attempt, candidateID, candidateDigest, requestDigest)
	if !validID(packetID) {
		return ExternalPacket{}, errors.New("external packet identity is invalid")
	}
	if packet, err := readExternalPacket(dir, packetID, store); err == nil {
		if packet.TaskID != task.ID || packet.AttemptID != attemptID || packet.ManagerID != task.ManagerID ||
			packet.Kind != meta.Kind || packet.Phase != meta.Phase || packet.Attempt != attempt ||
			packet.RequestDigest != requestDigest || packet.PlanDigest != plan.Digest ||
			packet.CandidateID != candidateID || packet.CandidateDigest != candidateDigest {
			return ExternalPacket{}, ErrStale
		}
		return packet, nil
	} else if !os.IsNotExist(err) {
		return ExternalPacket{}, err
	}
	invocation, wire, err := agentexec.PrepareInvocation(request)
	if err != nil {
		return ExternalPacket{}, err
	}
	packet := ExternalPacket{APIVersion: externalPacketVersion, ID: packetID, RunID: report.ID,
		TaskID: task.ID, AttemptID: attemptID, ManagerID: task.ManagerID, Kind: meta.Kind, Phase: meta.Phase, Attempt: attempt,
		CandidateID: candidateID, CandidateDigest: candidateDigest, CandidateRecordDigest: candidateRecordDigest,
		BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, PlanDigest: plan.Digest,
		RequestDigest: requestDigest, Invocation: invocation, InvocationBytes: wire}
	packet.Digest, err = externalPacketDigest(packet)
	if err != nil {
		return ExternalPacket{}, err
	}
	externalDir := filepath.Join(dir, "external")
	if err := ensureDirectory(externalDir, store); err != nil {
		return ExternalPacket{}, err
	}
	packetDir := filepath.Join(externalDir, "packets")
	if err := ensureDirectory(packetDir, store); err != nil {
		return ExternalPacket{}, err
	}
	path := filepath.Join(packetDir, packet.ID+".json")
	if err := writeImmutableJSON(path, packet, store); err != nil {
		if recordstore.WasPublished(err) || recordstore.OutcomeUnknown(err) {
			return ExternalPacket{}, err
		}
		if existing, readErr := readExternalPacket(dir, packet.ID, store); readErr == nil && existing.Digest == packet.Digest {
			return existing, nil
		}
		return ExternalPacket{}, err
	}
	return packet, nil
}

func validateExternalPacketFresh(store *runStore, dir string, plan PlanRecord, report RunReport, task ManagerTask, packet ExternalPacket) error {
	if packet.APIVersion != externalPacketVersion || packet.ID == "" || packet.RunID != report.ID || packet.TaskID == "" || !report.ExternalExecution || packet.PlanDigest != plan.Digest || packet.BaseRevision != plan.BaseRevision || packet.BaseSnapshot != plan.BaseSnapshot {
		return ErrStale
	}
	computed, err := externalPacketDigest(packet)
	if err != nil || computed != packet.Digest {
		return errors.New("stored external packet digest is invalid")
	}
	if packet.ID != externalPacketID(packet.RunID, packet.TaskID, packet.AttemptID, packet.Kind, packet.Phase, packet.Attempt,
		packet.CandidateID, packet.CandidateDigest, packet.RequestDigest) {
		return errors.New("stored external packet identity is invalid")
	}
	meta := externalRequestContext{Kind: packet.Kind, Phase: packet.Phase, Round: packet.Attempt}
	if task.ID != packet.TaskID || task.ManagerID != packet.ManagerID || packet.AttemptID == "" || externalAttempt(task, meta) != packet.Attempt {
		return ErrStale
	}
	if packet.Kind == "projectrun-review/v1" && (task.ReviewCandidateID != packet.CandidateID || task.ReviewRound != packet.Attempt) {
		return ErrStale
	}
	if packet.CandidateID != "" {
		candidate, err := store.readCandidate(dir, packet.CandidateID)
		if err != nil || candidate.Digest != packet.CandidateRecordDigest {
			return ErrStale
		}
	}
	return nil
}

func externalAttempt(task ManagerTask, meta externalRequestContext) int {
	if meta.Kind == "projectrun-review/v1" {
		if meta.Round > 0 {
			return meta.Round
		}
		return task.ReviewRound
	}
	switch meta.Phase {
	case "integrate":
		return task.IntegrationAttempts
	default:
		return task.WorkAttempts
	}
}

func externalPacketID(runID, taskID, attemptID, kind, phase string, attempt int, candidateID, candidateDigest, requestDigest string) string {
	base := struct {
		RunID         string `json:"runId"`
		TaskID        string `json:"taskId"`
		AttemptID     string `json:"attemptId"`
		Kind          string `json:"kind"`
		Phase         string `json:"phase"`
		Attempt       int    `json:"attempt"`
		CandidateID   string `json:"candidateId"`
		CandidateHash string `json:"candidateDigest"`
		RequestDigest string `json:"requestDigest"`
	}{runID, taskID, attemptID, kind, phase, attempt, candidateID, candidateDigest, requestDigest}
	packetID := strings.TrimPrefix(mustDigest(base), "sha256:")
	if len(packetID) >= 32 {
		return packetID[:32]
	}
	return ""
}

func decodeExternalRequestContext(raw json.RawMessage) (externalRequestContext, error) {
	var meta externalRequestContext
	if len(raw) == 0 {
		return meta, errors.New("external request has no context")
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, errors.New("external request context is invalid")
	}
	return meta, nil
}

func taskForPacket(report RunReport, packet ExternalPacket) ManagerTask {
	task := findTask(report.Tasks, packet.ManagerID)
	if task != nil {
		return *task
	}
	return ManagerTask{}
}

func readExternalPackets(dir string, owners ...*runStore) ([]ExternalPacket, error) {
	externalDir, err := requireRealSubdirectory(dir, "external", owners...)
	if os.IsNotExist(err) {
		return []ExternalPacket{}, nil
	}
	if err != nil {
		return nil, err
	}
	packetDir := filepath.Join(externalDir, "packets")
	entries, err := listRecordDirectory(packetDir, owners...)
	if err != nil {
		return nil, err
	}
	packets := make([]ExternalPacket, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pending-") {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("unknown external packet entry %q", entry.Name())
		}
		packet, err := readExternalPacketByPath(filepath.Join(packetDir, entry.Name()), owners...)
		if err != nil {
			return nil, err
		}
		if entry.Name() != packet.ID+".json" {
			return nil, errors.New("external packet filename does not match identity")
		}
		packets = append(packets, packet)
	}
	sort.Slice(packets, func(i, j int) bool { return packets[i].ID < packets[j].ID })
	return packets, nil
}

func readExternalPacket(dir, packetID string, owners ...*runStore) (ExternalPacket, error) {
	if !validID(packetID) {
		return ExternalPacket{}, ErrNotFound
	}
	externalDir, err := requireRealSubdirectory(dir, "external", owners...)
	if err != nil {
		return ExternalPacket{}, err
	}
	packetDir, err := requireRealSubdirectory(externalDir, "packets", owners...)
	if err != nil {
		return ExternalPacket{}, err
	}
	return readExternalPacketByPath(filepath.Join(packetDir, packetID+".json"), owners...)
}

func readExternalPacketByPath(path string, owners ...*runStore) (ExternalPacket, error) {
	var packet ExternalPacket
	if err := readJSON(path, &packet, owners...); err != nil {
		return packet, err
	}
	wire, err := json.Marshal(packet.Invocation)
	if err != nil || string(wire) != string(packet.InvocationBytes) {
		return packet, errors.New("external packet invocation bytes do not match its envelope")
	}
	computed, err := externalPacketDigest(packet)
	if err != nil || computed != packet.Digest || packet.APIVersion != externalPacketVersion || packet.ID == "" {
		return packet, errors.New("external packet identity or digest is invalid")
	}
	if packet.ID != externalPacketID(packet.RunID, packet.TaskID, packet.AttemptID, packet.Kind, packet.Phase, packet.Attempt,
		packet.CandidateID, packet.CandidateDigest, packet.RequestDigest) {
		return packet, errors.New("external packet identity is invalid")
	}
	return packet, nil
}

func readExternalSubmission(dir, packetID string, owners ...*runStore) (externalSubmission, error) {
	var submission externalSubmission
	if !validID(packetID) {
		return submission, ErrNotFound
	}
	externalDir, err := requireRealSubdirectory(dir, "external", owners...)
	if err != nil {
		return submission, err
	}
	responseDir, err := requireRealSubdirectory(externalDir, "responses", owners...)
	if err != nil {
		return submission, err
	}
	path := filepath.Join(responseDir, packetID+".json")
	if err := readJSON(path, &submission, owners...); err != nil {
		return submission, err
	}
	if submission.APIVersion != externalPacketVersion || submission.PacketID != packetID ||
		submission.ResponseHash != externalResponseDigest(submission.Response, submission.GitCandidate, submission.GitDelta) {
		return submission, errors.New("external submission identity or digest is invalid")
	}
	if (submission.GitCandidate == nil) != (submission.GitDelta == nil) {
		return submission, errors.New("external Git submission is incomplete")
	}
	if submission.GitDelta != nil && submission.GitDelta.Digest != externalGitDeltaDigest(*submission.GitDelta) {
		return submission, errors.New("external Git delta digest is invalid")
	}
	return submission, nil
}

func externalSubmissionPath(dir, packetID string) string {
	return filepath.Join(dir, "external", "responses", packetID+".json")
}

func externalReceipt(packet ExternalPacket, response agentexec.Response) agentexec.Receipt {
	contextDigest := sha256Digest(packet.Invocation.Request.Context)
	provider := "external-packet/v1"
	return agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: packet.Invocation.RunID,
		InputDigest: packet.Invocation.InputDigest, ContextDigest: contextDigest,
		ProviderVersion: provider, ProviderVersionDigest: sha256Digest([]byte(provider)),
		Outcome: response.Outcome, Usage: response.Usage}
}

func externalPacketDigest(packet ExternalPacket) (string, error) {
	packet.Digest = ""
	return digest(packet)
}

func mustDigest(value any) string {
	d, err := digest(value)
	if err != nil {
		return ""
	}
	return d
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
