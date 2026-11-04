package ledger

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "listJournalEntries", Perms: "nna",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				month := c.Num("month")
				out := []policies.Rec{}
				for _, e := range m.entries {
					if policies.MonthOf(e.At) == month {
						out = append(out, entryView(e))
					}
				}
				return policies.Rec{"month": c.Str("month"), "entries": out}, "", nil
			}},
		{Name: "getTrialBalance", Perms: "nna",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				rows, debit, credit := m.rowViews()
				return policies.Rec{"rows": rows, "totalDebitCents": debit, "totalCreditCents": credit}, "", nil
			}},
		{Name: "postAdjustment", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.Enum("debitAccount", Accounts...), policies.Enum("creditAccount", Accounts...),
				policies.Integer("amountCents", 1, 100000000), policies.Text("memo")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				if c.Str("debitAccount") == c.Str("creditAccount") {
					return nil, "", policies.Refuse("same_account", "The debit account and the credit account are the same.")
				}
				m.Post(c.Str("debitAccount"), c.Str("creditAccount"), c.Num("amountCents"), c.Str("memo"), c.At)
				e := m.entries[len(m.entries)-1]
				return policies.Rec{"entry": entryView(e)}, e.ID, nil
			}},
	}
}
