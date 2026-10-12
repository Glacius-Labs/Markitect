package projectbriefing

// Dismissal hides an event from one Manager's notifications.
type Dismissal struct {
	EventID   string `json:"eventId"`
	ManagerID string `json:"managerId"`
}

// Store is the durable briefing store value. Display-only working-tree hints
// are carried separately by ResolutionView.
type Store struct {
	APIVersion  string         `json:"apiVersion"`
	History     *HistoryCursor `json:"history,omitempty"`
	Briefings   []Bundle       `json:"briefings"`
	Dismissals  []Dismissal    `json:"dismissals"`
	Resolutions []Resolution   `json:"resolutions,omitempty"`
}

// VerifiedResolutionEvidence binds a completed full verification and applied
// candidate to its durable run, plan, checks, and complete Manager coverage.
type VerifiedResolutionEvidence struct {
	EventIDs           []string `json:"eventIds"`
	RunID              string   `json:"runId"`
	PlanDigest         string   `json:"planDigest"`
	CandidateID        string   `json:"candidateId"`
	CandidateDigest    string   `json:"candidateDigest"`
	VerificationDigest string   `json:"verificationDigest"`
	ApplyDigest        string   `json:"applyDigest"`
	FullVerifyPassed   bool     `json:"fullVerifyPassed"`
	CoveredManagerIDs  []string `json:"coveredManagerIds"`
	EvidenceRefs       []string `json:"evidenceRefs"`
	// Delivered is what Apply wrote. The resolution counts only while HEAD's
	// committed tree holds all of it.
	Delivered []DeliveredFile `json:"delivered"`
}

// DeliveredFile is one path a delivery wrote: its Git blob and mode, or its
// removal.
type DeliveredFile struct {
	Path    string `json:"path"`
	Mode    string `json:"mode,omitempty"`
	Object  string `json:"object,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}

// Resolution is an immutable status record. Dismissal and resolution remain
// separate: one changes visibility; the other cites fresh Host evidence.
type Resolution struct {
	EventID       string                     `json:"eventId"`
	ModelRevision string                     `json:"modelRevision"`
	ModelDigest   string                     `json:"modelDigest"`
	Evidence      VerifiedResolutionEvidence `json:"evidence"`
	Digest        string                     `json:"digest"`
}

type ResolutionStatus struct {
	Status     string      `json:"status"`
	Resolution *Resolution `json:"resolution,omitempty"`
}

// Status values reported by EventResolutionStatus.
const (
	StatusResolved   = "resolved"
	StatusUnresolved = "unresolved"
	// StatusDeliveredUncommitted means a delivery's result is in the working
	// tree but not yet committed. The event is not resolved.
	StatusDeliveredUncommitted = "delivered-uncommitted"
)

// HistoryCursor records the committed model history accepted by repository
// policy.
type HistoryCursor struct {
	Policy              string `json:"policy"`
	BaselineRevision    string `json:"baselineRevision"`
	BaselineModelDigest string `json:"baselineModelDigest"`
	Revision            string `json:"revision"`
	ModelDigest         string `json:"modelDigest"`
}

// EnsureReceipt is the fixed-revision result after accepted model history has
// been reconciled. Bundles are the immutable entries added during this call.
type EnsureReceipt struct {
	BaselineRevision    string   `json:"baselineRevision"`
	BaselineModelDigest string   `json:"baselineModelDigest"`
	Revision            string   `json:"revision"`
	ModelDigest         string   `json:"modelDigest"`
	StoreDigest         string   `json:"storeDigest"`
	Bundles             []Bundle `json:"bundles"`
}

type AcceptedHistory struct {
	// Receipt is what EnsureAcceptedHistory would return for the same revision.
	Receipt EnsureReceipt
	// State is the history accepted on the checked-out branch, as Read would
	// return it after EnsureAcceptedHistory; it is a view, not a store to write.
	State Store
	// PersistedDigest is the store digest State was computed from, and the
	// expected digest for writers that persist the prospective accepted state.
	PersistedDigest string
}
