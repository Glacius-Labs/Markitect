package government

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
	"regexp"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// PromotionRequest describes one compare-and-swap of a managed Government
// active ref. The caller must validate the candidate, fresh evidence, complete
// assent set, and decision before invoking Promote.
type PromotionRequest struct {
	Repo                string
	ActiveRef           string
	ExpectedOld         string
	NewCommit           string
	ExpectedTreeID      string
	MaterialCandidateID string
	EvidenceID          string
	DecisionID          string
	StateDirectory      string
	IdempotencyKey      string
}

// PromotionResult reports the durable intent and the observed ref state. An
// incomplete result means the caller must retain the report and inspect the
// intent/ref; it does not promise rollback or ledger atomicity.
type PromotionResult struct {
	Status         string `json:"status"`
	IntentPath     string `json:"intentPath,omitempty"`
	CompletionPath string `json:"completionPath,omitempty"`
	ExpectedOld    string `json:"expectedOld"`
	NewCommit      string `json:"newCommit"`
	ActualActive   string `json:"actualActive,omitempty"`
	FencingToken   string `json:"fencingToken,omitempty"`
}

type promotionIntent struct {
	Version             int       `json:"version"`
	ExpectedOld         string    `json:"expectedOld"`
	NewCommit           string    `json:"newCommit"`
	ExpectedTreeID      string    `json:"expectedTreeId"`
	MaterialCandidateID string    `json:"materialCandidateId"`
	EvidenceID          string    `json:"evidenceId"`
	DecisionID          string    `json:"decisionId"`
	IdempotencyKey      string    `json:"idempotencyKey"`
	ActiveRef           string    `json:"activeRef"`
	FencingToken        string    `json:"fencingToken"`
	CreatedAt           time.Time `json:"createdAt"`
}

var promotionComponent = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Promote installs a durable external intent, then compare-and-swaps exactly
// one dedicated managed ref. It never updates a checked-out branch.
func Promote(ctx context.Context, request PromotionRequest) (PromotionResult, error) {
	result := PromotionResult{Status: "rejected", ExpectedOld: request.ExpectedOld, NewCommit: request.NewCommit}
	if ctx == nil {
		return result, errors.New("promotion context is required")
	}
	if err := validatePromotionRequest(request); err != nil {
		return result, err
	}
	repo, err := realDirectory(request.Repo)
	if err != nil {
		return result, fmt.Errorf("repository: %w", err)
	}
	stateDir, err := realDirectory(request.StateDirectory)
	if err != nil {
		return result, fmt.Errorf("state directory: %w", err)
	}
	identity, err := source.IdentifyGit(repo)
	if err != nil {
		return result, fmt.Errorf("identify repository: %w", err)
	}
	commonDir, err := realDirectory(identity.CommonDir)
	if err != nil {
		return result, fmt.Errorf("Git common directory: %w", err)
	}
	gitDir, err := realDirectory(identity.GitDir)
	if err != nil {
		return result, fmt.Errorf("Git directory: %w", err)
	}
	if pathsOverlap(stateDir, repo) || pathsOverlap(stateDir, commonDir) || pathsOverlap(stateDir, gitDir) {
		return result, errors.New("state directory must be disjoint from repository and Git common directory")
	}
	if err := validatePromotionObjectFormat(request, identity.ObjectFormat); err != nil {
		return result, err
	}
	if err := validateCandidateCommit(ctx, repo, request); err != nil {
		return result, err
	}
	if err := rejectCheckedOutRef(ctx, repo, request.ActiveRef); err != nil {
		return result, err
	}
	actual, err := readPromotionRef(ctx, repo, request.ActiveRef)
	if err != nil {
		return result, err
	}
	result.ActualActive = actual
	if actual != request.ExpectedOld {
		result.Status = "stale-base"
		return result, fmt.Errorf("active ref is stale: expected %s, observed %s", request.ExpectedOld, actual)
	}

	lockPath := filepath.Join(commonDir, promotionLockName(request.ActiveRef))
	lock, err := AcquireLease(lockPath)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	result.FencingToken = lock.Token()
	if err := lock.Verify(); err != nil {
		return result, fmt.Errorf("promotion fencing lost before intent: %w", err)
	}
	actual, err = readPromotionRef(ctx, repo, request.ActiveRef)
	if err != nil {
		return result, err
	}
	result.ActualActive = actual
	if actual != request.ExpectedOld {
		result.Status = "stale-base"
		return result, fmt.Errorf("active ref changed before intent: expected %s, observed %s", request.ExpectedOld, actual)
	}
	if err := lock.Verify(); err != nil {
		return result, fmt.Errorf("promotion fencing lost before intent: %w", err)
	}
	intent := promotionIntent{
		Version: 2, ExpectedOld: request.ExpectedOld, NewCommit: request.NewCommit, ExpectedTreeID: request.ExpectedTreeID,
		MaterialCandidateID: request.MaterialCandidateID, EvidenceID: request.EvidenceID,
		DecisionID: request.DecisionID, IdempotencyKey: request.IdempotencyKey,
		ActiveRef: request.ActiveRef, FencingToken: lock.Token(), CreatedAt: time.Now().UTC(),
	}
	intentPath := filepath.Join(stateDir, promotionIntentName(request.IdempotencyKey))
	result.IntentPath = intentPath
	if err := writePromotionIntent(intentPath, intent); err != nil {
		if _, statErr := os.Lstat(intentPath); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			result.Status = "incomplete"
		}
		return result, fmt.Errorf("write durable promotion intent: %w", err)
	}
	result.Status = "incomplete"
	if err := lock.Verify(); err != nil {
		return result, fmt.Errorf("promotion fencing lost before compare-and-swap: %w", err)
	}
	// The Git worktree registry has no lock shared with unrelated Git clients.
	// This final check blocks cooperative writers; a non-cooperating process can
	// still check out the ref in the interval before update-ref.
	if err := rejectCheckedOutRef(ctx, repo, request.ActiveRef); err != nil {
		return result, fmt.Errorf("active ref became checked out before compare-and-swap: %w", err)
	}
	if err := lock.Verify(); err != nil {
		return result, fmt.Errorf("promotion fencing lost immediately before compare-and-swap: %w", err)
	}
	if _, err := runPromotionGit(ctx, repo, "update-ref", "--create-reflog", "-m", "markitect government promotion "+lock.Token(), request.ActiveRef, request.NewCommit, request.ExpectedOld); err != nil {
		observed, readErr := readPromotionRef(ctx, repo, request.ActiveRef)
		if readErr == nil {
			result.ActualActive = observed
			if observed != request.ExpectedOld {
				if observed != request.NewCommit {
					result.Status = "stale-base"
				}
				return result, fmt.Errorf("active ref compare-and-swap did not apply; observed %s: %w", observed, err)
			}
		}
		return result, fmt.Errorf("promotion compare-and-swap outcome is incomplete (readback error %v): %w", readErr, err)
	}
	result.ActualActive, err = readPromotionRef(ctx, repo, request.ActiveRef)
	if err != nil {
		return result, fmt.Errorf("promotion may have applied but active ref readback failed; intent retained: %w", err)
	}
	if result.ActualActive != request.NewCommit {
		return result, fmt.Errorf("promotion compare-and-swap returned success but active ref is %s; intent retained", result.ActualActive)
	}
	result.Status = "promoted"
	result.CompletionPath = filepath.Join(stateDir, promotionCompletionName(request.IdempotencyKey))
	if err := writePromotionCompletion(stateDir, intent); err != nil {
		result.Status = "incomplete"
		return result, fmt.Errorf("promotion applied but completion record could not be written; intent retained: %w", err)
	}
	return result, nil
}

func validatePromotionRequest(r PromotionRequest) error {
	if r.Repo == "" || r.StateDirectory == "" {
		return errors.New("repository and external state directory are required")
	}
	if !strings.HasPrefix(r.ActiveRef, "refs/markitect/government/active/") {
		return errors.New("active ref must be under refs/markitect/government/active")
	}
	component := strings.TrimPrefix(r.ActiveRef, "refs/markitect/government/active/")
	if !promotionComponent.MatchString(component) || strings.HasSuffix(component, ".lock") || strings.Contains(component, "..") {
		return errors.New("active ref must end in one safe managed component")
	}
	for _, field := range []struct{ name, value string }{
		{"expected old commit", r.ExpectedOld}, {"new commit", r.NewCommit}, {"expected tree", r.ExpectedTreeID},
	} {
		if !validFullRevision(field.value) {
			return fmt.Errorf("%s must be a full lowercase Git object ID", field.name)
		}
	}
	for _, field := range []struct{ name, value string }{
		{"material candidate", r.MaterialCandidateID}, {"evidence", r.EvidenceID}, {"decision", r.DecisionID},
	} {
		if !validDigest(field.value) {
			return fmt.Errorf("%s ID must be a sha256 digest", field.name)
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" || len(r.IdempotencyKey) > 256 {
		return errors.New("idempotency key must contain 1 to 256 non-whitespace bytes")
	}
	return nil
}

func validatePromotionObjectFormat(r PromotionRequest, format string) error {
	want := 40
	if format == "sha256" {
		want = 64
	} else if format != "sha1" {
		return fmt.Errorf("unsupported Git object format %q", format)
	}
	for _, field := range []struct{ name, value string }{
		{"expected old commit", r.ExpectedOld}, {"new commit", r.NewCommit}, {"expected tree", r.ExpectedTreeID},
	} {
		if len(field.value) != want {
			return fmt.Errorf("%s length does not match repository object format %s", field.name, format)
		}
	}
	return nil
}

func validateCandidateCommit(ctx context.Context, repo string, r PromotionRequest) error {
	resolved, err := runPromotionGit(ctx, repo, "rev-parse", "--verify", r.NewCommit+"^{commit}")
	if err != nil || strings.TrimSpace(resolved) != r.NewCommit {
		if err != nil {
			return fmt.Errorf("new commit is unavailable: %w", err)
		}
		return errors.New("new commit resolved to a different object ID")
	}
	tree, err := runPromotionGit(ctx, repo, "rev-parse", "--verify", r.NewCommit+"^{tree}")
	if err != nil {
		return fmt.Errorf("read candidate tree: %w", err)
	}
	if strings.TrimSpace(tree) != r.ExpectedTreeID {
		return errors.New("candidate commit tree differs from the reviewed expected tree")
	}
	parents, err := runPromotionGit(ctx, repo, "rev-list", "--parents", "-n", "1", r.NewCommit)
	if err != nil {
		return fmt.Errorf("read candidate parent: %w", err)
	}
	parts := strings.Fields(parents)
	if len(parts) != 2 || parts[0] != r.NewCommit || parts[1] != r.ExpectedOld {
		return errors.New("candidate commit must have exactly the expected active commit as its parent")
	}
	return nil
}

func rejectCheckedOutRef(ctx context.Context, repo, activeRef string) error {
	if _, err := runPromotionGit(ctx, repo, "check-ref-format", activeRef); err != nil {
		return fmt.Errorf("invalid managed active ref: %w", err)
	}
	if out, err := runPromotionGit(ctx, repo, "symbolic-ref", "--quiet", activeRef); err == nil && strings.TrimSpace(out) != "" {
		return errors.New("active ref is symbolic and cannot be promoted")
	}
	worktrees, err := runPromotionGit(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("inspect checked-out refs: %w", err)
	}
	for _, line := range strings.Split(worktrees, "\n") {
		if strings.TrimSpace(line) == "branch "+activeRef {
			return errors.New("managed active ref is checked out in a worktree")
		}
	}
	return nil
}

func readPromotionRef(ctx context.Context, repo, ref string) (string, error) {
	out, err := runPromotionGit(ctx, repo, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("read managed active ref %s: %w", ref, err)
	}
	return strings.TrimSpace(out), nil
}

func runPromotionGit(ctx context.Context, repo string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	hooksDir, err := os.MkdirTemp("", "markitect-government-no-hooks-")
	if err != nil {
		return "", fmt.Errorf("create empty Git hooks directory: %w", err)
	}
	defer os.Remove(hooksDir)
	gitArgs := append([]string{"--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(repo), "-c", "core.hooksPath=" + filepath.ToSlash(hooksDir), "-C", repo}, args...)
	cmd := exec.CommandContext(commandCtx, "git", gitArgs...)
	cmd.Env = source.CleanGitEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("git %s exceeded 15 second operation limit", strings.Join(args, " "))
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// WriteOperationalRecord durably publishes one exclusive JSON or report file
// outside the repository tree. It syncs file contents and the parent entry
// using the platform's durable publication primitive.
func WriteOperationalRecord(path string, data []byte) error {
	return writePromotionIntentDurable(path, data)
}

// CreateOperationalDirectory creates an exclusive directory beneath an
// existing trusted parent and durably publishes its name before returning.
func CreateOperationalDirectory(parent, prefix string) (string, error) {
	if prefix == "" || strings.TrimSpace(prefix) != prefix || prefix == "." || prefix == ".." || strings.ContainsAny(prefix, `/\\`) {
		return "", errors.New("operational directory prefix must be a plain non-empty name")
	}
	resolvedParent, err := realDirectory(parent)
	if err != nil {
		return "", fmt.Errorf("operational directory parent: %w", err)
	}
	return createOperationalDirectoryDurable(resolvedParent, prefix)
}

func realDirectory(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}
	path, err := normalizePromotionPath(path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	volume := filepath.VolumeName(abs)
	remainder := strings.TrimPrefix(abs, volume+string(filepath.Separator))
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return "", statErr
		}
		if promotionIsReparsePoint(info) {
			return "", fmt.Errorf("path contains a symlink or reparse point: %s", current)
		}
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	canonical, err = normalizePromotionPath(canonical)
	if err != nil {
		return "", err
	}
	canonical = filepath.Clean(canonical)
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return canonical, nil
}

// ResolveOperationalDirectory validates and returns an absolute existing
// directory while rejecting symlinks and platform reparse-point ancestors.
func ResolveOperationalDirectory(path string) (string, error) {
	return realDirectory(path)
}

func pathsOverlap(left, right string) bool {
	rel, err := filepath.Rel(left, right)
	if err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))) {
		return true
	}
	rel, err = filepath.Rel(right, left)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)))
}

func promotionLockName(activeRef string) string {
	digest := sha256.Sum256([]byte(activeRef))
	return "markitect-government-promotion-" + hex.EncodeToString(digest[:]) + ".lock"
}

func promotionIntentName(idempotencyKey string) string {
	digest := sha256.Sum256([]byte(idempotencyKey))
	return "promotion-" + hex.EncodeToString(digest[:]) + ".json"
}

func promotionCompletionName(idempotencyKey string) string {
	digest := sha256.Sum256([]byte(idempotencyKey))
	return "promotion-completion-" + hex.EncodeToString(digest[:]) + ".json"
}

func writePromotionIntent(path string, intent promotionIntent) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writePromotionIntentDurable(path, data)
}
