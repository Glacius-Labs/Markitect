package bookings

import "makerspace/policies"

func optInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func reservationView(r *reservation, at int) policies.Rec {
	return policies.Rec{"reservationId": r.id, "memberId": r.memberID, "machineId": r.machineID,
		"start": policies.FormatTime(r.start), "end": policies.FormatTime(r.end), "status": r.reportedStatus(at),
		"checkedInAt": policies.OptTime(r.checkedInAt), "checkedOutAt": policies.OptTime(r.checkedOutAt),
		"rateCentsPerMinute": optInt(r.rate), "meteredMinutes": optInt(r.metered),
		"amountCents": optInt(r.amount)}
}

func lockoutView(l *Lockout) policies.Rec {
	ids := make([]string, 0, len(l.CancelledIDs))
	ids = append(ids, l.CancelledIDs...)
	return policies.Rec{"lockoutId": l.ID, "incidentId": l.IncidentID, "machineId": l.MachineID,
		"start": policies.FormatTime(l.Start), "end": policies.FormatTime(l.End), "cancelledReservationIds": ids}
}
