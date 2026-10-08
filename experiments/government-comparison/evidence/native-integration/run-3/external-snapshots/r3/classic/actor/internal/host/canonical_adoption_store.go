package host

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// CanonicalAdoptionApply reports durable ledger selection. It deliberately
// contains no VerificationResult: adoption records remain unverified.
type CanonicalAdoptionApply struct {
	Status                string                    `json:"status"`
	PlanDigest            string                    `json:"planDigest"`
	Record                *records.ProjectionRecord `json:"record,omitempty"`
	LedgerHead            string                    `json:"ledgerHead,omitempty"`
	ActiveRecordIDs       []string                  `json:"activeRecordIds,omitempty"`
	ActiveSelectionStatus string                    `json:"activeSelectionStatus,omitempty"`
}

// CanonicalAdoptionEvidencePaths returns the exact immutable blobs needed by
// durable adoption: fixed canonical source, selected artifacts and runtime
// checkInputs. It intentionally does not inventory or read the whole repo.
func CanonicalAdoptionEvidencePaths(fixed *CanonicalSource, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig) []string {
	paths := []string{}
	if fixed != nil && fixed.Snapshot != nil {
		for name := range fixed.Snapshot.Files {
			paths = append(paths, name)
		}
	}
	paths = append(paths, selection.Artifacts...)
	paths = append(paths, cfg.CheckInputs...)
	sort.Strings(paths)
	unique := paths[:0]
	for _, name := range paths {
		if name != "" && (len(unique) == 0 || unique[len(unique)-1] != name) {
			unique = append(unique, name)
		}
	}
	return unique
}

// PrepareCanonicalAdoptionForRuntime reads the current external ledger without
// creating it and binds its exact state into the adoption plan.
func PrepareCanonicalAdoptionForRuntime(root string, fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig) (CanonicalAdoptionPlan, error) {
	if err := validateControllerConfig(cfg); err != nil {
		return CanonicalAdoptionPlan{}, err
	}
	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return CanonicalAdoptionPlan{}, err
	}
	return PrepareCanonicalAdoptionWithLedger(fixed, target, identity, selection, cfg, store, state, active)
}

// PrepareCanonicalAdoptionWithLedger binds adoption to the current external
// ledger state. A nil store means the path was validated and observed absent.
func PrepareCanonicalAdoptionWithLedger(fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig, store *recordstore.Store, state recordstore.State, active []records.ProjectionRecord) (CanonicalAdoptionPlan, error) {
	configDigest, err := digestCanonicalValue(cfg)
	if err != nil {
		return CanonicalAdoptionPlan{}, err
	}
	binding := CanonicalAdoptionLedgerBinding{Present: store != nil, ActiveRecordIDs: append([]string{}, state.ActiveSelection.RecordIDs...), ConfigDigest: configDigest}
	if store != nil {
		binding.StoreID = state.StoreID
		binding.Head = state.Head
	}
	if !binding.Present {
		binding.ActiveRecordIDs = []string{}
	}
	sort.Strings(binding.ActiveRecordIDs)
	return PrepareCanonicalDurableAdoption(fixed, target, identity, selection, binding, active)
}

// ApplyCanonicalAdoptionToLedger performs a digest-bound, no-artifact-write
// adoption under the same external lease as controller ledger operations.
// The supplied fixed snapshots are immutable Git evidence; the ledger state
// and runtime config are refreshed while the lease is held.
func ApplyCanonicalAdoptionToLedger(root string, fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig, expect string, write bool) (CanonicalAdoptionApply, error) {
	return applyCanonicalAdoptionToLedger(root, fixed, target, identity, selection, cfg, expect, write, nil)
}

type canonicalAdoptionSelectActiveFunc func(*recordstore.Store, string, []string) (recordstore.State, error)

func applyCanonicalAdoptionToLedger(root string, fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig, expect string, write bool, selectActive canonicalAdoptionSelectActiveFunc) (CanonicalAdoptionApply, error) {
	return applyCanonicalAdoptionToLedgerWithRecoveryRead(root, fixed, target, identity, selection, cfg, expect, write, selectActive, nil)
}

type canonicalAdoptionRecoveryReadFunc func(*recordstore.Store) (recordstore.State, error)

func applyCanonicalAdoptionToLedgerWithRecoveryRead(root string, fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, cfg CanonicalControllerConfig, expect string, write bool, selectActive canonicalAdoptionSelectActiveFunc, recoveryRead canonicalAdoptionRecoveryReadFunc) (CanonicalAdoptionApply, error) {
	result := CanonicalAdoptionApply{Status: "refused"}
	if !write || expect == "" {
		return result, errors.New("durable adoption requires explicit write and exact reviewed plan digest")
	}
	if err := validateControllerConfig(cfg); err != nil {
		return result, err
	}
	if _, _, _, err := readCanonicalControllerLedger(root, cfg); err != nil {
		return result, err
	}
	unlock, err := acquireCanonicalControllerLease(cfg)
	if err != nil {
		return result, err
	}
	defer unlock()

	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return result, err
	}
	plan, err := PrepareCanonicalAdoptionWithLedger(fixed, target, identity, selection, cfg, store, state, active)
	if err != nil {
		return result, err
	}
	result.PlanDigest = plan.PlanDigest
	if plan.PlanDigest != expect {
		return result, errors.New("durable adoption approval is stale for source, selected artifacts, runtime configuration or ledger state")
	}

	// This runs only the selected fixed command checks against the exact supplied
	// immutable snapshot. It does not invoke cfg.Executor or cfg.Verifier and its
	// result is not persisted as independent verification.
	verification, err := VerifyCanonicalProjection(fixed, target, plan.Record, records.VerifierIdentity{ID: "markitect.canonical-adoption-checks", Version: "fixed-command-checks/v1", Digest: plan.PlanDigest})
	if err != nil {
		return result, err
	}
	if verification.Result.Outcome != records.OutcomePassed {
		return result, errors.New("durable adoption requires all selected fixed command checks to pass")
	}

	// Recheck the absent/present state immediately before any store creation or
	// append. The shared lease coordinates Host writers; the store's own CAS
	// still detects direct concurrent recordstore writers.
	store, current, currentActive, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return result, err
	}
	currentPlan, err := PrepareCanonicalAdoptionWithLedger(fixed, target, identity, selection, cfg, store, current, currentActive)
	if err != nil {
		return result, err
	}
	if currentPlan.PlanDigest != expect {
		return result, errors.New("durable adoption ledger or evidence changed before append")
	}
	if store == nil {
		gitIdentity, identifyErr := source.IdentifyGit(root)
		if identifyErr != nil {
			return result, identifyErr
		}
		store, err = recordstore.Initialize(cfg.RecordStore, canonicalControllerForbiddenRoots(gitIdentity))
		if err != nil {
			return result, fmt.Errorf("initialize absent adoption ledger after all preconditions: %w", err)
		}
		current, err = store.Read()
		if err != nil {
			return result, fmt.Errorf("ledger initialized but could not be read; no adoption record was appended: %w", err)
		}
	}

	appended, err := store.AppendAttempt(current.Head, plan.Record)
	if err != nil {
		return result, fmt.Errorf("durable adoption append failed: %w", err)
	}
	selected := append([]string{}, current.ActiveSelection.RecordIDs...)
	selected = append(selected, plan.Record.ID)
	var updated recordstore.State
	if selectActive == nil {
		updated, err = store.SelectActive(appended.Head, selected)
	} else {
		updated, err = selectActive(store, appended.Head, selected)
	}
	if err != nil {
		result.Status = records.StatePartialFailure
		result.Record = &plan.Record
		if recoveryRead == nil {
			recoveryRead = (*recordstore.Store).Read
		}
		observed, readErr := recoveryRead(store)
		if readErr != nil {
			result.ActiveSelectionStatus = "unknown"
			return result, fmt.Errorf("adoption attempt %s was appended but active selection failed; current selection state is unknown because ledger recovery read failed: %w (recovery read: %v)", plan.Record.ID, err, readErr)
		}
		result.LedgerHead = observed.Head
		result.ActiveRecordIDs = append([]string{}, observed.ActiveSelection.RecordIDs...)
		result.ActiveSelectionStatus = "observed-not-selected"
		for _, id := range observed.ActiveSelection.RecordIDs {
			if id == plan.Record.ID {
				result.ActiveSelectionStatus = "observed-selected"
				break
			}
		}
		return result, fmt.Errorf("adoption attempt %s was appended but active selection call failed; current ledger selection was observed as %s: %w", plan.Record.ID, result.ActiveSelectionStatus, err)
	}
	result.Status = records.StateMaterializedUnverified
	result.Record = &plan.Record
	result.LedgerHead = updated.Head
	result.ActiveRecordIDs = append([]string{}, updated.ActiveSelection.RecordIDs...)
	return result, nil
}
