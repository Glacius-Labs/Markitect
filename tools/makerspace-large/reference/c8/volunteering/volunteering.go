// Package volunteering holds the volunteer rate and the time entries members
// log, and the operations on them.
package volunteering

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
)

// monthlyCap is the most approved minutes a member can have in one month.
const monthlyCap = 1200

// Entry is time a member volunteered. Date is a day; Credit is nil until the
// entry is approved.
type Entry struct {
	ID, MemberID, Note, Status string
	Date, Minutes              int
	Credit                     *int
}

// Module is the volunteering module: it owns the rate and every entry.
type Module struct {
	members *members.Module
	billing *billing.Module
	ids     policies.IDs
	rate    int // cents per hour
	byID    map[string]*Entry
	list    []*Entry
}

// New returns the module in the initial state: a rate of 0.
func New(mem *members.Module, bill *billing.Module) *Module {
	return &Module{members: mem, billing: bill, byID: map[string]*Entry{}}
}

// EntryExists reports whether the identifier names a time entry.
func (m *Module) EntryExists(id string) bool { _, ok := m.byID[id]; return ok }

// EntryOwner is the member a time entry belongs to.
func (m *Module) EntryOwner(id string) string { return m.byID[id].MemberID }

// approvedMinutes is the approved minutes of the member in the month.
func (m *Module) approvedMinutes(memberID string, month int) int {
	n := 0
	for _, e := range m.list {
		if e.MemberID == memberID && e.Status == "approved" && policies.MonthOf(e.Date*policies.MinPerDay) == month {
			n += e.Minutes
		}
	}
	return n
}

func entryView(e *Entry) policies.Rec {
	var credit any
	if e.Credit != nil {
		credit = *e.Credit
	}
	return policies.Rec{"entryId": e.ID, "memberId": e.MemberID, "date": policies.FormatDay(e.Date),
		"minutes": e.Minutes, "note": e.Note, "status": e.Status, "creditCents": credit}
}
