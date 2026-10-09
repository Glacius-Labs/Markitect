package projectrun

import (
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
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

type runStore struct {
	root string
	base string
}

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

func planDigest(plan PlanRecord) (string, error) {
	return digest(struct {
		APIVersion              string                       `json:"apiVersion"`
		ID                      string                       `json:"id"`
		Status                  string                       `json:"status"`
		Operation               string                       `json:"operation"`
		Goal                    string                       `json:"goal"`
		ExecuteAuthorized       bool                         `json:"executeAuthorized"`
		BaseRevision            string                       `json:"baseRevision"`
		ChangeBaseRevision      string                       `json:"changeBaseRevision,omitempty"`
		ChangeBaseSnapshot      string                       `json:"changeBaseSnapshot,omitempty"`
		ChangeBaseProjectDigest string                       `json:"changeBaseProjectDigest,omitempty"`
		ChangeBaseModelDigest   string                       `json:"changeBaseModelDigest,omitempty"`
		ChangeBaseReportDigest  string                       `json:"changeBaseReportDigest,omitempty"`
		ChangeImpact            *projectmodel.ChangeImpact   `json:"changeImpact,omitempty"`
		ChangeImpactDigest      string                       `json:"changeImpactDigest,omitempty"`
		TargetBranch            string                       `json:"targetBranch"`
		TargetHead              string                       `json:"targetHead"`
		RepositoryDigest        string                       `json:"repositoryDigest"`
		BaseSnapshot            string                       `json:"baseSnapshot"`
		WorkingSnapshot         string                       `json:"workingSnapshot"`
		BaseProjectDigest       string                       `json:"baseProjectDigest"`
		WorkingProjectDigest    string                       `json:"workingProjectDigest"`
		BaseModelDigest         string                       `json:"baseModelDigest"`
		ModelDigest             string                       `json:"modelDigest"`
		ReportDigest            string                       `json:"reportDigest"`
		RuntimeDigest           string                       `json:"runtimeDigest"`
		Managers                []ManagerTask                `json:"managers"`
		Strictness              map[string]StrictnessProfile `json:"strictness"`
		BriefingDigests         map[string]string            `json:"briefingDigests,omitempty"`
		Checks                  []CheckPlan                  `json:"checks"`
		ModelEdit               *EditPlan                    `json:"modelEdit,omitempty"`
		InitialCandidateID      string                       `json:"initialCandidateId"`
		RuntimeAgents           map[string]string            `json:"runtimeAgents"`
		Findings                []string                     `json:"findings,omitempty"`
		Blockers                []string                     `json:"blockers,omitempty"`
	}{plan.APIVersion, plan.ID, plan.Status, plan.Operation, plan.Goal, plan.ExecuteAuthorized, plan.BaseRevision, plan.ChangeBaseRevision, plan.ChangeBaseSnapshot, plan.ChangeBaseProjectDigest, plan.ChangeBaseModelDigest, plan.ChangeBaseReportDigest, plan.ChangeImpact, plan.ChangeImpactDigest, plan.TargetBranch, plan.TargetHead, plan.RepositoryDigest, plan.BaseSnapshot, plan.WorkingSnapshot, plan.BaseProjectDigest, plan.WorkingProjectDigest, plan.BaseModelDigest, plan.ModelDigest, plan.ReportDigest, plan.RuntimeDigest, plan.Managers, plan.Strictness, plan.BriefingDigests, plan.Checks, plan.ModelEdit, plan.InitialCandidateID, plan.RuntimeAgents, plan.Findings, plan.Blockers})
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
	if err := rejectReparsePath(abs, RuntimePath); err != nil {
		return nil, err
	}
	base := filepath.Join(abs, filepath.FromSlash(RunsPath))
	if info, statErr := os.Lstat(base); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("runtime state path %s must be a real directory", base)
		}
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("inspect runtime state path %s: %w", base, statErr)
	}
	return &runStore{root: abs, base: base}, nil
}

func ensureDirectory(path string) error {
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

func (s *runStore) lock() (func(), error) {
	if err := ensureDirectory(filepath.Dir(s.base)); err != nil {
		return nil, err
	}
	if err := ensureDirectory(s.base); err != nil {
		return nil, err
	}
	path := filepath.Join(s.base, ".write.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if os.IsExist(err) {
		removed, staleErr := removeStaleRuntimeLock(path, s.root)
		if staleErr != nil {
			return nil, staleErr
		}
		if removed {
			file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		}
	}
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("create runtime writer lock: %w", err)
	}
	lockData, marshalErr := json.Marshal(runtimeLock{PID: os.Getpid(), Root: s.root, StartedAt: time.Now().UTC()})
	if marshalErr != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, marshalErr
	}
	if _, err := file.Write(append(lockData, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write runtime writer lock: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("flush runtime writer lock: %w", err)
	}
	lockInfo, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, statErr
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close runtime writer lock: %w", err)
	}
	return func() {
		if current, err := os.Stat(path); err == nil && os.SameFile(current, lockInfo) {
			_ = os.Remove(path)
		}
	}, nil
}

type runtimeLock struct {
	PID       int       `json:"pid"`
	Root      string    `json:"root"`
	StartedAt time.Time `json:"startedAt"`
}

func removeStaleRuntimeLock(path, root string) (bool, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("runtime lock path is not a regular file")
	}
	var lock runtimeLock
	if err := readJSON(path, &lock); err != nil {
		return false, fmt.Errorf("runtime writer lock is unreadable and remains held: %w", err)
	}
	if lock.Root != root || lock.PID <= 0 {
		return false, fmt.Errorf("runtime writer lock owner is invalid and remains held")
	}
	alive, err := processAlive(lock.PID)
	if err != nil || alive {
		return false, ErrLocked
	}
	current, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !os.SameFile(before, current) {
		return false, ErrLocked
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("remove stale runtime lock: %w", err)
	}
	return true, nil
}

func (s *runStore) runDir(id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("invalid run ID %q", id)
	}
	dir := filepath.Join(s.base, id)
	if info, err := os.Lstat(dir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("runtime run path %s must be a real directory", dir)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return dir, nil
}

func requireRealSubdirectory(parent, name string) (string, error) {
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid runtime state directory %q", name)
	}
	path := filepath.Join(parent, name)
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
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", fmt.Errorf("create run directory: %w", err)
	}
	if err := ensureDirectory(filepath.Join(dir, "states")); err != nil {
		return "", err
	}
	if err := ensureDirectory(filepath.Join(dir, "candidates")); err != nil {
		return "", err
	}
	if err := ensureDirectory(filepath.Join(dir, "reports")); err != nil {
		return "", err
	}
	return dir, nil
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

func writeImmutableJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return writeImmutable(path, append(data, '\n'))
}

func writeImmutable(path string, data []byte) error {
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime parent %s is not a real directory", parent)
	}
	tmpID, err := newID()
	if err != nil {
		return err
	}
	tmp := filepath.Join(parent, ".pending-"+tmpID)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create pending runtime file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write pending runtime file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flush pending runtime file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close pending runtime file: %w", err)
	}
	if err := os.Link(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("publish runtime file: %w", err)
	}
	if err := os.Remove(tmp); err != nil {
		return fmt.Errorf("remove published runtime temporary file: %w", err)
	}
	return nil
}

func readJSON(path string, dest any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime file %s must be a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := validateNoDuplicateJSONKeys(data); err != nil {
		return fmt.Errorf("decode runtime file %s: %w", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("decode runtime file %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("runtime file %s has trailing data", path)
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
	if err := writeImmutableJSON(filepath.Join(dir, "plan.json"), plan); err != nil {
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
	if err := readJSON(filepath.Join(dir, "plan.json"), &plan); err != nil {
		if os.IsNotExist(err) {
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
	return writeImmutableJSON(path, report)
}

func (s *runStore) readLatestState(id string) (RunReport, error) {
	var latest RunReport
	dir, err := s.runDir(id)
	if err != nil {
		return latest, err
	}
	statesDir, err := requireRealSubdirectory(dir, "states")
	if err != nil {
		return latest, err
	}
	entries, err := os.ReadDir(statesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return latest, ErrNotFound
		}
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
		if err := readJSON(filepath.Join(statesDir, name), &state); err != nil {
			return latest, err
		}
		if state.ID != id || state.APIVersion != APIVersion {
			return latest, fmt.Errorf("stored run state identity is invalid")
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
	return writeImmutableJSON(filepath.Join(dir, "candidates", data.ID+".json"), data)
}

func (s *runStore) readCandidate(dir, id string) (candidateData, error) {
	var data candidateData
	if !validID(id) {
		return data, fmt.Errorf("invalid candidate ID")
	}
	candidateDir, err := requireRealSubdirectory(dir, "candidates")
	if err != nil {
		return data, err
	}
	if err := readJSON(filepath.Join(candidateDir, id+".json"), &data); err != nil {
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
