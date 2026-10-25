package ms_test

import "testing"

// sampleArgs gives well-formed arguments for every operation, naming records
// that do not exist unless the operation takes none.
var sampleArgs = map[string]m{
	"registerUser":           {"userId": "u1", "name": "U", "role": "staff"},
	"registerMember":         {"memberId": "m1", "name": "M", "tier": "basic"},
	"getMember":              {"memberId": "nobody"},
	"changeTier":             {"memberId": "nobody", "tier": "pro"},
	"setMemberStanding":      {"memberId": "nobody", "standing": "good"},
	"addCourse":              {"courseId": "c1", "familyId": "nofam", "title": "T", "feeCents": 0, "capacity": 1},
	"enrollInCourse":         {"courseId": "nocourse", "memberId": "ada"},
	"completeCourse":         {"courseId": "nocourse", "memberId": "ada"},
	"revokeCertification":    {"memberId": "nobody", "familyId": "laser"},
	"listCertifications":     {"memberId": "nobody"},
	"addFamily":              {"familyId": "f9", "name": "F"},
	"addMachine":             {"machineId": "m9", "name": "M", "familyId": "nofam", "rateCentsPerMinute": 1},
	"setMachineRate":         {"machineId": "nomachine", "rateCentsPerMinute": 1},
	"listMachines":           {},
	"scheduleMaintenance":    {"machineId": "nomachine", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z", "reason": "r"},
	"cancelMaintenance":      {"windowId": "mnt-99"},
	"createReservation":      {"memberId": "ada", "machineId": "nomachine", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"},
	"cancelReservation":      {"reservationId": "res-99"},
	"checkIn":                {"reservationId": "res-99"},
	"checkOut":               {"reservationId": "res-99"},
	"listMemberReservations": {"memberId": "nobody"},
	"getMachineSchedule":     {"machineId": "nomachine", "date": "2026-03-05"},
	"addMaterial":            {"materialId": "w1", "name": "Wood", "unitPriceCents": 100},
	"restockMaterial":        {"materialId": "nomat", "quantity": 1},
	"purchaseMaterial":       {"memberId": "ada", "materialId": "nomat", "quantity": 1},
	"listMaterials":          {},
	"generateInvoices":       {"month": "2026-01"},
	"listInvoices":           {"memberId": "nobody"},
	"recordPayment":          {"invoiceId": "inv-99", "amountCents": 1},
	"issueCredit":            {"memberId": "nobody", "amountCents": 1, "reason": "r"},
	"getAccount":             {"memberId": "nobody"},
	"reportIncident":         {"machineId": "nomachine", "severity": "low", "description": "d"},
	"resolveIncident":        {"incidentId": "inc-99", "resolution": "r"},
	"listIncidents":          {},
	"utilizationReport":      {"from": "2026-03-01", "to": "2026-03-08"},
	"revenueReport":          {"month": "2026-03"},
	"lockMachine":            {"incidentId": "inc-99", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"},
	"listAuditEntries":       {},
}

// matrix is the operation index of section 14: permissions of member, staff, admin.
var matrix = map[string]string{
	"registerUser": "nna", "registerMember": "naa", "getMember": "oaa", "changeTier": "naa",
	"setMemberStanding": "naa", "addCourse": "naa", "enrollInCourse": "oaa", "completeCourse": "naa",
	"revokeCertification": "naa", "listCertifications": "oaa", "addFamily": "nna", "addMachine": "nna",
	"setMachineRate": "nna", "listMachines": "aaa", "scheduleMaintenance": "naa", "cancelMaintenance": "naa",
	"createReservation": "oaa", "cancelReservation": "oaa", "checkIn": "oaa", "checkOut": "oaa",
	"listMemberReservations": "oaa", "getMachineSchedule": "aaa", "addMaterial": "naa",
	"restockMaterial": "naa", "purchaseMaterial": "oaa", "listMaterials": "aaa", "generateInvoices": "naa",
	"listInvoices": "oaa", "recordPayment": "oaa", "issueCredit": "nna", "getAccount": "oaa",
	"reportIncident": "naa", "resolveIncident": "naa", "listIncidents": "naa", "utilizationReport": "naa",
	"revenueReport": "naa", "listAuditEntries": "nna", "lockMachine": "naa",
}

func TestThereAre38Operations(t *testing.T) {
	if len(sampleArgs) != 38 || len(matrix) != 38 {
		t.Fatalf("operations: %d samples, %d permission rows", len(sampleArgs), len(matrix))
	}
}

func TestPermissionMatrix(t *testing.T) {
	roles := []struct{ actor, role string }{{"ada", "member"}, {"sam", "staff"}, {"admin", "admin"}}
	for op, perms := range matrix {
		for i, r := range roles {
			e := shop(t)
			resp := e.send(r.actor, "2026-03-02T09:00Z", op, sampleArgs[op])
			forbidden := resp["status"] == "refused" && resp["code"] == "forbidden"
			if (perms[i] == 'n') != forbidden {
				t.Errorf("%s as %s: permission %c, response %v", op, r.role, perms[i], resp)
			}
		}
	}
}

func TestOwnershipOfMemberOperations(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-02T09:00Z", "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	for _, op := range []string{"getMember", "listCertifications", "listMemberReservations", "listInvoices", "getAccount"} {
		e.refused("bob", "2026-03-02T09:00Z", op, m{"memberId": "ada"}, "forbidden")
		e.ok("ada", "2026-03-02T09:00Z", op, m{"memberId": "ada"})
	}
	e.refused("bob", "2026-03-02T09:00Z", "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"}, "forbidden")
	e.refused("bob", "2026-03-02T09:00Z", "purchaseMaterial", m{"memberId": "ada", "materialId": "x", "quantity": 1}, "not_found")
	r := e.ok("ada", "2026-03-04T10:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1",
		"start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"})
	rid := obj(r["reservation"])["reservationId"]
	for _, op := range []string{"cancelReservation", "checkIn", "checkOut"} {
		e.refused("bob", "2026-03-04T10:00Z", op, m{"reservationId": rid}, "forbidden")
	}
	e.refused("bob", "2026-03-04T10:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1",
		"start": "2026-03-05T12:00Z", "end": "2026-03-05T13:00Z"}, "forbidden")
}

func TestOwnershipOfInvoices(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-02T09:00Z", "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	e.refused("bob", "2026-04-01T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 100}, "forbidden")
	e.ok("ada", "2026-04-01T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 100})
}

func TestOrderOfChecks(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	// 1 arguments before 2 actor
	e.refused("ghost", at, "getMember", m{"memberId": 5}, "invalid_argument")
	// 2 actor before 3 time and 4 permission
	e.refused("ghost", "2025-01-01T00:00Z", "listAuditEntries", m{}, "unknown_actor")
	// 3 time before 4 permission
	e.refused("ada", "2026-03-01T09:00Z", "listAuditEntries", m{}, "time_regression")
	// 4 permission before 5 references
	e.refused("ada", at, "setMemberStanding", m{"memberId": "nobody", "standing": "good"}, "forbidden")
	// 5 references before 6 ownership
	e.refused("ada", at, "getMember", m{"memberId": "nobody"}, "not_found")
	// references are checked in argument order
	e.refused("sam", at, "enrollInCourse", m{"courseId": "nocourse", "memberId": "nobody"}, "not_found")
	// a staff user does not name a member
	e.refused("sam", at, "getMember", m{"memberId": "sam"}, "not_found")
	e.refused("sam", at, "changeTier", m{"memberId": "admin", "tier": "pro"}, "not_found")
	// 5 references before 7 operation rules
	e.refused("admin", at, "addCourse", m{"courseId": "laser-101", "familyId": "nofam", "title": "T", "feeCents": 1, "capacity": 1}, "not_found")
	e.refused("admin", at, "addCourse", m{"courseId": "laser-101", "familyId": "laser", "title": "T", "feeCents": 1, "capacity": 1}, "already_exists")
}

func TestRequestTimeOnlyMovesWithOk(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-05T10:00Z", "listMachines", m{})
	e.refused("sam", "2026-03-05T09:59Z", "listMachines", m{}, "time_regression")
	// an equal time is allowed
	e.ok("sam", "2026-03-05T10:00Z", "listMachines", m{})
	// a refusal does not move the request time
	e.refused("sam", "2026-03-06T10:00Z", "getMember", m{"memberId": "nobody"}, "not_found")
	e.ok("sam", "2026-03-05T10:30Z", "listMachines", m{})
	// an invalid request does not move it either
	e.raw(`{"id":"x","op":"nope","args":{}}`)
	e.ok("sam", "2026-03-05T10:30Z", "listMachines", m{})
}

func TestAuditEntries(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	e.ok("admin", at, "registerMember", m{"memberId": "ada", "name": "Ada", "tier": "basic"})
	e.ok("admin", at, "setMemberStanding", m{"memberId": "ada", "standing": "good"}) // no change
	e.ok("admin", at, "getMember", m{"memberId": "ada"})                             // read
	e.refused("admin", at, "registerMember", m{"memberId": "ada", "name": "Ada", "tier": "basic"}, "already_exists")
	e.raw(`not json`)
	entries := list(e.ok("admin", "2026-03-02T09:05Z", "listAuditEntries", m{})["entries"])
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %v", entries)
	}
	first, second := obj(entries[0]), obj(entries[1])
	assertKeys(t, first, "seq", "at", "actor", "actorRole", "op", "target", "outcome")
	if num(first["seq"]) != 1 || first["op"] != "registerMember" || first["target"] != "ada" ||
		first["actor"] != "admin" || first["at"] != at || first["actorRole"] != "admin" || first["outcome"] != "accepted" {
		t.Fatalf("first entry: %v", first)
	}
	if num(second["seq"]) != 2 || second["op"] != "setMemberStanding" {
		t.Fatalf("second entry: %v", second)
	}
}

func TestAuditTargets(t *testing.T) {
	e := shop(t)
	targets := map[string]string{}
	for _, en := range list(e.ok("admin", "2026-03-02T09:00Z", "listAuditEntries", m{})["entries"]) {
		targets[obj(en)["op"].(string)] = obj(en)["target"].(string)
	}
	want := map[string]string{"registerUser": "sam", "registerMember": "ada", "addFamily": "laser",
		"addMachine": "lc1", "addCourse": "laser-101", "enrollInCourse": "laser-101/ada",
		"completeCourse": "laser-101/ada"}
	for op, target := range want {
		if targets[op] != target {
			t.Errorf("%s target = %q, want %q", op, targets[op], target)
		}
	}
}
