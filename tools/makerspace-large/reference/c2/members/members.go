// Package members holds the registry of users, the members with their tier
// and standing, and the operations on them.
package members

import "makerspace/policies"

// User is a user of the program.
type User struct {
	ID, Name, Role string
}

// Member is a user with role member and a membership.
type Member struct {
	User         *User
	Tier         string
	Standing     string
	RegisteredAt int
}

// Module is the members module: it owns every user and member.
type Module struct {
	users   map[string]*User
	members map[string]*Member
}

// New returns the module in the initial state: one user, the admin.
func New() *Module {
	m := &Module{users: map[string]*User{}, members: map[string]*Member{}}
	m.users["admin"] = &User{ID: "admin", Name: "Administrator", Role: "admin"}
	return m
}

// User returns the user with the identifier, or nil.
func (m *Module) User(id string) *User { return m.users[id] }

// Member returns the member with the identifier, or nil.
func (m *Module) Member(id string) *Member { return m.members[id] }

// Exists reports whether the identifier names a member. The identifier of a
// staff or admin user does not.
func (m *Module) Exists(id string) bool {
	_, ok := m.members[id]
	return ok
}

// SortedIDs lists the identifiers of all members in ascending order.
func (m *Module) SortedIDs() []string { return policies.SortedKeys(m.members) }

// InGoodStanding reports whether the member is in good standing.
func (m *Module) InGoodStanding(id string) bool {
	mem := m.members[id]
	return mem != nil && mem.Standing == "good"
}

// SetStanding sets the standing of a member to good or suspended. It writes no
// audit entry: the operation that calls it is the one that is audited.
func (m *Module) SetStanding(id, standing string) {
	if mem := m.members[id]; mem != nil {
		mem.Standing = standing
	}
}
