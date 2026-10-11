package projectbriefing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const storePath = ".markitect/state/briefings/history.json"

var (
	ErrStaleStore       = errors.New("briefing store changed since it was read")
	ErrStaleModel       = errors.New("no briefing history is bound to the requested model digest")
	ErrAmbiguousHistory = errors.New("briefing history has multiple accepted paths to the requested model digest")
	ErrHistoryTooDeep   = errors.New("briefing history closure exceeds the bounded commit scan")
	ErrDismissal        = errors.New("dismissal requires an event and manager")
	ErrNoAcceptedModel  = errors.New("no valid committed project model exists at the selected revision")
	ErrResolution       = errors.New("verified resolution evidence is invalid or stale")
	ErrAlreadyResolved  = errors.New("model-change event already has different resolution evidence")
)

const (
	acceptedPolicy            = "committed-model policy: manifest-selected canonical .markitect/model YAML on the active first-parent branch is the accepted repository specification; exploration records and other drafts remain proposals; Git identity is not authenticated"
	maxAcceptedHistoryCommits = 4096
)

func validateStore(state Store) error {
	if state.History != nil {
		history := state.History
		if !fullObjectID(history.BaselineRevision) || strings.TrimSpace(history.BaselineModelDigest) == "" || !fullObjectID(history.Revision) || strings.TrimSpace(history.ModelDigest) == "" || history.Policy != acceptedPolicy {
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
		// The same change can be briefed at several commits, for example on a
		// topic and where it was merged; its content must then agree.
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
	if eventDigest(event.DefinitionID.Key(), event.Change, event.Before, event.After) != event.Digest {
		return fmt.Errorf("%w: event digest does not match its declared content", ErrInvalidBundle)
	}
	return nil
}

func briefingID(manager, revision, modelDigest string, eventIDs []string) string {
	payload := struct {
		Manager, Revision, ModelDigest string
		EventIDs                       []string
	}{manager, revision, modelDigest, eventIDs}
	digest := hash(payload)
	return "briefing-" + digest[:24]
}

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

// Dismissal hides an event from one Manager's notifications. It is keyed by
// the content-based event ID only, so it counts wherever that event is
// accepted: across merges, squashes, fast-forwards and on a detached HEAD.
type Dismissal struct {
	EventID   string `json:"eventId"`
	ManagerID string `json:"managerId"`
}

type Store struct {
	APIVersion  string         `json:"apiVersion"`
	History     *HistoryCursor `json:"history,omitempty"`
	Briefings   []Bundle       `json:"briefings"`
	Dismissals  []Dismissal    `json:"dismissals"`
	Resolutions []Resolution   `json:"resolutions,omitempty"`
	// uncommitted is a status hint Read fills: per accepted event without an
	// accepted resolution, the latest resolution whose delivered result is
	// in the working tree but not in HEAD's tree. It is never persisted and
	// never makes anything accepted.
	uncommitted map[string]Resolution
}

// VerifiedResolutionEvidence binds a completed full verification and applied
// candidate to its durable run, plan, checks, and complete Manager coverage.
// It records Host evidence; it is not human approval or authenticated identity.
type VerifiedResolutionEvidence struct {
	EventIDs           []string `json:"eventIds"`
	RunID              string   `json:"runId"`
	PlanDigest         string   `json:"planDigest"`
	CandidateID        string   `json:"candidateId"`
	CandidateDigest    string   `json:"candidateDigest"`
	VerificationDigest string   `json:"verificationDigest"`
	ApplyDigest        string   `json:"applyDigest"`
	FullVerifyPassed   bool     `json:"fullVerifyPassed"`
	CoveredManagerIDs  []string `json:"coveredManagerIds"`
	EvidenceRefs       []string `json:"evidenceRefs"`
	// Delivered is what Apply wrote. The resolution counts only while HEAD's
	// committed tree holds all of it.
	Delivered []DeliveredFile `json:"delivered"`
}

// DeliveredFile is one path a delivery wrote: its Git blob and mode, or its
// removal.
type DeliveredFile struct {
	Path    string `json:"path"`
	Mode    string `json:"mode,omitempty"`
	Object  string `json:"object,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}

// NewDeliveredFile records content written to filePath, or its removal, with
// the Git blob ID that content has when committed unchanged.
func NewDeliveredFile(root, filePath, mode string, content []byte, deleted bool) (DeliveredFile, error) {
	if deleted {
		return DeliveredFile{Path: filePath, Deleted: true}, nil
	}
	object, err := source.GitOutputInput(root, content, "hash-object", "--no-filters", "--stdin")
	if err != nil {
		return DeliveredFile{}, fmt.Errorf("hash delivered %s: %w", filePath, err)
	}
	return DeliveredFile{Path: filePath, Mode: mode, Object: strings.TrimSpace(string(object))}, nil
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

// Resolution is an immutable status record. Dismissal and resolution remain
// separate: one changes visibility; the other cites fresh Host evidence.
type Resolution struct {
	EventID       string                     `json:"eventId"`
	ModelRevision string                     `json:"modelRevision"`
	ModelDigest   string                     `json:"modelDigest"`
	Evidence      VerifiedResolutionEvidence `json:"evidence"`
	Digest        string                     `json:"digest"`
}

type ResolutionStatus struct {
	Status     string      `json:"status"`
	Resolution *Resolution `json:"resolution,omitempty"`
}

// Status values reported by EventResolutionStatus.
const (
	StatusResolved   = "resolved"
	StatusUnresolved = "unresolved"
	// StatusDeliveredUncommitted means a delivery's result is in the working
	// tree but not yet committed. The event is not resolved.
	StatusDeliveredUncommitted = "delivered-uncommitted"
)

// HistoryCursor records the committed model history accepted by repository
// policy. It is operational state, separate from draft/exploration content.
// The stored cursor is where accepted history was last reconciled; on another
// first-parent line it is provisional and that line's own cursor is derived.
type HistoryCursor struct {
	Policy              string `json:"policy"`
	BaselineRevision    string `json:"baselineRevision"`
	BaselineModelDigest string `json:"baselineModelDigest"`
	Revision            string `json:"revision"`
	ModelDigest         string `json:"modelDigest"`
}

// EnsureReceipt is the fixed-revision result after accepted model history has
// been reconciled. Bundles are the immutable entries added during this call.
type EnsureReceipt struct {
	BaselineRevision    string   `json:"baselineRevision"`
	BaselineModelDigest string   `json:"baselineModelDigest"`
	Revision            string   `json:"revision"`
	ModelDigest         string   `json:"modelDigest"`
	StoreDigest         string   `json:"storeDigest"`
	Bundles             []Bundle `json:"bundles"`
}

// Read validates the operational event history and returns the part of it
// accepted on the checked-out branch, with the persisted store's digest as the
// optimistic write token. Accepted history is the first-parent line of HEAD.
// A briefing counts when it was recorded at a commit on that line, and its
// events with it. A resolution of such an event counts when HEAD's committed
// tree holds everything its delivery wrote; of several, the latest on the
// line is returned. A dismissal counts wherever its event does. Other entries
// are provisional: they stay stored and count once those conditions hold. History is the line's cursor. A
// missing store has an empty deterministic state and digest.
func Read(root string) (Store, string, error) {
	state, digest, err := readStore(root)
	if err != nil || (state.History == nil && len(state.Briefings) == 0) {
		return state, digest, err
	}
	active, err := activeLine(root)
	if err != nil {
		return Store{}, "", err
	}
	view := active.accepted(state)
	view.uncommitted = active.uncommitted(state, view)
	return view, digest, nil
}

// readStore validates and returns the persisted store, provisional entries
// included, and its content digest.
func readStore(root string) (Store, string, error) {
	path, err := stateFile(root, false)
	if err != nil {
		return Store{}, "", err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		state := emptyStore()
		return state, StoreDigest(state), nil
	}
	if err != nil {
		return Store{}, "", err
	}
	if len(data) > 64<<20 {
		return Store{}, "", errors.New("briefing store exceeds size limit")
	}
	var state Store
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return Store{}, "", fmt.Errorf("decode briefing store: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Store{}, "", errors.New("briefing store contains trailing data")
	}
	if state.APIVersion != APIVersion {
		return Store{}, "", fmt.Errorf("briefing store %s has format %q, not %q; delete it and run a guided command to rebuild accepted history", storePath, state.APIVersion, APIVersion)
	}
	if state.Briefings == nil || state.Dismissals == nil {
		return Store{}, "", errors.New("briefing store has invalid shape")
	}
	if err := validateStore(state); err != nil {
		return Store{}, "", fmt.Errorf("validate briefing store: %w", err)
	}
	return state, StoreDigest(state), nil
}

// StoreDigest returns the canonical digest used as the optimistic write token.
func StoreDigest(state Store) string { return hash(state) }

// EventResolutionStatus reports conformity resolution independently of
// dismissal. Events remain available in manager context after either choice.
// Given a state from Read, it reports the latest resolution accepted on the
// checked-out branch as StatusResolved. Without one, it reports
// StatusDeliveredUncommitted, with that resolution, when a delivery's result
// is in the working tree but not committed; this is only a hint and the event
// stays unresolved. Otherwise it reports StatusUnresolved.
func EventResolutionStatus(state Store, eventID string) ResolutionStatus {
	for i := range state.Resolutions {
		if state.Resolutions[i].EventID == eventID {
			copy := state.Resolutions[i]
			return ResolutionStatus{Status: StatusResolved, Resolution: &copy}
		}
	}
	if resolution, ok := state.uncommitted[eventID]; ok {
		return ResolutionStatus{Status: StatusDeliveredUncommitted, Resolution: &resolution}
	}
	return ResolutionStatus{Status: StatusUnresolved}
}

// ResolveVerified records fresh full-Manager Verify and successful Apply
// evidence against the exact selected committed model. The caller is expected
// to invoke this only from the Host's successful after-Apply path, with
// evidence.Delivered listing what Apply wrote. The resolution counts once
// HEAD's committed tree holds that delivered result.
func ResolveVerified(root, modelRevision, modelDigest string, evidence VerifiedResolutionEvidence, expectedStoreDigest string) (string, error) {
	if evidence.Delivered == nil {
		return "", ErrResolution
	}
	evidence.Delivered = append([]DeliveredFile{}, evidence.Delivered...)
	sort.Slice(evidence.Delivered, func(i, j int) bool { return evidence.Delivered[i].Path < evidence.Delivered[j].Path })
	if strings.TrimSpace(modelRevision) == "" || strings.TrimSpace(modelDigest) == "" || !validVerifiedEvidence(evidence) {
		return "", ErrResolution
	}
	stored, storeDigest, err := readStore(root)
	if err != nil {
		return "", err
	}
	if storeDigest != expectedStoreDigest {
		return "", ErrStaleStore
	}
	active, err := activeLine(root)
	if err != nil {
		return "", err
	}
	state := active.accepted(stored)
	if state.History == nil {
		return "", ErrStaleModel
	}
	project, err := projectwork.Load(root, modelRevision)
	if err != nil || project == nil || project.Provisional || project.Model.Digest != modelDigest {
		return "", ErrStaleModel
	}
	cursorProjects, err := acceptedProjects(root, state.History.Revision)
	if err != nil {
		return "", err
	}
	if err := validateAcceptedPrefix(state, cursorProjects); err != nil {
		return "", err
	}
	if indexOf(projectRevisions(cursorProjects), modelRevision) < 0 {
		return "", ErrAmbiguousHistory
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", state.History.BaselineRevision, modelRevision); err != nil {
		return "", ErrAmbiguousHistory
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", modelRevision, state.History.Revision); err != nil {
		return "", ErrStaleModel
	}
	managerIDs := make([]string, 0, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		managerIDs = append(managerIDs, manager.ID)
	}
	sort.Strings(managerIDs)
	if !equalStrings(managerIDs, evidence.CoveredManagerIDs) {
		return "", fmt.Errorf("full verification did not cover every selected Manager: %w", ErrResolution)
	}
	events := make(map[string]Event, len(evidence.EventIDs))
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if !contains(evidence.EventIDs, event.ID) {
				continue
			}
			// The same change briefed again later on the line is not evidence
			// for this model.
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", bundle.Revision, modelRevision); err != nil {
				continue
			}
			events[event.ID] = event
		}
	}
	if len(events) != len(evidence.EventIDs) {
		return "", ErrStaleModel
	}
	for _, eventID := range evidence.EventIDs {
		for _, managerID := range events[eventID].AffectedManagers {
			if !contains(evidence.CoveredManagerIDs, managerID) {
				return "", fmt.Errorf("full verification omitted affected Manager %s: %w", managerID, ErrResolution)
			}
		}
	}
	resolutions := make([]Resolution, 0, len(evidence.EventIDs))
	for _, eventID := range evidence.EventIDs {
		resolution := Resolution{EventID: eventID, ModelRevision: modelRevision, ModelDigest: modelDigest, Evidence: evidence}
		resolution.Digest = resolutionDigest(resolution)
		resolutions = append(resolutions, resolution)
	}
	return update(root, expectedStoreDigest, func(current *Store) error {
		// A provisional resolution, for example one whose delivered result is
		// not committed here, does not block.
		accepted := active.accepted(*current)
		for _, resolution := range resolutions {
			for _, prior := range accepted.Resolutions {
				if prior.EventID != resolution.EventID {
					continue
				}
				if prior.Digest != resolution.Digest {
					return ErrAlreadyResolved
				}
			}
		}
		for _, resolution := range resolutions {
			found := false
			for _, prior := range current.Resolutions {
				if prior.Digest == resolution.Digest {
					found = true
					break
				}
			}
			if !found {
				current.Resolutions = append(current.Resolutions, resolution)
			}
		}
		// Stable, so resolutions of one event stay in recording order.
		sort.SliceStable(current.Resolutions, func(i, j int) bool { return current.Resolutions[i].EventID < current.Resolutions[j].EventID })
		return nil
	})
}

func validVerifiedEvidence(evidence VerifiedResolutionEvidence) bool {
	return sortedUniqueNonempty(evidence.EventIDs) && strings.TrimSpace(evidence.RunID) != "" && strings.TrimSpace(evidence.PlanDigest) != "" && strings.TrimSpace(evidence.CandidateID) != "" && strings.TrimSpace(evidence.CandidateDigest) != "" && strings.TrimSpace(evidence.VerificationDigest) != "" && strings.TrimSpace(evidence.ApplyDigest) != "" && evidence.FullVerifyPassed && sortedUniqueNonempty(evidence.CoveredManagerIDs) && sortedUniqueNonempty(evidence.EvidenceRefs) && evidence.Delivered != nil && validDelivered(evidence.Delivered)
}

func resolutionDigest(resolution Resolution) string {
	resolution.Digest = ""
	return hash(resolution)
}

func projectRevisions(projects []*projectwork.Project) []string {
	revisions := make([]string, 0, len(projects))
	for _, project := range projects {
		revisions = append(revisions, project.Revision)
	}
	return revisions
}

// EnsureAcceptedHistory reconciles the bounded first-parent history ending at
// targetRevision, which must lie on the first-parent line of HEAD. That line
// is the accepted history: its first valid committed project model is the
// baseline, and each subsequent canonical model-digest change gets one
// immutable briefing recorded at the line's own commit. Briefings recorded off
// the line stay provisional and neither count nor block, so a topic that was
// merged, squashed or rebased is briefed again where its change entered the
// line, with the same content-based event identities; only a fast-forward
// accepts the topic's own briefing. A briefing on the
// line that contradicts the line's transition is ambiguous and fails the call.
// Working-tree state is never inspected or accepted. Git commit identity is
// retained only as unauthenticated source metadata.
func EnsureAcceptedHistory(root, targetRevision string) (EnsureReceipt, error) {
	var receipt EnsureReceipt
	if !fullObjectID(targetRevision) {
		return receipt, ErrUncommittedModel
	}
	active, err := activeLine(root)
	if err != nil {
		return receipt, err
	}
	targetIndex := active.index(targetRevision)
	if targetIndex < 0 {
		return receipt, fmt.Errorf("accepted history target must be on the active first-parent branch: %w", ErrUncommittedModel)
	}
	projects, err := acceptedProjects(root, targetRevision)
	if err != nil {
		return receipt, err
	}
	if len(projects) == 0 || projects[len(projects)-1].Revision != targetRevision {
		return receipt, ErrNoAcceptedModel
	}
	state, digest, err := readStore(root)
	if err != nil {
		return receipt, err
	}
	if state.History == nil && len(state.Briefings) > 0 {
		return receipt, fmt.Errorf("existing briefing entries have no accepted-history cursor and cannot be safely rebased: %w", ErrAmbiguousHistory)
	}
	// Without a stored baseline on this line (no history yet, or a project
	// model that entered the line through a squash or rebase), the line starts
	// at its own first committed model.
	var baseline *HistoryCursor
	if active.cursor(state) == nil {
		first := projects[0]
		baseline = &HistoryCursor{Policy: acceptedPolicy, BaselineRevision: first.Revision, BaselineModelDigest: first.Model.Digest, Revision: first.Revision, ModelDigest: first.Model.Digest}
		state.History = baseline
	}
	view := active.acceptedHistory(state)
	accepted := projects
	if active.index(view.History.Revision) > targetIndex {
		if accepted, err = acceptedProjects(root, view.History.Revision); err != nil {
			return receipt, err
		}
	}
	if err := validateAcceptedPrefix(view, accepted); err != nil {
		return receipt, err
	}
	if baseline != nil {
		digest, err = update(root, digest, func(current *Store) error {
			current.History = baseline
			return nil
		})
		if err != nil {
			return receipt, err
		}
	}
	receipt.BaselineRevision = view.History.BaselineRevision
	receipt.BaselineModelDigest = view.History.BaselineModelDigest
	receipt.Revision = targetRevision
	receipt.ModelDigest = projects[len(projects)-1].Model.Digest
	start := indexOf(projectRevisions(projects), view.History.Revision)
	if start < 0 {
		// The line is already accepted beyond the target.
		receipt.StoreDigest = digest
		return receipt, nil
	}
	for i := start + 1; i < len(projects); i++ {
		previous, current := projects[i-1], projects[i]
		if current.Model.Digest != previous.Model.Digest {
			provenance, provenanceErr := gitProvenance(root, current.Revision)
			if provenanceErr != nil {
				return EnsureReceipt{}, provenanceErr
			}
			bundle, generateErr := Generate(root, previous.Revision, current.Revision, provenance)
			if generateErr != nil {
				return EnsureReceipt{}, fmt.Errorf("generate accepted model change at %s: %w", current.Revision, generateErr)
			}
			_, digest, err = readStore(root)
			if err != nil {
				return EnsureReceipt{}, err
			}
			digest, err = appendCanonicalBundle(root, active, bundle, digest)
			if err != nil {
				return EnsureReceipt{}, err
			}
			receipt.Bundles = append(receipt.Bundles, bundle)
		} else {
			digest, err = advanceCursor(root, active, previous, current, digest)
			if err != nil {
				return EnsureReceipt{}, err
			}
		}
	}
	state, digest, err = readStore(root)
	if err != nil {
		return EnsureReceipt{}, err
	}
	if cursor := active.cursor(state); cursor == nil || cursor.Revision != targetRevision || cursor.ModelDigest != receipt.ModelDigest {
		return EnsureReceipt{}, ErrStaleModel
	}
	receipt.StoreDigest = digest
	return receipt, nil
}

func acceptedProjects(root, targetRevision string) ([]*projectwork.Project, error) {
	shallow, err := source.GitOutput(root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, fmt.Errorf("inspect repository history completeness: %w", err)
	}
	if strings.TrimSpace(string(shallow)) != "false" {
		return nil, fmt.Errorf("accepted history requires complete Git ancestry: %w", ErrAmbiguousHistory)
	}
	revisions, err := firstParentRevisions(root, targetRevision)
	if err != nil {
		return nil, fmt.Errorf("enumerate accepted first-parent history: %w", err)
	}
	projects := make([]*projectwork.Project, 0, len(revisions))
	for _, revision := range revisions {
		paths, err := source.GitOutput(root, "ls-tree", "-r", "--name-only", revision, "--", projectwork.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("inspect project manifest at %s: %w", revision, err)
		}
		if strings.TrimSpace(string(paths)) == "" {
			if len(projects) == 0 {
				continue
			}
			return nil, fmt.Errorf("project manifest disappeared at %s: %w", revision, ErrAmbiguousHistory)
		}
		project, err := projectwork.Load(root, revision)
		if err != nil {
			return nil, fmt.Errorf("load committed project model at %s: %w", revision, err)
		}
		if project == nil || project.Provisional || project.Model.Digest == "" {
			return nil, fmt.Errorf("invalid committed model at %s: %w", revision, ErrAmbiguousHistory)
		}
		for _, finding := range project.Report.Findings {
			// `incomplete` findings describe absent implementation or inventory
			// evidence and do not invalidate the declared model. Error findings
			// describe declarations that cannot be accepted as a project
			// specification (for example unresolved references or invalid
			// ownership), even when Core compilation produced a model digest.
			if finding.Severity == "error" {
				return nil, fmt.Errorf("committed project model at %s has structural finding %s (%s): %w", revision, finding.Code, finding.Subject, ErrAmbiguousHistory)
			}
		}
		projects = append(projects, project)
	}
	return projects, nil
}

func firstParentRevisions(root, revision string) ([]string, error) {
	output, err := source.GitOutput(root, "rev-list", "--first-parent", "--reverse", "--topo-order", revision)
	if err != nil {
		return nil, err
	}
	revisions := strings.Fields(string(output))
	if len(revisions) == 0 || len(revisions) > maxAcceptedHistoryCommits {
		return nil, ErrHistoryTooDeep
	}
	return revisions, nil
}

func indexOf(revisions []string, revision string) int {
	for i, item := range revisions {
		if item == revision {
			return i
		}
	}
	return -1
}

func validateAcceptedPrefix(state Store, projects []*projectwork.Project) error {
	if state.History == nil {
		return nil
	}
	if len(projects) == 0 || projects[0].Revision != state.History.BaselineRevision || projects[0].Model.Digest != state.History.BaselineModelDigest {
		return ErrAmbiguousHistory
	}
	cursorIndex := -1
	for i, project := range projects {
		if project.Revision == state.History.Revision {
			cursorIndex = i
			break
		}
	}
	if cursorIndex < 0 || projects[cursorIndex].Model.Digest != state.History.ModelDigest {
		return ErrAmbiguousHistory
	}
	bundles := make(map[string]Bundle, len(state.Briefings))
	for _, bundle := range state.Briefings {
		bundles[bundle.Revision] = bundle
	}
	for i := 1; i <= cursorIndex; i++ {
		previous, current := projects[i-1], projects[i]
		bundle, exists := bundles[current.Revision]
		if current.Model.Digest != previous.Model.Digest {
			if !exists {
				return fmt.Errorf("accepted model transition %s..%s is not completely briefed: %w", previous.Revision, current.Revision, ErrStaleModel)
			}
			if bundle.SinceRevision != previous.Revision || bundle.SinceModelDigest != previous.Model.Digest || bundle.ModelDigest != current.Model.Digest {
				return fmt.Errorf("briefing at %s contradicts the accepted model transition %s..%s: %w", current.Revision, previous.Revision, current.Revision, ErrAmbiguousHistory)
			}
		} else if exists {
			return fmt.Errorf("no-op model revision %s has a briefing bundle: %w", current.Revision, ErrAmbiguousHistory)
		}
	}
	for revision := range bundles {
		found := false
		for i := 1; i <= cursorIndex; i++ {
			if projects[i].Revision == revision {
				found = true
				break
			}
		}
		if !found {
			return ErrAmbiguousHistory
		}
	}
	return nil
}

// line is the first-parent history of the checked-out HEAD, the accepted
// history. A briefing counts only when its commit lies on the line, and its
// events count with it. A resolution of such an event counts when HEAD's
// committed tree holds its delivered result; a dismissal whenever its event
// counts. Other entries are provisional: kept, but neither counted nor
// blocking.
type line struct {
	root      string
	head      string
	revisions []string
	positions map[string]int
	reachable map[string]bool
	// tree caches HEAD's committed entries by path.
	tree map[string]treeEntry
}

// treeEntry is HEAD's entry at a path: read tells whether HEAD's tree could
// be read for it, and an empty object means HEAD has no entry there.
type treeEntry struct {
	read         bool
	mode, object string
}

func activeLine(root string) (*line, error) {
	head, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve active accepted branch: %w", ErrUncommittedModel)
	}
	revisions, err := firstParentRevisions(root, strings.TrimSpace(string(head)))
	if err != nil {
		return nil, err
	}
	active := &line{root: root, head: revisions[len(revisions)-1], revisions: revisions, positions: make(map[string]int, len(revisions)), reachable: map[string]bool{}, tree: map[string]treeEntry{}}
	for i, revision := range revisions {
		active.positions[revision] = i
	}
	return active, nil
}

// index returns the position of revision on the line, or -1 off the line.
func (l *line) index(revision string) int {
	if i, ok := l.positions[revision]; ok {
		return i
	}
	return -1
}

// contains reports whether HEAD's history contains revision.
func (l *line) contains(revision string) bool {
	if l.index(revision) >= 0 {
		return true
	}
	if known, ok := l.reachable[revision]; ok {
		return known
	}
	_, err := source.GitOutput(l.root, "merge-base", "--is-ancestor", revision, l.head)
	l.reachable[revision] = err == nil
	return err == nil
}

// position orders revision along the line: its index on the line, the index
// of the line commit that merged it in, or -1 outside HEAD's history.
func (l *line) position(revision string) int {
	if i := l.index(revision); i >= 0 {
		return i
	}
	if !l.contains(revision) {
		return -1
	}
	low, high := 0, len(l.revisions)-1
	for low < high {
		middle := (low + high) / 2
		if _, err := source.GitOutput(l.root, "merge-base", "--is-ancestor", revision, l.revisions[middle]); err == nil {
			high = middle
		} else {
			low = middle + 1
		}
	}
	return low
}

// readTree loads HEAD's committed entries for paths into the cache, in
// batches so the command line stays short. A path HEAD lacks is read with no
// entry. When the read fails, its paths stay unread, and a delivery that
// depends on them is not verified.
func (l *line) readTree(paths []string) {
	pending := make([]string, 0, len(paths))
	for _, entry := range paths {
		if _, ok := l.tree[entry]; !ok {
			l.tree[entry] = treeEntry{}
			pending = append(pending, entry)
		}
	}
	for len(pending) > 0 {
		batch := pending
		if len(batch) > 200 {
			batch = batch[:200]
		}
		pending = pending[len(batch):]
		output, err := source.GitOutput(l.root, append([]string{"--literal-pathspecs", "ls-tree", "-r", "-z", l.head, "--"}, batch...)...)
		if err != nil {
			continue
		}
		for _, entry := range batch {
			l.tree[entry] = treeEntry{read: true}
		}
		for _, record := range bytes.Split(output, []byte{0}) {
			tab := bytes.IndexByte(record, '\t')
			if tab < 0 {
				continue
			}
			if fields := strings.Fields(string(record[:tab])); len(fields) == 3 {
				l.tree[string(record[tab+1:])] = treeEntry{read: true, mode: fields[0], object: fields[2]}
			}
		}
	}
}

// inHead reports whether HEAD's committed tree holds a delivered result: each
// written path with its blob and mode, and no entry at each removed path.
// known is false when HEAD's tree could not be read for one of the paths.
func (l *line) inHead(files []DeliveredFile) (present, known bool) {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	l.readTree(paths)
	present = true
	for _, file := range files {
		entry := l.tree[file.Path]
		if !entry.read {
			return false, false
		}
		if file.Deleted {
			if entry.object != "" {
				present = false
			}
		} else if entry.mode != file.Mode || entry.object != file.Object {
			present = false
		}
	}
	return present, true
}

// present reports whether HEAD's committed tree verifiably holds a delivered
// result; an unreadable tree never verifies one.
func (l *line) present(files []DeliveredFile) bool {
	present, known := l.inHead(files)
	return present && known
}

// uncommitted returns, per accepted event in view without an accepted
// resolution, the latest resolution whose delivered result the working tree
// holds although HEAD's tree does not. It reads the working tree only to tell
// users why an event is not resolved yet; it never makes anything accepted.
func (l *line) uncommitted(state, view Store) map[string]Resolution {
	events, resolved := map[string]bool{}, map[string]bool{}
	for _, bundle := range view.Briefings {
		for _, event := range bundle.Events {
			events[event.ID] = true
		}
	}
	for _, resolution := range view.Resolutions {
		resolved[resolution.EventID] = true
	}
	result := map[string]Resolution{}
	for _, resolution := range state.Resolutions {
		id := resolution.EventID
		if !events[id] || resolved[id] {
			continue
		}
		if committed, known := l.inHead(resolution.Evidence.Delivered); !known || committed || !l.inWorkingTree(resolution.Evidence.Delivered) {
			continue
		}
		if prior, ok := result[id]; ok && l.position(resolution.ModelRevision) < l.position(prior.ModelRevision) {
			continue
		}
		result[id] = resolution
	}
	return result
}

// inWorkingTree reports whether the working tree holds a delivered result:
// each written path as a regular file with the delivered blob, hashed like
// NewDeliveredFile, and each removed path absent. Modes are not compared;
// not every platform's working tree keeps them.
func (l *line) inWorkingTree(files []DeliveredFile) bool {
	written, objects := []string{}, []string{}
	for _, file := range files {
		info, err := os.Lstat(filepath.Join(l.root, filepath.FromSlash(file.Path)))
		if file.Deleted {
			if !os.IsNotExist(err) {
				return false
			}
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
		written, objects = append(written, file.Path), append(objects, file.Object)
	}
	if len(written) == 0 {
		return true
	}
	output, err := source.GitOutput(l.root, append([]string{"hash-object", "--no-filters", "--"}, written...)...)
	if err != nil {
		return false
	}
	return equalStrings(strings.Fields(string(output)), objects)
}

// cursor returns the accepted-history cursor on the line: the stored cursor
// when it lies on the line, else the baseline, moved up to the last briefing
// recorded on the line. It is nil when the stored baseline is off the line.
func (l *line) cursor(state Store) *HistoryCursor {
	if state.History == nil || l.index(state.History.BaselineRevision) < 0 {
		return nil
	}
	cursor := *state.History
	if l.index(cursor.Revision) < 0 {
		cursor.Revision, cursor.ModelDigest = cursor.BaselineRevision, cursor.BaselineModelDigest
	}
	for _, bundle := range state.Briefings {
		if l.index(bundle.Revision) > l.index(cursor.Revision) {
			cursor.Revision, cursor.ModelDigest = bundle.Revision, bundle.ModelDigest
		}
	}
	return &cursor
}

// acceptedHistory returns the line's cursor and the briefings recorded on it.
func (l *line) acceptedHistory(state Store) Store {
	view := Store{APIVersion: state.APIVersion, History: l.cursor(state), Briefings: []Bundle{}, Dismissals: []Dismissal{}, Resolutions: []Resolution{}}
	for _, bundle := range state.Briefings {
		if l.index(bundle.Revision) >= 0 {
			view.Briefings = append(view.Briefings, bundle)
		}
	}
	return view
}

// accepted returns acceptedHistory with the dismissals and resolutions of its
// events that are accepted on the line, keeping only the latest resolution of
// each event: the one recorded at the latest commit along the line, and of
// equals the one recorded last.
func (l *line) accepted(state Store) Store {
	view := l.acceptedHistory(state)
	events := map[string]bool{}
	for _, bundle := range view.Briefings {
		for _, event := range bundle.Events {
			events[event.ID] = true
		}
	}
	for _, dismissal := range state.Dismissals {
		if events[dismissal.EventID] {
			view.Dismissals = append(view.Dismissals, dismissal)
		}
	}
	paths := []string{}
	for _, resolution := range state.Resolutions {
		if events[resolution.EventID] {
			for _, file := range resolution.Evidence.Delivered {
				paths = append(paths, file.Path)
			}
		}
	}
	l.readTree(paths)
	latest := map[string]int{}
	for _, resolution := range state.Resolutions {
		if !events[resolution.EventID] || !l.present(resolution.Evidence.Delivered) {
			continue
		}
		i, seen := latest[resolution.EventID]
		if !seen {
			latest[resolution.EventID] = len(view.Resolutions)
			view.Resolutions = append(view.Resolutions, resolution)
		} else if l.position(resolution.ModelRevision) >= l.position(view.Resolutions[i].ModelRevision) {
			view.Resolutions[i] = resolution
		}
	}
	return view
}

func advanceCursor(root string, active *line, previous, current *projectwork.Project, expectedDigest string) (string, error) {
	if previous == nil || current == nil || previous.Model.Digest != current.Model.Digest {
		return "", ErrStaleModel
	}
	return update(root, expectedDigest, func(state *Store) error {
		cursor := active.cursor(*state)
		if cursor == nil || cursor.Revision != previous.Revision || cursor.ModelDigest != previous.Model.Digest {
			return ErrStaleStore
		}
		if err := requireNextFirstParent(root, previous.Revision, current.Revision); err != nil {
			return err
		}
		cursor.Revision, cursor.ModelDigest = current.Revision, current.Model.Digest
		state.History = cursor
		return nil
	})
}

func requireNextFirstParent(root, sinceRevision, revision string) error {
	output, err := source.GitOutput(root, "rev-list", "--first-parent", "--parents", "-n", "1", revision)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || fields[0] != revision || fields[1] != sinceRevision {
		return ErrAmbiguousHistory
	}
	return nil
}

func gitProvenance(root, revision string) (Provenance, error) {
	author, err := source.GitOutput(root, "show", "-s", "--format=%an", revision)
	if err != nil {
		return Provenance{}, fmt.Errorf("read accepted model commit metadata: %w", err)
	}
	name := strings.TrimSpace(string(author))
	if name == "" {
		return Provenance{}, ErrMissingProvenance
	}
	return Provenance{
		DecisionReference: "git-commit:" + revision,
		Actor:             "git-commit-author:" + name,
		Authority:         acceptedPolicy,
	}, nil
}

// Write appends an immutable briefing bundle. expectedDigest must be the token
// returned by Read; pass the empty-store digest for a not-yet-created store.
func Write(root string, bundle Bundle, expectedDigest string) (string, error) {
	if err := validateBundle(bundle); err != nil {
		return "", err
	}
	if bundle.SinceRevision == bundle.Revision {
		return "", ErrInvalidBundle
	}
	regenerated, err := Generate(root, bundle.SinceRevision, bundle.Revision, bundle.Provenance)
	if err != nil {
		return "", fmt.Errorf("regenerate briefing from fixed source revisions: %w", err)
	}
	if hash(regenerated) != hash(bundle) {
		return "", fmt.Errorf("%w: submitted bundle differs from fixed-source regeneration", ErrInvalidBundle)
	}
	if bundle.SinceModelDigest == bundle.ModelDigest && len(bundle.Events) != 0 {
		return "", fmt.Errorf("%w: unchanged model digest cannot carry events", ErrInvalidBundle)
	}
	active, err := activeLine(root)
	if err != nil {
		return "", err
	}
	if bundle.SinceModelDigest == bundle.ModelDigest {
		state, current, err := readStore(root)
		if err != nil {
			return "", err
		}
		if current != expectedDigest {
			return "", ErrStaleStore
		}
		if prior, exists := bundleForRevision(state, bundle.Revision); exists {
			if hash(prior) == hash(bundle) {
				return current, nil
			}
			return "", ErrAmbiguousHistory
		}
		if state.History != nil {
			if cursor := active.cursor(state); cursor == nil || cursor.Revision != bundle.SinceRevision || cursor.ModelDigest != bundle.SinceModelDigest {
				return "", ErrAmbiguousHistory
			}
			if err := requireNextFirstParent(root, bundle.SinceRevision, bundle.Revision); err != nil {
				return "", err
			}
		}
		return current, nil
	}
	state, currentDigest, err := readStore(root)
	if err != nil {
		return "", err
	}
	if currentDigest != expectedDigest {
		return "", ErrStaleStore
	}
	if prior, exists := bundleForRevision(state, bundle.Revision); exists {
		if hash(prior) == hash(bundle) {
			return currentDigest, nil
		}
		return "", ErrAmbiguousHistory
	}
	if state.History == nil {
		projects, err := acceptedProjects(root, bundle.Revision)
		if err != nil {
			return "", err
		}
		if len(projects) < 2 || projects[0].Revision != bundle.SinceRevision {
			return "", fmt.Errorf("first persisted briefing must begin at the earliest committed project model: %w", ErrAmbiguousHistory)
		}
		currentDigest, err = update(root, currentDigest, func(current *Store) error {
			if current.History != nil || len(current.Briefings) != 0 {
				return ErrStaleStore
			}
			baseline := projects[0]
			current.History = &HistoryCursor{Policy: acceptedPolicy, BaselineRevision: baseline.Revision, BaselineModelDigest: baseline.Model.Digest, Revision: baseline.Revision, ModelDigest: baseline.Model.Digest}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return appendCanonicalBundle(root, active, bundle, currentDigest)
}

func bundleForRevision(state Store, revision string) (Bundle, bool) {
	for _, bundle := range state.Briefings {
		if bundle.Revision == revision {
			return bundle, true
		}
	}
	return Bundle{}, false
}

func appendCanonicalBundle(root string, active *line, bundle Bundle, expectedDigest string) (string, error) {
	return update(root, expectedDigest, func(state *Store) error {
		cursor := active.cursor(*state)
		if cursor == nil {
			return ErrStaleModel
		}
		if cursor.Revision != bundle.SinceRevision || cursor.ModelDigest != bundle.SinceModelDigest {
			return fmt.Errorf("briefing must extend the accepted cursor: %w", ErrStaleModel)
		}
		if err := requireNextFirstParent(root, bundle.SinceRevision, bundle.Revision); err != nil {
			return err
		}
		before, err := projectwork.Load(root, bundle.SinceRevision)
		if err != nil {
			return err
		}
		after, err := projectwork.Load(root, bundle.Revision)
		if err != nil {
			return err
		}
		if before.Model.Digest != bundle.SinceModelDigest || after.Model.Digest != bundle.ModelDigest || before.Model.Digest == after.Model.Digest {
			return ErrStaleModel
		}
		for _, prior := range state.Briefings {
			if prior.Revision == bundle.Revision {
				if hash(prior) == hash(bundle) {
					return nil
				}
				return ErrAmbiguousHistory
			}
			if prior.Global.ID == bundle.Global.ID {
				if hash(prior) == hash(bundle) {
					return nil
				}
				return errors.New("briefing identity already exists with different content")
			}
		}
		for _, event := range bundle.Events {
			for _, prior := range state.Briefings {
				for _, old := range prior.Events {
					if old.ID == event.ID && old.Digest != event.Digest {
						return errors.New("model event identity already exists with different content")
					}
				}
			}
		}
		state.Briefings = append(state.Briefings, bundle)
		cursor.Revision, cursor.ModelDigest = bundle.Revision, bundle.ModelDigest
		state.History = cursor
		sort.Slice(state.Briefings, func(i, j int) bool {
			if state.Briefings[i].Revision != state.Briefings[j].Revision {
				return state.Briefings[i].Revision < state.Briefings[j].Revision
			}
			return state.Briefings[i].Global.ID < state.Briefings[j].Global.ID
		})
		return nil
	})
}

// Dismiss records a local visibility choice for an event accepted on the
// checked-out branch. It has no field that could resolve an event or change
// its conformity state.
func Dismiss(root, eventID, managerID, expectedDigest string) (string, error) {
	if strings.TrimSpace(eventID) == "" || strings.TrimSpace(managerID) == "" {
		return "", ErrDismissal
	}
	active, err := activeLine(root)
	if err != nil {
		return "", err
	}
	return update(root, expectedDigest, func(state *Store) error {
		accepted := active.accepted(*state)
		found := false
		for _, bundle := range accepted.Briefings {
			for _, event := range bundle.Events {
				if event.ID == eventID && contains(event.AffectedManagers, managerID) {
					found = true
				}
			}
		}
		if !found {
			return errors.New("event is not available to the selected manager")
		}
		for _, d := range accepted.Dismissals {
			if d.EventID == eventID && d.ManagerID == managerID {
				return nil
			}
		}
		state.Dismissals = append(state.Dismissals, Dismissal{EventID: eventID, ManagerID: managerID})
		sort.Slice(state.Dismissals, func(i, j int) bool {
			if state.Dismissals[i].EventID != state.Dismissals[j].EventID {
				return state.Dismissals[i].EventID < state.Dismissals[j].EventID
			}
			return state.Dismissals[i].ManagerID < state.Dismissals[j].ManagerID
		})
		return nil
	})
}

// LoadForManager returns all manager briefing records that are exactly bound to
// modelDigest, with their events and a digest over the returned immutable data.
// Only briefings accepted on the checked-out branch count, as in Read. It
// rejects a stale digest instead of silently switching to a newer model.
func LoadForManager(root, modelDigest, managerID string, requestedRevision ...string) ([]Briefing, []Event, string, error) {
	if strings.TrimSpace(modelDigest) == "" || strings.TrimSpace(managerID) == "" {
		return nil, nil, "", ErrStaleModel
	}
	if len(requestedRevision) > 1 {
		return nil, nil, "", ErrStaleModel
	}
	state, _, err := Read(root)
	if err != nil {
		return nil, nil, "", err
	}
	var chain []Bundle
	if len(requestedRevision) == 1 {
		project, err := projectwork.Load(root, requestedRevision[0])
		if err != nil {
			return nil, nil, "", fmt.Errorf("load manager briefing basis: %w", err)
		}
		if project.Provisional || project.Model.Digest != modelDigest {
			return nil, nil, "", ErrStaleModel
		}
		line, err := source.GitOutput(root, "rev-list", "--reverse", "--topo-order", project.Revision)
		if err != nil {
			return nil, nil, "", fmt.Errorf("enumerate accepted revision ancestry: %w", err)
		}
		rank := make(map[string]int)
		for i, revision := range strings.Split(strings.TrimSpace(string(line)), "\n") {
			if revision != "" {
				rank[revision] = i
			}
		}
		for _, bundle := range state.Briefings {
			if _, ancestor := rank[bundle.Revision]; ancestor {
				if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", bundle.SinceRevision, bundle.Revision); err != nil {
					return nil, nil, "", ErrAmbiguousHistory
				}
				chain = append(chain, bundle)
			}
		}
		if len(chain) == 0 {
			if state.History == nil || state.History.BaselineModelDigest != modelDigest {
				return nil, nil, "", ErrStaleModel
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", state.History.BaselineRevision, project.Revision); err != nil {
				return nil, nil, "", ErrAmbiguousHistory
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", project.Revision, state.History.Revision); err != nil {
				return nil, nil, "", ErrStaleModel
			}
			return []Briefing{}, []Event{}, hash(struct {
				ModelDigest, ManagerID, Revision string
			}{modelDigest, managerID, project.Revision}), nil
		}
		sort.Slice(chain, func(i, j int) bool { return rank[chain[i].Revision] < rank[chain[j].Revision] })
		if chain[len(chain)-1].ModelDigest != modelDigest {
			return nil, nil, "", ErrStaleModel
		}
		for i := 1; i < len(chain); i++ {
			if chain[i].SinceModelDigest != chain[i-1].ModelDigest {
				return nil, nil, "", ErrAmbiguousHistory
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", chain[i-1].Revision, chain[i].Revision); err != nil {
				return nil, nil, "", ErrAmbiguousHistory
			}
		}
		if err := verifyHistoryClosure(root, chain[len(chain)-1], project); err != nil {
			return nil, nil, "", err
		}
	} else {
		var err error
		chain, err = legacyDigestChain(state.Briefings, modelDigest)
		if err != nil {
			return nil, nil, "", err
		}
	}
	briefings := make([]Briefing, 0)
	events := make([]Event, 0)
	seenEvents := map[string]bool{}
	for _, bundle := range chain {
		for _, briefing := range bundle.Managers {
			if briefing.ManagerID != managerID {
				continue
			}
			briefings = append(briefings, briefing)
			for _, event := range bundle.Events {
				if contains(briefing.EventIDs, event.ID) && !seenEvents[event.ID] {
					events = append(events, event)
					seenEvents[event.ID] = true
				}
			}
		}
	}
	return briefings, events, hash(struct {
		ModelDigest, ManagerID string
		Briefings              []Briefing
		Events                 []Event
	}{modelDigest, managerID, briefings, events}), nil
}

func verifyHistoryClosure(root string, latest Bundle, target *projectwork.Project) error {
	if target == nil || target.Provisional || target.Model.Digest != latest.ModelDigest {
		return ErrStaleModel
	}
	base, err := projectwork.Load(root, latest.Revision)
	if err != nil {
		return fmt.Errorf("load last briefed model revision: %w", err)
	}
	if base.Model.Digest != latest.ModelDigest {
		return ErrStaleModel
	}
	paths := map[string]bool{projectwork.ManifestPath: true}
	for _, path := range base.Config.ModelFiles {
		paths[path] = true
	}
	for _, path := range target.Config.ModelFiles {
		paths[path] = true
	}
	seenCommits := map[string]bool{}
	for pass := 0; pass < 16; pass++ {
		args := []string{"log", "--full-history", "--format=%H", latest.Revision + ".." + target.Revision, "--"}
		for _, path := range sortedKeys(paths) {
			args = append(args, ":(literal)"+path)
		}
		output, err := source.GitOutput(root, args...)
		if err != nil {
			return fmt.Errorf("inspect model-input history after %s: %w", latest.Revision, err)
		}
		commits := strings.Fields(string(output))
		if len(commits) > 512 {
			return ErrHistoryTooDeep
		}
		addedPath := false
		for _, commit := range commits {
			if seenCommits[commit] {
				continue
			}
			seenCommits[commit] = true
			project, err := projectwork.Load(root, commit)
			if err != nil {
				return fmt.Errorf("load model-input history commit %s: %w", commit, err)
			}
			if project.Model.Digest != latest.ModelDigest {
				return fmt.Errorf("%w: accepted model digest changed at unbriefed revision %s", ErrStaleModel, commit)
			}
			for _, path := range project.Config.ModelFiles {
				if !paths[path] {
					paths[path] = true
					addedPath = true
				}
			}
		}
		if !addedPath {
			return nil
		}
	}
	return ErrHistoryTooDeep
}

func legacyDigestChain(bundles []Bundle, modelDigest string) ([]Bundle, error) {
	chain := make([]Bundle, 0)
	cursor := modelDigest
	seen := map[string]bool{}
	for cursor != "" {
		if seen[cursor] {
			return nil, ErrAmbiguousHistory
		}
		seen[cursor] = true
		matches := make([]Bundle, 0)
		for _, bundle := range bundles {
			if bundle.ModelDigest == cursor {
				matches = append(matches, bundle)
			}
		}
		if len(matches) == 0 {
			break
		}
		if len(matches) > 1 {
			return nil, ErrAmbiguousHistory
		}
		parents := map[string]bool{}
		for _, bundle := range matches {
			parents[bundle.SinceModelDigest] = true
		}
		if len(parents) != 1 {
			return nil, ErrAmbiguousHistory
		}
		chain = append(chain, matches...)
		for parent := range parents {
			cursor = parent
		}
	}
	if len(chain) == 0 {
		return nil, ErrStaleModel
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

func emptyStore() Store {
	return Store{APIVersion: APIVersion, Briefings: []Bundle{}, Dismissals: []Dismissal{}, Resolutions: []Resolution{}}
}

func update(root, expected string, mutate func(*Store) error) (string, error) {
	path, err := stateFile(root, true)
	if err != nil {
		return "", err
	}
	lockPath := filepath.Join(filepath.Dir(path), ".briefings.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("acquire briefing store lock: %w", err)
	}
	defer func() { _ = lock.Close(); _ = os.Remove(lockPath) }()
	state, current, err := readStore(root)
	if err != nil {
		return "", err
	}
	if current != expected {
		return "", ErrStaleStore
	}
	if err := mutate(&state); err != nil {
		return "", err
	}
	if StoreDigest(state) == current {
		return current, nil
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	tmp := filepath.Join(filepath.Dir(path), fmt.Sprintf(".briefings-%d.tmp", time.Now().UnixNano()))
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return StoreDigest(state), nil
}

func stateFile(root string, create bool) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("briefing store root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("briefing store root must be a real directory")
	}
	meta := filepath.Join(abs, ".markitect")
	stateDir := filepath.Join(meta, "state")
	briefingsDir := filepath.Join(stateDir, "briefings")
	for _, dir := range []string{meta, stateDir, briefingsDir} {
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) && create {
			if err = os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
				return "", err
			}
			info, err = os.Lstat(dir)
		}
		if err != nil {
			if os.IsNotExist(err) && !create {
				return filepath.Join(abs, filepath.FromSlash(storePath)), nil
			}
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsafe briefing state directory %s", dir)
		}
	}
	path := filepath.Join(abs, filepath.FromSlash(storePath))
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("briefing store target is not a regular file")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return path, nil
}

// DigestBytes exposes a stable digest for CLI optimistic-concurrency prompts.
func DigestBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
