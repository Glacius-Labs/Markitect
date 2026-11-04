package ledger

import "makerspace/policies"

func entryView(e *Entry) policies.Rec {
	return policies.Rec{"entryId": e.ID, "at": policies.FormatTime(e.At), "debitAccount": e.Debit,
		"creditAccount": e.Credit, "amountCents": e.Amount, "memo": e.Memo}
}

// rowViews gives one trial balance row per account, and the two totals.
func (m *Module) rowViews() ([]policies.Rec, int, int) {
	debits, credits := map[string]int{}, map[string]int{}
	totalDebit, totalCredit := 0, 0
	for _, e := range m.entries {
		debits[e.Debit] += e.Amount
		credits[e.Credit] += e.Amount
		totalDebit += e.Amount
		totalCredit += e.Amount
	}
	rows := make([]policies.Rec, 0, len(Accounts))
	for _, a := range Accounts {
		d, c := debits[a], credits[a]
		side, balance := "none", 0
		if d > c {
			side, balance = "debit", d-c
		} else if c > d {
			side, balance = "credit", c-d
		}
		rows = append(rows, policies.Rec{"account": a, "debitCents": d, "creditCents": c,
			"balanceCents": balance, "side": side})
	}
	return rows, totalDebit, totalCredit
}
