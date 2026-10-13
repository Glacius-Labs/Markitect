package projecthistory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	projectbriefing "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const storePath = ".markitect/records/briefing/accepted/record.json"

var (
	ErrStaleStore      = errors.New("briefing store changed since it was read")
	ErrHistoryTooDeep  = errors.New("briefing history closure exceeds the bounded commit scan")
	ErrDismissal       = errors.New("dismissal requires an event and manager")
	ErrNoAcceptedModel = errors.New("no valid committed project model exists at the selected revision")
	ErrAlreadyResolved = errors.New("model-change event already has different resolution evidence")
)

const (
	acceptedPolicy            = projectbriefing.AcceptedPolicy
	maxAcceptedHistoryCommits = 4096
	// maxEnsureAttempts bounds how often EnsureAcceptedHistory recomputes after
	// another writer changed the store between its read and its write.
	maxEnsureAttempts = 8
	// storeLockWait bounds how long a writer waits for another writer's lock.
	storeLockWait = 10 * time.Second
)

// Read returns the accepted store value, raw persisted CAS digest, and
// process-only status view derived from actual working-tree observations.
type ReadResult struct {
	Accepted        projectbriefing.Store
	PersistedDigest string
	StatusView      projectbriefing.ResolutionView
}

func Read(root string) (ReadResult, error) {
	state, digest, err := readStore(root)
	if err != nil {
		return ReadResult{}, err
	}
	if state.History == nil && len(state.Briefings) == 0 {
		return ReadResult{Accepted: state, PersistedDigest: digest, StatusView: projectbriefing.NewResolutionView(state, nil)}, nil
	}
	active, err := activeLine(root)
	if err != nil {
		return ReadResult{}, err
	}
	accepted := active.accepted(state)
	hintsByEvent := active.uncommitted(state, accepted)
	hints := make([]projectbriefing.Resolution, 0, len(hintsByEvent))
	for _, resolution := range hintsByEvent {
		hints = append(hints, resolution)
	}
	return ReadResult{Accepted: accepted, PersistedDigest: digest, StatusView: projectbriefing.NewResolutionView(accepted, hints)}, nil
}

// readStore validates and returns the persisted store, provisional entries
// included, and its content digest.
func historyName() recordstore.Name {
	return recordstore.Name{Kind: recordstore.Briefing, ID: "accepted", Role: "record.json"}
}
func readStore(root string) (projectbriefing.Store, string, error) {
	store, err := recordstore.Open(root)
	if err != nil {
		return projectbriefing.Store{}, "", err
	}
	defer store.Close()
	var file recordstore.File
	err = waitWhileBusy(func() (bool, error) {
		var e error
		file, e = store.Read(historyName(), 64<<20)
		return windowsBusy(e), e
	})
	if err != nil {
		return projectbriefing.Store{}, "", err
	}
	if !file.Exists {
		state := emptyStore()
		return state, projectbriefing.StoreDigest(state), nil
	}
	return decodeHistoryStore(file.Bytes)
}
func decodeHistoryStore(data []byte) (projectbriefing.Store, string, error) {
	var state projectbriefing.Store
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return projectbriefing.Store{}, "", fmt.Errorf("decode briefing store: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return projectbriefing.Store{}, "", errors.New("briefing store contains trailing data")
	}
	if state.APIVersion != projectbriefing.APIVersion {
		return projectbriefing.Store{}, "", fmt.Errorf("briefing store %s has format %q, not %q; delete it and run a guided command to rebuild it: accepted history is re-briefed from the first committed model, and the stored dismissals and resolutions are lost", storePath, state.APIVersion, projectbriefing.APIVersion)
	}
	if state.Briefings == nil || state.Dismissals == nil {
		return projectbriefing.Store{}, "", errors.New("briefing store has invalid shape")
	}
	if err := projectbriefing.ValidateStore(state); err != nil {
		return projectbriefing.Store{}, "", fmt.Errorf("validate briefing store: %w", err)
	}
	return state, projectbriefing.StoreDigest(state), nil
}

// NewDeliveredFile records the exact Git blob corresponding to bytes that
// Apply wrote, or records a delivered removal without reading the filesystem.
func NewDeliveredFile(root, filePath, mode string, content []byte, deleted bool) (projectbriefing.DeliveredFile, error) {
	if deleted {
		return projectbriefing.DeliveredFile{Path: filePath, Deleted: true}, nil
	}
	object, err := source.GitOutputInput(root, content, "hash-object", "--no-filters", "--stdin")
	if err != nil {
		return projectbriefing.DeliveredFile{}, fmt.Errorf("hash delivered %s: %w", filePath, err)
	}
	return projectbriefing.DeliveredFile{Path: filePath, Mode: mode, Object: strings.TrimSpace(string(object))}, nil
}

// ResolveVerified records fresh full-Manager Verify and successful Apply
// evidence against the exact selected committed model. The caller is expected
// to invoke this only from the Host's successful after-Apply path, with
// evidence.Delivered listing what Apply wrote. The resolution counts once
// HEAD's committed tree holds that delivered result.
func ResolveVerified(root, modelRevision, modelDigest string, evidence projectbriefing.VerifiedResolutionEvidence, expectedStoreDigest string) (string, error) {
	if evidence.Delivered == nil {
		return "", projectbriefing.ErrResolution
	}
	evidence.Delivered = append([]projectbriefing.DeliveredFile{}, evidence.Delivered...)
	sort.Slice(evidence.Delivered, func(i, j int) bool { return evidence.Delivered[i].Path < evidence.Delivered[j].Path })
	if strings.TrimSpace(modelRevision) == "" || strings.TrimSpace(modelDigest) == "" || !projectbriefing.ValidVerifiedEvidence(evidence) {
		return "", projectbriefing.ErrResolution
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
		return "", projectbriefing.ErrStaleModel
	}
	project, err := projectwork.Load(root, modelRevision)
	if err != nil || project == nil || project.Provisional || project.Model.Digest != modelDigest {
		return "", projectbriefing.ErrStaleModel
	}
	cursorProjects, err := acceptedProjects(root, state.History.Revision)
	if err != nil {
		return "", err
	}
	if err := validateAcceptedPrefix(state, cursorProjects); err != nil {
		return "", err
	}
	if indexOf(projectRevisions(cursorProjects), modelRevision) < 0 {
		return "", projectbriefing.ErrAmbiguousHistory
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", state.History.BaselineRevision, modelRevision); err != nil {
		return "", projectbriefing.ErrAmbiguousHistory
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", modelRevision, state.History.Revision); err != nil {
		return "", projectbriefing.ErrStaleModel
	}
	managerIDs := make([]string, 0, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		managerIDs = append(managerIDs, manager.ID)
	}
	sort.Strings(managerIDs)
	if !equalStrings(managerIDs, evidence.CoveredManagerIDs) {
		return "", fmt.Errorf("full verification did not cover every selected Manager: %w", projectbriefing.ErrResolution)
	}
	events := make(map[string]projectbriefing.Event, len(evidence.EventIDs))
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
		return "", projectbriefing.ErrStaleModel
	}
	for _, eventID := range evidence.EventIDs {
		for _, managerID := range events[eventID].AffectedManagers {
			if !contains(evidence.CoveredManagerIDs, managerID) {
				return "", fmt.Errorf("full verification omitted affected Manager %s: %w", managerID, projectbriefing.ErrResolution)
			}
		}
	}
	resolutions := make([]projectbriefing.Resolution, 0, len(evidence.EventIDs))
	for _, eventID := range evidence.EventIDs {
		resolution := projectbriefing.Resolution{EventID: eventID, ModelRevision: modelRevision, ModelDigest: modelDigest, Evidence: evidence}
		resolution.Digest = projectbriefing.ResolutionDigest(resolution)
		resolutions = append(resolutions, resolution)
	}
	return update(root, expectedStoreDigest, func(current *projectbriefing.Store) error {
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

func projectRevisions(projects []*projectvalue.Project) []string {
	revisions := make([]string, 0, len(projects))
	for _, project := range projects {
		revisions = append(revisions, project.Revision)
	}
	return revisions
}

// ReadAcceptedHistory computes in memory the accepted history that
// EnsureAcceptedHistory would persist for targetRevision. It reads the store
// once, takes no writer lock and creates no files. A missing store or a
// baseline off the checked-out line starts at the line's first committed
// model, a cursor behind the target is extended and one already past it is
// kept, exactly as EnsureAcceptedHistory would do. It never fails because the
// persisted store is behind or another writer is active; it fails closed on
// ambiguous or invalid history with the same errors as EnsureAcceptedHistory.
func ReadAcceptedHistory(root, targetRevision string) (projectbriefing.AcceptedHistory, error) {
	if !projectbriefing.ValidObjectID(targetRevision) {
		return projectbriefing.AcceptedHistory{}, projectbriefing.ErrUncommittedModel
	}
	active, err := activeLine(root)
	if err != nil {
		return projectbriefing.AcceptedHistory{}, err
	}
	stored, digest, err := readStore(root)
	if err != nil {
		return projectbriefing.AcceptedHistory{}, err
	}
	next, receipt, err := acceptHistory(root, active, targetRevision, stored)
	if err != nil {
		return projectbriefing.AcceptedHistory{}, err
	}
	receipt.StoreDigest = projectbriefing.StoreDigest(next)
	return projectbriefing.AcceptedHistory{Receipt: receipt, State: active.accepted(next), PersistedDigest: digest}, nil
}

// EnsureAcceptedHistory reconciles the bounded first-parent history ending at
// targetRevision, which must lie on the first-parent line of HEAD, and
// persists it in one write. That line is the accepted history: its first valid
// committed project model is the baseline, and each subsequent canonical
// model-digest change gets one immutable briefing recorded at the line's own
// commit. Briefings recorded off the line stay provisional and neither count
// nor block, so a topic that was merged, squashed or rebased is briefed again
// where its change entered the line, with the same content-based event
// identities; only a fast-forward accepts the topic's own briefing. A briefing
// on the line that contradicts the line's transition is ambiguous and fails
// the call. When another writer changes the store first, the history is
// recomputed from what that writer persisted, so a cursor it already moved to
// or past the target is success. Working-tree state is never inspected or
// accepted. Git commit identity is retained only as unauthenticated source
// metadata.
func EnsureAcceptedHistory(root, targetRevision string) (projectbriefing.EnsureReceipt, error) {
	if !projectbriefing.ValidObjectID(targetRevision) {
		return projectbriefing.EnsureReceipt{}, projectbriefing.ErrUncommittedModel
	}
	active, err := activeLine(root)
	if err != nil {
		return projectbriefing.EnsureReceipt{}, err
	}
	for attempt := 1; ; attempt++ {
		stored, digest, err := readStore(root)
		if err != nil {
			return projectbriefing.EnsureReceipt{}, err
		}
		next, receipt, err := acceptHistory(root, active, targetRevision, stored)
		if err != nil {
			return projectbriefing.EnsureReceipt{}, err
		}
		if projectbriefing.StoreDigest(next) == digest {
			receipt.StoreDigest = digest
			return receipt, nil
		}
		written, err := update(root, digest, func(current *projectbriefing.Store) error {
			*current = next
			return nil
		})
		if errors.Is(err, ErrStaleStore) && attempt < maxEnsureAttempts {
			continue
		}
		if err != nil {
			return projectbriefing.EnsureReceipt{}, err
		}
		receipt.StoreDigest = written
		return receipt, nil
	}
}

// acceptHistory is the write-free reconciliation shared by ReadAcceptedHistory
// and EnsureAcceptedHistory. From the persisted store it returns the store
// that accepts targetRevision on the active line, provisional entries kept,
// and a receipt whose Bundles are the entries it added; the receipt's
// StoreDigest field is left to the caller.
func acceptHistory(root string, active *line, targetRevision string, state projectbriefing.Store) (projectbriefing.Store, projectbriefing.EnsureReceipt, error) {
	var receipt projectbriefing.EnsureReceipt
	targetIndex := active.index(targetRevision)
	if targetIndex < 0 {
		return projectbriefing.Store{}, receipt, fmt.Errorf("accepted history target must be on the active first-parent branch: %w", projectbriefing.ErrUncommittedModel)
	}
	projects, err := acceptedProjects(root, targetRevision)
	if err != nil {
		return projectbriefing.Store{}, receipt, err
	}
	if len(projects) == 0 || projects[len(projects)-1].Revision != targetRevision {
		return projectbriefing.Store{}, receipt, ErrNoAcceptedModel
	}
	if state.History == nil && len(state.Briefings) > 0 {
		return projectbriefing.Store{}, receipt, fmt.Errorf("existing briefing entries have no accepted-history cursor and cannot be safely rebased: %w", projectbriefing.ErrAmbiguousHistory)
	}
	// Work on a copy: the cursor is shared by pointer and briefings are re-sorted.
	if state.History != nil {
		cursor := *state.History
		state.History = &cursor
	}
	state.Briefings = append([]projectbriefing.Bundle{}, state.Briefings...)
	// Without a stored baseline on this line (no history yet, or a project
	// model that entered the line through a squash or rebase), the line starts
	// at its own first committed model.
	if active.cursor(state) == nil {
		first := projects[0]
		state.History = &projectbriefing.HistoryCursor{Policy: acceptedPolicy, BaselineRevision: first.Revision, BaselineModelDigest: first.Model.Digest, Revision: first.Revision, ModelDigest: first.Model.Digest}
	}
	view := active.acceptedHistory(state)
	accepted := projects
	if active.index(view.History.Revision) > targetIndex {
		if accepted, err = acceptedProjects(root, view.History.Revision); err != nil {
			return projectbriefing.Store{}, receipt, err
		}
	}
	if err := validateAcceptedPrefix(view, accepted); err != nil {
		return projectbriefing.Store{}, receipt, err
	}
	receipt.BaselineRevision = view.History.BaselineRevision
	receipt.BaselineModelDigest = view.History.BaselineModelDigest
	receipt.Revision = targetRevision
	receipt.ModelDigest = projects[len(projects)-1].Model.Digest
	start := indexOf(projectRevisions(projects), view.History.Revision)
	if start < 0 {
		// The line is already accepted beyond the target.
		return state, receipt, nil
	}
	for i := start + 1; i < len(projects); i++ {
		previous, current := projects[i-1], projects[i]
		if current.Model.Digest == previous.Model.Digest {
			if err := advanceCursor(root, active, &state, previous, current); err != nil {
				return projectbriefing.Store{}, receipt, err
			}
			continue
		}
		provenance, err := gitProvenance(root, current.Revision)
		if err != nil {
			return projectbriefing.Store{}, receipt, err
		}
		// Chain to the events this computation already added, not only the
		// persisted ones.
		bundle, err := generate(root, previous.Revision, current.Revision, provenance, &state)
		if err != nil {
			return projectbriefing.Store{}, receipt, fmt.Errorf("generate accepted model change at %s: %w", current.Revision, err)
		}
		if err := appendBundle(root, active, &state, bundle); err != nil {
			return projectbriefing.Store{}, receipt, err
		}
		receipt.Bundles = append(receipt.Bundles, bundle)
	}
	if cursor := active.cursor(state); cursor == nil || cursor.Revision != targetRevision || cursor.ModelDigest != receipt.ModelDigest {
		return projectbriefing.Store{}, receipt, projectbriefing.ErrStaleModel
	}
	if err := projectbriefing.ValidateStore(state); err != nil {
		return projectbriefing.Store{}, receipt, fmt.Errorf("validate accepted history: %w", err)
	}
	return state, receipt, nil
}

func acceptedProjects(root, targetRevision string) ([]*projectvalue.Project, error) {
	shallow, err := source.GitOutput(root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, fmt.Errorf("inspect repository history completeness: %w", err)
	}
	if strings.TrimSpace(string(shallow)) != "false" {
		return nil, fmt.Errorf("accepted history requires complete Git ancestry: %w", projectbriefing.ErrAmbiguousHistory)
	}
	revisions, err := firstParentRevisions(root, targetRevision)
	if err != nil {
		return nil, fmt.Errorf("enumerate accepted first-parent history: %w", err)
	}
	projects := make([]*projectvalue.Project, 0, len(revisions))
	for _, revision := range revisions {
		paths, err := source.GitOutput(root, "ls-tree", "-r", "--name-only", revision, "--", projectvalue.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("inspect project manifest at %s: %w", revision, err)
		}
		if strings.TrimSpace(string(paths)) == "" {
			if len(projects) == 0 {
				continue
			}
			return nil, fmt.Errorf("project manifest disappeared at %s: %w", revision, projectbriefing.ErrAmbiguousHistory)
		}
		project, err := projectwork.Load(root, revision)
		if err != nil {
			return nil, fmt.Errorf("load committed project model at %s: %w", revision, err)
		}
		if project == nil || project.Provisional || project.Model.Digest == "" {
			return nil, fmt.Errorf("invalid committed model at %s: %w", revision, projectbriefing.ErrAmbiguousHistory)
		}
		for _, finding := range project.Report.Findings {
			// `incomplete` findings describe absent implementation or inventory
			// evidence and do not invalidate the declared model. Error findings
			// describe declarations that cannot be accepted as a project
			// specification (for example unresolved references or invalid
			// ownership), even when Core compilation produced a model digest.
			if finding.Severity == "error" {
				return nil, fmt.Errorf("committed project model at %s has structural finding %s (%s): %w", revision, finding.Code, finding.Subject, projectbriefing.ErrAmbiguousHistory)
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

func validateAcceptedPrefix(state projectbriefing.Store, projects []*projectvalue.Project) error {
	if state.History == nil {
		return nil
	}
	if len(projects) == 0 || projects[0].Revision != state.History.BaselineRevision || projects[0].Model.Digest != state.History.BaselineModelDigest {
		return projectbriefing.ErrAmbiguousHistory
	}
	cursorIndex := -1
	for i, project := range projects {
		if project.Revision == state.History.Revision {
			cursorIndex = i
			break
		}
	}
	if cursorIndex < 0 || projects[cursorIndex].Model.Digest != state.History.ModelDigest {
		return projectbriefing.ErrAmbiguousHistory
	}
	bundles := make(map[string]projectbriefing.Bundle, len(state.Briefings))
	for _, bundle := range state.Briefings {
		bundles[bundle.Revision] = bundle
	}
	// last tracks, per definition, the newest event on the line; each event
	// must continue it.
	last := map[string]string{}
	for i := 1; i <= cursorIndex; i++ {
		previous, current := projects[i-1], projects[i]
		bundle, exists := bundles[current.Revision]
		if current.Model.Digest != previous.Model.Digest {
			if !exists {
				return fmt.Errorf("accepted model transition %s..%s is not completely briefed: %w", previous.Revision, current.Revision, projectbriefing.ErrStaleModel)
			}
			if bundle.SinceRevision != previous.Revision || bundle.SinceModelDigest != previous.Model.Digest || bundle.ModelDigest != current.Model.Digest {
				return fmt.Errorf("briefing at %s contradicts the accepted model transition %s..%s: %w", current.Revision, previous.Revision, current.Revision, projectbriefing.ErrAmbiguousHistory)
			}
			for _, event := range bundle.Events {
				key := event.DefinitionID.Key()
				if event.Predecessor != last[key] {
					return fmt.Errorf("event %s at %s does not continue its definition's accepted events: %w", event.ID, current.Revision, projectbriefing.ErrAmbiguousHistory)
				}
				last[key] = event.ID
			}
		} else if exists {
			return fmt.Errorf("no-op model revision %s has a briefing bundle: %w", current.Revision, projectbriefing.ErrAmbiguousHistory)
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
			return projectbriefing.ErrAmbiguousHistory
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
		return nil, fmt.Errorf("resolve active accepted branch: %w", projectbriefing.ErrUncommittedModel)
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

// rank orders a resolution along the line for latest-wins: where its commit
// lies on the line or was merged in, else, for a delivery that reached the
// line through a squash or rebase, where its delivered result entered it.
func (l *line) rank(resolution projectbriefing.Resolution) int {
	if i := l.position(resolution.ModelRevision); i >= 0 {
		return i
	}
	return l.entered(resolution.Evidence.Delivered)
}

// entered returns the line position of the newest first-parent commit that
// wrote one of the delivered blobs at its path or removed one of the removed
// paths, or -1 when none did or the history cannot be read.
func (l *line) entered(files []projectbriefing.DeliveredFile) int {
	at := -1
	for _, file := range files {
		filter := "--find-object=" + file.Object
		if file.Deleted {
			filter = "--diff-filter=D"
		}
		output, err := source.GitOutput(l.root, "--literal-pathspecs", "log", "--first-parent", "-1", "--format=%H", filter, l.head, "--", file.Path)
		if err != nil {
			return -1
		}
		if i := l.index(strings.TrimSpace(string(output))); i > at {
			at = i
		}
	}
	return at
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
func (l *line) inHead(files []projectbriefing.DeliveredFile) (present, known bool) {
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
func (l *line) present(files []projectbriefing.DeliveredFile) bool {
	present, known := l.inHead(files)
	return present && known
}

// uncommitted returns, per accepted event in view without an accepted
// resolution, the latest resolution whose delivered result the working tree
// holds although HEAD's tree does not. It reads the working tree only to tell
// users why an event is not resolved yet; it never makes anything accepted.
func (l *line) uncommitted(state, view projectbriefing.Store) map[string]projectbriefing.Resolution {
	events, resolved := map[string]bool{}, map[string]bool{}
	for _, bundle := range view.Briefings {
		for _, event := range bundle.Events {
			events[event.ID] = true
		}
	}
	for _, resolution := range view.Resolutions {
		resolved[resolution.EventID] = true
	}
	result := map[string]projectbriefing.Resolution{}
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
func (l *line) inWorkingTree(files []projectbriefing.DeliveredFile) bool {
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
func (l *line) cursor(state projectbriefing.Store) *projectbriefing.HistoryCursor {
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
func (l *line) acceptedHistory(state projectbriefing.Store) projectbriefing.Store {
	view := projectbriefing.Store{APIVersion: state.APIVersion, History: l.cursor(state), Briefings: []projectbriefing.Bundle{}, Dismissals: []projectbriefing.Dismissal{}, Resolutions: []projectbriefing.Resolution{}}
	for _, bundle := range state.Briefings {
		if l.index(bundle.Revision) >= 0 {
			view.Briefings = append(view.Briefings, bundle)
		}
	}
	return view
}

// accepted returns acceptedHistory with the dismissals and resolutions of its
// events that are accepted on the line, keeping only the latest resolution of
// each event by rank, and of equals the one recorded last.
func (l *line) accepted(state projectbriefing.Store) projectbriefing.Store {
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
		} else if l.rank(resolution) >= l.rank(view.Resolutions[i]) {
			view.Resolutions[i] = resolution
		}
	}
	return view
}

// advanceCursor moves the line's cursor over one first-parent commit that
// leaves the accepted model digest unchanged.
func advanceCursor(root string, active *line, state *projectbriefing.Store, previous, current *projectvalue.Project) error {
	if previous == nil || current == nil || previous.Model.Digest != current.Model.Digest {
		return projectbriefing.ErrStaleModel
	}
	cursor := active.cursor(*state)
	if cursor == nil || cursor.Revision != previous.Revision || cursor.ModelDigest != previous.Model.Digest {
		return projectbriefing.ErrStaleModel
	}
	if err := requireNextFirstParent(root, previous.Revision, current.Revision); err != nil {
		return err
	}
	cursor.Revision, cursor.ModelDigest = current.Revision, current.Model.Digest
	state.History = cursor
	return nil
}

func requireNextFirstParent(root, sinceRevision, revision string) error {
	output, err := source.GitOutput(root, "rev-list", "--first-parent", "--parents", "-n", "1", revision)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || fields[0] != revision || fields[1] != sinceRevision {
		return projectbriefing.ErrAmbiguousHistory
	}
	return nil
}

func gitProvenance(root, revision string) (projectbriefing.Provenance, error) {
	author, err := source.GitOutput(root, "show", "-s", "--format=%an", revision)
	if err != nil {
		return projectbriefing.Provenance{}, fmt.Errorf("read accepted model commit metadata: %w", err)
	}
	name := strings.TrimSpace(string(author))
	if name == "" {
		return projectbriefing.Provenance{}, projectbriefing.ErrMissingProvenance
	}
	return projectbriefing.Provenance{
		DecisionReference: "git-commit:" + revision,
		Actor:             "git-commit-author:" + name,
		Authority:         acceptedPolicy,
	}, nil
}

// Write appends an immutable briefing bundle. expectedDigest must be the token
// returned by Read; pass the empty-store digest for a not-yet-created store.
func Write(root string, bundle projectbriefing.Bundle, expectedDigest string) (string, error) {
	if err := projectbriefing.ValidateBundle(bundle); err != nil {
		return "", err
	}
	if bundle.SinceRevision == bundle.Revision {
		return "", projectbriefing.ErrInvalidBundle
	}
	regenerated, err := Generate(root, bundle.SinceRevision, bundle.Revision, bundle.Provenance)
	if err != nil {
		return "", fmt.Errorf("regenerate briefing from fixed source revisions: %w", err)
	}
	if hash(regenerated) != hash(bundle) {
		return "", fmt.Errorf("%w: submitted bundle differs from fixed-source regeneration", projectbriefing.ErrInvalidBundle)
	}
	if bundle.SinceModelDigest == bundle.ModelDigest && len(bundle.Events) != 0 {
		return "", fmt.Errorf("%w: unchanged model digest cannot carry events", projectbriefing.ErrInvalidBundle)
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
			return "", projectbriefing.ErrAmbiguousHistory
		}
		if state.History != nil {
			if cursor := active.cursor(state); cursor == nil || cursor.Revision != bundle.SinceRevision || cursor.ModelDigest != bundle.SinceModelDigest {
				return "", projectbriefing.ErrAmbiguousHistory
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
		return "", projectbriefing.ErrAmbiguousHistory
	}
	if state.History == nil {
		projects, err := acceptedProjects(root, bundle.Revision)
		if err != nil {
			return "", err
		}
		if len(projects) < 2 || projects[0].Revision != bundle.SinceRevision {
			return "", fmt.Errorf("first persisted briefing must begin at the earliest committed project model: %w", projectbriefing.ErrAmbiguousHistory)
		}
		currentDigest, err = update(root, currentDigest, func(current *projectbriefing.Store) error {
			if current.History != nil || len(current.Briefings) != 0 {
				return ErrStaleStore
			}
			baseline := projects[0]
			current.History = &projectbriefing.HistoryCursor{Policy: acceptedPolicy, BaselineRevision: baseline.Revision, BaselineModelDigest: baseline.Model.Digest, Revision: baseline.Revision, ModelDigest: baseline.Model.Digest}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return appendCanonicalBundle(root, active, bundle, currentDigest)
}

func bundleForRevision(state projectbriefing.Store, revision string) (projectbriefing.Bundle, bool) {
	for _, bundle := range state.Briefings {
		if bundle.Revision == revision {
			return bundle, true
		}
	}
	return projectbriefing.Bundle{}, false
}

func appendCanonicalBundle(root string, active *line, bundle projectbriefing.Bundle, expectedDigest string) (string, error) {
	return update(root, expectedDigest, func(state *projectbriefing.Store) error {
		return appendBundle(root, active, state, bundle)
	})
}

// appendBundle adds a briefing that extends the line's cursor by one
// first-parent commit and moves the cursor onto it.
func appendBundle(root string, active *line, state *projectbriefing.Store, bundle projectbriefing.Bundle) error {
	cursor := active.cursor(*state)
	if cursor == nil {
		return projectbriefing.ErrStaleModel
	}
	if cursor.Revision != bundle.SinceRevision || cursor.ModelDigest != bundle.SinceModelDigest {
		return fmt.Errorf("briefing must extend the accepted cursor: %w", projectbriefing.ErrStaleModel)
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
		return projectbriefing.ErrStaleModel
	}
	for _, prior := range state.Briefings {
		if prior.Revision == bundle.Revision {
			if hash(prior) == hash(bundle) {
				return nil
			}
			return projectbriefing.ErrAmbiguousHistory
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
	return update(root, expectedDigest, func(state *projectbriefing.Store) error {
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
		state.Dismissals = append(state.Dismissals, projectbriefing.Dismissal{EventID: eventID, ManagerID: managerID})
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
func LoadForManager(root, modelDigest, managerID string, requestedRevision ...string) ([]projectbriefing.Briefing, []projectbriefing.Event, string, error) {
	if strings.TrimSpace(modelDigest) == "" || strings.TrimSpace(managerID) == "" || len(requestedRevision) > 1 {
		return nil, nil, "", projectbriefing.ErrStaleModel
	}
	read, err := Read(root)
	if err != nil {
		return nil, nil, "", err
	}
	return LoadForManagerFrom(root, read.Accepted, modelDigest, managerID, requestedRevision...)
}

// LoadForManagerFrom is LoadForManager over an accepted view that was already
// read: the state from Read or the State of ReadAcceptedHistory.
func LoadForManagerFrom(root string, state projectbriefing.Store, modelDigest, managerID string, requestedRevision ...string) ([]projectbriefing.Briefing, []projectbriefing.Event, string, error) {
	if strings.TrimSpace(modelDigest) == "" || strings.TrimSpace(managerID) == "" || len(requestedRevision) > 1 {
		return nil, nil, "", projectbriefing.ErrStaleModel
	}
	var chain []projectbriefing.Bundle
	var project *projectvalue.Project
	var err error
	if len(requestedRevision) == 1 {
		project, err = projectwork.Load(root, requestedRevision[0])
		if err != nil {
			return nil, nil, "", fmt.Errorf("load manager briefing basis: %w", err)
		}
		if project.Provisional || project.Model.Digest != modelDigest {
			return nil, nil, "", projectbriefing.ErrStaleModel
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
					return nil, nil, "", projectbriefing.ErrAmbiguousHistory
				}
				chain = append(chain, bundle)
			}
		}
		if len(chain) == 0 {
			if state.History == nil || state.History.BaselineModelDigest != modelDigest {
				return nil, nil, "", projectbriefing.ErrStaleModel
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", state.History.BaselineRevision, project.Revision); err != nil {
				return nil, nil, "", projectbriefing.ErrAmbiguousHistory
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", project.Revision, state.History.Revision); err != nil {
				return nil, nil, "", projectbriefing.ErrStaleModel
			}
			return []projectbriefing.Briefing{}, []projectbriefing.Event{}, hash(struct {
				ModelDigest, ManagerID, Revision string
			}{modelDigest, managerID, project.Revision}), nil
		}
		sort.Slice(chain, func(i, j int) bool { return rank[chain[i].Revision] < rank[chain[j].Revision] })
		if chain[len(chain)-1].ModelDigest != modelDigest {
			return nil, nil, "", projectbriefing.ErrStaleModel
		}
		for i := 1; i < len(chain); i++ {
			if chain[i].SinceModelDigest != chain[i-1].ModelDigest {
				return nil, nil, "", projectbriefing.ErrAmbiguousHistory
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", chain[i-1].Revision, chain[i].Revision); err != nil {
				return nil, nil, "", projectbriefing.ErrAmbiguousHistory
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
		project, err = projectwork.Load(root, chain[len(chain)-1].Revision)
		if err != nil {
			return nil, nil, "", fmt.Errorf("load manager briefing visibility: %w", err)
		}
	}
	currentView, err := projectbriefing.NewManagerView(project.Report, managerID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("load manager briefing visibility: %w", err)
	}
	views := map[string]projectbriefing.ManagerView{}
	viewAt := func(revision string) (projectbriefing.ManagerView, error) {
		if view, ok := views[revision]; ok {
			return view, nil
		}
		view, err := loadManagerView(root, revision, managerID)
		if errors.Is(err, projectmodel.ErrManagerNotFound) {
			view = emptyManagerView()
			err = nil
		}
		if err != nil {
			return projectbriefing.ManagerView{}, err
		}
		views[revision] = view
		return view, nil
	}
	views[project.Revision] = currentView
	briefings := make([]projectbriefing.Briefing, 0)
	events := make([]projectbriefing.Event, 0)
	seenEvents := map[string]bool{}
	for _, bundle := range chain {
		for _, briefing := range bundle.Managers {
			if briefing.ManagerID != managerID {
				continue
			}
			briefings = append(briefings, briefing)
			beforeView, err := viewAt(bundle.SinceRevision)
			if err != nil {
				return nil, nil, "", err
			}
			afterView, err := viewAt(bundle.Revision)
			if err != nil {
				return nil, nil, "", err
			}
			for _, event := range bundle.Events {
				if contains(briefing.EventIDs, event.ID) && !seenEvents[event.ID] {
					projected, visible := projectbriefing.ProjectEvent(event, beforeView, afterView)
					if !visible {
						continue
					}
					events = append(events, projected)
					seenEvents[event.ID] = true
				}
			}
		}
	}
	return briefings, events, hash(struct {
		ModelDigest, ManagerID string
		Briefings              []projectbriefing.Briefing
		Events                 []projectbriefing.Event
	}{modelDigest, managerID, briefings, events}), nil
}

func verifyHistoryClosure(root string, latest projectbriefing.Bundle, target *projectvalue.Project) error {
	if target == nil || target.Provisional || target.Model.Digest != latest.ModelDigest {
		return projectbriefing.ErrStaleModel
	}
	base, err := projectwork.Load(root, latest.Revision)
	if err != nil {
		return fmt.Errorf("load last briefed model revision: %w", err)
	}
	if base.Model.Digest != latest.ModelDigest {
		return projectbriefing.ErrStaleModel
	}
	paths := map[string]bool{projectvalue.ManifestPath: true}
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
				return fmt.Errorf("%w: accepted model digest changed at unbriefed revision %s", projectbriefing.ErrStaleModel, commit)
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

func legacyDigestChain(bundles []projectbriefing.Bundle, modelDigest string) ([]projectbriefing.Bundle, error) {
	chain := make([]projectbriefing.Bundle, 0)
	cursor := modelDigest
	seen := map[string]bool{}
	for cursor != "" {
		if seen[cursor] {
			return nil, projectbriefing.ErrAmbiguousHistory
		}
		seen[cursor] = true
		matches := make([]projectbriefing.Bundle, 0)
		for _, bundle := range bundles {
			if bundle.ModelDigest == cursor {
				matches = append(matches, bundle)
			}
		}
		if len(matches) == 0 {
			break
		}
		if len(matches) > 1 {
			return nil, projectbriefing.ErrAmbiguousHistory
		}
		parents := map[string]bool{}
		for _, bundle := range matches {
			parents[bundle.SinceModelDigest] = true
		}
		if len(parents) != 1 {
			return nil, projectbriefing.ErrAmbiguousHistory
		}
		chain = append(chain, matches...)
		for parent := range parents {
			cursor = parent
		}
	}
	if len(chain) == 0 {
		return nil, projectbriefing.ErrStaleModel
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

func emptyStore() projectbriefing.Store {
	return projectbriefing.Store{APIVersion: projectbriefing.APIVersion, Briefings: []projectbriefing.Bundle{}, Dismissals: []projectbriefing.Dismissal{}, Resolutions: []projectbriefing.Resolution{}}
}

func update(root, expected string, mutate func(*projectbriefing.Store) error) (returnedDigest string, resultErr error) {
	store, err := recordstore.Open(root)
	if err != nil {
		return "", err
	}
	defer store.Close()
	lease, err := store.Exclusive(recordstore.Briefing, "accepted", ".briefings.lock", nil, storeLockWait, nil)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("acquire briefing store lock: %s is still held after %s; if no Markitect process is running, delete that file: %w", filepath.Join(root, filepath.Dir(filepath.FromSlash(storePath)), ".briefings.lock"), storeLockWait, err)
		}
		return "", err
	}
	defer func() {
		if err := lease.Close(); err != nil {
			resultErr = errors.Join(resultErr, &recordstore.PublicationError{Path: filepath.ToSlash(filepath.Join(filepath.Dir(storePath), ".briefings.lock")), Outcome: recordstore.Published, Cause: err})
		}
	}()
	prior, err := store.Read(historyName(), 64<<20)
	if err != nil {
		return "", err
	}
	state := emptyStore()
	current := projectbriefing.StoreDigest(state)
	if prior.Exists {
		state, current, err = decodeHistoryStore(prior.Bytes)
		if err != nil {
			return "", err
		}
	}
	if current != expected {
		return "", ErrStaleStore
	}
	if err = mutate(&state); err != nil {
		return "", err
	}
	next := projectbriefing.StoreDigest(state)
	if next == current {
		return current, nil
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err = store.Replace(historyName(), prior, data, lease.Check); err != nil {
		return "", err
	}
	if err := observePublishedHistory(store, data, next); err != nil {
		return "", err
	}
	return next, nil
}

func observePublishedHistory(store *recordstore.Store, data []byte, next string) error {
	observed, err := store.Read(historyName(), 64<<20)
	if err != nil {
		return &recordstore.PublicationError{Path: storePath, Outcome: recordstore.Published, Cause: err}
	}
	if !observed.Exists || !bytes.Equal(observed.Bytes, data) {
		return &recordstore.PublicationError{Path: storePath, Outcome: recordstore.Published, Cause: errors.New("published history bytes differ")}
	}
	_, digest, err := decodeHistoryStore(observed.Bytes)
	if err != nil || digest != next {
		if err == nil {
			err = errors.New("published history domain digest differs")
		}
		return &recordstore.PublicationError{Path: storePath, Outcome: recordstore.Published, Cause: err}
	}
	return nil
}

// waitWhileBusy retries attempt with backoff for at most storeLockWait while it
// reports the store as busy, and returns its last error.
func waitWhileBusy(attempt func() (busy bool, err error)) error {
	deadline := time.Now().Add(storeLockWait)
	delay := time.Millisecond
	for {
		busy, err := attempt()
		if !busy || time.Now().After(deadline) {
			return err
		}
		time.Sleep(delay)
		if delay < 50*time.Millisecond {
			delay *= 2
		}
	}
}

// Windows reports these transiently while another process opens, replaces or
// deletes the same store or lock file.
const (
	windowsAccessDenied     = syscall.Errno(5)
	windowsSharingViolation = syscall.Errno(32)
)

// windowsBusy reports a transient Windows file-sharing conflict with another
// reader or writer of the store.
func windowsBusy(err error) bool {
	var errno syscall.Errno
	return runtime.GOOS == "windows" && errors.As(err, &errno) && (errno == windowsAccessDenied || errno == windowsSharingViolation)
}

func stateFile(root string, create bool) (string, error) {
	store, err := recordstore.Open(root)
	if err != nil {
		return "", err
	}
	defer store.Close()
	if err = store.CheckCutover(recordstore.Briefing); err != nil {
		return "", err
	}
	if create {
		if err = store.Ensure(recordstore.Briefing, "accepted", ""); err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, filepath.FromSlash(storePath)), nil
}

// DigestBytes exposes a stable digest for CLI optimistic-concurrency prompts.
func DigestBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
