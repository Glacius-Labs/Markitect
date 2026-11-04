// Package billing holds the charges, the invoices, the payments and the credit
// of each member, and the operations on them.
package billing

import (
	"makerspace/households"
	"makerspace/ledger"
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
	ledger       *ledger.Module
	households   *households.Module
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
func New(mem *members.Module, led *ledger.Module, hh *households.Module) *Module {
	return &Module{
		members:    mem,
		ledger:     led,
		households: hh,
		invoices:   map[string]*Invoice{},
		invoiced:   map[int]bool{},
		credit:     map[string]int{},
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

// GrantCredit adds the credit an admin issues to the balance of a member, posts
// it as goodwill and counts it as a credit issued for the revenue report.
func (b *Module) GrantCredit(memberID string, amount, at int) {
	b.credit[memberID] += amount
	b.creditEvents = append(b.creditEvents, CreditEvent{At: at, Amount: amount})
	b.ledger.Post("expenseGoodwill", "creditLiability", amount, memberID, at)
}

// AddCredit adds credit to the balance of a member on behalf of another
// module: the entry that debits the named account and credits creditLiability
// by the amount is posted with the memo and the request time. An amount of 0
// changes nothing. The credit does not count as a credit issued by an admin.
func (b *Module) AddCredit(memberID string, amount int, debitAccount, memo string, at int) {
	if amount <= 0 {
		return
	}
	b.credit[memberID] += amount
	b.ledger.Post(debitAccount, "creditLiability", amount, memo, at)
}

// CreditBalance is the credit balance of a member.
func (b *Module) CreditBalance(memberID string) int { return b.credit[memberID] }

// OpenInvoices lists the invoices whose status is open, in ascending invoice
// number.
func (b *Module) OpenInvoices() []*Invoice {
	var out []*Invoice
	for _, i := range b.invoiceList {
		if i.Status() == "open" {
			out = append(out, i)
		}
	}
	return out
}

// OverdueInvoices lists the invoices that are overdue at the time, in
// ascending invoice number.
func (b *Module) OverdueInvoices(at int) []*Invoice {
	var out []*Invoice
	for _, i := range b.invoiceList {
		if i.Overdue(at) {
			out = append(out, i)
		}
	}
	return out
}

// ChargeOnPaidInvoice reports whether a line with the kind and ref is on an
// invoice whose status is paid.
func (b *Module) ChargeOnPaidInvoice(kind, ref string) bool {
	for _, i := range b.invoiceList {
		if i.Status() != "paid" {
			continue
		}
		for _, l := range i.Lines {
			if l.Kind == kind && l.Ref == ref {
				return true
			}
		}
	}
	return false
}

// Invoice is the invoice with the identifier, or nil.
func (b *Module) Invoice(id string) *Invoice { return b.invoices[id] }

// InvoiceOfMonth is the invoice a member has for a month, or nil.
func (b *Module) InvoiceOfMonth(memberID string, month int) *Invoice {
	for _, i := range b.invoiceList {
		if i.MemberID == memberID && i.Month == month {
			return i
		}
	}
	return nil
}

// InvoicesOfMonth lists the invoices of a month in ascending invoice number.
func (b *Module) InvoicesOfMonth(month int) []*Invoice {
	var out []*Invoice
	for _, i := range b.invoiceList {
		if i.Month == month {
			out = append(out, i)
		}
	}
	return out
}

// Account is the account figures of a member at the time.
type Account struct {
	Credit, Open, Overdue, Unbilled int
}

// AccountOf gives the credit balance, the open balance, the overdue balance
// and the unbilled amount of a member at the time.
func (b *Module) AccountOf(memberID string, at int) Account {
	a := Account{Credit: b.credit[memberID]}
	for _, i := range b.invoiceList {
		if i.MemberID != memberID {
			continue
		}
		if i.Status() == "open" {
			a.Open += i.Balance
		}
		if i.Overdue(at) {
			a.Overdue += i.Balance
		}
	}
	for _, ch := range b.memberCharges(memberID) {
		if !ch.Billed {
			a.Unbilled += ch.Amount
		}
	}
	return a
}

// PaymentsInMonth is the sum of the payments whose time lies in the month.
func (b *Module) PaymentsInMonth(month int) int {
	n := 0
	for _, p := range b.payments {
		if policies.MonthOf(p.At) == month {
			n += p.Amount
		}
	}
	return n
}

// CreditsIssuedInMonth is the sum of the credits issued with issueCredit whose
// request time lies in the month.
func (b *Module) CreditsIssuedInMonth(month int) int {
	n := 0
	for _, e := range b.creditEvents {
		if policies.MonthOf(e.At) == month {
			n += e.Amount
		}
	}
	return n
}
