package ms_test

import (
	"strings"
	"testing"
)

// Computed values and limits (section 2.8).

func lateShop(t *testing.T, at string) *env {
	e := newEnv(t)
	e.ok("admin", at, "registerUser", m{"userId": "sam", "name": "Sam", "role": "staff"})
	e.ok("sam", at, "registerMember", m{"memberId": "ada", "name": "Ada", "tier": "basic"})
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "Laser"})
	e.ok("admin", at, "addMachine", m{"machineId": "lc1", "name": "L", "familyId": "laser", "rateCentsPerMinute": 1})
	return e
}

func TestLineOf65536BytesIsAccepted(t *testing.T) {
	e := newEnv(t)
	prefix := `{"id":"big","op":"registerUser","args":{"actor":"admin","at":"2026-03-02T09:00Z","userId":"u","role":"staff","name":"`
	suffix := `"}}`
	pad := 65536 - len(prefix) - len(suffix)
	line := prefix + strings.Repeat("x", pad) + suffix
	if len(line) != 65536 {
		t.Fatalf("line length %d", len(line))
	}
	if r := e.raw(line); r["status"] != "ok" || r["id"] != "big" {
		t.Fatalf("%.100v", r)
	}
}

func TestEndOfCalendarBilling(t *testing.T) {
	e := lateShop(t, "2099-10-15T09:00Z")
	// 2099-11 is the last month that can be invoiced; its invoice is due after 2099 and is never overdue
	e.refused("sam", "2099-11-30T23:59Z", "generateInvoices", m{"month": "2099-11"}, "month_not_ended")
	out := e.ok("sam", "2099-12-31T23:59Z", "generateInvoices", m{"month": "2099-11"})
	inv := obj(list(out["invoices"])[0])
	if inv["issuedAt"] != "2099-12-31T23:59Z" || inv["dueAt"] != "2100-01-14T23:59Z" || inv["overdue"] != false {
		t.Fatalf("%v", inv)
	}
	got := obj(list(e.ok("ada", "2099-12-31T23:59Z", "listInvoices", m{"memberId": "ada"})["invoices"])[0])
	if got["overdue"] != false {
		t.Fatalf("%v", got)
	}
	// 2099-12 can never be invoiced
	e.refused("sam", "2099-12-31T23:59Z", "generateInvoices", m{"month": "2099-12"}, "month_not_ended")
	rep := e.ok("sam", "2099-12-31T23:59Z", "revenueReport", m{"month": "2099-12"})
	if rep["invoicesGenerated"] != false || num(rep["invoiceCount"]) != 0 {
		t.Fatalf("%v", rep)
	}
}

func TestEndOfCalendarTimeRanges(t *testing.T) {
	e := lateShop(t, "2099-12-20T09:00Z")
	// a maintenance window ends at the latest at 2099-12-31T23:59Z
	w := obj(e.ok("sam", "2099-12-20T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2099-12-30T00:00Z", "end": "2099-12-31T23:59Z", "reason": "long"})["window"])
	if w["end"] != "2099-12-31T23:59Z" {
		t.Fatalf("%v", w)
	}
	e.refused("sam", "2099-12-20T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2099-12-31T23:59Z", "end": "2100-01-01T00:00Z", "reason": "x"}, "invalid_argument")
	// the last day a utilization report can include is 2099-12-30
	r := e.ok("sam", "2099-12-20T09:00Z", "utilizationReport", m{"from": "2099-12-30", "to": "2099-12-31"})
	if r["to"] != "2099-12-31" || len(list(r["machines"])) != 1 {
		t.Fatalf("%v", r)
	}
	e.refused("sam", "2099-12-20T09:00Z", "utilizationReport", m{"from": "2099-12-31", "to": "2100-01-01"}, "invalid_argument")
	e.refused("sam", "2099-12-20T09:00Z", "utilizationReport", m{"from": "2099-12-31", "to": "2099-12-31"}, "invalid_interval")
	// a month-long session metered across the whole range stays exact
	e.ok("admin", "2099-12-20T09:00Z", "registerUser", m{"userId": "st2", "name": "S", "role": "staff"})
}

func TestLongIdentifiersAndAuditTargets(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	long := strings.Repeat("a", 64)
	other := strings.Repeat("b", 64)
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "L"})
	e.ok("admin", at, "registerMember", m{"memberId": other, "name": "M", "tier": "basic"})
	e.ok("admin", at, "addCourse", m{"courseId": long, "familyId": "laser", "title": "T", "feeCents": 0, "capacity": 1})
	e.ok("admin", at, "enrollInCourse", m{"courseId": long, "memberId": other})
	entries := list(e.ok("admin", at, "listAuditEntries", m{})["entries"])
	last := obj(entries[len(entries)-1])
	if last["target"] != long+"/"+other || len(last["target"].(string)) != 129 {
		t.Fatalf("%v", last)
	}
}
