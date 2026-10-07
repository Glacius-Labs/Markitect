package government

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

type promotionCompletion struct {
	Version     int             `json:"version"`
	Intent      promotionIntent `json:"intent"`
	ObservedRef string          `json:"observedRef"`
	CompletedAt time.Time       `json:"completedAt"`
}

func writePromotionCompletion(stateDir string, intent promotionIntent) error {
	path := filepath.Join(stateDir, promotionCompletionName(intent.IdempotencyKey))
	completion := promotionCompletion{Version: 1, Intent: intent, ObservedRef: intent.NewCommit, CompletedAt: time.Now().UTC()}
	data, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := WriteOperationalRecord(path, data); err != nil {
		stored, readErr := readBoundPromotionFile(path, 64*1024)
		if readErr != nil {
			return err
		}
		var previous promotionCompletion
		if decodeErr := json.Unmarshal(stored, &previous); decodeErr != nil || previous.Version != 1 || !samePromotionIntent(previous.Intent, intent) || previous.ObservedRef != intent.NewCommit {
			return fmt.Errorf("existing completion record is not the same promotion: %w", err)
		}
	}
	return nil
}

// RecoverPromotion reconstructs an outcome from the exact intent and raw Git
// ref history. It never retries the compare-and-swap.
func RecoverPromotion(ctx context.Context, request PromotionRequest) (PromotionResult, error) {
	result := PromotionResult{Status: "incomplete", ExpectedOld: request.ExpectedOld, NewCommit: request.NewCommit}
	if ctx == nil {
		return result, errors.New("promotion recovery context is required")
	}
	if err := validatePromotionRequest(request); err != nil {
		result.Status = "rejected"
		return result, err
	}
	repo, err := realDirectory(request.Repo)
	if err != nil {
		result.Status = "rejected"
		return result, fmt.Errorf("repository: %w", err)
	}
	stateDir, err := realDirectory(request.StateDirectory)
	if err != nil {
		result.Status = "rejected"
		return result, fmt.Errorf("state directory: %w", err)
	}
	identity, err := source.IdentifyGit(repo)
	if err != nil {
		result.Status = "rejected"
		return result, fmt.Errorf("identify repository: %w", err)
	}
	commonDir, err := realDirectory(identity.CommonDir)
	if err != nil {
		return result, fmt.Errorf("Git common directory: %w", err)
	}
	if pathsOverlap(stateDir, repo) || pathsOverlap(stateDir, commonDir) {
		result.Status = "rejected"
		return result, errors.New("state directory must be disjoint from repository and Git common directory")
	}
	if err := validatePromotionObjectFormat(request, identity.ObjectFormat); err != nil {
		result.Status = "rejected"
		return result, err
	}
	result.IntentPath = filepath.Join(stateDir, promotionIntentName(request.IdempotencyKey))
	result.CompletionPath = filepath.Join(stateDir, promotionCompletionName(request.IdempotencyKey))
	data, err := readBoundPromotionFile(result.IntentPath, 32*1024)
	if err != nil {
		return result, fmt.Errorf("promotion intent unavailable; effect remains incomplete: %w", err)
	}
	var intent promotionIntent
	if err := json.Unmarshal(data, &intent); err != nil {
		return result, fmt.Errorf("promotion intent is unreadable; effect remains incomplete: %w", err)
	}
	if !matchesPromotionRequest(intent, request) {
		result.Status = "rejected"
		return result, errors.New("promotion intent does not match the exact recovery request")
	}
	if err := validateCandidateCommit(ctx, repo, request); err != nil {
		return result, fmt.Errorf("recovery candidate no longer matches the bound tree and parent: %w", err)
	}
	result.FencingToken = intent.FencingToken
	lease, err := AcquireLease(filepath.Join(commonDir, promotionLockName(request.ActiveRef)))
	if err != nil {
		return result, fmt.Errorf("cannot recover while promotion lease is held or unrecognized: %w", err)
	}
	defer lease.Close()
	if err := lease.Verify(); err != nil {
		return result, fmt.Errorf("recovery fencing lost: %w", err)
	}
	result.ActualActive, err = readPromotionRef(ctx, repo, request.ActiveRef)
	if err != nil {
		return result, fmt.Errorf("active ref unavailable; effect remains incomplete: %w", err)
	}
	completionExists := false
	if data, err := readBoundPromotionFile(result.CompletionPath, 64*1024); err == nil {
		var completion promotionCompletion
		if decodeErr := json.Unmarshal(data, &completion); decodeErr != nil || completion.Version != 1 || !samePromotionIntent(completion.Intent, intent) || completion.ObservedRef != intent.NewCommit {
			return result, errors.New("promotion completion record is malformed or bound to different inputs")
		}
		completionExists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("read promotion completion record: %w", err)
	}
	matched, err := hasExactPromotionReflogEdge(ctx, commonDir, request.ActiveRef, intent, result.ActualActive)
	if err != nil {
		return result, fmt.Errorf("promotion ref history is incomplete or ambiguous: %w", err)
	}
	if !matched {
		return result, errors.New("no exact tokenized promotion edge exists; outcome remains incomplete and will not be replayed")
	}
	latest, err := readPromotionRef(ctx, repo, request.ActiveRef)
	if err != nil || latest != result.ActualActive {
		return result, errors.New("active ref changed during recovery; outcome remains incomplete")
	}
	if err := lease.Verify(); err != nil {
		return result, fmt.Errorf("recovery fencing lost before completion record: %w", err)
	}
	if !completionExists {
		if err := writePromotionCompletion(stateDir, intent); err != nil {
			return result, fmt.Errorf("exact promotion edge found but completion record could not be published: %w", err)
		}
	}
	result.Status = "promoted"
	return result, nil
}

func matchesPromotionRequest(i promotionIntent, r PromotionRequest) bool {
	return i.Version == 2 && i.ExpectedOld == r.ExpectedOld && i.NewCommit == r.NewCommit && i.ExpectedTreeID == r.ExpectedTreeID && i.MaterialCandidateID == r.MaterialCandidateID && i.EvidenceID == r.EvidenceID && i.DecisionID == r.DecisionID && i.IdempotencyKey == r.IdempotencyKey && i.ActiveRef == r.ActiveRef && !i.CreatedAt.IsZero() && validFencingToken(i.FencingToken)
}

func validFencingToken(token string) bool {
	if len(token) != 64 {
		return false
	}
	for _, ch := range token {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}

func samePromotionIntent(a, b promotionIntent) bool {
	return a.Version == b.Version && a.ExpectedOld == b.ExpectedOld && a.NewCommit == b.NewCommit && a.ExpectedTreeID == b.ExpectedTreeID && a.MaterialCandidateID == b.MaterialCandidateID && a.EvidenceID == b.EvidenceID && a.DecisionID == b.DecisionID && a.IdempotencyKey == b.IdempotencyKey && a.ActiveRef == b.ActiveRef && a.FencingToken == b.FencingToken && a.CreatedAt.Equal(b.CreatedAt)
}

func hasExactPromotionReflogEdge(ctx context.Context, commonDir, activeRef string, intent promotionIntent, current string) (bool, error) {
	path := filepath.Join(commonDir, "logs", filepath.FromSlash(activeRef))
	data, err := readBoundPromotionFile(path, 16*1024*1024)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read raw reflog %s: %w", path, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	if len(data) == 0 || (data[len(data)-1] != '\n' && data[len(data)-1] != '\r') {
		return false, errors.New("raw reflog has a partial trailing record")
	}
	if len(lines) == 0 || lines[0] == "" {
		return false, errors.New("raw reflog has no entries")
	}
	type edge struct{ old, next, message string }
	edges := make([]edge, 0, len(lines))
	for n, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		message := ""
		if len(parts) == 2 {
			message = parts[1]
		}
		header := strings.Fields(parts[0])
		if len(header) < 6 || !validReflogObjectID(header[0], len(intent.ExpectedOld)) || !validReflogObjectID(header[1], len(intent.ExpectedOld)) || !strings.HasPrefix(header[len(header)-3], "<") || !strings.HasSuffix(header[len(header)-3], ">") || !validReflogTimestamp(header[len(header)-2]) || !validReflogTimezone(header[len(header)-1]) {
			return false, fmt.Errorf("raw reflog line %d is not a complete Git identity and old/new transition", n+1)
		}
		edges = append(edges, edge{header[0], header[1], message})
	}
	if edges[len(edges)-1].next != current {
		return false, errors.New("active ref does not match the tail of its raw reflog")
	}
	targetMessage := "markitect government promotion " + intent.FencingToken
	found := -1
	for n, e := range edges {
		if e.old == intent.ExpectedOld && e.next == intent.NewCommit && e.message == targetMessage {
			if found >= 0 {
				return false, errors.New("multiple identical tokenized promotion edges are ambiguous")
			}
			found = n
		}
	}
	if found < 0 {
		return false, nil
	}
	for n := found + 1; n < len(edges); n++ {
		if edges[n].old != edges[n-1].next {
			return false, errors.New("raw reflog has a discontinuity after the promotion edge")
		}
	}
	return true, nil
}

func readBoundPromotionFile(path string, maxBytes int64) ([]byte, error) {
	parent, err := realDirectory(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("validate promotion evidence parent: %w", err)
	}
	if filepath.Clean(parent) != filepath.Clean(filepath.Dir(path)) {
		return nil, errors.New("promotion evidence parent is not canonical")
	}
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if promotionIsReparsePoint(linkInfo) {
		return nil, errors.New("promotion evidence path is a reparse point")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(linkInfo, fileInfo) {
		return nil, errors.New("promotion evidence path changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("promotion evidence exceeds size limit")
	}
	return data, nil
}

func validReflogTimestamp(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}

func validReflogTimezone(value string) bool {
	if len(value) != 5 || (value[0] != '+' && value[0] != '-') {
		return false
	}
	for n := 1; n < len(value); n++ {
		if value[n] < '0' || value[n] > '9' {
			return false
		}
	}
	hours := int(value[1]-'0')*10 + int(value[2]-'0')
	minutes := int(value[3]-'0')*10 + int(value[4]-'0')
	return hours <= 23 && minutes <= 59
}

func validReflogObjectID(value string, wantLength int) bool {
	if len(value) != wantLength {
		return false
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}
