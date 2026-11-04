package jobs

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	jobRef := policies.IDRef("jobId", m.JobExists)
	return []*policies.Operation{
		{Name: "startJob", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("quoteId", m.quotes.QuoteExists), policies.IDRef("makerId", m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				q, _ := m.quotes.QuoteOf(c.Str("quoteId"))
				maker := c.Str("makerId")
				if q.Status != "accepted" {
					return nil, "", policies.Refuse("quote_not_accepted", "The quote is not accepted.")
				}
				if m.ofQuote[q.ID] != nil {
					return nil, "", policies.Refuse("already_started", "A job already exists for the quote.")
				}
				if !m.members.InGoodStanding(maker) {
					return nil, "", policies.Refuse("member_suspended", "The maker is suspended.")
				}
				day := policies.DayOf(c.At)
				for _, l := range q.Lines {
					if l.Kind == "machine" && !m.training.Certified(maker, m.equipment.Machine(l.Ref).FamilyID, day) {
						return nil, "", policies.Refuse("maker_not_certified", "The maker is not certified for machine %s.", l.Ref)
					}
				}
				j := &job{id: m.ids.Next("job"), quoteID: q.ID, customerID: q.CustomerID, makerID: maker, total: q.TotalCents}
				m.byID[j.id] = j
				m.list = append(m.list, j)
				m.ofQuote[q.ID] = j
				return policies.Rec{"job": jobView(j)}, j.id, nil
			}},
		{Name: "completeJob", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{jobRef},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				j := m.byID[c.Str("jobId")]
				if j.completed {
					return nil, "", policies.Refuse("invalid_state", "The job is not started.")
				}
				credit := (j.total*30 + 50) / 100
				j.completed, j.makerCredit = true, &credit
				m.billing.AddCredit(j.makerID, credit, "expenseMaker", j.id, c.At)
				return policies.Rec{"job": jobView(j)}, j.id, nil
			}},
		{Name: "recordJobPayment", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{jobRef, policies.Integer("amountCents", 1, 100000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				j, amount := m.byID[c.Str("jobId")], c.Num("amountCents")
				if !j.completed {
					return nil, "", policies.Refuse("job_not_completed", "The job is not completed.")
				}
				if j.paid+amount > j.total {
					return nil, "", policies.Refuse("overpayment", "The payment would exceed the total of the job.")
				}
				j.paid += amount
				m.ledger.Post("cash", "revenueJobs", amount, j.id, c.At)
				return policies.Rec{"job": jobView(j)}, j.id, nil
			}},
		{Name: "listJobs", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("customerId", m.quotes.CustomerExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid := c.Str("customerId")
				out := []policies.Rec{}
				for _, j := range m.list {
					if j.customerID == cid {
						out = append(out, jobView(j))
					}
				}
				return policies.Rec{"customerId": cid, "jobs": out}, "", nil
			}},
	}
}
