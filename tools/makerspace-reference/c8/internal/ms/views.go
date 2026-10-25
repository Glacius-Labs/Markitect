package ms

// rec is a JSON object with exactly the members of one record.
type rec = map[string]any

func optTime(p *int) any {
	if p == nil {
		return nil
	}
	return formatTime(*p)
}

func optInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func (s *state) userView(u *user) rec {
	return rec{"userId": u.id, "name": u.name, "role": u.role}
}

func (s *state) memberView(m *member) rec {
	return rec{"memberId": m.user.id, "name": m.user.name, "tier": m.tier,
		"standing": m.standing, "registeredAt": formatTime(m.registeredAt)}
}

func familyView(f *family) rec { return rec{"familyId": f.id, "name": f.name} }

func (s *state) courseView(c *course) rec {
	count := 0
	for _, e := range s.enrollments {
		if e.courseID == c.id && e.status == "enrolled" {
			count++
		}
	}
	return rec{"courseId": c.id, "familyId": c.familyID, "title": c.title,
		"feeCents": c.fee, "capacity": c.capacity, "enrolledCount": count}
}

func enrollmentView(e *enrollment) rec {
	return rec{"courseId": e.courseID, "memberId": e.memberID, "status": e.status,
		"enrolledAt": formatTime(e.enrolledAt), "completedAt": optTime(e.completedAt)}
}

func machineView(m *machine) rec {
	return rec{"machineId": m.id, "name": m.name, "familyId": m.familyID,
		"rateCentsPerMinute": m.rate}
}

func windowView(w *window) rec {
	status := "scheduled"
	if w.cancelled {
		status = "cancelled"
	}
	return rec{"windowId": w.id, "machineId": w.machineID, "start": formatTime(w.start),
		"end": formatTime(w.end), "reason": w.reason, "status": status}
}

func reservationView(r *reservation, at int) rec {
	return rec{"reservationId": r.id, "memberId": r.memberID, "machineId": r.machineID,
		"start": formatTime(r.start), "end": formatTime(r.end), "status": r.reportedStatus(at),
		"checkedInAt": optTime(r.checkedInAt), "checkedOutAt": optTime(r.checkedOutAt),
		"rateCentsPerMinute": optInt(r.rate), "meteredMinutes": optInt(r.metered),
		"amountCents": optInt(r.amount)}
}

func chargeView(c *charge) rec {
	return rec{"kind": c.kind, "ref": c.ref, "quantity": c.quantity, "unitPriceCents": c.unit,
		"discountPercent": c.discount, "amountCents": c.amount, "at": formatTime(c.at)}
}

func materialView(m *material) rec {
	return rec{"materialId": m.id, "name": m.name, "unitPriceCents": m.unit, "stock": m.stock}
}

func purchaseView(p *purchase) rec {
	return rec{"purchaseId": p.id, "memberId": p.memberID, "materialId": p.materialID,
		"quantity": p.quantity, "unitPriceCents": p.unit, "discountPercent": p.discount,
		"amountCents": p.amount, "at": formatTime(p.at)}
}

func invoiceView(i *invoice, at int) rec {
	lines := make([]rec, 0, len(i.lines))
	for k := range i.lines {
		lines = append(lines, chargeView(&i.lines[k]))
	}
	return rec{"invoiceId": i.id, "memberId": i.memberID, "month": formatMonth(i.month),
		"issuedAt": formatTime(i.issuedAt), "dueAt": formatTime(i.dueAt), "lines": lines,
		"totalCents": i.total, "creditAppliedCents": i.creditApplied, "paidCents": i.paid,
		"balanceCents": i.balance, "status": i.status(), "overdue": i.overdue(at)}
}

func paymentView(p *payment) rec {
	return rec{"paymentId": p.id, "invoiceId": p.invoiceID, "memberId": p.memberID,
		"amountCents": p.amount, "appliedCents": p.applied, "creditedCents": p.credit,
		"at": formatTime(p.at)}
}

func incidentView(i *incident) rec {
	status := "open"
	var resolution, resolvedAt any
	if i.resolved {
		status = "resolved"
		resolution = i.resolution
		resolvedAt = formatTime(i.resolvedAt)
	}
	return rec{"incidentId": i.id, "machineId": i.machineID, "severity": i.severity,
		"description": i.description, "reportedBy": i.reportedBy,
		"reportedAt": formatTime(i.reportedAt), "status": status,
		"resolution": resolution, "resolvedAt": resolvedAt}
}

func auditView(e *auditEntry) rec {
	var target any = e.target
	if e.outcome == "refused" {
		target = nil
	}
	return rec{"seq": e.seq, "at": formatTime(e.at), "actor": e.actor, "actorRole": e.actorRole,
		"op": e.op, "target": target, "outcome": e.outcome}
}

func (s *state) certView(c *certification, at int) rec {
	status := "valid"
	if c.revoked {
		status = "revoked"
	}
	return rec{"memberId": c.memberID, "familyId": c.familyID, "issuedOn": formatDay(c.issuedOn),
		"status": status}
}

func lockoutView(l *lockout) rec {
	ids := make([]string, 0, len(l.cancelledIDs))
	ids = append(ids, l.cancelledIDs...)
	return rec{"lockoutId": l.id, "incidentId": l.incidentID, "machineId": l.machineID,
		"start": formatTime(l.start), "end": formatTime(l.end), "cancelledReservationIds": ids}
}
