// Package loans holds the tools members can borrow for a fortnight and their
// loans, the late fee of an overdue tool, and the operations on them.
package loans

import (
	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/guardians"
	"makerspace/members"
	"makerspace/notices"
	"makerspace/policies"
	"makerspace/training"
	"makerspace/waivers"
)

// loanDays is the number of days a tool may be kept.
const loanDays = 14

// maxActive is the number of active loans a member may hold.
const maxActive = 3

// maxRenewals is the number of times a loan may be renewed.
const maxRenewals = 2

// Tool is a tool that can be lent out; it belongs to a tool family.
type Tool struct {
	ID, Name, FamilyID string
	LateFeePerDay      int
	OnLoan             bool
}

// Loan is the loan of one tool to one member. DueOn is a date.
type Loan struct {
	ID, ToolID, MemberID string
	BorrowedAt, DueOn    int
	ReturnedAt           *int
	ChargedDays          int
	Renewals             int
}

// Status is active or returned.
func (l *Loan) Status() string {
	if l.ReturnedAt == nil {
		return "active"
	}
	return "returned"
}

// OverdueOn reports whether the loan is overdue on the date: it is active and
// the date is after its due date.
func (l *Loan) OverdueOn(day int) bool { return l.ReturnedAt == nil && day > l.DueOn }

// Module is the loans module: it owns every tool and loan.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	training  *training.Module
	guardians *guardians.Module
	waivers   *waivers.Module
	billing   *billing.Module
	notices   *notices.Module
	ids       policies.IDs
	tools     map[string]*Tool
	byID      map[string]*Loan
	list      []*Loan
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module, tr *training.Module, gd *guardians.Module, wv *waivers.Module,
	bill *billing.Module, ntc *notices.Module) *Module {
	return &Module{members: mem, equipment: eq, training: tr, guardians: gd, waivers: wv,
		billing: bill, notices: ntc, tools: map[string]*Tool{}, byID: map[string]*Loan{}}
}

// ToolExists reports whether the identifier names a tool.
func (m *Module) ToolExists(id string) bool { _, ok := m.tools[id]; return ok }

// LoanExists reports whether the identifier names a loan.
func (m *Module) LoanExists(id string) bool { _, ok := m.byID[id]; return ok }

// LoanOwner is the member a loan belongs to.
func (m *Module) LoanOwner(id string) string { return m.byID[id].MemberID }

// HasOverdue reports whether the member has a loan that is overdue on the date.
func (m *Module) HasOverdue(memberID string, day int) bool {
	for _, l := range m.list {
		if l.MemberID == memberID && l.OverdueOn(day) {
			return true
		}
	}
	return false
}

// chargeLateDays charges the days of late fee that the loan has not paid yet at
// the request time at and answers the charge. It changes nothing and answers
// nil when the loan is not overdue or every day is charged.
func (m *Module) chargeLateDays(l *Loan, at int) *billing.Charge {
	day := policies.DayOf(at)
	days := day - l.DueOn - l.ChargedDays
	if !l.OverdueOn(day) || days <= 0 {
		return nil
	}
	ch := m.billing.AddCharge(l.MemberID, "lateFee", l.ID, days, m.tools[l.ToolID].LateFeePerDay, 0, at)
	l.ChargedDays += days
	return ch
}

func toolView(t *Tool) policies.Rec {
	return policies.Rec{"toolId": t.ID, "name": t.Name, "familyId": t.FamilyID, "lateFeeCentsPerDay": t.LateFeePerDay,
		"onLoan": t.OnLoan}
}

func loanView(l *Loan, at int) policies.Rec {
	return policies.Rec{"loanId": l.ID, "toolId": l.ToolID, "memberId": l.MemberID,
		"borrowedAt": policies.FormatTime(l.BorrowedAt), "dueOn": policies.FormatDay(l.DueOn),
		"returnedAt": policies.OptTime(l.ReturnedAt), "status": l.Status(), "overdue": l.OverdueOn(policies.DayOf(at)),
		"chargedDays": l.ChargedDays, "renewals": l.Renewals}
}
