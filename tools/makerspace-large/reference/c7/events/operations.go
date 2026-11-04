package events

import (
	"makerspace/policies"
	"makerspace/rooms"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "createEvent", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("eventId"), policies.Text("title"), policies.IDRef("roomId", m.rooms.RoomExists),
				policies.Stamp("start"), policies.Stamp("end"), policies.Integer("capacity", 1, 1000),
				policies.Integer("priceCents", 0, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id, room := c.Str("eventId"), c.Str("roomId")
				start, end := c.Num("start"), c.Num("end")
				if m.events[id] != nil {
					return nil, "", policies.Refuse("already_exists", "An event with identifier %s already exists.", id)
				}
				if end <= start {
					return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.At {
					return nil, "", policies.Refuse("starts_in_past", "The event starts before the request time.")
				}
				if c.Num("capacity") > m.rooms.Capacity(room) {
					return nil, "", policies.Refuse("capacity_too_large", "The capacity is above the capacity of the room.")
				}
				if !m.rooms.Free(room, start, end) {
					return nil, "", policies.Refuse("room_taken", "The room is not free for the interval.")
				}
				e := &Event{ID: id, Title: c.Str("title"), RoomID: room, Start: start, End: end,
					Capacity: c.Num("capacity"), Price: c.Num("priceCents")}
				e.holdID = m.rooms.HoldFor(room, start, end, "event "+id, rooms.OriginEvent).ID
				m.events[id] = e
				return policies.Rec{"event": eventView(e)}, id, nil
			}},
		{Name: "sellTicket", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.IDRef("eventId", m.EventExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				e := m.events[c.Str("eventId")]
				mid := c.Str("memberId")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				if e.Cancelled || e.Start <= c.At {
					return nil, "", policies.Refuse("event_closed", "The event is cancelled or has started.")
				}
				for _, t := range m.list {
					if t.EventID == e.ID && t.MemberID == mid && !t.Cancelled {
						return nil, "", policies.Refuse("already_ticketed", "The member holds a valid ticket.")
					}
				}
				if e.Sold >= e.Capacity {
					return nil, "", policies.Refuse("event_sold_out", "All places are taken.")
				}
				t := &Ticket{ID: m.ids.Next("tkt"), EventID: e.ID, MemberID: mid}
				m.tickets[t.ID] = t
				m.list = append(m.list, t)
				e.Sold++
				m.billing.AddCharge(mid, "event", t.ID, 1, e.Price, 0, c.At)
				return policies.Rec{"ticket": ticketView(t), "event": eventView(e)}, t.ID, nil
			}},
		{Name: "cancelTicket", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("ticketId", m.TicketExists, m.TicketOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				t := m.tickets[c.Str("ticketId")]
				e := m.events[t.EventID]
				if t.Cancelled {
					return nil, "", policies.Refuse("invalid_state", "The ticket is cancelled already.")
				}
				if e.Start-c.At < ticketLeadMinutes {
					return nil, "", policies.Refuse("too_late", "The event starts in less than %d minutes.", ticketLeadMinutes)
				}
				m.cancelTicket(t, c.At)
				return policies.Rec{"ticket": ticketView(t), "event": eventView(e)}, t.ID, nil
			}},
		{Name: "cancelEvent", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("eventId", m.EventExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				e := m.events[c.Str("eventId")]
				if e.Cancelled || e.Start <= c.At {
					return nil, "", policies.Refuse("invalid_state", "The event is cancelled or has started.")
				}
				e.Cancelled = true
				m.rooms.ReleaseHold(e.holdID)
				out := []policies.Rec{}
				for _, t := range m.list {
					if t.EventID == e.ID && !t.Cancelled {
						m.cancelTicket(t, c.At)
						out = append(out, ticketView(t))
					}
				}
				return policies.Rec{"event": eventView(e), "tickets": out}, e.ID, nil
			}},
		{Name: "listEventTickets", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("eventId", m.EventExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("eventId")
				out := []policies.Rec{}
				for _, t := range m.list {
					if t.EventID == id {
						out = append(out, ticketView(t))
					}
				}
				return policies.Rec{"eventId": id, "tickets": out}, "", nil
			}},
	}
}
