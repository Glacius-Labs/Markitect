package run

import (
	"context"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

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
	ID        CheckID
	ImageID   string
	Tree      snapshot.TreeID
	Started   bool
	Exited    bool
	ExitCode  int
	TimedOut  bool
	Output    []byte
	Truncated bool
	Uncertain bool
	Reason    string
}

// Passed reports whether the receipt establishes a passing run of exactly this
// request: an observed zero exit that neither timed out nor is uncertain, for
// the same check, image and tree.
func (r CheckReceipt) Passed(req CheckRequest) bool {
	return r.ID == req.ID && r.ImageID == req.ImageID && r.Tree == req.Tree.Digest() &&
		r.Started && r.Exited && r.ExitCode == 0 && !r.TimedOut && !r.Uncertain
}
