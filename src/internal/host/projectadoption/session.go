package projectadoption

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const (
	BrownfieldSessionVersion = "markitect.example.org/brownfield-session/v1alpha1"
	BrownfieldSessionPath    = ".markitect/drafts/brownfield"
)

// BrownfieldSession is the durable ledger for iterative reverse modeling.
// Discovery and TargetContext are immutable starting bases; each later stage
// is append-only and binds the exact prior stage digests.
type BrownfieldSession struct {
	APIVersion    string                    `json:"apiVersion"`
	ID            string                    `json:"id"`
	Source        Discovery                 `json:"source"`
	Target        SessionTarget             `json:"target"`
	TargetContext DistillationTargetContext `json:"targetContext"`
	Scopes        []ScopeStatus             `json:"scopes"`
	Iterations    []ReverseIteration        `json:"iterations"`
	Adoptions     []SessionAdoption         `json:"adoptions"`
	Digest        string                    `json:"digest"`
}

type SessionTarget struct {
	Root          string `json:"root"`
	Revision      string `json:"revision"`
	ProjectDigest string `json:"projectDigest"`
	ModelDigest   string `json:"modelDigest"`
}

// ScopeStatus keeps discovery progress separate from repository conformance.
// Transitional and unresolved scopes can never be reported as conforming.
type ScopeStatus struct {
	ScopeID string `json:"scopeId"`
	Status  string `json:"status"` // observed, modeled, transitional, unresolved, adopted
	Reason  string `json:"reason"`
}

type SessionConflict struct {
	ID          string   `json:"id"`
	ScopeID     string   `json:"scopeId"`
	QuestionID  string   `json:"questionId"`
	Description string   `json:"description"`
	EvidenceIDs []string `json:"evidenceIds"`
	Disposition string   `json:"disposition"` // Manager reports only unresolved; the owner resolves or defers separately
	Reason      string   `json:"reason"`
}

type ReverseIterationRequest struct {
	ID                    string   `json:"id"`
	ParentIterationID     string   `json:"parentIterationId,omitempty"`
	SupersedesIterationID string   `json:"supersedesIterationId,omitempty"`
	ManagerID             string   `json:"managerId"`
	EvidenceIDs           []string `json:"evidenceIds"`
	DelegationEvidenceIDs []string `json:"delegationEvidenceIds"`
	Purpose               string   `json:"purpose"`
	Review                string   `json:"review"`
}

type ManagerProposal struct {
	ManagerID       string                  `json:"managerId"`
	EvidenceIDs     []string                `json:"evidenceIds"`
	Hierarchy       []ProposedManager       `json:"hierarchy"`
	PublicContracts []ManagerPublicContract `json:"publicContracts"`
	Report          Distillation            `json:"report"`
	Digest          string                  `json:"digest"`
}

// ProposedManager is one child responsibility proposed by the assigned
// Manager. Its evidence IDs are an explicit future assignment, not an
// automatic ownership inference.
type ProposedManager struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Purpose               string   `json:"purpose"`
	ParentID              string   `json:"parentId"`
	EvidenceIDs           []string `json:"evidenceIds"`
	DelegationEvidenceIDs []string `json:"delegationEvidenceIds"`
}

type ManagerIntegration struct {
	ManagerID               string                     `json:"managerId"`
	ChildProposalDigests    []string                   `json:"childProposalDigests"`
	ChildIntegrationDigests []ChildIntegrationDigest   `json:"childIntegrationDigests,omitempty"`
	ChildContracts          []IntegratedChildContracts `json:"childContracts"`
	Report                  Distillation               `json:"report"`
	Conflicts               []SessionConflict          `json:"conflicts"`
	Digest                  string                     `json:"digest"`
}

type IntegratedChildContracts struct {
	ManagerID      string                  `json:"managerId"`
	ProposalDigest string                  `json:"proposalDigest"`
	Contracts      []ManagerPublicContract `json:"contracts"`
}

// ManagerPublicContract is a proposed child-facing contract grounded in the
// Manager's own cited claims. Accepted target contracts use the narrower
// DistillationTargetContract type directly.
type ManagerPublicContract struct {
	Contract DistillationTargetContract `json:"contract"`
	ClaimIDs []string                   `json:"claimIds"`
}

type ReverseIteration struct {
	ID                    string              `json:"id"`
	ParentIterationID     string              `json:"parentIterationId,omitempty"`
	SupersedesIterationID string              `json:"supersedesIterationId,omitempty"`
	ManagerID             string              `json:"managerId"`
	EvidenceIDs           []string            `json:"evidenceIds"`
	DelegationEvidenceIDs []string            `json:"delegationEvidenceIds"`
	Purpose               string              `json:"purpose"`
	Review                string              `json:"review"`
	TargetContextDigest   string              `json:"targetContextDigest"`
	Proposal              *ManagerProposal    `json:"proposal,omitempty"`
	Integration           *ManagerIntegration `json:"integration,omitempty"`
	Resolution            *Resolution         `json:"resolution,omitempty"`
}

type SessionAdoption struct {
	IterationID string          `json:"iterationId"`
	Plan        AdoptionPlan    `json:"plan"`
	Receipt     AdoptionReceipt `json:"receipt"`
}

type Readiness struct {
	SessionDigest       string            `json:"sessionDigest"`
	SourceCurrent       bool              `json:"sourceCurrent"`
	TargetCurrent       bool              `json:"targetCurrent"`
	Scopes              []ScopeStatus     `json:"scopes"`
	UnresolvedConflicts []SessionConflict `json:"unresolvedConflicts"`
	DeferredConflicts   []SessionConflict `json:"deferredConflicts"`
	BlockingQuestions   []Question        `json:"blockingQuestions"`
	CoverageAccounted   bool              `json:"coverageAccounted"`
	CoverageConforming  bool              `json:"coverageConforming"`
	Ready               bool              `json:"ready"`
	Findings            []string          `json:"findings"`
}

// StartBrownfieldSession freezes the source Discovery and accepted target
// model as two independent bases. It performs no model/provider call.
func StartBrownfieldSession(sourceRoot string, target *projectwork.Project, discovery Discovery, scopeStates []ScopeStatus) (BrownfieldSession, error) {
	if err := ValidateDiscovery(discovery); err != nil {
		return BrownfieldSession{}, err
	}
	if _, err := RefreshDiscovery(sourceRoot, discovery); err != nil {
		return BrownfieldSession{}, err
	}
	if target == nil || target.Snapshot == nil || target.Provisional || !validFullCommit(target.Revision) || target.Snapshot.ID != target.Revision || !validDigest(target.Digest) {
		return BrownfieldSession{}, errors.New("Brownfield session requires an accepted fixed target project snapshot")
	}
	if !sameFilesystemPath(sourceRoot, discovery.Identity.Root) {
		return BrownfieldSession{}, errors.New("Brownfield source repository root differs from Discovery")
	}
	context, err := TargetContextForProject(target)
	if err != nil {
		return BrownfieldSession{}, err
	}
	if scopeStates == nil {
		scopeStates = []ScopeStatus{}
	}
	if err := validateScopeStatuses(scopeStates, discovery); err != nil {
		return BrownfieldSession{}, err
	}
	session := BrownfieldSession{
		APIVersion: BrownfieldSessionVersion, ID: discovery.ID, Source: discovery,
		Target:        SessionTarget{Root: target.Root, Revision: target.Revision, ProjectDigest: target.Digest, ModelDigest: context.ModelDigest},
		TargetContext: context, Scopes: append([]ScopeStatus{}, scopeStates...), Iterations: []ReverseIteration{}, Adoptions: []SessionAdoption{},
	}
	sealSession(&session)
	return session, nil
}

func ValidateBrownfieldSession(session BrownfieldSession) error {
	if session.APIVersion != BrownfieldSessionVersion || !validID(session.ID) || session.ID != session.Source.ID {
		return errors.New("Brownfield session has invalid version or identity")
	}
	if err := ValidateDiscovery(session.Source); err != nil {
		return fmt.Errorf("session source: %w", err)
	}
	if session.Target.Root == "" || !validFullCommit(session.Target.Revision) || !validDigest(session.Target.ProjectDigest) || !validDigest(session.Target.ModelDigest) {
		return errors.New("Brownfield session must bind its fixed target root, revision, project and model digests")
	}
	if err := ValidateTargetContext(session.TargetContext); err != nil {
		return fmt.Errorf("session target context: %w", err)
	}
	if session.TargetContext.ProjectDigest != session.Target.ProjectDigest || session.TargetContext.Revision != session.Target.Revision || session.TargetContext.ModelDigest != session.Target.ModelDigest {
		return errors.New("Brownfield session target context differs from its target basis")
	}
	if session.Scopes == nil || session.Iterations == nil || session.Adoptions == nil {
		return errors.New("Brownfield session lists must be explicit")
	}
	if err := validateScopeStatuses(session.Scopes, session.Source); err != nil {
		return err
	}
	seen := map[string]bool{}
	seenChildManagers := map[string]bool{}
	knownScopes := map[string]bool{}
	lastRootIterationID := ""
	for _, iteration := range session.Iterations {
		if !validID(iteration.ID) || seen[iteration.ID] || strings.TrimSpace(iteration.ManagerID) == "" || strings.TrimSpace(iteration.Purpose) == "" || strings.TrimSpace(iteration.Review) == "" || iteration.TargetContextDigest != session.TargetContext.Digest || iteration.EvidenceIDs == nil || len(iteration.EvidenceIDs) == 0 {
			return fmt.Errorf("invalid or duplicate reverse iteration %q", iteration.ID)
		}
		seen[iteration.ID] = true
		if iteration.ParentIterationID != "" {
			childKey := iteration.ParentIterationID + "\x00" + iteration.ManagerID
			if seenChildManagers[childKey] {
				return fmt.Errorf("parent iteration %q has duplicate child iterations for Manager %q", iteration.ParentIterationID, iteration.ManagerID)
			}
			seenChildManagers[childKey] = true
			if iteration.SupersedesIterationID != "" {
				return fmt.Errorf("child iteration %q cannot supersede a root iteration", iteration.ID)
			}
			if !seen[iteration.ParentIterationID] {
				return fmt.Errorf("iteration %q refers to a missing or later parent iteration", iteration.ID)
			}
			parent, _ := findIteration(session, iteration.ParentIterationID)
			assignment, exists := proposedManager(parent, iteration.ManagerID)
			if !exists || assignment.ParentID != parent.ManagerID || !sameStrings(assignment.EvidenceIDs, iteration.EvidenceIDs) || !sameStrings(assignment.DelegationEvidenceIDs, iteration.DelegationEvidenceIDs) {
				return fmt.Errorf("iteration %q Manager was not proposed by its parent", iteration.ID)
			}
			if accepted, isAccepted := targetManager(session.TargetContext, iteration.ManagerID); isAccepted {
				if accepted.Parent != assignment.ParentID || accepted.Name != assignment.Name || accepted.Purpose != assignment.Purpose {
					return fmt.Errorf("accepted Manager assignment %q differs from the fixed target tree", iteration.ManagerID)
				}
			} else if !validID(iteration.ManagerID) {
				return fmt.Errorf("proposed Manager ID %q is invalid", iteration.ManagerID)
			}
		} else {
			if iteration.ManagerID != session.TargetContext.RootManagerID {
				return fmt.Errorf("root iteration %q must use the accepted root Manager", iteration.ID)
			}
			if lastRootIterationID == "" && iteration.SupersedesIterationID != "" || lastRootIterationID != "" && iteration.SupersedesIterationID != lastRootIterationID {
				return fmt.Errorf("root iteration %q must bind the immediately prior root pass", iteration.ID)
			}
			if iteration.SupersedesIterationID != "" {
				prior, exists := findIteration(session, iteration.SupersedesIterationID)
				if !exists || prior.ManagerID != iteration.ManagerID || prior.ParentIterationID != "" || prior.Integration == nil {
					return fmt.Errorf("root iteration %q must supersede an integrated prior root pass", iteration.ID)
				}
			}
			lastRootIterationID = iteration.ID
		}
		if err := validateEvidencePools(iteration.EvidenceIDs, iteration.DelegationEvidenceIDs, session.Source); err != nil {
			return fmt.Errorf("iteration %q: %w", iteration.ID, err)
		}
		if iteration.Proposal != nil {
			if err := validateManagerProposal(*iteration.Proposal, iteration, session); err != nil {
				return err
			}
			seenChildren := map[string]bool{}
			for _, proposed := range iteration.Proposal.Hierarchy {
				key := strings.ToLower(proposed.ID)
				if seenChildren[key] {
					return fmt.Errorf("Manager proposal repeats child %q", proposed.ID)
				}
				seenChildren[key] = true
				if accepted, exists := targetManager(session.TargetContext, proposed.ID); exists {
					if accepted.Parent != proposed.ParentID || accepted.Name != proposed.Name || accepted.Purpose != proposed.Purpose {
						return fmt.Errorf("accepted child Manager assignment %q differs from fixed target tree", proposed.ID)
					}
				} else if !validID(proposed.ID) {
					return fmt.Errorf("proposed Manager ID %q is invalid", proposed.ID)
				}
			}
			for _, scope := range iteration.Proposal.Report.Scopes {
				knownScopes[scope.ID] = true
			}
		}
		if iteration.Integration != nil {
			if err := validateManagerIntegration(*iteration.Integration, iteration, session); err != nil {
				return err
			}
			for _, scope := range iteration.Integration.Report.Scopes {
				knownScopes[scope.ID] = true
			}
		}
		if iteration.Resolution != nil {
			if iteration.Integration == nil {
				return fmt.Errorf("iteration %q resolution requires integrated report", iteration.ID)
			}
			if err := ValidateResolution(session.Source, iteration.Integration.Report, *iteration.Resolution); err != nil {
				return err
			}
			if iteration.Resolution.TargetBasis != session.Target.ProjectDigest {
				return fmt.Errorf("iteration %q resolution does not bind target project", iteration.ID)
			}
		}
	}
	for _, scope := range session.Scopes {
		if !knownScopes[scope.ScopeID] {
			return fmt.Errorf("scope status %q has no recorded reverse-model proposal", scope.ScopeID)
		}
	}
	for _, adoption := range session.Adoptions {
		if !seen[adoption.IterationID] {
			return errors.New("session adoption refers to an unknown iteration")
		}
		if err := ValidateAdoptionPlan(adoption.Plan); err != nil {
			return err
		}
		if adoption.Plan.TargetBasis != session.Target.ProjectDigest || adoption.Plan.DiscoveryDigest != session.Source.Digest {
			return errors.New("session adoption differs from fixed source or target basis")
		}
		if adoption.Receipt.DiscoveryDigest != session.Source.Digest || adoption.Receipt.PriorTargetBasis != session.Target.ProjectDigest || adoption.Receipt.CandidateDigest == "" {
			return errors.New("session adoption receipt does not bind fixed bases")
		}
	}
	copy := session
	copy.Digest = ""
	if !validDigest(session.Digest) || digestValue(copy) != session.Digest {
		return errors.New("Brownfield session digest mismatch")
	}
	return nil
}

func EncodeBrownfieldSession(session BrownfieldSession) ([]byte, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return nil, err
	}
	return encodeClosedJSON(session)
}

func DecodeBrownfieldSession(data []byte) (BrownfieldSession, error) {
	var value BrownfieldSession
	if err := decodeClosedJSON(data, &value); err != nil {
		return BrownfieldSession{}, err
	}
	if err := ValidateBrownfieldSession(value); err != nil {
		return BrownfieldSession{}, err
	}
	return value, nil
}

// WriteBrownfieldSession persists an immutable ledger by compare-and-swap.
// expectedDigest is empty only when creating a previously absent session.
func WriteBrownfieldSession(sourceRoot string, session BrownfieldSession, expectedDigest string) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	dir, err := sessionDirectory(sourceRoot, session.ID, true)
	if err != nil {
		return BrownfieldSession{}, err
	}
	unlock, err := acquireSessionLock(dir)
	if err != nil {
		return BrownfieldSession{}, err
	}
	defer unlock()
	if err := ensureNoIncompleteAtomicWrite(dir, "session.json"); err != nil {
		return BrownfieldSession{}, err
	}
	file := filepath.Join(dir, "session.json")
	old, readErr := readRegularLedger(file)
	if expectedDigest == session.Digest {
		if readErr == nil {
			return BrownfieldSession{}, errors.New("Brownfield session already exists")
		}
		if !errors.Is(readErr, fs.ErrNotExist) {
			return BrownfieldSession{}, readErr
		}
	} else {
		if readErr != nil {
			return BrownfieldSession{}, fmt.Errorf("read prior Brownfield session: %w", readErr)
		}
		prior, err := DecodeBrownfieldSession(old)
		if err != nil {
			return BrownfieldSession{}, err
		}
		if prior.Digest != expectedDigest {
			return BrownfieldSession{}, errors.New("Brownfield session changed; reload before writing")
		}
	}
	data, err := EncodeBrownfieldSession(session)
	if err != nil {
		return BrownfieldSession{}, err
	}
	temp := filepath.Join(dir, ".session.json.tmp")
	tempFile, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return BrownfieldSession{}, err
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(temp)
		return BrownfieldSession{}, err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(temp)
		return BrownfieldSession{}, err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(temp)
		return BrownfieldSession{}, err
	}
	if err := atomicReplaceFile(temp, file); err != nil {
		return BrownfieldSession{}, err
	}
	return session, nil
}

func LoadBrownfieldSession(sourceRoot, id string) (BrownfieldSession, error) {
	if !validID(id) {
		return BrownfieldSession{}, errors.New("invalid Brownfield session ID")
	}
	dir, err := sessionDirectory(sourceRoot, id, false)
	if err != nil {
		return BrownfieldSession{}, err
	}
	unlock, err := acquireSessionLock(dir)
	if err != nil {
		return BrownfieldSession{}, err
	}
	defer unlock()
	if err := ensureNoIncompleteAtomicWrite(dir, "session.json"); err != nil {
		return BrownfieldSession{}, err
	}
	data, err := readRegularLedger(filepath.Join(dir, "session.json"))
	if err != nil {
		return BrownfieldSession{}, err
	}
	return DecodeBrownfieldSession(data)
}

func readRegularLedger(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("unsafe Brownfield ledger file %q", path)
	}
	return os.ReadFile(path)
}

func acquireSessionLock(dir string) (func(), error) {
	unlock, err := acquireProcessFileLock(filepath.Join(dir, "session.lock"))
	if err != nil {
		return nil, fmt.Errorf("Brownfield session is being updated: %w", err)
	}
	return unlock, nil
}

// ResumeBrownfieldSession verifies both original fixed bases and returns
// readiness against the current accepted project model; it never replays an
// iteration or invokes a Manager.
func ResumeBrownfieldSession(sourceRoot, targetRoot, id string) (BrownfieldSession, Readiness, error) {
	session, err := LoadBrownfieldSession(sourceRoot, id)
	if err != nil {
		return BrownfieldSession{}, Readiness{}, err
	}
	if _, err := RefreshDiscovery(sourceRoot, session.Source); err != nil {
		return BrownfieldSession{}, Readiness{}, fmt.Errorf("source basis is stale: %w", err)
	}
	if !sameFilesystemPath(targetRoot, session.Target.Root) {
		return BrownfieldSession{}, Readiness{}, errors.New("target repository root differs from session basis")
	}
	fixedTarget, err := projectwork.Load(targetRoot, session.Target.Revision)
	if err != nil {
		return BrownfieldSession{}, Readiness{}, fmt.Errorf("reload fixed target basis: %w", err)
	}
	if fixedTarget.Digest != session.Target.ProjectDigest {
		return BrownfieldSession{}, Readiness{}, errors.New("target basis is stale; start a new session against the accepted target model")
	}
	headBytes, err := source.GitOutput(targetRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return BrownfieldSession{}, Readiness{}, fmt.Errorf("resolve current target HEAD: %w", err)
	}
	headRevision := strings.TrimSpace(string(headBytes))
	acceptedTarget, err := projectwork.Load(targetRoot, headRevision)
	if err != nil {
		return BrownfieldSession{}, Readiness{}, fmt.Errorf("reload current accepted target: %w", err)
	}
	workingTarget, err := projectwork.Load(targetRoot, "")
	if err != nil {
		return BrownfieldSession{}, Readiness{}, fmt.Errorf("reload current target worktree: %w", err)
	}
	coverage := workingTarget.Coverage
	targetCurrent := false
	worktreeMatchesHead := acceptedTarget.Snapshot != nil && workingTarget.Snapshot != nil && acceptedTarget.Snapshot.Digest() == workingTarget.Snapshot.Digest()
	if len(session.Adoptions) == 0 {
		targetCurrent = acceptedTarget.Digest == session.Target.ProjectDigest && worktreeMatchesHead
	} else {
		latest := session.Adoptions[len(session.Adoptions)-1]
		acceptedContext, contextErr := TargetContextForProject(acceptedTarget)
		if contextErr != nil {
			return BrownfieldSession{}, Readiness{}, fmt.Errorf("derive accepted target model context: %w", contextErr)
		}
		expectedModelDigest := strings.TrimPrefix(latest.Plan.Edit.Report.ModelDigest, "sha256:")
		workingModelDigest := strings.TrimPrefix(workingTarget.Report.ModelDigest, "sha256:")
		targetCurrent = expectedModelDigest != "" && acceptedContext.ModelDigest == expectedModelDigest && workingModelDigest == expectedModelDigest && worktreeMatchesHead
	}
	if targetCurrent {
		coverage = acceptedTarget.Coverage
	}
	readiness := AssessReadiness(session, coverage)
	readiness.SourceCurrent = true
	readiness.TargetCurrent = targetCurrent
	if !targetCurrent {
		if len(session.Adoptions) > 0 {
			readiness.Findings = append(readiness.Findings, "latest applied model candidate is not the clean accepted HEAD")
		} else {
			readiness.Findings = append(readiness.Findings, "current target HEAD or worktree differs from the fixed target basis")
		}
	}
	sort.Strings(readiness.Findings)
	finalizeReadiness(&readiness, session)
	return session, readiness, nil
}

func sessionDirectory(root, id string, create bool) (string, error) {
	if !validID(id) {
		return "", errors.New("invalid Brownfield session ID")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for _, relative := range []string{".markitect", filepath.Join(".markitect", "drafts"), filepath.Join(".markitect", "drafts", "brownfield"), filepath.Join(BrownfieldSessionPath, id)} {
		current := filepath.Join(abs, relative)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) && create {
			if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
				return "", err
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return "", statErr
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsafe Brownfield ledger directory %q", relative)
		}
	}
	return filepath.Join(abs, filepath.FromSlash(BrownfieldSessionPath), id), nil
}

func validateScopeStatuses(scopes []ScopeStatus, discovery Discovery) error {
	_ = discovery
	seen := map[string]bool{}
	for _, scope := range scopes {
		if !validID(scope.ScopeID) || seen[scope.ScopeID] || strings.TrimSpace(scope.Reason) == "" {
			return fmt.Errorf("scope status %q requires a stable unique scope ID and reason", scope.ScopeID)
		}
		switch scope.Status {
		case "observed", "modeled", "transitional", "unresolved", "adopted":
		default:
			return fmt.Errorf("unsupported Brownfield scope status %q", scope.Status)
		}
		seen[scope.ScopeID] = true
	}
	return nil
}

func sealSession(session *BrownfieldSession) {
	copy := *session
	copy.Digest = ""
	session.Digest = digestValue(copy)
}

func sortedUniqueStrings(values []string) ([]string, error) {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for i, value := range result {
		if strings.TrimSpace(value) == "" || (i > 0 && result[i-1] == value) {
			return nil, errors.New("list must contain unique nonempty values")
		}
	}
	return result, nil
}

func sameStrings(a, b []string) bool {
	aa, _ := sortedUniqueStringsOrEmpty(a)
	bb, _ := sortedUniqueStringsOrEmpty(b)
	return bytes.Equal([]byte(strings.Join(aa, "\x00")), []byte(strings.Join(bb, "\x00")))
}
func sortedUniqueStringsOrEmpty(values []string) ([]string, error) {
	if values == nil {
		return []string{}, nil
	}
	return sortedUniqueStrings(values)
}
