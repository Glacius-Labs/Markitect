// Package notices holds the notices recorded for members, and the operations
// on them. The program only records notices; it sends nothing.
package notices

import (
	"makerspace/members"
	"makerspace/policies"
)

// Notice is a message recorded for a member.
type Notice struct {
	ID, MemberID, Kind, Subject string
	Ref                         *string
	At                          int
	AcknowledgedAt              *int
}

// Module is the notices module: it owns every notice.
type Module struct {
	members *members.Module
	ids     policies.IDs
	byID    map[string]*Notice
	list    []*Notice
}

// New returns the module in the initial state.
func New(mem *members.Module) *Module {
	return &Module{members: mem, byID: map[string]*Notice{}}
}

// NoticeExists reports whether the identifier names a notice.
func (m *Module) NoticeExists(id string) bool { _, ok := m.byID[id]; return ok }

// NoticeOwner is the member a notice belongs to.
func (m *Module) NoticeOwner(id string) string { return m.byID[id].MemberID }

func (m *Module) add(memberID, kind, subject string, ref *string, at int) *Notice {
	n := &Notice{ID: m.ids.Next("ntc"), MemberID: memberID, Kind: kind, Subject: subject, Ref: ref, At: at}
	m.byID[n.ID] = n
	m.list = append(m.list, n)
	return n
}

// Post records a notice that an operation of another module posts for the
// member: kind is dunning, suspension, waitlist or cancellation, and ref the
// invoice, course or reservation identifier. The subject is the one of the
// kind. Standing is not checked and no audit entry is written.
func (m *Module) Post(memberID, kind, ref string, at int) {
	var subject string
	switch kind {
	case "dunning":
		subject = "Invoice " + ref + " is overdue"
	case "suspension":
		subject = "Membership suspended: invoice " + ref + " is overdue"
	case "waitlist":
		subject = "Enrolled from the waitlist of course " + ref
	case "cancellation":
		subject = "Reservation " + ref + " was cancelled"
	}
	m.add(memberID, kind, subject, &ref, at)
}
