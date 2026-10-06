// Package recordstore persists Host-owned projection and verification records.
// Stored facts do not authenticate their authors or imply semantic acceptance.
package recordstore

import (
	"bytes"
	"crypto/rand"
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

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const (
	APIVersion = "markitect.example.org/projection-record-store/v1alpha1"
	maxEvents  = 100_000
	maxBytes   = 256 << 20
	maxEvent   = 8 << 20
	marker     = "store.json"
	eventsDir  = "events"
	lockName   = ".store.lock"
	pending    = ".pending-"
)

var (
	ErrStaleHead  = errors.New("projection record store head is stale")
	ErrLocked     = errors.New("projection record store is locked")
	ErrIncomplete = errors.New("projection record store is incomplete")
)

type Store struct {
	root      string
	forbidden []string
	storeID   string
	genesis   string
}

type State struct {
	APIVersion      string
	StoreID         string
	Head            string
	Sequence        uint64
	Records         []records.ProjectionRecord
	Verifications   []records.VerificationResult
	ActiveSelection ActiveSelection
	ActiveHistory   []ActiveSelectionReceipt
}

// ActiveSelection is an explicit complete ownership set. It says nothing about
// verification outcome or semantic acceptance.
type ActiveSelection struct {
	Generation uint64
	RecordIDs  []string
	Digest     string
}

type ActiveSelectionReceipt struct {
	Sequence  uint64
	EventID   string
	Selection ActiveSelection
}

type StaleHeadError struct{ Expected, Actual string }

func (e *StaleHeadError) Error() string {
	return fmt.Sprintf("%v: expected %s, current %s", ErrStaleHead, e.Expected, e.Actual)
}
func (e *StaleHeadError) Unwrap() error { return ErrStaleHead }

// Initialize creates an absent external root exclusively. Host must supply the
// source, Git metadata, and materialization roots that this store must avoid.
func Initialize(root string, forbiddenRoots []string) (*Store, error) {
	p, err := preparePaths(root, forbiddenRoots, true)
	if err != nil {
		return nil, err
	}
	if _, err = os.Lstat(p.root); err == nil {
		return nil, fmt.Errorf("store root already exists: %s", p.root)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err = os.Mkdir(p.root, 0700); err != nil {
		return nil, fmt.Errorf("create store root exclusively %s: %w", p.root, err)
	}
	created := []string{p.root}
	fail := func(path string, cause error) (*Store, error) {
		return nil, fmt.Errorf("initialization stopped at %s; created paths retained: %s: %w", path, strings.Join(created, ", "), cause)
	}
	events := filepath.Join(p.root, eventsDir)
	if err = os.Mkdir(events, 0700); err != nil {
		return fail(events, err)
	}
	created = append(created, events)
	var randomID [16]byte
	if _, err = rand.Read(randomID[:]); err != nil {
		return fail(p.root, err)
	}
	storeID := hex.EncodeToString(randomID[:])
	body := markerBody{APIVersion: APIVersion, StoreID: storeID}
	encoded, _ := json.Marshal(body)
	meta := markerEnvelope{APIVersion: body.APIVersion, StoreID: storeID, ContentDigest: hash(encoded)}
	data, err := canonicalJSON(meta)
	if err != nil {
		return fail(p.root, err)
	}
	path := filepath.Join(p.root, marker)
	if err = writeExclusive(path, data); err != nil {
		return fail(path, err)
	}
	created = append(created, path)
	return &Store{root: p.root, forbidden: p.forbidden, storeID: storeID, genesis: meta.ContentDigest}, nil
}

// Open validates an existing external store without creating or repairing it.
func Open(root string, forbiddenRoots []string) (*Store, error) {
	p, err := preparePaths(root, forbiddenRoots, false)
	if err != nil {
		return nil, err
	}
	meta, genesis, err := readMarker(p.root)
	if err != nil {
		return nil, err
	}
	return &Store{root: p.root, forbidden: p.forbidden, storeID: meta.StoreID, genesis: genesis}, nil
}

// Read returns a complete validated history without creating or changing files.
// Existing locks and pending files are surfaced; a concurrent append may leave
// this call with a valid prior head, which callers must bind with expectedHead.
func (s *Store) Read() (State, error) {
	if err := s.refuseWriterLock(); err != nil {
		return State{}, err
	}
	state, err := s.readUnlocked()
	if err != nil {
		return State{}, err
	}
	if err = s.refuseWriterLock(); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Store) refuseWriterLock() error {
	if err := s.validatePaths(); err != nil {
		return err
	}
	path := filepath.Join(s.root, lockName)
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%w at %s; inspect writer before owner recovery: lock exists", ErrLocked, path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect writer lock %s: %w", path, err)
	}
	return nil
}

// AppendAttempt persists any valid materialization outcome. A repeated record
// content ID is idempotent. Partial/escalated records remain history only.
func (s *Store) AppendAttempt(expectedHead string, record records.ProjectionRecord) (State, error) {
	if err := records.ValidateProjectionRecord(record); err != nil {
		return State{}, fmt.Errorf("validate ProjectionRecord: %w", err)
	}
	unlock, err := s.lock()
	if err != nil {
		return State{}, err
	}
	defer unlock()
	state, err := s.readUnlocked()
	if err != nil {
		return State{}, err
	}
	for _, old := range state.Records {
		if old.ID == record.ID {
			return state, nil
		}
	}
	if state.Head != expectedHead {
		return State{}, &StaleHeadError{expectedHead, state.Head}
	}
	if len(state.Records) >= 50_000 {
		return State{}, errors.New("ProjectionRecord limit exceeded")
	}
	if record.PriorRecordID != "" {
		prior, ok := findRecord(state.Records, record.PriorRecordID)
		if !ok || prior.ProjectionID != record.ProjectionID {
			return State{}, errors.New("priorRecordId must reference an earlier record for the same Projection")
		}
	}
	return s.commit(eventBody{APIVersion: APIVersion, Sequence: state.Sequence + 1, PreviousEventDigest: state.Head, Kind: "attempt", Record: &record})
}

// AppendVerification persists a result bound to an earlier record. Host must
// first check current declared verifier/check identities. Outcomes never
// promote or remove active ownership. Repeated result IDs are idempotent.
func (s *Store) AppendVerification(expectedHead string, result records.VerificationResult) (State, error) {
	if err := records.ValidateVerificationResult(result); err != nil {
		return State{}, fmt.Errorf("validate VerificationResult: %w", err)
	}
	unlock, err := s.lock()
	if err != nil {
		return State{}, err
	}
	defer unlock()
	state, err := s.readUnlocked()
	if err != nil {
		return State{}, err
	}
	for _, old := range state.Verifications {
		if old.ID == result.ID {
			return state, nil
		}
	}
	if state.Head != expectedHead {
		return State{}, &StaleHeadError{expectedHead, state.Head}
	}
	record, ok := findRecord(state.Records, result.RecordID)
	if !ok {
		return State{}, fmt.Errorf("unknown ProjectionRecord %q", result.RecordID)
	}
	if err := validateResultBinding(result, record); err != nil {
		return State{}, err
	}
	return s.commit(eventBody{APIVersion: APIVersion, Sequence: state.Sequence + 1, PreviousEventDigest: state.Head, Kind: "verification", Verification: &result})
}

// SelectActive replaces the whole active set with an exact list of complete
// materializations. An explicit empty slice clears ownership; nil is refused.
// Verification PASS is not required and never changes this selection.
func (s *Store) SelectActive(expectedHead string, recordIDs []string) (State, error) {
	if recordIDs == nil {
		return State{}, errors.New("active selection must be explicit; pass an empty slice to select none")
	}
	unlock, err := s.lock()
	if err != nil {
		return State{}, err
	}
	defer unlock()
	state, err := s.readUnlocked()
	if err != nil {
		return State{}, err
	}
	selection, err := normalizeSelection(recordIDs, state.Records, state.ActiveSelection.Generation+1)
	if err != nil {
		return State{}, err
	}
	if equalStrings(selection.RecordIDs, state.ActiveSelection.RecordIDs) {
		return state, nil
	}
	if state.Head != expectedHead {
		return State{}, &StaleHeadError{expectedHead, state.Head}
	}
	return s.commit(eventBody{APIVersion: APIVersion, Sequence: state.Sequence + 1, PreviousEventDigest: state.Head, Kind: "active-selection", Selection: &selectionBody{Generation: selection.Generation, RecordIDs: selection.RecordIDs}})
}

func (s *Store) commit(body eventBody) (State, error) {
	event := envelopeFor(body)
	data, err := canonicalJSON(event)
	if err != nil {
		return State{}, err
	}
	if len(data) > maxEvent {
		return State{}, errors.New("event byte limit exceeded")
	}
	if body.Sequence == 0 || body.Sequence > maxEvents {
		return State{}, errors.New("event count limit exceeded")
	}
	events := filepath.Join(s.root, eventsDir)
	entries, err := os.ReadDir(events)
	if err != nil {
		return State{}, err
	}
	var currentBytes int64
	var eventCount int
	for _, entry := range entries {
		path := filepath.Join(events, entry.Name())
		if strings.HasPrefix(entry.Name(), pending) {
			return State{}, fmt.Errorf("%w at %s; owner recovery required", ErrIncomplete, path)
		}
		if _, ok := parseEventName(entry.Name()); !ok {
			return State{}, fmt.Errorf("malformed event filename: %s", path)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return State{}, err
		}
		if err = rejectReparse(path, info); err != nil {
			return State{}, err
		}
		if !info.Mode().IsRegular() || info.Size() < 2 || info.Size() > maxEvent {
			return State{}, fmt.Errorf("event must be bounded regular file: %s", path)
		}
		eventCount++
		currentBytes += info.Size()
	}
	if eventCount+1 != int(body.Sequence) {
		return State{}, errors.New("event sequence does not match current event count")
	}
	if err = checkAppendCapacity(eventCount, currentBytes, int64(len(data))); err != nil {
		return State{}, err
	}
	f, err := os.CreateTemp(events, pending+"*.json")
	if err != nil {
		return State{}, err
	}
	stage := f.Name()
	if _, err = f.Write(data); err != nil {
		f.Close()
		return State{}, fmt.Errorf("partial event retained at %s: %w", stage, err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return State{}, fmt.Errorf("partial event retained at %s: %w", stage, err)
	}
	if err = f.Close(); err != nil {
		return State{}, fmt.Errorf("event retained at %s: %w", stage, err)
	}
	name := eventName(body.Sequence, event.EventDigest)
	dest := filepath.Join(events, name)
	if _, err = os.Lstat(dest); err == nil {
		return State{}, fmt.Errorf("event destination exists: %s", dest)
	} else if !os.IsNotExist(err) {
		return State{}, err
	}
	if err = os.Rename(stage, dest); err != nil {
		return State{}, fmt.Errorf("event staging file retained at %s: %w", stage, err)
	}
	state, err := s.readUnlocked()
	if err != nil {
		return State{}, fmt.Errorf("event %s committed; readback failed: %w", dest, err)
	}
	return state, nil
}

func checkAppendCapacity(eventCount int, currentBytes, nextEventBytes int64) error {
	if eventCount >= maxEvents {
		return errors.New("event count limit exceeded")
	}
	if currentBytes < 0 || nextEventBytes < 0 || currentBytes > maxBytes-nextEventBytes {
		return errors.New("store byte limit exceeded")
	}
	return nil
}

func (s *Store) readUnlocked() (State, error) {
	if err := s.validatePaths(); err != nil {
		return State{}, err
	}
	meta, genesis, err := readMarker(s.root)
	if err != nil {
		return State{}, err
	}
	if meta.StoreID != s.storeID || genesis != s.genesis {
		return State{}, errors.New("store identity changed")
	}
	if err = s.validateRootEntries(); err != nil {
		return State{}, err
	}
	dir := filepath.Join(s.root, eventsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return State{}, err
	}

	type file struct {
		seq        uint64
		name, path string
	}
	files := make([]file, 0, len(entries))
	aliases := map[string]string{}
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		fold := strings.ToLower(name)
		if old, ok := aliases[fold]; ok && old != name {
			return State{}, fmt.Errorf("event case alias %q / %q", old, name)
		}
		aliases[fold] = name
		path := filepath.Join(dir, name)
		if strings.HasPrefix(name, pending) {
			return State{}, fmt.Errorf("%w at %s; owner recovery required", ErrIncomplete, path)
		}
		seq, ok := parseEventName(name)
		if !ok {
			return State{}, fmt.Errorf("malformed event filename: %s", path)
		}
		info, e := os.Lstat(path)
		if e != nil {
			return State{}, e
		}
		if e = rejectReparse(path, info); e != nil {
			return State{}, e
		}
		if !info.Mode().IsRegular() || info.Size() < 2 || info.Size() > maxEvent {
			return State{}, fmt.Errorf("event must be bounded regular file: %s", path)
		}
		total += info.Size()
		if total > maxBytes {
			return State{}, errors.New("store byte limit exceeded")
		}
		files = append(files, file{seq, name, path})
	}
	if len(files) > maxEvents {
		return State{}, errors.New("event count limit exceeded")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].seq < files[j].seq })
	state := State{APIVersion: APIVersion, StoreID: s.storeID, Head: genesis, Records: []records.ProjectionRecord{}, Verifications: []records.VerificationResult{}}
	state.ActiveSelection = ActiveSelection{RecordIDs: []string{}}
	state.ActiveSelection.Digest = selectionDigest(0, state.ActiveSelection.RecordIDs)
	recordByID := map[string]records.ProjectionRecord{}
	resultIDs := map[string]bool{}
	for i, item := range files {
		if item.seq != uint64(i+1) {
			return State{}, fmt.Errorf("noncontiguous sequence at %s", item.path)
		}
		data, e := readRegular(item.path, maxEvent)
		if e != nil {
			return State{}, e
		}
		var env eventEnvelope
		if e = decodeCanonical(data, &env); e != nil {
			return State{}, fmt.Errorf("decode %s: %w", item.path, e)
		}
		body := env.body()
		want := eventDigest(body)
		if env.APIVersion != APIVersion || env.Sequence != item.seq || env.PreviousEventDigest != state.Head || env.EventDigest != want || item.name != eventName(item.seq, env.EventDigest) {
			return State{}, fmt.Errorf("event hash or chain mismatch at %s", item.path)
		}
		if e = applyEvent(&state, body, env.EventDigest, recordByID, resultIDs); e != nil {
			return State{}, fmt.Errorf("invalid event %s: %w", item.path, e)
		}
		state.Sequence = item.seq
		state.Head = env.EventDigest
	}
	if err = records.ValidateRecordReferences(state.Records); err != nil {
		return State{}, err
	}
	return state, nil
}

func applyEvent(s *State, b eventBody, eventID string, recordByID map[string]records.ProjectionRecord, resultIDs map[string]bool) error {
	if b.Sequence != s.Sequence+1 || b.PreviousEventDigest != s.Head {
		return errors.New("event does not extend current head")
	}
	switch b.Kind {
	case "attempt":
		if b.Record == nil || b.Verification != nil || b.Selection != nil {
			return errors.New("attempt event has wrong payload shape")
		}
		r := *b.Record
		if err := records.ValidateProjectionRecord(r); err != nil {
			return err
		}
		if _, ok := recordByID[r.ID]; ok {
			return errors.New("duplicate ProjectionRecord ID")
		}
		if r.PriorRecordID != "" {
			prior, ok := recordByID[r.PriorRecordID]
			if !ok || prior.ProjectionID != r.ProjectionID {
				return errors.New("dangling or cross-Projection priorRecordId")
			}
		}
		recordByID[r.ID] = r
		s.Records = append(s.Records, r)
		if len(s.Records) > 50_000 {
			return errors.New("ProjectionRecord limit exceeded")
		}
	case "verification":
		if b.Record != nil || b.Verification == nil || b.Selection != nil {
			return errors.New("verification event has wrong payload shape")
		}
		v := *b.Verification
		if err := records.ValidateVerificationResult(v); err != nil {
			return err
		}
		if resultIDs[v.ID] {
			return errors.New("duplicate VerificationResult ID")
		}
		r, ok := recordByID[v.RecordID]
		if !ok {
			return errors.New("dangling VerificationResult recordId")
		}
		if err := validateResultBinding(v, r); err != nil {
			return err
		}
		resultIDs[v.ID] = true
		s.Verifications = append(s.Verifications, v)
		if len(s.Verifications) > 100_000 {
			return errors.New("VerificationResult limit exceeded")
		}
	case "active-selection":
		if b.Record != nil || b.Verification != nil || b.Selection == nil {
			return errors.New("active-selection event has wrong payload shape")
		}
		if b.Selection.Generation != s.ActiveSelection.Generation+1 {
			return errors.New("active selection generation mismatch")
		}
		next, err := normalizeSelection(b.Selection.RecordIDs, s.Records, b.Selection.Generation)
		if err != nil {
			return err
		}
		if !equalStrings(next.RecordIDs, b.Selection.RecordIDs) {
			return errors.New("active record IDs are not sorted")
		}
		s.ActiveSelection = next
		s.ActiveHistory = append(s.ActiveHistory, ActiveSelectionReceipt{Sequence: b.Sequence, EventID: eventID, Selection: next})
	default:
		return fmt.Errorf("unknown event kind %q", b.Kind)
	}
	return nil
}

func normalizeSelection(ids []string, all []records.ProjectionRecord, generation uint64) (ActiveSelection, error) {
	if ids == nil {
		return ActiveSelection{}, errors.New("recordIds must be an explicit array")
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	byID := map[string]records.ProjectionRecord{}
	for _, r := range all {
		byID[r.ID] = r
	}
	seenProjection := map[string]bool{}
	active := make([]records.ProjectionRecord, 0, len(sorted))
	for i, id := range sorted {
		if id == "" || (i > 0 && sorted[i-1] == id) {
			return ActiveSelection{}, errors.New("active record IDs must be nonempty and unique")
		}
		r, ok := byID[id]
		if !ok {
			return ActiveSelection{}, fmt.Errorf("unknown active record %q", id)
		}
		if r.State != records.StateMaterializedUnverified {
			return ActiveSelection{}, fmt.Errorf("record %q with state %q cannot be active", id, r.State)
		}
		if seenProjection[r.ProjectionID] {
			return ActiveSelection{}, fmt.Errorf("multiple active records for Projection %q", r.ProjectionID)
		}
		seenProjection[r.ProjectionID] = true
		active = append(active, r)
	}
	if _, err := records.BuildOwnershipIndex(active, nil); err != nil {
		return ActiveSelection{}, err
	}
	sel := ActiveSelection{Generation: generation, RecordIDs: sorted}
	sel.Digest = selectionDigest(generation, sorted)
	return sel, nil
}

func validateResultBinding(v records.VerificationResult, r records.ProjectionRecord) error {
	if v.RecordID != r.ID || v.Revision != r.Revision || v.ModelDigest != r.ModelDigest || v.TargetSnapshotDigest != r.TargetSnapshotDigest {
		return errors.New("verification source/model/target binding differs from its record")
	}
	checks := make([]records.CheckIdentity, 0, len(v.Checks))
	for _, c := range v.Checks {
		checks = append(checks, records.CheckIdentity{ID: c.ID, Version: c.Version, Digest: c.Digest})
	}
	fresh := records.Freshness{Revision: v.Revision, ModelDigest: v.ModelDigest, RecordID: r.ID, TargetSnapshotDigest: v.TargetSnapshotDigest, Verifier: v.Verifier, Checks: checks}
	return records.ValidateVerificationFreshness(v, r, fresh)
}

func (s *Store) lock() (func(), error) {
	if err := s.validatePaths(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.root, lockName)
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	value := hex.EncodeToString(nonce[:]) + "\n"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("%w at %s; inspect writer before owner recovery: %v", ErrLocked, path, err)
	}
	if _, err = f.WriteString(value); err != nil {
		f.Close()
		return nil, fmt.Errorf("lock retained at %s: %w", path, err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return nil, fmt.Errorf("lock retained at %s: %w", path, err)
	}
	if err = f.Close(); err != nil {
		return nil, fmt.Errorf("lock retained at %s: %w", path, err)
	}
	return func() {
		current, e := os.ReadFile(path)
		if e == nil && string(current) == value {
			_ = os.Remove(path)
		}
	}, nil
}

func (s *Store) validatePaths() error {
	p, err := preparePaths(s.root, s.forbidden, false)
	if err != nil {
		return err
	}
	if p.root != s.root {
		return errors.New("store root identity changed")
	}
	return nil
}
func (s *Store) validateRootEntries() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	allowed := map[string]bool{marker: true, eventsDir: true, lockName: true}
	seen := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		fold := strings.ToLower(name)
		if old, ok := seen[fold]; ok && old != name {
			return fmt.Errorf("store path case alias %q / %q", old, name)
		}
		seen[fold] = name
		if !allowed[name] {
			return fmt.Errorf("unknown store entry %s", filepath.Join(s.root, name))
		}
		path := filepath.Join(s.root, name)
		info, er := os.Lstat(path)
		if er != nil {
			return er
		}
		if er = rejectReparse(path, info); er != nil {
			return er
		}
		if name == eventsDir && !info.IsDir() {
			return fmt.Errorf("events path is not a directory: %s", path)
		}
		if name != eventsDir && !info.Mode().IsRegular() {
			return fmt.Errorf("store entry is not a regular file: %s", path)
		}
	}
	if _, ok := seen[marker]; !ok {
		return fmt.Errorf("%w: missing marker under %s", ErrIncomplete, s.root)
	}
	if _, ok := seen[eventsDir]; !ok {
		return fmt.Errorf("%w: missing marker or events directory under %s", ErrIncomplete, s.root)
	}
	return nil
}

func readMarker(root string) (markerEnvelope, string, error) {
	if err := checkDirectory(root); err != nil {
		return markerEnvelope{}, "", err
	}
	path := filepath.Join(root, marker)
	data, err := readRegular(path, maxEvent)
	if err != nil {
		return markerEnvelope{}, "", err
	}
	var m markerEnvelope
	if err = decodeCanonical(data, &m); err != nil {
		return m, "", err
	}
	body := markerBody{APIVersion: m.APIVersion, StoreID: m.StoreID}
	encoded, _ := json.Marshal(body)
	genesis := hash(encoded)
	if m.APIVersion != APIVersion || len(m.StoreID) != 32 || m.ContentDigest != genesis {
		return m, "", fmt.Errorf("invalid store marker: %s", path)
	}
	if _, err = hex.DecodeString(m.StoreID); err != nil {
		return m, "", fmt.Errorf("invalid store ID: %s", path)
	}
	return m, genesis, nil
}
func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
func readRegular(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if err = rejectReparse(path, info); err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > max {
		return nil, fmt.Errorf("expected bounded regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != info.Size() {
		return nil, fmt.Errorf("file changed while reading: %s", path)
	}
	return data, nil
}
func canonicalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
func decodeCanonical[T any](data []byte, out *T) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := d.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	want, err := canonicalJSON(out)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, want) {
		return errors.New("noncanonical JSON, duplicate key, or alternate encoding")
	}
	return nil
}
func hash(data []byte) string { x := sha256.Sum256(data); return "sha256:" + hex.EncodeToString(x[:]) }
func selectionDigest(g uint64, ids []string) string {
	b, _ := json.Marshal(selectionBody{Generation: g, RecordIDs: ids})
	return hash(b)
}
func findRecord(all []records.ProjectionRecord, id string) (records.ProjectionRecord, bool) {
	for _, r := range all {
		if r.ID == id {
			return r, true
		}
	}
	return records.ProjectionRecord{}, false
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

type markerBody struct {
	APIVersion string `json:"apiVersion"`
	StoreID    string `json:"storeId"`
}
type markerEnvelope struct {
	APIVersion    string `json:"apiVersion"`
	StoreID       string `json:"storeId"`
	ContentDigest string `json:"contentDigest"`
}
type selectionBody struct {
	Generation uint64   `json:"generation"`
	RecordIDs  []string `json:"recordIds"`
}
type eventBody struct {
	APIVersion          string                      `json:"apiVersion"`
	Sequence            uint64                      `json:"sequence"`
	PreviousEventDigest string                      `json:"previousEventDigest"`
	Kind                string                      `json:"kind"`
	Record              *records.ProjectionRecord   `json:"record,omitempty"`
	Verification        *records.VerificationResult `json:"verification,omitempty"`
	Selection           *selectionBody              `json:"selection,omitempty"`
}
type eventEnvelope struct {
	APIVersion          string                      `json:"apiVersion"`
	Sequence            uint64                      `json:"sequence"`
	PreviousEventDigest string                      `json:"previousEventDigest"`
	Kind                string                      `json:"kind"`
	Record              *records.ProjectionRecord   `json:"record,omitempty"`
	Verification        *records.VerificationResult `json:"verification,omitempty"`
	Selection           *selectionBody              `json:"selection,omitempty"`
	EventDigest         string                      `json:"eventDigest"`
}

func envelopeFor(b eventBody) eventEnvelope {
	raw, _ := json.Marshal(b)
	return eventEnvelope{APIVersion: b.APIVersion, Sequence: b.Sequence, PreviousEventDigest: b.PreviousEventDigest, Kind: b.Kind, Record: b.Record, Verification: b.Verification, Selection: b.Selection, EventDigest: hash(raw)}
}
func eventDigest(b eventBody) string { raw, _ := json.Marshal(b); return hash(raw) }
func (e eventEnvelope) body() eventBody {
	return eventBody{APIVersion: e.APIVersion, Sequence: e.Sequence, PreviousEventDigest: e.PreviousEventDigest, Kind: e.Kind, Record: e.Record, Verification: e.Verification, Selection: e.Selection}
}
func eventName(seq uint64, id string) string {
	return fmt.Sprintf("%020d-%s.json", seq, strings.TrimPrefix(id, "sha256:"))
}
func parseEventName(name string) (uint64, bool) {
	if len(name) != 90 || name[20] != '-' || !strings.HasSuffix(name, ".json") {
		return 0, false
	}
	for _, r := range name[:20] {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	if _, e := hex.DecodeString(name[21:85]); e != nil {
		return 0, false
	}
	var seq uint64
	if _, e := fmt.Sscanf(name[:20], "%d", &seq); e != nil || seq == 0 {
		return 0, false
	}
	return seq, true
}
