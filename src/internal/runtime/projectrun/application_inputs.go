package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

// These concrete inputs cross the application/runtime seam in memory only.
// Their identities are persisted by the existing plan and receipt records.
type DeliveryScope struct {
	ExplorationID string
	ScopeID       string
	Name          string
	Goal          string
	Operation     string
	ManagerIDs    []string
}

type ExplorationReadiness struct {
	Ready    bool
	Digest   string
	Blockers []string
}

type ExplorationAdmission struct {
	Binding           projectvalue.Binding
	ExplorationDigest string
	ReadinessDigest   string
}

type DeliveredResolution struct {
	Plan            PlanRecord
	CandidateID     string
	CandidateDigest string
	Files           map[string]File
	Verification    VerifyReport
	Apply           ApplyReport
}

type ExplorationCompletion struct {
	Plan            PlanRecord
	CandidateID     string
	CandidateDigest string
	Verification    VerifyReport
	Apply           ApplyReport
}
