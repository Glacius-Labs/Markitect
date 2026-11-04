// Package jobs holds the jobs a member does for accepted quotes, their credit
// and their payments, and the operations on them.
package jobs

import (
	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/ledger"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/quotes"
	"makerspace/training"
)

type job struct {
	id, quoteID, customerID, makerID string
	completed                        bool
	total                            int
	makerCredit                      *int
	paid                             int
}

func (j *job) status() string {
	if j.completed {
		return "completed"
	}
	return "started"
}

// Module is the jobs module: it owns every job.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	billing   *billing.Module
	ledger    *ledger.Module
	training  *training.Module
	quotes    *quotes.Module
	ids       policies.IDs
	byID      map[string]*job
	list      []*job
	ofQuote   map[string]*job
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module, bill *billing.Module, led *ledger.Module,
	tr *training.Module, q *quotes.Module) *Module {
	return &Module{members: mem, equipment: eq, billing: bill, ledger: led, training: tr, quotes: q,
		byID: map[string]*job{}, ofQuote: map[string]*job{}}
}

// JobExists reports whether the identifier names a job.
func (m *Module) JobExists(id string) bool { _, ok := m.byID[id]; return ok }

func jobView(j *job) policies.Rec {
	var credit any
	if j.makerCredit != nil {
		credit = *j.makerCredit
	}
	return policies.Rec{"jobId": j.id, "quoteId": j.quoteID, "customerId": j.customerID, "makerId": j.makerID,
		"status": j.status(), "totalCents": j.total, "makerCreditCents": credit, "paidCents": j.paid}
}
