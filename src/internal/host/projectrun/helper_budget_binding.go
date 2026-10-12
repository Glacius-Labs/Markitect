package projectrun

import (
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
)

// bindRunHelperBudget clones the native executor with this run's durable
// helper reservation policy while leaving its stable fingerprint options intact.
func bindRunHelperBudget(invoker Invoker, host Host, limits projectconfig.Limits, store *runStore, report *RunReport) (Invoker, *RoleStartBudget, error) {
	budget, err := NewRoleStartBudget(limits.MaxStarts, func() RunReport { return *report }, func(record RoleStartReservation) error {
		found := false
		for index, prior := range report.RoleStartReservations {
			if prior.Key == record.Key {
				report.RoleStartReservations[index] = record
				found = true
				break
			}
		}
		if !found {
			report.RoleStartReservations = append(report.RoleStartReservations, record)
		}
		return persistState(store, report)
	})
	if err != nil {
		return invoker, nil, err
	}
	if binder, ok := invoker.(projectexecutor.HelperBinder); ok {
		invoker = binder.WithHelperPolicy(NativeHelperPolicy{Workspaces: host.Workspaces, Limits: limits,
			MaxStartRequests: limits.MaxStarts, Reserve: budget.ReserveHelper})
	}
	return invoker, budget, nil
}
