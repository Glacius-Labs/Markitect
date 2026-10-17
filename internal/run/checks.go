package run

import (
	"context"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CheckID is the Host identity of one check run.
type CheckID string

// Checks runs one allowed project check on an exact tree, in its own container
// from the check profile's image, on a disposable copy of that tree. It knows
// no reviews or findings: run decides what a receipt establishes. A nonzero
// exit is a failed check. A startup or observation failure is reported
// separately, and cancellation never turns an unknown process into a pass.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}

// CheckRequest is one check with every input fixed. Executable is the path
// inside the image; Args are the profile's argument prefix followed by the
// project arguments, passed literally and never through a shell. Directory is
// relative to the tree root; empty means the root.
type CheckRequest struct {
	ID             CheckID
	Tree           snapshot.Tree
	ImageID        string
	Executable     string
	Args           []string
	Env            []string
	Directory      string
	Timeout        time.Duration
	MaxOutputBytes int64
}

// Valid reports whether the request is complete and bounded.
func (r CheckRequest) Valid() bool {
	return identifier(string(r.ID)) && imageIdentity(r.ImageID) && identifier(r.Executable) &&
		r.Timeout > 0 && r.MaxOutputBytes > 0 && r.MaxOutputBytes <= maxRecordBytes
}

// CheckReceipt is what the runtime observed. ImageID is the image that ran.
// Output is the combined output up to the request's limit; Truncated marks a
// cut. Exited with ExitCode zero is a pass only when the run is not Uncertain.
type CheckReceipt struct {
	ID        CheckID
	ImageID   string
	Started   bool
	Exited    bool
	ExitCode  int
	TimedOut  bool
	Output    []byte
	Truncated bool
	Uncertain bool
	Reason    string
}

// Passed reports whether the receipt establishes a passing check.
func (r CheckReceipt) Passed() bool {
	return r.Started && r.Exited && r.ExitCode == 0 && !r.TimedOut && !r.Uncertain
}
