// Package quotes holds the customers of the workshop's made-to-order work and
// the quotes offered to them, and the operations on them.
package quotes

import (
	"makerspace/equipment"
	"makerspace/materials"
	"makerspace/policies"
)

// validDays is how long a sent quote can be accepted, in days after the date of
// sending.
const validDays = 30

type customer struct{ id, name string }

type line struct {
	kind, ref string
	quantity  int
	unit      *int // nil until the quote is sent
}

type quote struct {
	id, customerID, title, status string
	lines                         []*line
	subtotal, total               int
	sentAt                        *int
	validUntil                    *int // day
}

// Line is a copy of a quote line. UnitPriceCents is nil until the quote is
// sent.
type Line struct {
	Kind, Ref      string
	Quantity       int
	UnitPriceCents *int
}

// Quote is a copy of a quote for other modules.
type Quote struct {
	ID, CustomerID, Status string
	TotalCents             int
	Lines                  []Line
}

// Module is the quotes module: it owns every customer and quote.
type Module struct {
	equipment *equipment.Module
	materials *materials.Module
	ids       policies.IDs
	customers map[string]*customer
	quotes    map[string]*quote
}

// New returns the module in the initial state.
func New(eq *equipment.Module, mat *materials.Module) *Module {
	return &Module{equipment: eq, materials: mat, customers: map[string]*customer{}, quotes: map[string]*quote{}}
}

// CustomerExists reports whether the identifier names a customer.
func (m *Module) CustomerExists(id string) bool { _, ok := m.customers[id]; return ok }

// QuoteExists reports whether the identifier names a quote.
func (m *Module) QuoteExists(id string) bool { _, ok := m.quotes[id]; return ok }

// QuoteOf is a copy of the quote with the identifier: its customer, status,
// total and lines. The last result is false when the identifier names no
// quote.
func (m *Module) QuoteOf(id string) (Quote, bool) {
	q := m.quotes[id]
	if q == nil {
		return Quote{}, false
	}
	out := Quote{ID: q.id, CustomerID: q.customerID, Status: q.status, TotalCents: q.total}
	for _, l := range q.lines {
		out.Lines = append(out.Lines, Line{Kind: l.kind, Ref: l.ref, Quantity: l.quantity, UnitPriceCents: l.unit})
	}
	return out, true
}

func customerView(c *customer) policies.Rec {
	return policies.Rec{"customerId": c.id, "name": c.name}
}

func quoteView(q *quote) policies.Rec {
	lines := make([]policies.Rec, 0, len(q.lines))
	for _, l := range q.lines {
		var unit any
		if l.unit != nil {
			unit = *l.unit
		}
		lines = append(lines, policies.Rec{"lineKind": l.kind, "ref": l.ref, "quantity": l.quantity,
			"unitPriceCents": unit})
	}
	var validUntil any
	if q.validUntil != nil {
		validUntil = policies.FormatDay(*q.validUntil)
	}
	return policies.Rec{"quoteId": q.id, "customerId": q.customerID, "title": q.title, "status": q.status,
		"lines": lines, "subtotalCents": q.subtotal, "totalCents": q.total,
		"sentAt": policies.OptTime(q.sentAt), "validUntil": validUntil}
}
