package projectrun

// helperHostBinder keeps protocol fixtures and alternate process invokers usable
// while binding the normal native router to this run's durable quota ledger.
type helperHostBinder interface {
	WithHelperHost(TransportHelperHost) Invoker
}

func bindRunHelperBudget(invoker Invoker, host Host, limits Limits, store *runStore, report *RunReport) (Invoker, *RoleStartBudget, error) {
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
	if binder, ok := invoker.(helperHostBinder); ok {
		invoker = binder.WithHelperHost(TransportHelperHost{Workspaces: host.Workspaces, Limits: limits, MaxStartRequests: limits.MaxStarts, Reserve: budget.ReserveHelper})
	}
	return invoker, budget, nil
}
