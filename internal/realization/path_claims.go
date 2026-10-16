package realization

import (
	"errors"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInvalidPathClaim = errors.New("invalid path claim request")

// ClaimID is an opaque Host-assigned key, not a credential or runtime identity.
type ClaimID string

// PathRequest fixes the proposed owner, exact new file and declared form.
type PathRequest struct {
	id    ClaimID
	owner Owner
	path  snapshot.Path
	form  Form
}

func NewPathRequest(id ClaimID, owner Owner, path snapshot.Path, form Form) (PathRequest, error) {
	if strings.TrimSpace(string(id)) == "" || !owner.valid() || path.String() == "" {
		return PathRequest{}, ErrInvalidPathClaim
	}
	switch form {
	case Code, Test, Doc, Build, Config:
	default:
		return PathRequest{}, ErrInvalidPathClaim
	}
	return PathRequest{id, owner, path, form}, nil
}
func (r PathRequest) ID() ClaimID         { return r.id }
func (r PathRequest) Owner() Owner        { return r.owner }
func (r PathRequest) Path() snapshot.Path { return r.path }
func (r PathRequest) Form() Form          { return r.form }

// PathClaim records only structural reservation facts and their exact basis.
// It is not a live capability. The Host owns actual admission, task/run bindings,
// ordering, durable records and release/transfer; this package implements none.
type PathClaim struct {
	request PathRequest
	basis   *pathBasis
}

func (c PathClaim) Request() PathRequest { return c.request }

// PathClaims is a fixed set supplied by the Host. Zero is empty. Reservation
// returns a new set without changing the old one; concurrent callers must still
// be serialized by the Host. It is not a journal or a second lifecycle/ledger.
type PathClaims struct{ entries []PathClaim }

func (c PathClaims) Entries() []PathClaim { return slices.Clone(c.entries) }

type PathStatus uint8

const (
	PathIncomplete PathStatus = iota
	PathRefused
	PathAdmissible
)

// PathDecision describes conditional admissibility under supplied facts. Even
// PathAdmissible cannot authorize an effect on an actual workspace by itself.
type PathDecision struct {
	status   PathStatus
	findings []Finding
	claim    PathClaim
	repeated bool
}

func (d PathDecision) Status() PathStatus  { return d.status }
func (d PathDecision) Findings() []Finding { return slices.Clone(d.findings) }
func (d PathDecision) Repeated() bool      { return d.repeated }
func (d PathDecision) Claim() (PathClaim, bool) {
	return d.claim, d.status == PathAdmissible && d.claim.request.id != ""
}

func refused(code string, owner Owner, path snapshot.Path, message string) PathDecision {
	return PathDecision{status: PathRefused, findings: []Finding{{Code: code, Path: path, Owner: owner, Message: message}}}
}

func checkPath(facts PathFacts, owner Owner, path snapshot.Path) PathDecision {
	if facts.basis == nil {
		return PathDecision{status: PathIncomplete, findings: []Finding{{Code: "claim.facts_unavailable", Message: "Complete fixed claim facts are required."}}}
	}
	if path.String() == "" {
		return refused("claim.invalid_request", owner, path, "An exact file path is required.")
	}
	state, _ := snapshot.NewPath(".markitect/state")
	if pathsOverlap(state, path) {
		return refused("claim.protected", owner, path, "Operational state is outside artifact authority.")
	}
	for _, p := range facts.basis.protections {
		if p.covers(path) {
			return refused("claim.protected", owner, path, p.why)
		}
	}
	if !owner.valid() {
		return refused("claim.invalid_request", owner, path, "An explicit owner is required.")
	}
	if !owner.root && !facts.basis.owners[owner.packageName] {
		return refused("claim.unknown_owner", owner, path, "The owner is not a current local Package; any takeover must be explicit Host-supplied ownership facts.")
	}
	return PathDecision{status: PathAdmissible}
}

// CheckOwnedPath checks the concrete Registry owner and protection of one path.
// It does not require Accounted or resolve the artifact's Definition basis: a
// dangling/orphaned artifact may need deletion or explicit rejustification.
// A disappeared owner is never silently replaced by an ancestor or root. The
// supplied Registry must already reflect any recorded Host takeover decision.
// Creation areas cannot grant or revoke existing ownership.
func CheckOwnedPath(facts PathFacts, owner Owner, path snapshot.Path) PathDecision {
	decision := checkPath(facts, owner, path)
	if decision.status != PathAdmissible {
		return decision
	}
	entry, found := facts.basis.registry.Lookup(path)
	if !found {
		return refused("claim.not_registered", owner, path, "The path has no existing registration; unexplained files require intake.")
	}
	if entry.owner != owner {
		return refused("claim.not_owner", owner, path, "The exact existing file belongs to another owner.")
	}
	for _, entry := range facts.basis.tree.Entries() {
		if entry.Path() != path && pathsOverlap(entry.Path(), path) {
			return refused("claim.file_directory_conflict", owner, path, "Observed files conflict with the owned exact-file path.")
		}
	}
	return decision
}

// ReserveNewPath checks a new exact file against current facts and existing
// claims. Repeated IDs must have identical request and basis and are rechecked
// against current protections/ownership/claims before being recognized. A later
// reservation never excuses an already-present unregistered file.
func ReserveNewPath(facts PathFacts, claims PathClaims, request PathRequest) (PathClaims, PathDecision) {
	if request.id == "" {
		return claims, refused("claim.invalid_request", request.owner, request.path, "An explicit Host claim key is required.")
	}
	decision := checkPath(facts, request.owner, request.path)
	if decision.status != PathAdmissible {
		return claims, decision
	}
	var repeated *PathClaim
	for i := range claims.entries {
		claim := &claims.entries[i]
		if claim.request.id == request.id {
			if claim.request != request {
				return claims, refused("claim.id_conflict", request.owner, request.path, "The claim key was already used for different request facts.")
			}
			if !claim.basis.same(facts.basis) {
				return claims, refused("claim.basis_changed", request.owner, request.path, "The original claim basis differs; no renewed authority is implied.")
			}
			repeated = claim
		} else if pathsOverlap(claim.request.path, request.path) {
			return claims, refused("claim.exclusive_conflict", request.owner, request.path, "Another exact-file claim conflicts with this path.")
		}
	}
	for _, entry := range facts.basis.registry.entries {
		if pathsOverlap(entry.artifact.path, request.path) {
			return claims, refused("claim.registered_conflict", request.owner, request.path, "Existing registration cannot be acquired through a new-path claim.")
		}
	}
	for _, entry := range facts.basis.tree.Entries() {
		if pathsOverlap(entry.Path(), request.path) {
			return claims, refused("claim.intake_required", request.owner, request.path, "The current observed tree already occupies this path or an ancestor/descendant; intake or repair is required.")
		}
	}
	inside := false
	for _, area := range facts.basis.areas {
		if area.owner == request.owner && area.directory.ContainsDescendant(request.path) {
			inside = true
			break
		}
	}
	if !inside {
		return claims, refused("claim.outside_creation_area", request.owner, request.path, "The new file is not a strict descendant of an explicit creation area for this owner.")
	}
	if repeated != nil {
		return claims, PathDecision{status: PathAdmissible, claim: *repeated, repeated: true}
	}
	claim := PathClaim{request: request, basis: facts.basis}
	entries := slices.Clone(claims.entries)
	entries = append(entries, claim)
	slices.SortFunc(entries, func(a, b PathClaim) int { return strings.Compare(a.request.path.String(), b.request.path.String()) })
	return PathClaims{entries: entries}, PathDecision{status: PathAdmissible, claim: claim}
}
