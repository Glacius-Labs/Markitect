// Package billing holds the charges, the invoices, the payments and the credit
// of each member, and the operations on them.
package billing

import (
	"makerspace/members"
	"makerspace/policies"
)

// Charge is one invoice line; Billed is false until it is on an invoice.
type Charge struct {
	MemberID, Kind, Ref string
	Quantity, Unit      int
	Discount, Amount    int
	At                  int
	Billed              bool
}

// Invoice is the invoice of one member for one month.
type Invoice struct {
	ID, MemberID  string
	Month         int
	IssuedAt      int
	DueAt         int
	Lines         []Charge
	Total         int
	CreditApplied int
	Paid, Balance int
}

// Status is paid when nothing is left to pay, otherwise open.
func (i *Invoice) Status() string {
	if i.Balance == 0 {
		return "paid"
	}
	return "open"
}

// Overdue reports whether the invoice is open and due at the given time.
func (i *Invoice) Overdue(at int) bool { return i.Status() == "open" && at >= i.DueAt }

// Payment is one payment of an invoice.
type Payment struct {
	ID, InvoiceID, MemberID string
	Amount, Applied, Credit int
	At                      int
}

// CreditEvent is credit granted to a member, counted in the revenue report.
type CreditEvent struct {
	At, Amount int
}

// Module is the billing module: it owns every charge, invoice, payment and
// credit balance.
type Module struct {
	members      *members.Module
	ids          policies.IDs
	charges      []*Charge
	invoices     map[string]*Invoice
	invoiceList  []*Invoice
	payments     []*Payment
	creditEvents []CreditEvent
	invoiced     map[int]bool
	credit       map[string]int // credit balance of each member
}

// New returns the module in the initial state.
func New(mem *members.Module) *Module {
	return &Module{
		members:  mem,
		invoices: map[string]*Invoice{},
		invoiced: map[int]bool{},
		credit:   map[string]int{},
	}
}

// InvoiceExists reports whether the identifier names an invoice.
func (b *Module) InvoiceExists(id string) bool { _, ok := b.invoices[id]; return ok }

// InvoiceOwner is the member an invoice belongs to.
func (b *Module) InvoiceOwner(id string) string { return b.invoices[id].MemberID }

// Invoices lists all invoices in creation order.
func (b *Module) Invoices() []*Invoice { return b.invoiceList }

// Payments lists all payments in creation order.
func (b *Module) Payments() []*Payment { return b.payments }

// CreditEvents lists all credits granted to members.
func (b *Module) CreditEvents() []CreditEvent { return b.creditEvents }

// Invoiced reports whether the invoices of the month have been generated.
func (b *Module) Invoiced(month int) bool { return b.invoiced[month] }

func (b *Module) memberCharges(memberID string) []*Charge {
	var out []*Charge
	for _, c := range b.charges {
		if c.MemberID == memberID {
			out = append(out, c)
		}
	}
	return out
}

// AddCharge records a charge for a member and returns it. The amount follows
// the money rule.
func (b *Module) AddCharge(memberID, kind, ref string, quantity, unit, discount, at int) *Charge {
	c := &Charge{MemberID: memberID, Kind: kind, Ref: ref, Quantity: quantity, Unit: unit,
		Discount: discount, Amount: policies.ChargeAmount(quantity, unit, discount), At: at}
	b.charges = append(b.charges, c)
	return c
}

// GrantCredit adds credit to the balance of a member.
func (b *Module) GrantCredit(memberID string, amount, at int) {
	b.credit[memberID] += amount
	b.creditEvents = append(b.creditEvents, CreditEvent{At: at, Amount: amount})
}
