// Package runner starts Apply in the Linux runner image and returns its
// verified result as a branch (DEC-077). The launcher runs on the owner's
// host: it spells the runner's mounts for the engine CLI, starts one runner
// container and fetches the returned Git bundle into the owner's repository.
// Inside the image, the entrypoint clones the read-only checkout, runs
// markitect apply against that clone and writes the hand-back. The runner
// knows no models, Runs or reviews; Apply inside the image keeps its own
// contracts. See docs/design/linux-runner.md.
package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Fixed paths inside the runner image. The launcher mounts the owner's inputs
// at these targets; the entrypoint and Apply inside the image use them.
const (
	SourceDir       = "/markitect/source.git"
	InvocationDir   = "/markitect/invocation"
	CredentialDir   = "/markitect/credentials"
	OutputDir       = "/markitect/output"
	StateDir        = "/markitect/state"
	CloneDir        = "/markitect/work/repository"
	EngineSocket    = "/var/run/docker.sock"
	EngineBinary    = "/usr/local/bin/docker"
	MarkitectBinary = "/usr/local/bin/markitect"
)

// Files of the invocation and hand-back areas.
const (
	RunConfigFile      = "run.json"
	RuntimeProfileFile = "runtime-profile.yaml"
	ReportFile         = "apply-report.json"
	ResultFile         = "result.json"
	BundleFile         = "result.bundle"
)

// WorkBranch is the clone's own branch. Apply publishes into its working
// tree, the entrypoint commits there, and the bundle carries it as ResultRef.
const (
	WorkBranch = "markitect/runner"
	ResultRef  = "refs/heads/" + WorkBranch
)

// Apply statuses as the Apply report states them.
const (
	StatusPassed       = "passed"
	StatusFailed       = "failed"
	StatusNotCompleted = "not_completed"
)

// CredentialPath is where the runner sees the n-th mounted login file, from 1.
func CredentialPath(n int) string { return CredentialDir + "/" + strconv.Itoa(n) + ".json" }

// Layout names the paths the entrypoint works with. ContainerLayout returns
// the image's fixed paths; tests substitute temporary directories.
type Layout struct {
	// Source is the read-only mount of the checkout's Git directory, the clone source.
	Source string
	// Invocation holds the translated run configuration and runtime profile.
	Invocation string
	// Output is the writable hand-back area.
	Output string
	// Clone is the runner's own clone; it must not exist yet.
	Clone string
	// Markitect is the binary that runs Apply.
	Markitect string
}

// ContainerLayout returns the runner image's fixed layout.
func ContainerLayout() Layout {
	return Layout{Source: SourceDir, Invocation: InvocationDir, Output: OutputDir, Clone: CloneDir, Markitect: MarkitectBinary}
}

// Result is the runner's hand-back record in ResultFile. ApplyStatus is the
// status of the Apply report. Error names a failure of the runner itself or of
// the hand-back; then no result is returned, whatever Apply reported.
type Result struct {
	FormatVersion int      `json:"formatVersion"`
	ApplyStatus   string   `json:"applyStatus"`
	ApplyID       string   `json:"applyID,omitempty"`
	ApplyExit     int      `json:"applyExit"`
	Base          string   `json:"base"`
	NoChange      bool     `json:"noChange"`
	Commit        string   `json:"commit,omitempty"`
	Ref           string   `json:"ref,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	Error         string   `json:"error,omitempty"`
}

const resultFormat = 1

var (
	objectIDPattern = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
	applyIDPattern  = regexp.MustCompile(`^[0-9a-f]{32}$`)
	imageIDPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// Engine volume and container names.
	engineNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
)

func validObjectID(id string) bool { return objectIDPattern.MatchString(id) }

// RepositoryRelative reports whether p is a clean slash-separated path inside
// the repository: not absolute, without "..", backslash, drive or empty element.
// With dot, "." (the repository root) is accepted too.
func RepositoryRelative(p string, dot bool) bool {
	if p == "." {
		return dot
	}
	if p == "" || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\:\x00\r\n") || path.Clean(p) != p {
		return false
	}
	for _, element := range strings.Split(p, "/") {
		if element == ".." || element == "." {
			return false
		}
	}
	return true
}

// validate checks the record's own consistency, not what it claims.
func (r Result) validate() error {
	if r.FormatVersion != resultFormat || !validObjectID(r.Base) {
		return fmt.Errorf("runner result has an unknown format or base")
	}
	switch r.ApplyStatus {
	case StatusPassed, StatusFailed, StatusNotCompleted:
	default:
		return fmt.Errorf("runner result has an unknown Apply status")
	}
	if r.ApplyID != "" && !applyIDPattern.MatchString(r.ApplyID) {
		return fmt.Errorf("runner result has an invalid Apply ID")
	}
	if r.Error != "" || r.ApplyStatus != StatusPassed || r.NoChange {
		if r.Commit != "" || r.Ref != "" || len(r.Paths) != 0 {
			return fmt.Errorf("runner result names a branch without a verified change")
		}
		return nil
	}
	if r.ApplyID == "" || !validObjectID(r.Commit) || r.Ref != ResultRef || len(r.Paths) == 0 {
		return fmt.Errorf("runner result is incomplete")
	}
	seen := map[string]bool{}
	for _, p := range r.Paths {
		if !RepositoryRelative(p, false) || seen[p] {
			return fmt.Errorf("runner result names an invalid or repeated path")
		}
		seen[p] = true
	}
	return nil
}

// readBoundedFile reads one regular file of at most limit bytes.
func readBoundedFile(name string, limit int64) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(name))
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", filepath.Base(name), limit)
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", filepath.Base(name), limit)
	}
	return data, nil
}

func readResult(name string) (Result, error) {
	data, err := readBoundedFile(name, 1<<20)
	if err != nil {
		return Result{}, err
	}
	var r Result
	if err := decodeStrict(data, &r); err != nil {
		return Result{}, fmt.Errorf("runner result: %w", err)
	}
	return r, r.validate()
}

func writeResult(dir string, r Result) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ResultFile), append(data, '\n'), 0o644)
}
