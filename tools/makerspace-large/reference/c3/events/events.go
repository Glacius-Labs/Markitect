// Package events holds the one-off events with their tickets, and the
// operations on them.
package events

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/rooms"
)

// ticketLeadMinutes is how long before the start of an event a ticket can be
// cancelled.
const ticketLeadMinutes = 1440

// Event is a one-off event in a room.
type Event struct {
	ID, Title, RoomID string
	Start, End        int
	Capacity, Price   int
	Cancelled         bool
	Sold              int
	holdID            string
}

// Status is open or cancelled.
func (e *Event) Status() string {
	if e.Cancelled {
		return "cancelled"
	}
	return "open"
}

// Ticket is the ticket of one member for one event.
type Ticket struct {
	ID, EventID, MemberID string
	Cancelled             bool
}

// Status is valid or cancelled.
func (t *Ticket) Status() string {
	if t.Cancelled {
		return "cancelled"
	}
	return "valid"
}

// Module is the events module: it owns every event and ticket.
type Module struct {
	members *members.Module
	rooms   *rooms.Module
	billing *billing.Module
	ids     policies.IDs
	events  map[string]*Event
	tickets map[string]*Ticket
	list    []*Ticket
}

// New returns the module in the initial state.
func New(mem *members.Module, rm *rooms.Module, bill *billing.Module) *Module {
	return &Module{members: mem, rooms: rm, billing: bill, events: map[string]*Event{}, tickets: map[string]*Ticket{}}
}

// EventExists reports whether the identifier names an event.
func (m *Module) EventExists(id string) bool { _, ok := m.events[id]; return ok }

// TicketExists reports whether the identifier names a ticket.
func (m *Module) TicketExists(id string) bool { _, ok := m.tickets[id]; return ok }

// TicketOwner is the member a ticket belongs to.
func (m *Module) TicketOwner(id string) string { return m.tickets[id].MemberID }

// cancelTicket cancels a valid ticket: the event gives the place back and the
// member gets the price as credit, posted against revenueRefunds.
func (m *Module) cancelTicket(t *Ticket, at int) {
	e := m.events[t.EventID]
	t.Cancelled = true
	e.Sold--
	m.billing.AddCredit(t.MemberID, e.Price, "revenueRefunds", t.ID, at)
}

func eventView(e *Event) policies.Rec {
	return policies.Rec{"eventId": e.ID, "title": e.Title, "roomId": e.RoomID, "start": policies.FormatTime(e.Start),
		"end": policies.FormatTime(e.End), "capacity": e.Capacity, "priceCents": e.Price, "status": e.Status(),
		"sold": e.Sold}
}

func ticketView(t *Ticket) policies.Rec {
	return policies.Rec{"ticketId": t.ID, "eventId": t.EventID, "memberId": t.MemberID, "status": t.Status()}
}
