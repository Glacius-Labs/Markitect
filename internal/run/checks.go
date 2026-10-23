package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ValidateCheckObservations is shared by fixed Run and publication checkpoints.
// A physical observation failure remains technical; a known nonzero exit is
// candidate invalidity and cannot be waived by an approval.
func ValidateCheckObservations(declarations []project.ExecutionCheck, observed []CheckObservation, fixed Commit, tree snapshot.Tree, dirs []snapshot.Path) error {
	expected := map[string]bool{}
	for _, d := range declarations {
		expected[d.ID.Value] = true
	}
	for _, c := range observed {
		r := c.Receipt
		if !expected[c.ID] || c.Commit != fixed.ID || r.ID != CheckID(c.ID) || r.Tree != tree.Digest() || !sameDirectories(r.Directories, dirs) {
			return fmt.Errorf("%w: check %s fixed identity, directory inventory or multiplicity mismatch", ErrObservationIncomplete, c.ID)
		}
		if !r.Started || !r.Exited || r.Uncertain || r.TimedOut || !r.CopiesClosed || !imageIdentity(r.ImageID) || c.Error != "" {
			return fmt.Errorf("%w: check %s physical observations", ErrObservationIncomplete, c.ID)
		}
		if r.ExitCode != 0 {
			return fmt.Errorf("%w: check %s exited %d", project.ErrCandidateInvalid, c.ID, r.ExitCode)
		}
		delete(expected, c.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("%w: declared check observations incomplete", ErrObservationIncomplete)
	}
	return nil
}

// CheckID is the Orchestrator's identity of one check run.
type CheckID string

// Checks runs one allowed project check on an exact tree, in its own container
// from the check profile's image, on a disposable copy of that tree, within
// the request's container and termination limits. It knows no reviews or
// findings: run decides what a receipt establishes. A nonzero exit is a failed
// check. A startup or observation failure is reported separately, and
// cancellation never turns an unknown process into a pass.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}

// CheckLimits are the container limits of one check run.
type CheckLimits struct {
	MilliCPUs   int64
	MemoryBytes int64
	Processes   int64
}

// CheckRequest is one check with every input fixed. Executable is an absolute
// path inside the image. Args are the profile's argument prefix followed by the
// project arguments, passed literally and never through a shell. Env holds
// unique KEY=VALUE entries from the profile's fixed environment policy.
// Directory is a boundary inside the tree.
type CheckRequest struct {
	ID             CheckID
	Tree           snapshot.Tree
	Directories    []snapshot.Path
	ImageID        string
	Executable     string
	Args           []string
	Env            []string
	Directory      snapshot.DirectoryBoundary
	Limits         CheckLimits
	Timeout        time.Duration
	MaxOutputBytes int64
}

// Valid reports whether the request is complete and bounded.
func (r CheckRequest) Valid() bool {
	if !identifier(string(r.ID)) || !imageIdentity(r.ImageID) || !identifier(r.Executable) || !strings.HasPrefix(r.Executable, "/") || r.Directory.String() == "" {
		return false
	}
	if r.Limits.MilliCPUs <= 0 || r.Limits.MemoryBytes <= 0 || r.Limits.Processes <= 0 || r.Timeout <= 0 || r.MaxOutputBytes <= 0 || r.MaxOutputBytes > maxRecordBytes {
		return false
	}
	dirs := map[string]bool{}
	for _, d := range r.Directories {
		if d.String() == "" || dirs[d.String()] {
			return false
		}
		if _, exists := r.Tree.Lookup(d); exists {
			return false
		}
		dirs[d.String()] = true
	}
	keys := map[string]bool{}
	for _, entry := range r.Env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" || keys[key] || !text(entry) {
			return false
		}
		keys[key] = true
	}
	return true
}

// CheckReceipt is what the runtime observed. ImageID is the image that ran and
// Tree the identity of the copy it ran on. Output is the combined output up to
// the request's limit; Truncated marks a cut.
type CheckReceipt struct {
	ID           CheckID
	ImageID      string
	Tree         snapshot.TreeID
	Directories  []string
	Started      bool
	Exited       bool
	ExitCode     int
	TimedOut     bool
	Output       []byte
	Truncated    bool
	Uncertain    bool
	CopiesClosed bool
	Reason       string
}

// Passed reports whether the receipt establishes a passing run of exactly this
// request: an observed zero exit that neither timed out nor is uncertain, for
// the same check, image and tree.
func (r CheckReceipt) Passed(req CheckRequest) bool {
	return r.ID == req.ID && r.ImageID == req.ImageID && r.Tree == req.Tree.Digest() &&
		r.Started && r.Exited && r.ExitCode == 0 && !r.TimedOut && !r.Uncertain && r.CopiesClosed && sameDirectories(r.Directories, req.Directories)
}

func sameDirectories(observed []string, required []snapshot.Path) bool {
	if len(observed) != len(required) {
		return false
	}
	for i, p := range required {
		if observed[i] != p.String() {
			return false
		}
	}
	return true
}
