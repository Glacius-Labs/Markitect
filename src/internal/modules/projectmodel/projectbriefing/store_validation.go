package projectbriefing

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
)

var (
	ErrUncommittedModel    = errors.New("briefings require explicit committed model revisions")
	ErrMissingProvenance   = errors.New("accepted model change requires explicit decision provenance")
	ErrRevisionNotAncestor = errors.New("since revision is not an ancestor of the accepted revision")
	ErrInvalidBundle       = errors.New("briefing bundle is invalid")
	ErrAmbiguousHistory    = errors.New("briefing history has multiple accepted paths to the requested model digest")
	ErrStaleModel          = errors.New("no briefing history is bound to the requested model digest")
	ErrResolution          = errors.New("verified resolution evidence is invalid or stale")
)

// AcceptedPolicy is the persisted policy label used by accepted history cursors.
const AcceptedPolicy = "committed-model policy: manifest-selected canonical .markitect/model YAML on the active first-parent branch is the accepted repository specification; exploration records and other drafts remain proposals; Git identity is not authenticated"

// ValidateStore checks durable briefing records without reading repository
// state. History acceptance and persistence remain Application decisions.
func ValidateStore(state Store) error {
	if state.History != nil {
		history := state.History
		if !fullObjectID(history.BaselineRevision) || strings.TrimSpace(history.BaselineModelDigest) == "" || !fullObjectID(history.Revision) || strings.TrimSpace(history.ModelDigest) == "" || history.Policy != AcceptedPolicy {
			return errors.New("accepted history cursor has invalid revisions or model digests")
		}
	}
	bundlesByID := map[string]bool{}
	bundleByRevision := map[string]Bundle{}
	eventsByID := map[string]Event{}
	managerByEvent := map[string]map[string]bool{}
	for _, bundle := range state.Briefings {
		if err := validateBundle(bundle); err != nil {
			return err
		}
		if bundle.SinceModelDigest == bundle.ModelDigest {
			return fmt.Errorf("%w: no-op briefings are not history entries", ErrInvalidBundle)
		}
		if bundlesByID[bundle.Global.ID] {
			return fmt.Errorf("duplicate briefing bundle %s", bundle.Global.ID)
		}
		bundlesByID[bundle.Global.ID] = true
		if prior, exists := bundleByRevision[bundle.Revision]; exists {
			if hash(prior) != hash(bundle) {
				return ErrAmbiguousHistory
			}
			return fmt.Errorf("duplicate briefing bundle for revision %s", bundle.Revision)
		}
		bundleByRevision[bundle.Revision] = bundle
		for _, event := range bundle.Events {
			if prior, exists := eventsByID[event.ID]; exists && prior.Digest != event.Digest {
				return fmt.Errorf("event %s has conflicting definitions", event.ID)
			}
			eventsByID[event.ID] = event
			if managerByEvent[event.ID] == nil {
				managerByEvent[event.ID] = make(map[string]bool)
			}
			for _, manager := range event.AffectedManagers {
				managerByEvent[event.ID][manager] = true
			}
		}
	}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if event.Predecessor != "" && eventsByID[event.Predecessor].DefinitionID.Key() != event.DefinitionID.Key() {
				return fmt.Errorf("event %s continues unknown or unrelated event %s", event.ID, event.Predecessor)
			}
		}
	}
	dismissals := map[string]bool{}
	for _, dismissal := range state.Dismissals {
		key := dismissal.EventID + "\x00" + dismissal.ManagerID
		if dismissal.EventID == "" || dismissal.ManagerID == "" || dismissals[key] {
			return errors.New("invalid or duplicate dismissal")
		}
		if !managerByEvent[dismissal.EventID][dismissal.ManagerID] {
			return fmt.Errorf("dismissal references unknown event-manager pair %s", dismissal.EventID)
		}
		dismissals[key] = true
	}
	resolutions := map[string]bool{}
	for _, resolution := range state.Resolutions {
		if resolution.EventID == "" || !fullObjectID(resolution.ModelRevision) || strings.TrimSpace(resolution.ModelDigest) == "" || !validVerifiedEvidence(resolution.Evidence) || resolution.Digest != resolutionDigest(resolution) {
			return ErrResolution
		}
		if resolutions[resolution.Digest] || eventsByID[resolution.EventID].ID == "" {
			return ErrResolution
		}
		if !contains(resolution.Evidence.EventIDs, resolution.EventID) {
			return ErrResolution
		}
		for _, eventID := range resolution.Evidence.EventIDs {
			if eventsByID[eventID].ID == "" {
				return ErrResolution
			}
		}
		resolutions[resolution.Digest] = true
	}
	return nil
}

func validateBundle(bundle Bundle) error {
	if bundle.APIVersion != APIVersion || !fullObjectID(bundle.SinceRevision) || !fullObjectID(bundle.Revision) || bundle.SinceRevision == bundle.Revision || bundle.SinceModelDigest == "" || bundle.ModelDigest == "" {
		return fmt.Errorf("%w: invalid revisions, model digests, or API version", ErrInvalidBundle)
	}
	if strings.TrimSpace(bundle.Provenance.DecisionReference) == "" || strings.TrimSpace(bundle.Provenance.Actor) == "" || strings.TrimSpace(bundle.Provenance.Authority) == "" {
		return fmt.Errorf("%w: incomplete decision provenance", ErrInvalidBundle)
	}
	if bundle.SinceReportDigest == "" || bundle.ReportDigest == "" || bundle.Impact.BaseDigest != bundle.SinceReportDigest || bundle.Impact.CandidateDigest != bundle.ReportDigest || bundle.Impact.Digest == "" {
		return fmt.Errorf("%w: impact is not bound to the bundle report digests", ErrInvalidBundle)
	}
	eventIDs := make([]string, 0, len(bundle.Events))
	managerSet := map[string]bool{}
	seenEvents := map[string]bool{}
	for _, event := range bundle.Events {
		if err := validateEvent(bundle, event); err != nil {
			return err
		}
		if seenEvents[event.ID] {
			return fmt.Errorf("%w: duplicate event %s", ErrInvalidBundle, event.ID)
		}
		seenEvents[event.ID] = true
		eventIDs = append(eventIDs, event.ID)
		for _, manager := range event.AffectedManagers {
			managerSet[manager] = true
		}
	}
	global := bundle.Global
	if global.ManagerID != "" || global.Revision != bundle.Revision || global.ModelDigest != bundle.ModelDigest || !equalStrings(global.EventIDs, eventIDs) || global.ID != briefingID("", global.Revision, global.ModelDigest, global.EventIDs) {
		return fmt.Errorf("%w: global briefing identity or event references do not match", ErrInvalidBundle)
	}
	if len(bundle.Managers) != len(managerSet) {
		return fmt.Errorf("%w: manager briefing set does not match affected managers", ErrInvalidBundle)
	}
	seenManagers := map[string]bool{}
	for _, briefing := range bundle.Managers {
		if briefing.ManagerID == "" || seenManagers[briefing.ManagerID] || !managerSet[briefing.ManagerID] || briefing.Revision != bundle.Revision || briefing.ModelDigest != bundle.ModelDigest || briefing.ID != briefingID(briefing.ManagerID, briefing.Revision, briefing.ModelDigest, briefing.EventIDs) {
			return fmt.Errorf("%w: invalid manager briefing identity", ErrInvalidBundle)
		}
		seenManagers[briefing.ManagerID] = true
		wantIDs := make([]string, 0)
		for _, event := range bundle.Events {
			if contains(event.AffectedManagers, briefing.ManagerID) {
				wantIDs = append(wantIDs, event.ID)
			}
		}
		if !equalStrings(briefing.EventIDs, wantIDs) {
			return fmt.Errorf("%w: manager briefing references unrelated events", ErrInvalidBundle)
		}
	}
	if bundle.Digest == "" || bundle.Digest != FullBundleDigest(bundle) {
		return fmt.Errorf("%w: full bundle digest mismatch", ErrInvalidBundle)
	}
	return nil
}

// ValidateBundle checks one durable briefing bundle's identities and digest.
func ValidateBundle(bundle Bundle) error { return validateBundle(bundle) }

func validateEvent(bundle Bundle, event Event) error {
	if event.ID == "" || len(event.Digest) != 64 || event.ID != "model-change-"+event.Digest[:24] || !fullHexDigest(event.Digest) {
		return fmt.Errorf("%w: invalid event ID or digest", ErrInvalidBundle)
	}
	if !event.DefinitionID.Valid() {
		return fmt.Errorf("%w: invalid definition identity", ErrInvalidBundle)
	}
	if event.Provenance != bundle.Provenance || event.Category == "" || event.Severity == "" {
		return fmt.Errorf("%w: event provenance or notification fields do not match", ErrInvalidBundle)
	}
	for _, values := range [][]string{event.AffectedManagers, event.AffectedArtifacts} {
		if !sortedUniqueNonempty(values) {
			return fmt.Errorf("%w: affected references are not sorted and unique", ErrInvalidBundle)
		}
	}
	switch event.Change {
	case "added":
		if event.Before != nil || event.After == nil || event.After.Identity() != event.DefinitionID {
			return fmt.Errorf("%w: invalid added definition payload", ErrInvalidBundle)
		}
	case "removed":
		if event.Before == nil || event.After != nil || event.Before.Identity() != event.DefinitionID {
			return fmt.Errorf("%w: invalid removed definition payload", ErrInvalidBundle)
		}
	case "modified":
		if event.Before == nil || event.After == nil || event.Before.Identity() != event.DefinitionID || event.After.Identity() != event.DefinitionID || sameDefinition(*event.Before, *event.After) {
			return fmt.Errorf("%w: invalid modified definition payload", ErrInvalidBundle)
		}
	default:
		return fmt.Errorf("%w: unknown model change %q", ErrInvalidBundle, event.Change)
	}
	if event.Predecessor != "" && !validEventID(event.Predecessor) {
		return fmt.Errorf("%w: invalid predecessor event ID", ErrInvalidBundle)
	}
	if eventDigest(event.Predecessor, event.DefinitionID.Key(), event.Change, event.Before, event.After) != event.Digest {
		return fmt.Errorf("%w: event digest does not match its declared content", ErrInvalidBundle)
	}
	return nil
}

func validDelivered(files []DeliveredFile) bool {
	for i, file := range files {
		if file.Path == "" || file.Path != path.Clean(file.Path) || path.IsAbs(file.Path) || file.Path == ".." || strings.HasPrefix(file.Path, "../") || strings.Contains(file.Path, "\\") || (i > 0 && files[i-1].Path >= file.Path) {
			return false
		}
		if file.Deleted {
			if file.Mode != "" || file.Object != "" {
				return false
			}
		} else if (file.Mode != "100644" && file.Mode != "100755") || !fullObjectID(file.Object) {
			return false
		}
	}
	return true
}

// ValidVerifiedEvidence checks the finite evidence shape without repository
// reads. Application separately verifies freshness, ancestry and result bytes.
func ValidVerifiedEvidence(evidence VerifiedResolutionEvidence) bool {
	return validVerifiedEvidence(evidence)
}

func validVerifiedEvidence(evidence VerifiedResolutionEvidence) bool {
	return len(evidence.EventIDs) > 0 && sortedUniqueNonempty(evidence.EventIDs) && evidence.RunID != "" && fullHexDigest(evidence.PlanDigest) && evidence.CandidateID != "" && fullHexDigest(evidence.CandidateDigest) && fullHexDigest(evidence.VerificationDigest) && fullHexDigest(evidence.ApplyDigest) && evidence.FullVerifyPassed && sortedUniqueNonempty(evidence.CoveredManagerIDs) && sortedUniqueNonempty(evidence.EvidenceRefs) && validDelivered(evidence.Delivered)
}

func resolutionDigest(resolution Resolution) string {
	resolution.Digest = ""
	return hash(resolution)
}

// ResolutionDigest returns the immutable evidence digest for a resolution.
func ResolutionDigest(resolution Resolution) string { return resolutionDigest(resolution) }

// StoreDigest returns the canonical digest used as the optimistic write token.
func StoreDigest(state Store) string { return hash(state) }

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedUniqueNonempty(values []string) bool {
	for i, value := range values {
		if value == "" || (i > 0 && values[i-1] >= value) {
			return false
		}
	}
	return true
}

func fullHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// ValidObjectID accepts the supported full Git object-ID encodings.
func ValidObjectID(id string) bool { return fullObjectID(id) }
