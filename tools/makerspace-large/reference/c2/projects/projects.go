// Package projects holds the projects of members: an owner, collaborators and
// the reservations and purchases attached to them, whose cost can be read
// together, and the operations on them.
package projects

import (
	"sort"

	"makerspace/bookings"
	"makerspace/materials"
	"makerspace/members"
	"makerspace/policies"
)

type project struct {
	id, ownerID, title string
	collaborators      []string // ascending
	closed             bool
	reservationIDs     []string
	purchaseIDs        []string
}

func (p *project) status() string {
	if p.closed {
		return "closed"
	}
	return "open"
}

// isMember reports whether the member is the owner or a collaborator.
func (p *project) isMember(memberID string) bool {
	if memberID == p.ownerID {
		return true
	}
	i := sort.SearchStrings(p.collaborators, memberID)
	return i < len(p.collaborators) && p.collaborators[i] == memberID
}

// Module is the projects module: it owns every project.
type Module struct {
	members   *members.Module
	bookings  *bookings.Module
	materials *materials.Module
	byID      map[string]*project
	// attached to a project: a reservation or a purchase belongs to at most one.
	reservationOf map[string]string
	purchaseOf    map[string]string
}

// New returns the module in the initial state.
func New(mem *members.Module, bk *bookings.Module, mat *materials.Module) *Module {
	return &Module{members: mem, bookings: bk, materials: mat, byID: map[string]*project{},
		reservationOf: map[string]string{}, purchaseOf: map[string]string{}}
}

// ProjectExists reports whether the identifier names a project.
func (m *Module) ProjectExists(id string) bool { _, ok := m.byID[id]; return ok }

// ProjectOwner is the owner of a project.
func (m *Module) ProjectOwner(id string) string { return m.byID[id].ownerID }

func projectView(p *project) policies.Rec {
	return policies.Rec{"projectId": p.id, "ownerId": p.ownerID, "title": p.title,
		"collaboratorIds": append([]string{}, p.collaborators...), "status": p.status(),
		"reservationIds": append([]string{}, p.reservationIDs...),
		"purchaseIds":    append([]string{}, p.purchaseIDs...)}
}
