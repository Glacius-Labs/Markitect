// Package rooms holds the rooms and their holds, and the operations on them.
package rooms

import "makerspace/policies"

// Origins of a hold. A hold made for a class or an event is released only
// together with that class or event.
const (
	OriginDirect = "direct"
	OriginClass  = "class"
	OriginEvent  = "event"
)

type room struct {
	id, name string
	capacity int
}

// Hold keeps a room for [Start,End) while Released is false.
type Hold struct {
	ID, RoomID, Purpose, Origin string
	Start, End                  int
	Released                    bool
}

// Module is the rooms module: it owns every room and hold.
type Module struct {
	ids      policies.IDs
	rooms    map[string]*room
	holds    map[string]*Hold
	holdList []*Hold
}

// New returns the module in the initial state.
func New() *Module {
	return &Module{rooms: map[string]*room{}, holds: map[string]*Hold{}}
}

// RoomExists reports whether the identifier names a room.
func (m *Module) RoomExists(id string) bool { _, ok := m.rooms[id]; return ok }

// HoldExists reports whether the identifier names a hold.
func (m *Module) HoldExists(id string) bool { _, ok := m.holds[id]; return ok }

// Capacity is the capacity of the room.
func (m *Module) Capacity(roomID string) int { return m.rooms[roomID].capacity }

// Free reports whether none of the room's held holds overlaps [start,end).
func (m *Module) Free(roomID string, start, end int) bool {
	for _, h := range m.holdList {
		if h.RoomID == roomID && !h.Released && policies.Overlaps(start, end, h.Start, h.End) {
			return false
		}
	}
	return true
}

// HoldFor holds the room for [start,end) on behalf of another module. The
// caller has checked that the room is free for the interval; origin is
// OriginClass or OriginEvent. It returns the new hold.
func (m *Module) HoldFor(roomID string, start, end int, purpose, origin string) *Hold {
	h := &Hold{ID: m.ids.Next("hld"), RoomID: roomID, Start: start, End: end, Purpose: purpose, Origin: origin}
	m.holds[h.ID] = h
	m.holdList = append(m.holdList, h)
	return h
}

// ReleaseHold releases a hold that was made for another module.
func (m *Module) ReleaseHold(holdID string) { m.holds[holdID].Released = true }
