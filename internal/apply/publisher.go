package apply

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrUnpublishable reports a change set that the target cannot represent, such
// as case aliases or modes the repository does not track. Preflight reports a
// source that changed since its capture with an error that wraps
// ErrCaptureChanged. Any other error is not a definite refusal; apply retries
// the publication later.
var ErrUnpublishable = errors.New("the change set cannot be published to the target")

// PublicationID is the Host identity of one publication.
type PublicationID string

// ControlCheck fences a publication to the live repository controller. A
// ControlSession satisfies it.
type ControlCheck interface {
	Check(context.Context) error
}

// Publisher writes a verified result into the owner's working tree. It knows no
// reviews, runs or evidence: apply decides what may be published, and voids the
// permit of a finished run. Preflight rechecks the captured source against its
// guard and the controller, refuses changes the target cannot represent (such
// as case aliases or modes the repository does not track), and returns a permit
// bound to that request. Publish checks guard and controller before its first
// and after its last write, writes exactly the requested changes the way Git
// checks files out in that repository, and reads every written path back in
// Git's clean form, so line-ending conversion does not count as a mismatch. It
// never commits, merges or pushes. Multi-file publication is not atomic: a
// partial write stays visible and is reported. Inspect classifies the paths of
// an earlier request, which apply journaled before Publish, after an
// interruption. Receipts are returned with known observations also on error.
type Publisher interface {
	Preflight(context.Context, PublicationRequest, ControlCheck) (PublicationPermit, error)
	Publish(context.Context, PublicationPermit, ControlCheck) (PublicationReceipt, error)
	Inspect(context.Context, PublicationRequest) (PublicationReceipt, error)
}

// PublicationRequest names one publication. Repository is the same explicit
// acquisition location the capture used. Changes are the exact artifact and
// registry changes against the captured tree; apply limits them to the paths
// the artifact-only target permits. An empty change set is valid and publishes
// nothing.
type PublicationRequest struct {
	ID         PublicationID
	Run        string
	Repository string
	Guard      PublicationGuard
	Changes    []snapshot.Change
}

// PublicationPermit binds one preflighted request. Token is issued by the
// adapter and opaque to apply.
type PublicationPermit struct {
	Request PublicationRequest
	Token   string
}

// PublicationOutcome is the observed result of a publication. Its zero value
// is invalid.
type PublicationOutcome string

const (
	PublicationApplied  PublicationOutcome = "applied"   // every change was written and read back, and the final check held
	PublicationNoChange PublicationOutcome = "no_change" // the change set was empty and the guard held
	PublicationStale    PublicationOutcome = "stale"     // the guard failed before any write
	PublicationPartial  PublicationOutcome = "partial"   // some, not all, changes were written
	PublicationUnknown  PublicationOutcome = "unknown"   // the state of some path, or the final check, is not known
	PublicationFailed   PublicationOutcome = "failed"    // a definite failure without a write
)

// PathState classifies one requested path as the adapter observed it.
type PathState string

const (
	PathAfter  PathState = "after"  // holds the requested content
	PathBefore PathState = "before" // still holds the captured content
	PathOther  PathState = "other"  // holds something else or could not be read
)

// PublishedPath is one requested path and its observed state.
type PublishedPath struct {
	Path  snapshot.Path
	State PathState
}

// PublicationReceipt is what the adapter observed. Paths classifies every
// requested path; on any outcome but applied or no_change, Reason says what is
// known. After the writes, the guard holds when every path outside the change
// set is unchanged; if the check after the last write fails, the outcome is
// unknown.
type PublicationReceipt struct {
	ID      PublicationID
	Outcome PublicationOutcome
	Paths   []PublishedPath
	Reason  string
}
