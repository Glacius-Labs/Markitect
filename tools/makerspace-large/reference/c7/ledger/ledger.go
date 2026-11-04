// Package ledger holds the double-entry journal: the fixed accounts, the
// journal entries and the trial balance, and the operations on them.
package ledger

import "makerspace/policies"

// Accounts lists the ledger accounts in ascending (listing) order.
var Accounts = []string{"cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker",
	"expensePayout", "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable",
	"revenueCourse", "revenueDeposits", "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees",
	"revenueLockers", "revenueMaterial", "revenueMembership", "revenueNoShows", "revenuePacks", "revenueRefunds",
	"revenueUsage"}

// Entry is one posting: an amount that debits one account and credits another.
type Entry struct {
	ID, Debit, Credit string
	Amount, At        int
	Memo              string
}

// Module is the ledger module: it owns every journal entry.
type Module struct {
	ids     policies.IDs
	entries []*Entry
}

// New returns the module in the initial state: an empty journal.
func New() *Module { return &Module{} }

// Post records one entry at the request time. An amount of 0 posts nothing and
// consumes no entry number.
func (m *Module) Post(debit, credit string, amount int, memo string, at int) {
	if amount <= 0 {
		return
	}
	m.entries = append(m.entries, &Entry{ID: m.ids.Next("jrn"), Debit: debit, Credit: credit,
		Amount: amount, At: at, Memo: memo})
}

// EntriesInMonth is the number of entries whose time lies in the month.
func (m *Module) EntriesInMonth(month int) int {
	n := 0
	for _, e := range m.entries {
		if policies.MonthOf(e.At) == month {
			n++
		}
	}
	return n
}

// Balanced reports whether the total of all debits equals the total of all
// credits.
func (m *Module) Balanced() bool {
	_, debit, credit := m.rowViews()
	return debit == credit
}
