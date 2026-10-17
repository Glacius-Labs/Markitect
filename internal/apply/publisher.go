package apply

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// PublicationID is the Host identity of one publication.
type PublicationID string

// Publisher writes a verified result into the owner's working tree. It knows no
// reviews or evidence: apply decides what may be published. Preflight rechecks
// the captured source against its guard and the live controller, and returns a
// permit bound to that request. Publish revalidates guard and controller at the
// write boundary, writes exactly the requested changes the way Git checks files
// out in that repository, and reads every written path back. It never commits,
// merges or pushes. Multi-file publication is not atomic: a partial write stays
// visible and is reported. Inspect observes an earlier publication after an
// interruption. Receipts are returned with known observations also on error.
type Publisher interface {
	Preflight(context.Context, PublicationRequest) (PublicationPermit, error)
	Publish(context.Context, PublicationPermit) (PublicationReceipt, error)
	Inspect(context.Context, PublicationID) (PublicationReceipt, error)
}

// PublicationRequest names one publication. Changes are the exact artifact and
// registry changes against the captured tree; apply limits them to paths the
// artifact-only target permits. The run and controller bind the request, so a
// finished run or a lost controller cannot publish.
type PublicationRequest struct {
	ID         PublicationID
	Run        string
	Repository string
	Control    ControlBinding
	Guard      PublicationGuard
	Changes    []snapshot.Change
}

// PublicationPermit binds one preflighted request. Token is issued by the
// adapter and opaque to apply. A permit is void once its run has finished.
type PublicationPermit struct {
	Request PublicationRequest
	Token   string
}

type PublicationOutcome string

const (
	PublicationApplied  PublicationOutcome = "applied"
	PublicationNoChange PublicationOutcome = "no-change"
	PublicationStale    PublicationOutcome = "stale"
	PublicationPartial  PublicationOutcome = "partial"
	PublicationUnknown  PublicationOutcome = "unknown"
	PublicationFailed   PublicationOutcome = "failed"
)

// PublicationReceipt is what the adapter observed. Written lists the paths
// whose read-back matched the requested change; on any outcome but applied or
// no-change, Reason says what is known.
type PublicationReceipt struct {
	ID      PublicationID
	Outcome PublicationOutcome
	Written []snapshot.Path
	Reason  string
}
