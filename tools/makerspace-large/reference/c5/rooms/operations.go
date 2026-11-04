package rooms

import (
	"sort"

	"makerspace/policies"
)

func roomView(r *room) policies.Rec {
	return policies.Rec{"roomId": r.id, "name": r.name, "capacity": r.capacity}
}

// HoldView is the hold as it appears in results.
func HoldView(h *Hold) policies.Rec {
	status := "held"
	if h.Released {
		status = "released"
	}
	return policies.Rec{"holdId": h.ID, "roomId": h.RoomID, "start": policies.FormatTime(h.Start),
		"end": policies.FormatTime(h.End), "purpose": h.Purpose, "status": status}
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addRoom", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.ID("roomId"), policies.Text("name"), policies.Integer("capacity", 1, 1000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("roomId")
				if m.rooms[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A room with identifier %s already exists.", id)
				}
				r := &room{id: id, name: c.Str("name"), capacity: c.Num("capacity")}
				m.rooms[id] = r
				return policies.Rec{"room": roomView(r)}, id, nil
			}},
		{Name: "holdRoom", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("roomId", m.RoomExists), policies.Stamp("start"), policies.Stamp("end"),
				policies.Text("purpose")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				rid, start, end := c.Str("roomId"), c.Num("start"), c.Num("end")
				if end <= start {
					return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.At {
					return nil, "", policies.Refuse("starts_in_past", "The hold starts before the request time.")
				}
				if !m.Free(rid, start, end) {
					return nil, "", policies.Refuse("room_taken", "A held hold of the room overlaps the interval.")
				}
				h := m.HoldFor(rid, start, end, c.Str("purpose"), OriginDirect)
				return policies.Rec{"hold": HoldView(h)}, h.ID, nil
			}},
		{Name: "releaseRoomHold", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("holdId", m.HoldExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				h := m.holds[c.Str("holdId")]
				if h.Released {
					return nil, "", policies.Refuse("invalid_state", "The hold is already released.")
				}
				if h.Origin != OriginDirect {
					return nil, "", policies.Refuse("hold_not_direct", "The hold was made for a class or an event.")
				}
				h.Released = true
				return policies.Rec{"hold": HoldView(h)}, h.ID, nil
			}},
		{Name: "getRoomSchedule", Perms: "aaa",
			Args: []policies.ArgSpec{policies.IDRef("roomId", m.RoomExists), policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				rid := c.Str("roomId")
				from := c.Num("date") * policies.MinPerDay
				var list []*Hold
				for _, h := range m.holdList {
					if h.RoomID == rid && !h.Released && policies.Overlaps(h.Start, h.End, from, from+policies.MinPerDay) {
						list = append(list, h)
					}
				}
				sort.SliceStable(list, func(i, j int) bool { return list[i].Start < list[j].Start })
				out := make([]policies.Rec, 0, len(list))
				for _, h := range list {
					out = append(out, HoldView(h))
				}
				return policies.Rec{"roomId": rid, "date": c.Str("date"), "holds": out}, "", nil
			}},
	}
}
