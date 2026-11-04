package quotes

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	quoteRef := policies.IDRef("quoteId", m.QuoteExists)
	return []*policies.Operation{
		{Name: "addCustomer", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("customerId"), policies.Text("name")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("customerId")
				if m.customers[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A customer with identifier %s already exists.", id)
				}
				cu := &customer{id: id, name: c.Str("name")}
				m.customers[id] = cu
				return policies.Rec{"customer": customerView(cu)}, id, nil
			}},
		{Name: "createQuote", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("customerId", m.CustomerExists), policies.Text("title")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				q := &quote{id: m.ids.Next("quo"), customerID: c.Str("customerId"), title: c.Str("title"), status: "draft"}
				m.quotes[q.id] = q
				return policies.Rec{"quote": quoteView(q)}, q.id, nil
			}},
		{Name: "addQuoteLine", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{quoteRef, policies.Enum("lineKind", "machine", "material"), policies.ID("ref"),
				policies.Integer("quantity", 1, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				q := m.quotes[c.Str("quoteId")]
				kind, ref := c.Str("lineKind"), c.Str("ref")
				if q.status != "draft" {
					return nil, "", policies.Refuse("invalid_state", "The quote is not a draft.")
				}
				known := m.equipment.MachineExists(ref)
				if kind == "material" {
					known = m.materials.MaterialExists(ref)
				}
				if !known {
					return nil, "", policies.Refuse("unknown_line_ref", "No %s has the identifier %s.", kind, ref)
				}
				q.lines = append(q.lines, &line{kind: kind, ref: ref, quantity: c.Num("quantity")})
				return policies.Rec{"quote": quoteView(q)}, q.id, nil
			}},
		{Name: "sendQuote", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{quoteRef},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				q := m.quotes[c.Str("quoteId")]
				if q.status != "draft" {
					return nil, "", policies.Refuse("invalid_state", "The quote is not a draft.")
				}
				if len(q.lines) == 0 {
					return nil, "", policies.Refuse("empty_quote", "The quote has no line.")
				}
				subtotal := 0
				for _, l := range q.lines {
					var unit int
					if l.kind == "machine" {
						unit = m.equipment.Machine(l.ref).Rate
					} else {
						unit = m.materials.UnitPrice(l.ref)
					}
					l.unit = &unit
					subtotal += l.quantity * unit
				}
				at, valid := c.At, policies.DayOf(c.At)+validDays
				q.status, q.subtotal, q.total = "sent", subtotal, (subtotal*120+50)/100
				q.sentAt, q.validUntil = &at, &valid
				return policies.Rec{"quote": quoteView(q)}, q.id, nil
			}},
		{Name: "decideQuote", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{quoteRef, policies.Enum("decision", "accepted", "rejected")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				q := m.quotes[c.Str("quoteId")]
				decision := c.Str("decision")
				if q.status != "sent" {
					return nil, "", policies.Refuse("invalid_state", "The quote is not sent.")
				}
				if decision == "accepted" && policies.DayOf(c.At) > *q.validUntil {
					return nil, "", policies.Refuse("quote_expired", "The quote was valid until %s.", policies.FormatDay(*q.validUntil))
				}
				q.status = decision
				return policies.Rec{"quote": quoteView(q)}, q.id, nil
			}},
		{Name: "getQuote", Perms: "naa",
			Args: []policies.ArgSpec{quoteRef},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				return policies.Rec{"quote": quoteView(m.quotes[c.Str("quoteId")])}, "", nil
			}},
	}
}
