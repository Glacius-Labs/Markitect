package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
)

type runStore struct {
	root            string
	base            string
	storage         *recordstore.Store
	identity        os.FileInfo
	beforeStateRead func() error
}

// Only a successful common read with no named record creates this signal.
var errRuntimeRecordMissing = errors.New("validated Runtime record is missing")

type candidateData struct {
	APIVersion string          `json:"apiVersion"`
	ID         string          `json:"id"`
	Parents    []string        `json:"parents,omitempty"`
	Files      map[string]File `json:"files"`
	Digest     string          `json:"digest"`
}

type File struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Content []byte `json:"content"`
	Delete  bool   `json:"delete,omitempty"`
}

// PlanPreviewDigest identifies a plan independent of the run identity,
// initial candidate identity and authorization that persisting assigns, so a
// write can be bound to the reviewed preview.
func PlanPreviewDigest(plan PlanRecord) (string, error) {
	stable := plan
	stable.ID, stable.Status, stable.InitialCandidateID = "", "", ""
	stable.ExecuteAuthorized = false
	return planDigest(stable)
}

func planDigest(plan PlanRecord) (string, error) {
	base, err := digest(struct {
		APIVersion              string                                     `json:"apiVersion"`
		ID                      string                                     `json:"id"`
		Status                  string                                     `json:"status"`
		Operation               string                                     `json:"operation"`
		Goal                    string                                     `json:"goal"`
		ExecuteAuthorized       bool                                       `json:"executeAuthorized"`
		BaseRevision            string                                     `json:"baseRevision"`
		ChangeBaseRevision      string                                     `json:"changeBaseRevision,omitempty"`
		ChangeBaseSnapshot      string                                     `json:"changeBaseSnapshot,omitempty"`
		ChangeBaseProjectDigest string                                     `json:"changeBaseProjectDigest,omitempty"`
		ChangeBaseModelDigest   string                                     `json:"changeBaseModelDigest,omitempty"`
		ChangeBaseReportDigest  string                                     `json:"changeBaseReportDigest,omitempty"`
		ChangeImpact            *projectmodel.ChangeImpact                 `json:"changeImpact,omitempty"`
		ChangeImpactDigest      string                                     `json:"changeImpactDigest,omitempty"`
		TargetBranch            string                                     `json:"targetBranch"`
		TargetHead              string                                     `json:"targetHead"`
		RepositoryDigest        string                                     `json:"repositoryDigest"`
		BaseSnapshot            string                                     `json:"baseSnapshot"`
		WorkingSnapshot         string                                     `json:"workingSnapshot"`
		BaseProjectDigest       string                                     `json:"baseProjectDigest"`
		WorkingProjectDigest    string                                     `json:"workingProjectDigest"`
		BaseModelDigest         string                                     `json:"baseModelDigest"`
		ModelDigest             string                                     `json:"modelDigest"`
		ReportDigest            string                                     `json:"reportDigest"`
		RuntimeDigest           string                                     `json:"runtimeDigest"`
		RuntimeBindingVersion   int                                        `json:"runtimeBindingVersion,omitempty"`
		Managers                []ManagerTask                              `json:"managers"`
		Strictness              map[string]projectconfig.StrictnessProfile `json:"strictness"`
		BriefingDigests         map[string]string                          `json:"briefingDigests,omitempty"`
		Checks                  []CheckPlan                                `json:"checks"`
		ModelEdit               *projectvalue.EditPlan                     `json:"modelEdit,omitempty"`
		InitialCandidateID      string                                     `json:"initialCandidateId"`
		RuntimeAgents           map[string]string                          `json:"runtimeAgents"`
		Findings                []string                                   `json:"findings,omitempty"`
		Blockers                []string                                   `json:"blockers,omitempty"`
	}{plan.APIVersion, plan.ID, plan.Status, plan.Operation, plan.Goal, plan.ExecuteAuthorized, plan.BaseRevision, plan.ChangeBaseRevision, plan.ChangeBaseSnapshot, plan.ChangeBaseProjectDigest, plan.ChangeBaseModelDigest, plan.ChangeBaseReportDigest, plan.ChangeImpact, plan.ChangeImpactDigest, plan.TargetBranch, plan.TargetHead, plan.RepositoryDigest, plan.BaseSnapshot, plan.WorkingSnapshot, plan.BaseProjectDigest, plan.WorkingProjectDigest, plan.BaseModelDigest, plan.ModelDigest, plan.ReportDigest, plan.RuntimeDigest, plan.RuntimeBindingVersion, plan.Managers, plan.Strictness, plan.BriefingDigests, plan.Checks, plan.ModelEdit, plan.InitialCandidateID, plan.RuntimeAgents, plan.Findings, plan.Blockers})
	if err != nil || plan.ExplorationBinding == nil && plan.ExplorationID == "" && plan.ScopeID == "" && plan.ExplorationDigest == "" && plan.ReadinessDigest == "" {
		return base, err
	}
	return digest(struct {
		Base, ExplorationID, ScopeID, ExplorationDigest, ReadinessDigest string
		Binding                                                          any
	}{base, plan.ExplorationID, plan.ScopeID, plan.ExplorationDigest, plan.ReadinessDigest, plan.ExplorationBinding})
}

func newID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("create project run ID: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode digest input: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func newRunStore(root string) (*runStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	if err := rejectReparsePath(abs, projectconfig.RuntimePath); err != nil {
		return nil, err
	}
	base := filepath.Join(abs, filepath.FromSlash(RunsPath))
	physical, err := recordstore.Open(abs)
	if err != nil {
		return nil, err
	}
	defer physical.Close()
	if err = physical.CheckCutover(recordstore.Run); err != nil {
		return nil, err
	}
	if _, err = physical.List(recordstore.Run, "", ""); err != nil {
		return nil, err
	}
	return &runStore{root: abs, base: base, identity: physical.Identity()}, nil
}

func ensureDirectory(path string, owners ...*runStore) error {
	if strings.Contains(filepath.ToSlash(path), "/"+recordstore.RootPath+"/") {
		store, kind, id, role, done, err := recordDirectoryOwner(path, owners...)
		if err != nil {
			return err
		}
		defer done()
		return store.Ensure(kind, id, role)
	}
	info, err := os.Lstat(path)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("runtime state path %s must be a real directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect runtime state directory %s: %w", path, err)
	}
	if err := os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create runtime state directory %s: %w", path, err)
	}
	info, err = os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime state directory %s changed during creation", path)
	}
	return nil
}

func (s *runStore) physical() (*recordstore.Store, func(), error) {
	if s.storage != nil {
		if err := s.storage.CheckIdentity(); err != nil {
			return nil, nil, err
		}
		return s.storage, func() {}, nil
	}
	store, err := recordstore.Open(s.root)
	if err != nil {
		return nil, nil, err
	}
	if s.identity != nil && !os.SameFile(s.identity, store.Identity()) {
		store.Close()
		return nil, nil, errors.New("Runtime record root identity changed")
	}
	return store, func() { store.Close() }, nil
}
func (s *runStore) lock() (func() error, error) {
	store, done, err := s.physical()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(runtimeLock{PID: os.Getpid(), Root: s.root, StartedAt: time.Now().UTC()})
	if err != nil {
		done()
		return nil, err
	}
	lease, err := store.Exclusive(recordstore.Run, "", ".write.lock", append(data, '\n'), 0, func(raw []byte) (bool, error) {
		var owner runtimeLock
		if err := decodeRuntimeJSON(raw, &owner); err != nil {
			return false, fmt.Errorf("runtime writer lock unreadable and remains held: %w", err)
		}
		if owner.Root != s.root || owner.PID <= 0 {
			return false, errors.New("runtime lock owner invalid and remains held")
		}
		alive, err := processAlive(owner.PID)
		if err != nil || alive {
			return false, ErrLocked
		}
		return true, nil
	})
	if err != nil {
		done()
		if errors.Is(err, os.ErrExist) {
			return nil, ErrLocked
		}
		return nil, err
	}
	s.storage = store
	return func() error {
		err := lease.Close()
		s.storage = nil
		done()
		if err != nil {
			return &recordstore.PublicationError{Path: RunsPath + "/.write.lock", Outcome: recordstore.Published, Cause: err}
		}
		return nil
	}, nil
}

type runtimeLock struct {
	PID       int       `json:"pid"`
	Root      string    `json:"root"`
	StartedAt time.Time `json:"startedAt"`
}

func (s *runStore) runDir(id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("invalid run ID %q", id)
	}
	dir := filepath.Join(s.base, id)
	physical, done, err := s.physical()
	if err != nil {
		return "", err
	}
	defer done()
	if _, err := physical.List(recordstore.Run, id, ""); err != nil {
		return "", err
	}
	return dir, nil
}

func requireRealSubdirectory(parent, name string, owners ...*runStore) (string, error) {
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid runtime state directory %q", name)
	}
	path := filepath.Join(parent, name)
	if strings.Contains(filepath.ToSlash(path), "/"+recordstore.RootPath+"/") {
		store, kind, id, role, done, err := recordDirectoryOwner(path, owners...)
		if err != nil {
			return "", err
		}
		defer done()
		if _, err := store.List(kind, id, role); err != nil {
			return "", err
		}
		return path, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("runtime state directory %s must be real", path)
	}
	return path, nil
}

func (s *runStore) createRun(id string) (string, error) {
	dir, err := s.runDir(id)
	if err != nil {
		return "", err
	}
	store, done, err := s.physical()
	if err != nil {
		return "", err
	}
	defer done()
	if err = store.CreateContainer(recordstore.Run, id); err != nil {
		return "", err
	}
	for _, role := range []string{"states", "candidates", "reports"} {
		if err = store.Ensure(recordstore.Run, id, role); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// RunSummary is one durable run as the project overview shows it.
type RunSummary struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	Operation string    `json:"operation"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ListRuns reads the recorded runs without taking the writer lock or changing
// anything. A run without a recorded state is planned; a run whose records do
// not validate is reported as invalid instead of failing the listing.
func ListRuns(root string) ([]RunSummary, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	// Listing needs no runtime configuration and never creates records or leases.
	base := filepath.Join(abs, filepath.FromSlash(RunsPath))
	physical, err := recordstore.Open(abs)
	if err != nil {
		return nil, err
	}
	defer physical.Close()
	s := &runStore{root: abs, base: base, identity: physical.Identity(), storage: physical}
	entries, err := physical.List(recordstore.Run, "", "")
	if err != nil {
		return nil, err
	}
	runs := []RunSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			if entry.Name() == ".write.lock" {
				continue
			}
			return nil, fmt.Errorf("unknown Run container entry %q", entry.Name())
		}
		if strings.HasPrefix(entry.Name(), "full-") {
			if _, err := (recordstore.Name{Kind: recordstore.Run, ID: entry.Name(), Role: "record.json"}).Path(); err != nil {
				return nil, err
			}
			continue
		}
		if !validID(entry.Name()) {
			return nil, fmt.Errorf("unknown Run container ID %q", entry.Name())
		}
		plan, err := s.readPlan(entry.Name())
		if errors.Is(err, ErrNotFound) {
			state, stateErr := s.readLatestState(entry.Name())
			if stateErr == nil && state.ManagerStageBinding != nil {
				continue
			}
		}
		if err != nil {
			runs = append(runs, RunSummary{ID: entry.Name(), Status: "invalid"})
			continue
		}
		summary := RunSummary{ID: plan.ID, Goal: plan.Goal, Operation: plan.Operation, Status: StatusPlanned, UpdatedAt: plan.PlannedAt}
		state, err := s.readLatestState(plan.ID)
		switch {
		case err == nil:
			summary.Status, summary.UpdatedAt = state.Status, state.UpdatedAt
		case !errors.Is(err, ErrNotFound):
			summary.Status = "invalid"
		}
		runs = append(runs, summary)
	}
	sort.Slice(runs, func(i, j int) bool {
		if !runs[i].UpdatedAt.Equal(runs[j].UpdatedAt) {
			return runs[i].UpdatedAt.After(runs[j].UpdatedAt)
		}
		return runs[i].ID < runs[j].ID
	})
	return runs, nil
}

func validID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, char := range id {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func recordPathOwner(absolute string, owners ...*runStore) (*recordstore.Store, recordstore.Name, func(), error) {
	slash := filepath.ToSlash(filepath.Clean(absolute))
	marker := "/" + recordstore.RootPath + "/"
	pos := strings.LastIndex(slash, marker)
	if pos < 0 {
		return nil, recordstore.Name{}, nil, errors.New("runtime record outside canonical store")
	}
	root := filepath.FromSlash(slash[:pos])
	name, err := recordstore.Parse(slash[pos+1:])
	if err != nil {
		return nil, name, nil, err
	}
	if len(owners) > 0 && owners[0] != nil {
		if !sameRecoveryPath(root, owners[0].root) {
			return nil, name, nil, errors.New("runtime record differs from owning root")
		}
		store, done, err := owners[0].physical()
		return store, name, done, err
	}
	store, err := recordstore.Open(root)
	return store, name, func() {
		if store != nil {
			store.Close()
		}
	}, err
}
func recordDirectoryOwner(absolute string, owners ...*runStore) (*recordstore.Store, recordstore.Kind, string, string, func(), error) {
	slash := filepath.ToSlash(filepath.Clean(absolute))
	marker := "/" + recordstore.RootPath + "/"
	pos := strings.LastIndex(slash, marker)
	if pos < 0 {
		return nil, "", "", "", nil, errors.New("runtime directory outside canonical records")
	}
	parts := strings.Split(slash[pos+len(marker):], "/")
	kind := recordstore.Kind(parts[0])
	id, role := "", ""
	if len(parts) > 1 {
		id = parts[1]
	}
	if len(parts) > 2 {
		role = strings.Join(parts[2:], "/")
	}
	root := filepath.FromSlash(slash[:pos])
	if len(owners) > 0 && owners[0] != nil {
		if !sameRecoveryPath(root, owners[0].root) {
			return nil, kind, id, role, nil, errors.New("runtime directory differs from owning root")
		}
		store, done, err := owners[0].physical()
		return store, kind, id, role, done, err
	}
	store, err := recordstore.Open(root)
	return store, kind, id, role, func() {
		if store != nil {
			store.Close()
		}
	}, err
}
func listRecordDirectory(absolute string, owners ...*runStore) ([]os.DirEntry, error) {
	store, kind, id, role, done, err := recordDirectoryOwner(absolute, owners...)
	if err != nil {
		return nil, err
	}
	defer done()
	return store.List(kind, id, role)
}
func readRecordBytes(absolute string, owners ...*runStore) ([]byte, error) {
	store, name, done, err := recordPathOwner(absolute, owners...)
	if err != nil {
		return nil, err
	}
	defer done()
	file, err := store.Read(name, 0)
	if err != nil {
		return nil, err
	}
	if !file.Exists {
		return nil, errRuntimeRecordMissing
	}
	return file.Bytes, nil
}
func writeImmutableJSON(path string, value any, owners ...*runStore) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeImmutable(path, append(data, '\n'), owners...)
}
func writeImmutable(path string, data []byte, owners ...*runStore) error {
	store, name, done, err := recordPathOwner(path, owners...)
	if err != nil {
		return err
	}
	defer done()
	return store.Create(name, data)
}
func readJSON(path string, dest any, owners ...*runStore) error {
	store, name, done, err := recordPathOwner(path, owners...)
	if err != nil {
		return err
	}
	defer done()
	file, err := store.Read(name, 0)
	if err != nil {
		return err
	}
	if !file.Exists {
		return errRuntimeRecordMissing
	}
	return decodeRuntimeJSON(file.Bytes, dest)
}
func decodeRuntimeJSON(data []byte, dest any) error {
	if err := validateNoDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("runtime record has trailing data")
	}
	return nil
}
func validateNoDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			folded := strings.ToLower(key)
			if seen[folded] {
				return fmt.Errorf("duplicate or case-aliased JSON key %q", key)
			}
			seen[folded] = true
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func (s *runStore) writePlan(plan PlanRecord) error {
	dir, err := s.runDir(plan.ID)
	if err != nil {
		return err
	}
	if err := writeImmutableJSON(filepath.Join(dir, "plan.json"), plan, s); err != nil {
		return err
	}
	return nil
}

func (s *runStore) readPlan(id string) (PlanRecord, error) {
	var plan PlanRecord
	dir, err := s.runDir(id)
	if err != nil {
		return plan, err
	}
	if err := readJSON(filepath.Join(dir, "plan.json"), &plan, s); err != nil {
		if errors.Is(err, errRuntimeRecordMissing) {
			return plan, ErrNotFound
		}
		return plan, err
	}
	if plan.ID != id || plan.APIVersion != APIVersion {
		return plan, fmt.Errorf("stored plan identity is invalid")
	}
	computed, err := planDigest(plan)
	if err != nil {
		return plan, err
	}
	if computed != plan.Digest {
		return plan, fmt.Errorf("stored plan digest mismatch")
	}
	return plan, nil
}

func (s *runStore) appendState(report RunReport) error {
	dir, err := s.runDir(report.ID)
	if err != nil {
		return err
	}
	report.Digest = ""
	computed, err := digest(report)
	if err != nil {
		return err
	}
	report.Digest = computed
	path := filepath.Join(dir, "states", fmt.Sprintf("%08d.json", report.Revision))
	return writeImmutableJSON(path, report, s)
}

func (s *runStore) readLatestState(id string) (RunReport, error) {
	var latest RunReport
	dir, err := s.runDir(id)
	if err != nil {
		return latest, err
	}
	statesDir, err := requireRealSubdirectory(dir, "states", s)
	if err != nil {
		return latest, err
	}
	entries, err := listRecordDirectory(statesDir, s)
	if err != nil {
		return latest, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".pending-") {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			return latest, fmt.Errorf("unknown state entry %q", name)
		}
		paths = append(paths, name)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return latest, ErrNotFound
	}
	for _, name := range paths {
		var state RunReport
		if s.beforeStateRead != nil {
			if err := s.beforeStateRead(); err != nil {
				return latest, err
			}
		}
		if err := readJSON(filepath.Join(statesDir, name), &state, s); err != nil {
			return latest, err
		}
		if state.ID != id || state.APIVersion != APIVersion {
			return latest, fmt.Errorf("stored run state identity is invalid")
		}
		if state.Revision <= 0 || name != fmt.Sprintf("%08d.json", state.Revision) {
			return latest, errors.New("stored run revision filename differs")
		}
		storedDigest := state.Digest
		state.Digest = ""
		computed, err := digest(state)
		if err != nil {
			return latest, err
		}
		if storedDigest == "" || storedDigest != computed {
			return latest, fmt.Errorf("stored run state digest mismatch")
		}
		state.Digest = storedDigest
		if state.Revision <= latest.Revision {
			return latest, fmt.Errorf("stored run state revisions are not increasing")
		}
		if err := validateManagerStageState(s, state, latest); err != nil {
			return latest, err
		}
		latest = state
	}
	return latest, nil
}

func (s *runStore) writeCandidate(dir string, data candidateData) error {
	if !validID(data.ID) {
		return fmt.Errorf("invalid candidate ID")
	}
	data.APIVersion = APIVersion
	computed, err := digest(struct {
		ID      string          `json:"id"`
		Parents []string        `json:"parents,omitempty"`
		Files   map[string]File `json:"files"`
	}{ID: data.ID, Parents: data.Parents, Files: data.Files})
	if err != nil {
		return err
	}
	data.Digest = computed
	return writeImmutableJSON(filepath.Join(dir, "candidates", data.ID+".json"), data, s)
}

func (s *runStore) readCandidate(dir, id string) (candidateData, error) {
	var data candidateData
	if !validID(id) {
		return data, fmt.Errorf("invalid candidate ID")
	}
	candidateDir, err := requireRealSubdirectory(dir, "candidates", s)
	if err != nil {
		return data, err
	}
	if err := readJSON(filepath.Join(candidateDir, id+".json"), &data, s); err != nil {
		return data, err
	}
	if data.ID != id || data.APIVersion != APIVersion {
		return data, fmt.Errorf("stored candidate identity is invalid")
	}
	for path, file := range data.Files {
		if path != file.Path {
			return data, fmt.Errorf("candidate file key/path mismatch")
		}
		if !safeRepoPath(path) {
			return data, fmt.Errorf("candidate contains unsafe path %q", path)
		}
	}
	computed, err := digest(struct {
		ID      string          `json:"id"`
		Parents []string        `json:"parents,omitempty"`
		Files   map[string]File `json:"files"`
	}{ID: data.ID, Parents: data.Parents, Files: data.Files})
	if err != nil {
		return data, err
	}
	if computed != data.Digest {
		return data, fmt.Errorf("candidate digest mismatch")
	}
	return data, nil
}

func snapshotWithCandidate(base *snapshot.Snapshot, candidate candidateData) (*snapshot.Snapshot, error) {
	if base == nil {
		return nil, fmt.Errorf("base snapshot is required")
	}
	copy := &snapshot.Snapshot{ID: base.ID, Provisional: base.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range base.Files {
		copy.Files[path] = append([]byte(nil), content...)
		copy.Modes[path] = base.Modes[path]
	}
	for path, file := range candidate.Files {
		if !safeRepoPath(path) || path != file.Path {
			return nil, fmt.Errorf("invalid candidate path %q", path)
		}
		if file.Delete {
			delete(copy.Files, path)
			delete(copy.Modes, path)
			continue
		}
		copy.Files[path] = append([]byte(nil), file.Content...)
		copy.Modes[path] = file.Mode
	}
	return copy, nil
}

func sortedFileKeys(files map[string]File) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
