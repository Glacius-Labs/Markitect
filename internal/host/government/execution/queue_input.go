package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func readQueueBacklog(repo, path string) (QueueBacklog, []byte, error) {
	if path == "" || !filepath.IsAbs(path) {
		return QueueBacklog{}, nil, errors.New("backlog path must be absolute")
	}
	data, err := readExternalQueueFile(repo, path)
	if err != nil {
		return QueueBacklog{}, nil, err
	}
	if err := rejectRuntimeDuplicateKeys(data); err != nil {
		return QueueBacklog{}, nil, err
	}
	if !utf8.Valid(data) {
		return QueueBacklog{}, nil, errors.New("backlog must be valid UTF-8")
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(data, &shape); err != nil {
		return QueueBacklog{}, nil, err
	}
	for _, key := range []string{"apiVersion", "stateDirectory", "limits", "jobs"} {
		value, ok := shape[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return QueueBacklog{}, nil, fmt.Errorf("backlog requires non-null %q", key)
		}
	}
	var jobRows []json.RawMessage
	if err := json.Unmarshal(shape["jobs"], &jobRows); err != nil || jobRows == nil {
		return QueueBacklog{}, nil, errors.New("backlog jobs must be an explicit array")
	}
	for i, row := range jobRows {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(row, &fields); err != nil {
			return QueueBacklog{}, nil, fmt.Errorf("jobs[%d]: %w", i, err)
		}
		for _, key := range []string{"id", "configPath", "orderPath", "runtimePath", "dependsOn"} {
			value, ok := fields[key]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return QueueBacklog{}, nil, fmt.Errorf("jobs[%d] requires non-null %q", i, key)
			}
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var backlog QueueBacklog
	if err := dec.Decode(&backlog); err != nil {
		return QueueBacklog{}, nil, fmt.Errorf("decode backlog: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return QueueBacklog{}, nil, errors.New("backlog contains trailing data")
	}
	return backlog, data, nil
}

func validateQueueBacklog(repo string, b QueueBacklog, raw []byte) error {
	if b.APIVersion != QueueVersion {
		return fmt.Errorf("apiVersion must be %q", QueueVersion)
	}
	if len(raw) == 0 || len(raw) > maxRuntimeJSONBytes {
		return errors.New("backlog is empty or exceeds 4 MiB")
	}
	if len(b.Jobs) > 128 {
		return errors.New("backlog must contain at most 128 finite jobs")
	}
	if b.Limits.ActorStarts < 1 || b.Limits.ActorStarts > 4096 || b.Limits.MaxRepairs < 0 || b.Limits.MaxRepairs > 64 || b.Limits.MaxWallTimeSeconds < 1 || b.Limits.MaxWallTimeSeconds > 86400 || b.Limits.MaxParallelism < 1 || b.Limits.MaxParallelism > 16 {
		return errors.New("queue limits are outside their finite bounds")
	}
	state, err := realDirectory(b.StateDirectory)
	if err != nil {
		return fmt.Errorf("stateDirectory: %w", err)
	}
	if !filepath.IsAbs(b.StateDirectory) || filepath.Clean(state) != filepath.Clean(b.StateDirectory) {
		return errors.New("stateDirectory must be a normalized absolute existing directory")
	}
	if _, err := queueRepo(repo, state); err != nil {
		return err
	}
	seen := map[string]bool{}
	positions := map[string]int{}
	for i, j := range b.Jobs {
		if !queueIDPattern.MatchString(j.ID) || seen[j.ID] {
			return fmt.Errorf("job ID %q is invalid or duplicated", j.ID)
		}
		seen[j.ID] = true
		positions[j.ID] = i
		if !government.SafePath(j.ConfigPath) || !government.SafePath(j.OrderPath) {
			return fmt.Errorf("job %q configPath and orderPath must be safe repository-relative paths", j.ID)
		}
		if !filepath.IsAbs(j.RuntimePath) {
			return fmt.Errorf("job %q runtimePath must be absolute", j.ID)
		}
		if len(j.DependsOn) > 128 {
			return fmt.Errorf("job %q has too many dependencies", j.ID)
		}
		depSeen := map[string]bool{}
		for _, d := range j.DependsOn {
			if d == j.ID || depSeen[d] {
				return fmt.Errorf("job %q has invalid or duplicate dependency %q", j.ID, d)
			}
			depSeen[d] = true
		}
	}
	for _, j := range b.Jobs {
		for _, d := range j.DependsOn {
			if !seen[d] {
				return fmt.Errorf("job %q depends on unknown job %q", j.ID, d)
			}
			if positions[d] >= positions[j.ID] {
				return fmt.Errorf("job %q must follow its explicit dependency %q", j.ID, d)
			}
		}
	}
	if err := validateQueueDAG(b.Jobs); err != nil {
		return err
	}
	return nil
}

func queueRepo(path, state string) (string, error) {
	repo, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	repo, err = realDirectory(repo)
	if err != nil {
		return "", err
	}
	id, err := source.IdentifyGit(repo)
	if err != nil {
		return "", err
	}
	common, err := realDirectory(id.CommonDir)
	if err != nil {
		return "", err
	}
	if directoriesOverlap(repo, state) || directoriesOverlap(common, state) {
		return "", errors.New("queue state must be outside repository and Git metadata")
	}
	return repo, nil
}

func queueRepositoryIdentity(path string) (string, error) {
	repo, err := realDirectory(path)
	if err != nil {
		return "", err
	}
	id, err := source.IdentifyGit(repo)
	if err != nil {
		return "", err
	}
	common, err := realDirectory(id.CommonDir)
	if err != nil {
		return "", err
	}
	gitDir, err := realDirectory(id.GitDir)
	if err != nil {
		return "", err
	}
	return repo + "\n" + common + "\n" + gitDir, nil
}

func readExternalQueueFile(repo, path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("external queue input path must be absolute")
	}
	root, err := realDirectory(repo)
	if err != nil {
		return nil, err
	}
	id, err := source.IdentifyGit(root)
	if err != nil {
		return nil, err
	}
	parent, err := realDirectory(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	full := filepath.Join(parent, filepath.Base(path))
	for _, candidate := range []string{root, id.CommonDir, id.GitDir} {
		resolved, e := realDirectory(candidate)
		if e != nil {
			return nil, e
		}
		if directoriesOverlap(full, resolved) {
			return nil, errors.New("backlog and runtime inputs must be external to source and Git metadata")
		}
	}
	return inventory.ReadInput(parent, filepath.Base(path), maxRuntimeJSONBytes)
}

func revalidateFrozenJobs(repo string, b QueueBacklog, s *queueState) error {
	if len(b.Jobs) != len(s.Jobs) {
		return errors.New("job set changed")
	}
	for _, job := range b.Jobs {
		frozen, ok := s.Jobs[job.ID]
		if !ok || frozen.QueueJob.ID != job.ID || frozen.ConfigPath != job.ConfigPath || frozen.OrderPath != job.OrderPath || frozen.RuntimePath != job.RuntimePath || strings.Join(frozen.DependsOn, "\x00") != strings.Join(job.DependsOn, "\x00") {
			return fmt.Errorf("job %q bindings changed", job.ID)
		}
		data, err := readExternalQueueFile(repo, job.RuntimePath)
		if err != nil {
			return err
		}
		if digestBytes(data) != frozen.RuntimeDigest {
			return fmt.Errorf("job %q runtime bytes changed", job.ID)
		}
		rt, err := ReadRuntime(repo, job.RuntimePath)
		if err != nil {
			return err
		}
		pin, err := FingerprintRuntime(rt)
		if err != nil || pin != frozen.RuntimePin || rt.TimeoutSeconds != frozen.TimeoutSeconds {
			return fmt.Errorf("job %q runtime fingerprint changed: %v", job.ID, err)
		}
		tool, err := toolPins(rt)
		if err != nil || tool != frozen.ToolPins {
			return fmt.Errorf("job %q tool pins changed: %v", job.ID, err)
		}
		frozen.Runtime = rt
		s.Jobs[job.ID] = frozen
	}
	return nil
}

func validateQueueDAG(jobs []QueueJob) error {
	deps := map[string][]string{}
	for _, j := range jobs {
		deps[j.ID] = j.DependsOn
	}
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("queue dependency cycle includes %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, d := range deps[id] {
			if err := visit(d); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, j := range jobs {
		if err := visit(j.ID); err != nil {
			return err
		}
	}
	return nil
}

func rehydrateFrozenJobs(repo string, b QueueBacklog, s *queueState) error {
	if len(b.Jobs) != len(s.Jobs) {
		return errors.New("frozen job set is missing or changed")
	}
	for _, job := range b.Jobs {
		frozen, ok := s.Jobs[job.ID]
		if !ok {
			return fmt.Errorf("job %q has no durable runtime binding", job.ID)
		}
		rt, err := ReadRuntime(repo, job.RuntimePath)
		if err != nil {
			return err
		}
		frozen.Runtime = rt
		s.Jobs[job.ID] = frozen
	}
	return nil
}

var queueIDPattern = regexpMustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func regexpMustCompile(s string) *regexp.Regexp { return regexp.MustCompile(s) }

func runtimeParallelism(runtime Runtime) int {
	if runtime.Recursion == nil {
		return 1
	}
	return runtime.Recursion.Parallelism
}
