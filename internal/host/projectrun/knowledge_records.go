package projectrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// KnowledgeRecordState describes whether one selected immutable run record
// was available to the read-only knowledge adapter.
type KnowledgeRecordState string

const (
	KnowledgeRecordPresent KnowledgeRecordState = "present"
	KnowledgeRecordMissing KnowledgeRecordState = "missing"
	KnowledgeRecordUnknown KnowledgeRecordState = "unknown"
)

// KnowledgeCaptureState says whether the selected records remained stable
// across the port's read fence. It does not describe the freshness of their
// contents against the current repository or provider runtime.
type KnowledgeCaptureState string

const (
	KnowledgeCaptureConsistent KnowledgeCaptureState = "consistent"
	KnowledgeCaptureUnknown    KnowledgeCaptureState = "unknown"
)

// KnowledgeRuntimeBindingState is explicit because KnowledgeRecords does not
// load or compare a current model/provider runtime configuration.
type KnowledgeRuntimeBindingState string

const KnowledgeRuntimeBindingNotCompared KnowledgeRuntimeBindingState = "not-compared"

// KnowledgeRecordSet is a bounded projection of one caller-selected run.
// Missing records remain missing; this type never promotes absence to success.
// ApplyContentDigest is computed from the receipt content for read-fence
// comparison only. ApplyReport has no native digest and is not authenticated.
type KnowledgeRecordSet struct {
	Status StatusReport `json:"status"`

	PlanState      KnowledgeRecordState `json:"planState"`
	PlanAPIVersion string               `json:"planApiVersion,omitempty"`
	PlanDigest     string               `json:"planDigest,omitempty"`

	RunState      KnowledgeRecordState `json:"runState"`
	RunAPIVersion string               `json:"runApiVersion,omitempty"`
	RunDigest     string               `json:"runDigest,omitempty"`
	RunRevision   uint64               `json:"runRevision,omitempty"`

	CandidateState      KnowledgeRecordState `json:"candidateState"`
	CandidateID         string               `json:"candidateId,omitempty"`
	CandidateAPIVersion string               `json:"candidateApiVersion,omitempty"`
	// CandidateHash is the validated candidateData.Digest (the candidate's
	// canonical content digest), not a hash of the candidate ID or snapshot.
	CandidateHash string `json:"candidateHash,omitempty"`

	VerificationState      KnowledgeRecordState `json:"verificationState"`
	Verification           *VerifyReport        `json:"verification,omitempty"`
	VerificationAPIVersion string               `json:"verificationApiVersion,omitempty"`
	VerificationDigest     string               `json:"verificationDigest,omitempty"`

	ApplyState         KnowledgeRecordState `json:"applyState"`
	Apply              *ApplyReport         `json:"apply,omitempty"`
	ApplyAPIVersion    string               `json:"applyApiVersion,omitempty"`
	ApplyContentDigest string               `json:"applyContentDigest,omitempty"`

	RuntimeDigest       string                       `json:"runtimeDigest,omitempty"`
	RuntimeBindingState KnowledgeRuntimeBindingState `json:"runtimeBindingState"`
	CaptureState        KnowledgeCaptureState        `json:"captureState"`
}

// KnowledgeRecords loads validated records for exactly runID. It performs no
// provider invocation, replay, lock acquisition, directory creation, or
// repository scan. It re-reads selected record identities after loading to
// detect concurrent run-state changes; a changed fence is returned as unknown.
func KnowledgeRecords(root, runID string) (KnowledgeRecordSet, error) {
	out := KnowledgeRecordSet{
		PlanState: KnowledgeRecordMissing, RunState: KnowledgeRecordMissing,
		CandidateState: KnowledgeRecordMissing, VerificationState: KnowledgeRecordMissing,
		ApplyState: KnowledgeRecordMissing, RuntimeBindingState: KnowledgeRuntimeBindingNotCompared,
		CaptureState: KnowledgeCaptureConsistent,
	}
	store, err := newRunStore(root)
	if err != nil {
		return out, err
	}
	if !validID(runID) {
		return out, fmt.Errorf("invalid run ID %q", runID)
	}

	plan, planErr := store.readPlan(runID)
	if errors.Is(planErr, ErrNotFound) || errors.Is(planErr, os.ErrNotExist) {
		if _, err := store.readPlan(runID); !errors.Is(err, ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
			out.CaptureState = KnowledgeCaptureUnknown
		}
		return out, nil
	}
	if planErr != nil {
		return out, fmt.Errorf("read selected run plan: %w", planErr)
	}
	out.PlanState = KnowledgeRecordPresent
	out.Status.Plan = plan
	out.PlanAPIVersion, out.PlanDigest = plan.APIVersion, plan.Digest
	out.RuntimeDigest = plan.RuntimeDigest

	run, runErr := store.readLatestState(runID)
	if errors.Is(runErr, ErrNotFound) || errors.Is(runErr, os.ErrNotExist) {
		// A planned run can have an immutable initial candidate before its first
		// state record. Keep that candidate visible while keeping RunState missing.
		out.CandidateID = plan.InitialCandidateID
	} else if runErr != nil {
		return out, fmt.Errorf("read selected run state: %w", runErr)
	} else {
		if err := validateKnowledgeRunLinks(plan, runID, run); err != nil {
			return out, err
		}
		out.Status.Run = run
		out.RunState = KnowledgeRecordPresent
		out.RunAPIVersion, out.RunDigest, out.RunRevision = run.APIVersion, run.Digest, run.Revision
		out.CandidateID = run.Candidate.ID
	}

	dir, err := store.runDir(runID)
	if err != nil {
		return out, err
	}
	var candidate candidateData
	if out.CandidateID != "" {
		candidate, err = store.readCandidate(dir, out.CandidateID)
		if errors.Is(err, os.ErrNotExist) {
			out.CandidateState = KnowledgeRecordMissing
		} else if err != nil {
			return out, fmt.Errorf("read selected candidate: %w", err)
		} else {
			out.CandidateState = KnowledgeRecordPresent
			out.CandidateAPIVersion, out.CandidateHash = candidate.APIVersion, candidate.Digest
			if out.RunState == KnowledgeRecordPresent {
				if run.Candidate.Snapshot != candidate.Digest || !reflect.DeepEqual(run.Candidate, candidateRef(candidate, run.Candidate.Integrated)) {
					return out, fmt.Errorf("run state candidate reference does not match the validated candidate")
				}
			}
		}
	}

	if out.CandidateState == KnowledgeRecordPresent {
		verification, verifyErr := latestVerification(dir, out.CandidateID)
		switch {
		case knowledgeVerificationMissing(verifyErr, out.CandidateID):
			out.VerificationState = KnowledgeRecordMissing
		case verifyErr != nil:
			return out, fmt.Errorf("read selected candidate verification: %w", verifyErr)
		default:
			if err := validateKnowledgeVerification(runID, candidate, verification); err != nil {
				return out, err
			}
			out.VerificationState = KnowledgeRecordPresent
			out.Verification = &verification
			out.VerificationAPIVersion, out.VerificationDigest = verification.APIVersion, verification.Digest
		}
	}

	apply, applyDigest, applyState, err := readAppliedKnowledgeReceipt(dir, runID, out.CandidateID)
	if err != nil {
		return out, fmt.Errorf("read selected candidate Apply receipt: %w", err)
	}
	out.ApplyState, out.Apply, out.ApplyContentDigest = applyState, apply, applyDigest
	if apply != nil {
		out.ApplyAPIVersion = apply.APIVersion
	}

	if out.RunState == KnowledgeRecordPresent && run.Status == StatusApplied && out.ApplyState != KnowledgeRecordPresent {
		// A status label alone is not proof of Apply. Preserve the record states
		// and mark their relationship unresolved.
		out.CaptureState = KnowledgeCaptureUnknown
	}
	if out.RunState == KnowledgeRecordPresent && run.Status != StatusApplied && out.Apply != nil && out.Apply.Status == StatusApplied {
		out.CaptureState = KnowledgeCaptureUnknown
	}
	if out.RunState == KnowledgeRecordPresent && out.CandidateState != KnowledgeRecordPresent {
		out.CaptureState = KnowledgeCaptureUnknown
	}

	if !knowledgeFenceUnchanged(store, runID, out) {
		out.CaptureState = KnowledgeCaptureUnknown
	}
	return out, nil
}

func validateKnowledgeRunLinks(plan PlanRecord, runID string, run RunReport) error {
	if run.ID != runID || run.PlanID != plan.ID || run.APIVersion != plan.APIVersion || run.APIVersion != APIVersion {
		return fmt.Errorf("run state does not link to the selected plan")
	}
	if run.Operation != "" && plan.Operation != "" && run.Operation != plan.Operation {
		return fmt.Errorf("run state operation differs from its selected plan")
	}
	if run.BaseRevision != plan.BaseRevision || run.BaseSnapshot != plan.BaseSnapshot || run.ModelDigest != plan.ModelDigest || run.RuntimeDigest != plan.RuntimeDigest {
		return fmt.Errorf("run state basis differs from its selected plan")
	}
	if !validID(run.Candidate.ID) {
		return fmt.Errorf("run state has no valid candidate identity")
	}
	return nil
}

func validateKnowledgeVerification(runID string, candidate candidateData, report VerifyReport) error {
	if report.APIVersion != APIVersion || report.RunID != runID || report.CandidateID != candidate.ID || report.CandidateHash != candidate.Digest {
		return fmt.Errorf("verification report does not link to the selected run and candidate")
	}
	computed, err := verificationDigest(report)
	if err != nil {
		return err
	}
	if report.Digest == "" || report.Digest != computed {
		return fmt.Errorf("verification report digest mismatch")
	}
	return nil
}

// readAppliedKnowledgeReceipt returns only one successful receipt for the
// selected candidate. ApplyReport has no native digest; the derived digest is
// used solely to compare the captured content on the read fence.
func readAppliedKnowledgeReceipt(dir, runID, candidateID string) (*ApplyReport, string, KnowledgeRecordState, error) {
	if candidateID == "" {
		return nil, "", KnowledgeRecordMissing, nil
	}
	applyDir := filepath.Join(dir, "apply")
	info, err := os.Lstat(applyDir)
	if os.IsNotExist(err) {
		return nil, "", KnowledgeRecordMissing, nil
	}
	if err != nil {
		return nil, "", KnowledgeRecordUnknown, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, "", KnowledgeRecordUnknown, fmt.Errorf("runtime apply path must be a real directory")
	}
	entries, err := os.ReadDir(applyDir)
	if err != nil {
		return nil, "", KnowledgeRecordUnknown, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var selected *ApplyReport
	var selectedDigest string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".pending-") {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			return nil, "", KnowledgeRecordUnknown, fmt.Errorf("unknown Apply receipt entry %q", name)
		}
		var report ApplyReport
		if err := readJSON(filepath.Join(applyDir, name), &report); err != nil {
			return nil, "", KnowledgeRecordUnknown, err
		}
		if report.APIVersion != APIVersion || report.RunID != runID || !validID(report.CandidateID) {
			return nil, "", KnowledgeRecordUnknown, fmt.Errorf("Apply receipt %q has invalid identity or schema", name)
		}
		if report.CandidateID != candidateID || report.Status != StatusApplied {
			continue
		}
		if report.AppliedAt.IsZero() {
			return nil, "", KnowledgeRecordUnknown, fmt.Errorf("successful Apply receipt %q has no applied timestamp", name)
		}
		computed, err := digest(report)
		if err != nil {
			return nil, "", KnowledgeRecordUnknown, err
		}
		if selected != nil {
			a, _ := json.Marshal(selected)
			b, _ := json.Marshal(report)
			if !bytes.Equal(a, b) {
				return nil, "", KnowledgeRecordUnknown, fmt.Errorf("ambiguous successful Apply receipts")
			}
			continue
		}
		copy := report
		selected, selectedDigest = &copy, computed
	}
	if selected == nil {
		return nil, "", KnowledgeRecordMissing, nil
	}
	return selected, selectedDigest, KnowledgeRecordPresent, nil
}

func knowledgeFenceUnchanged(store *runStore, runID string, captured KnowledgeRecordSet) bool {
	plan, err := store.readPlan(runID)
	if err != nil || captured.PlanDigest != plan.Digest || captured.PlanAPIVersion != plan.APIVersion {
		return false
	}
	run, err := store.readLatestState(runID)
	if captured.RunState == KnowledgeRecordMissing {
		if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			// Keep fencing the planned candidate and any independently persisted
			// verification or Apply records below.
		} else {
			return false
		}
	} else if err != nil || captured.RunDigest != run.Digest || captured.RunRevision != run.Revision {
		return false
	}
	dir, err := store.runDir(runID)
	if err != nil {
		return false
	}
	if captured.CandidateID != "" && captured.CandidateState == KnowledgeRecordMissing {
		if _, err := store.readCandidate(dir, captured.CandidateID); !errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	if captured.CandidateState == KnowledgeRecordPresent {
		candidate, err := store.readCandidate(dir, captured.CandidateID)
		if err != nil || candidate.Digest != captured.CandidateHash {
			return false
		}
	}
	if captured.CandidateState == KnowledgeRecordPresent {
		verify, err := latestVerification(dir, captured.CandidateID)
		if captured.VerificationState == KnowledgeRecordMissing {
			if !knowledgeVerificationMissing(err, captured.CandidateID) {
				return false
			}
		} else if err != nil || verify.Digest != captured.VerificationDigest {
			return false
		}
	}
	apply, applyDigest, applyState, err := readAppliedKnowledgeReceipt(dir, runID, captured.CandidateID)
	if err != nil || applyState != captured.ApplyState || applyDigest != captured.ApplyContentDigest {
		return false
	}
	if (apply == nil) != (captured.Apply == nil) {
		return false
	}
	return true
}

func knowledgeVerificationMissing(err error, candidateID string) bool {
	return errors.Is(err, os.ErrNotExist) || err != nil && err.Error() == fmt.Sprintf("no verification report for candidate %s", candidateID)
}
