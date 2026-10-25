package ms_test

import "testing"

func TestStewardWorkflow(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("stu", now, "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z", "reason": "x"})
	e.ok("stu", now, "cancelMaintenance", m{"windowId": "mnt-1"})
	inc := obj(e.ok("stu", now, "reportIncident", m{"machineId": "lc1", "severity": "low", "description": "d"})["incident"])
	if inc["reportedBy"] != "stu" {
		t.Fatalf("%v", inc)
	}
	e.ok("stu", now, "listIncidents", m{})
	e.ok("stu", now, "listMachines", m{})
	e.ok("stu", now, "getMachineSchedule", m{"machineId": "lc1", "date": "2026-03-05"})
	e.ok("stu", now, "listMaterials", m{})
	e.ok("stu", now, "resolveIncident", m{"incidentId": "inc-1", "resolution": "ok"})
	e.ok("stu", now, "reportIncident", m{"machineId": "lc1", "severity": "high", "description": "d"})
	// a lockout by a steward grants the same credit
	e.ok("sam", now, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	out := e.ok("stu", now, "lockMachine", m{"incidentId": "inc-2", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	if len(list(out["cancelledReservations"])) != 1 {
		t.Fatalf("%v", out)
	}
	if num(obj(e.ok("admin", now, "getAccount", m{"memberId": "ada"})["account"])["creditBalanceCents"]) != 1000 {
		t.Fatal("lockout credit")
	}
}

func TestStewardHasNoOtherAccess(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	for _, op := range []string{"generateInvoices", "listInvoices", "recordPayment", "issueCredit", "getAccount",
		"revenueReport", "utilizationReport", "listAuditEntries", "registerUser", "addFamily", "addMachine",
		"setMachineRate", "getMember", "registerMember", "createReservation", "purchaseMaterial", "addMaterial"} {
		e.refused("stu", now, op, sampleArgs[op], "forbidden")
	}
}

func TestStewardIsNotAMember(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.refused("sam", now, "getMember", m{"memberId": "stu"}, "not_found")
	e.refused("sam", now, "setMemberStanding", m{"memberId": "stu", "standing": "suspended"}, "not_found")
	e.refused("sam", now, "registerMember", m{"memberId": "stu", "name": "x", "tier": "basic"}, "already_exists")
	e.ok("admin", now, "registerUser", m{"userId": "stu2", "name": "Stu Two", "role": "steward"})
	u := obj(e.ok("admin", now, "registerUser", m{"userId": "stu3", "name": "Stu Three", "role": "steward"})["user"])
	if u["role"] != "steward" {
		t.Fatalf("%v", u)
	}
	e.refused("admin", now, "registerUser", m{"userId": "stu4", "name": "x", "role": "Steward"}, "invalid_argument")
	// a steward has no invoices: billing names no such member
	e.refused("admin", now, "listInvoices", m{"memberId": "stu"}, "not_found")
}

func TestStewardAuditEntries(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("stu", now, "reportIncident", m{"machineId": "lc1", "severity": "low", "description": "d"})
	e.refused("stu", now, "getAccount", m{"memberId": "ada"}, "forbidden")
	log := auditLog(e, now)
	acc, ref := log[len(log)-2], log[len(log)-1]
	if acc["actor"] != "stu" || acc["actorRole"] != "steward" || acc["outcome"] != "accepted" || acc["target"] != "inc-1" ||
		ref["actorRole"] != "steward" || ref["outcome"] != "refused" || ref["op"] != "getAccount" || ref["target"] != nil {
		t.Fatalf("%v %v", acc, ref)
	}
}
