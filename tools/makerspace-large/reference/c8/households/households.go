// Package households holds the households of members who share a payer, and
// the operations on them.
package households

import (
	"sort"

	"makerspace/members"
	"makerspace/policies"
)

// maxMembers is the size limit of a household.
const maxMembers = 6

// Household is a group of members with one payer.
type Household struct {
	ID, Name, PayerID string
	MemberIDs         []string // ascending
}

// Module is the households module: it owns every household.
type Module struct {
	members *members.Module
	byID    map[string]*Household
	of      map[string]*Household // household of each member
}

// New returns the module in the initial state.
func New(mem *members.Module) *Module {
	return &Module{members: mem, byID: map[string]*Household{}, of: map[string]*Household{}}
}

// HouseholdExists reports whether the identifier names a household.
func (m *Module) HouseholdExists(id string) bool { _, ok := m.byID[id]; return ok }

// OfMember gives the household the member belongs to: its payer and its
// members in ascending order. The second result is false when the member
// belongs to none. The member count is the length of the list.
func (m *Module) OfMember(memberID string) (payerID string, memberIDs []string, ok bool) {
	h := m.of[memberID]
	if h == nil {
		return "", nil, false
	}
	return h.PayerID, append([]string(nil), h.MemberIDs...), true
}

func (h *Household) add(memberID string) {
	h.MemberIDs = append(h.MemberIDs, memberID)
	sort.Strings(h.MemberIDs)
}

func (h *Household) remove(memberID string) {
	for i, id := range h.MemberIDs {
		if id == memberID {
			h.MemberIDs = append(h.MemberIDs[:i:i], h.MemberIDs[i+1:]...)
			return
		}
	}
}

func householdView(h *Household) policies.Rec {
	return policies.Rec{"householdId": h.ID, "name": h.Name, "payerId": h.PayerID,
		"memberIds": append([]string{}, h.MemberIDs...)}
}

// MemberIDsOf lists the members of the household in ascending order.
func (m *Module) MemberIDsOf(householdID string) []string {
	return append([]string(nil), m.byID[householdID].MemberIDs...)
}
