package agentexec

// Lifecycle is transport-observed operational evidence. It is never accepted
// from Response/reportJson. Nil in Receipt means the adapter did not observe
// this protocol; it does not mean zero sessions or zero helper requests.
type Lifecycle struct {
	Provider  string           `json:"provider"`
	SessionID string           `json:"sessionId,omitempty"`
	TurnID    string           `json:"turnId,omitempty"`
	State     string           `json:"state"` // completed, interrupted, failed, running, unknown
	Requested SessionSettings  `json:"requested"`
	Effective *SessionSettings `json:"effective,omitempty"`
	// StartRequests includes the root request and every observed child request,
	// including failures. A request is not evidence that a child actually started.
	StartRequests  []RoleStartRequest `json:"startRequests"`
	Accounting     string             `json:"accounting"` // complete, partial, unavailable
	EventLogDigest string             `json:"eventLogDigest,omitempty"`
}

// SessionSettings keeps requested and observed values separate. Empty observed
// fields mean unavailable; policy strings record settings and grant no authority.
type SessionSettings struct {
	Model             string `json:"model,omitempty"`
	ReasoningEffort   string `json:"reasoningEffort,omitempty"`
	PermissionProfile string `json:"permissionProfile,omitempty"`
	CWD               string `json:"cwd,omitempty"`
	InstructionDigest string `json:"instructionDigest,omitempty"`
}

type RoleStartRequest struct {
	RequestID       string `json:"requestId"`
	ParentSessionID string `json:"parentSessionId,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
	Role            string `json:"role"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
	State           string `json:"state"` // requested, started, failed, interrupted, completed, unknown
}
