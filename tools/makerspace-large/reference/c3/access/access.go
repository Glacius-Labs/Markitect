// Package access holds the door badges of members and the door events, and
// the operations on them.
package access

import (
	"makerspace/billing"
	"makerspace/bookings"
	"makerspace/guests"
	"makerspace/members"
	"makerspace/policies"
)

// Badge is the door badge of a member.
type Badge struct {
	ID, MemberID string
	Revoked      bool
	Inside       bool
}

// Status is active or revoked.
func (b *Badge) Status() string {
	if b.Revoked {
		return "revoked"
	}
	return "active"
}

// DoorEvent is one passage at the door.
type DoorEvent struct {
	ID, Kind, BadgeID string
	GuestPassID       *string
	At                int
}

// Module is the access module: it owns every badge and door event.
type Module struct {
	members  *members.Module
	billing  *billing.Module
	guests   *guests.Module
	bookings *bookings.Module
	ids      policies.IDs
	byID     map[string]*Badge
	badges   []*Badge
	events   []*DoorEvent
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module, gst *guests.Module, bk *bookings.Module) *Module {
	return &Module{members: mem, billing: bill, guests: gst, bookings: bk, byID: map[string]*Badge{}}
}

// BadgeExists reports whether the identifier names a badge.
func (m *Module) BadgeExists(id string) bool { _, ok := m.byID[id]; return ok }

// BadgeOwner is the member a badge belongs to.
func (m *Module) BadgeOwner(id string) string { return m.byID[id].MemberID }

func (m *Module) activeBadge(memberID string) *Badge {
	for _, b := range m.badges {
		if b.MemberID == memberID && !b.Revoked {
			return b
		}
	}
	return nil
}

// BadgeOutside reports whether the member has an active badge that is not
// inside (premises gate). A member with no active badge is not gated.
func (m *Module) BadgeOutside(memberID string) bool {
	b := m.activeBadge(memberID)
	return b != nil && !b.Inside
}

func (m *Module) addEvent(kind, badgeID string, guestPassID *string, at int) *DoorEvent {
	e := &DoorEvent{ID: m.ids.Next("dor"), Kind: kind, BadgeID: badgeID, GuestPassID: guestPassID, At: at}
	m.events = append(m.events, e)
	return e
}

func badgeView(b *Badge) policies.Rec {
	return policies.Rec{"badgeId": b.ID, "memberId": b.MemberID, "status": b.Status(), "inside": b.Inside}
}

func eventView(e *DoorEvent) policies.Rec {
	var pass any
	if e.GuestPassID != nil {
		pass = *e.GuestPassID
	}
	return policies.Rec{"eventId": e.ID, "kind": e.Kind, "badgeId": e.BadgeID, "guestPassId": pass,
		"at": policies.FormatTime(e.At)}
}
