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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const storePath = ".markitect/state/briefings/history.json"

var (
	ErrStaleStore       = errors.New("briefing store changed since it was read")
	ErrStaleModel       = errors.New("no briefing history is bound to the requested model digest")
	ErrAmbiguousHistory = errors.New("briefing history has multiple accepted paths to the requested model digest")
	ErrHistoryTooDeep   = errors.New("briefing history closure exceeds the bounded commit scan")
	ErrDismissal        = errors.New("dismissal requires an event and manager")
)

func validateStore(state Store) error {
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
			if prior, exists := eventsByID[event.ID]; exists {
				if hash(prior) != hash(event) {
					return fmt.Errorf("event %s has conflicting definitions", event.ID)
				}
				return fmt.Errorf("duplicate model event %s", event.ID)
			}
			eventsByID[event.ID] = event
			managerByEvent[event.ID] = make(map[string]bool)
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
	if event.DefinitionID.APIVersion == "" || event.DefinitionID.Kind == "" || event.DefinitionID.Namespace == "" || event.DefinitionID.Name == "" {
		return fmt.Errorf("%w: incomplete definition identity", ErrInvalidBundle)
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
	payload := struct {
		Since, Revision, Key, Change string
		Before, After                *core.Definition
	}{bundle.SinceRevision, bundle.Revision, event.DefinitionID.Key(), event.Change, event.Before, event.After}
	if hash(payload) != event.Digest {
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

type Dismissal struct {
	EventID   string `json:"eventId"`
	ManagerID string `json:"managerId"`
}

type Store struct {
	APIVersion string      `json:"apiVersion"`
	Briefings  []Bundle    `json:"briefings"`
	Dismissals []Dismissal `json:"dismissals"`
}

// Read validates and returns the operational event history and its current
// content digest. A missing store has an empty deterministic state and digest.
func Read(root string) (Store, string, error) {
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
	if state.APIVersion != APIVersion || state.Briefings == nil || state.Dismissals == nil {
		return Store{}, "", errors.New("briefing store has invalid shape or API version")
	}
	if err := validateStore(state); err != nil {
		return Store{}, "", fmt.Errorf("validate briefing store: %w", err)
	}
	return state, StoreDigest(state), nil
}

// StoreDigest returns the canonical digest used as the optimistic write token.
func StoreDigest(state Store) string { return hash(state) }

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
	if bundle.SinceModelDigest == bundle.ModelDigest {
		_, current, err := Read(root)
		if err != nil {
			return "", err
		}
		if current != expectedDigest {
			return "", ErrStaleStore
		}
		return current, nil
	}
	return update(root, expectedDigest, func(state *Store) error {
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
					if old.ID == event.ID && hash(old) != hash(event) {
						return errors.New("model event identity already exists with different content")
					}
				}
			}
		}
		state.Briefings = append(state.Briefings, bundle)
		sort.Slice(state.Briefings, func(i, j int) bool {
			if state.Briefings[i].Revision != state.Briefings[j].Revision {
				return state.Briefings[i].Revision < state.Briefings[j].Revision
			}
			return state.Briefings[i].Global.ID < state.Briefings[j].Global.ID
		})
		return nil
	})
}

// Dismiss records a local visibility choice. It has no field that could resolve
// an event or change its conformity state.
func Dismiss(root, eventID, managerID, expectedDigest string) (string, error) {
	if strings.TrimSpace(eventID) == "" || strings.TrimSpace(managerID) == "" {
		return "", ErrDismissal
	}
	return update(root, expectedDigest, func(state *Store) error {
		found := false
		for _, bundle := range state.Briefings {
			for _, event := range bundle.Events {
				if event.ID == eventID && contains(event.AffectedManagers, managerID) {
					found = true
				}
			}
		}
		if !found {
			return errors.New("event is not available to the selected manager")
		}
		for _, d := range state.Dismissals {
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
// It rejects a stale digest instead of silently switching to a newer model.
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
			return nil, nil, "", ErrStaleModel
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
	eventsByID := map[string]Event{}
	for _, bundle := range chain {
		for _, briefing := range bundle.Managers {
			if briefing.ManagerID != managerID {
				continue
			}
			briefings = append(briefings, briefing)
			for _, event := range bundle.Events {
				if contains(briefing.EventIDs, event.ID) {
					eventsByID[event.ID] = event
				}
			}
		}
	}
	events := make([]Event, 0, len(eventsByID))
	for _, event := range eventsByID {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].DefinitionID.Key() != events[j].DefinitionID.Key() {
			return events[i].DefinitionID.Key() < events[j].DefinitionID.Key()
		}
		return events[i].ID < events[j].ID
	})
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
	return Store{APIVersion: APIVersion, Briefings: []Bundle{}, Dismissals: []Dismissal{}}
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
	state, current, err := Read(root)
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
