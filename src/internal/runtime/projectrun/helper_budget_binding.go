package projectrun

import (
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
)

// bindRunHelperBudget clones the native executor with this run's durable
// helper reservation policy while leaving its stable fingerprint options intact.
func bindRunHelperBudget(invoker Invoker, host Host, limits projectconfig.Limits, store *runStore, report *RunReport) (Invoker, *RoleStartBudget, error) {
	journal, err := newManagerStageJournal(store, report, limits)
	if err != nil {
		return invoker, nil, err
	}
	return bindManagerJournalHelpers(invoker, host, limits, journal), journal.budget, nil
}

func bindManagerJournalHelpers(invoker Invoker, host Host, limits projectconfig.Limits, journal *managerStageJournal) Invoker {
	budget := journal.budget
	if binder, ok := invoker.(projectexecutor.HelperBinder); ok {
		invoker = binder.WithHelperPolicy(NativeHelperPolicy{Workspaces: host.Workspaces, Limits: limits,
			MaxStartRequests: limits.MaxStarts, Reserve: budget.ReserveHelper})
	}
	return invoker
}
