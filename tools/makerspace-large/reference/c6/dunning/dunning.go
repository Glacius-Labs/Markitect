// Package dunning holds the dunning ladder and the cases of overdue invoices,
// and the operations on them.
package dunning

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/notices"
	"makerspace/policies"
)

// Ladder is the steps against an overdue invoice and the days they are taken.
type Ladder struct {
	NoticeDays, FeeDays, FeeCents, SuspendDays int
}

// Case is the record of the steps taken for one overdue invoice.
type Case struct {
	ID, InvoiceID, MemberID              string
	NoticedAt, FeeChargedAt, SuspendedAt *int
}

// Module is the dunning module: it owns the ladder and every case.
type Module struct {
	members *members.Module
	billing *billing.Module
	notices *notices.Module
	ids     policies.IDs
	ladder  *Ladder
	byInv   map[string]*Case
	list    []*Case
}

// New returns the module in the initial state: no ladder.
func New(mem *members.Module, bill *billing.Module, ntc *notices.Module) *Module {
	return &Module{members: mem, billing: bill, notices: ntc, byInv: map[string]*Case{}}
}

func ladderView(l *Ladder) policies.Rec {
	return policies.Rec{"noticeDays": l.NoticeDays, "feeDays": l.FeeDays, "feeCents": l.FeeCents,
		"suspendDays": l.SuspendDays}
}

func caseView(c *Case) policies.Rec {
	return policies.Rec{"caseId": c.ID, "invoiceId": c.InvoiceID, "memberId": c.MemberID,
		"noticedAt": policies.OptTime(c.NoticedAt), "feeChargedAt": policies.OptTime(c.FeeChargedAt),
		"suspendedAt": policies.OptTime(c.SuspendedAt)}
}
