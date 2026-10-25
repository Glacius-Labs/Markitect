package ms_test

import "testing"

// overdueShop has an open invoice for ada, issued 2026-04-01T08:00Z, due 2026-04-15T08:00Z.
func overdueShop(t *testing.T) *env {
	e := usageMonth(t)
	e.ok("sam", "2026-03-20T09:00Z", "addMaterial", m{"materialId": "a", "name": "A", "unitPriceCents": 100})
	e.ok("sam", "2026-03-20T09:00Z", "restockMaterial", m{"materialId": "a", "quantity": 50})
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	return e
}

func tryBook(e *env, at, start, end string) resp {
	return e.send("sam", at, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": start, "end": end})
}

func tryBuy(e *env, at string) resp {
	return e.send("ada", at, "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 1})
}

func blockedAt(e *env, at string) bool {
	return obj(e.ok("ada", at, "getAccount", m{"memberId": "ada"})["account"])["blocked"].(bool)
}

func TestBlockBoundary(t *testing.T) {
	e := overdueShop(t)
	// due 2026-04-15T08:00Z; due plus 30 days is 2026-05-15T08:00Z
	if r := tryBook(e, "2026-05-15T08:00Z", "2026-05-18T10:00Z", "2026-05-18T11:00Z"); r["status"] != "ok" {
		t.Fatalf("not yet blocked: %v", r)
	}
	if blockedAt(e, "2026-05-15T08:00Z") {
		t.Fatal("blocked at exactly due plus 30 days")
	}
	if r := tryBook(e, "2026-05-15T08:01Z", "2026-05-19T10:00Z", "2026-05-19T11:00Z"); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	if r := tryBuy(e, "2026-05-15T08:01Z"); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	if !blockedAt(e, "2026-05-15T08:01Z") {
		t.Fatal("not blocked one minute later")
	}
}

func TestBlockAppliesWhoeverActsAndOnlyToReservationsAndPurchases(t *testing.T) {
	e := overdueShop(t)
	at := "2026-06-01T09:00Z"
	if r := e.send("ada", at, "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-06-02T10:00Z", "end": "2026-06-02T11:00Z"}); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	if r := tryBook(e, at, "2026-06-02T10:00Z", "2026-06-02T11:00Z"); r["code"] != "account_overdue" {
		t.Fatalf("staff acting: %v", r)
	}
	if r := e.send("admin", at, "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 1}); r["code"] != "account_overdue" {
		t.Fatalf("admin acting: %v", r)
	}
	// the standing stays good and nothing else is blocked
	if mem := obj(e.ok("ada", at, "getMember", m{"memberId": "ada"})["member"]); mem["standing"] != "good" {
		t.Fatalf("%v", mem)
	}
	e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("sam", at, "generateInvoices", m{"month": "2026-04"})
	e.ok("ada", at, "recordPayment", m{"invoiceId": "inv-2", "amountCents": 1})
}

func TestBlockRuleOrder(t *testing.T) {
	e := overdueShop(t)
	at := "2026-06-01T09:00Z"
	// account_overdue comes after member_suspended and before not_certified / insufficient_stock
	e.ok("admin", at, "addFamily", m{"familyId": "cnc", "name": "CNC"})
	e.ok("admin", at, "addMachine", m{"machineId": "cnc1", "name": "C", "familyId": "cnc", "rateCentsPerMinute": 5})
	if r := e.send("sam", at, "createReservation", m{"memberId": "ada", "machineId": "cnc1", "start": "2026-06-02T10:00Z", "end": "2026-06-02T11:00Z"}); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	if r := e.send("sam", at, "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 1000}); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	if r := tryBook(e, at, "2026-06-02T10:00Z", "2026-06-02T11:00Z"); r["code"] != "member_suspended" {
		t.Fatalf("%v", r)
	}
	if r := tryBuy(e, at); r["code"] != "member_suspended" {
		t.Fatalf("%v", r)
	}
	// slot and opening-hours rules come first
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "good"})
	if r := tryBook(e, at, "2026-06-02T10:10Z", "2026-06-02T11:00Z"); r["code"] != "invalid_slot" {
		t.Fatalf("%v", r)
	}
}

func TestExistingReservationsAndCheckInSurviveTheBlock(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-02T09:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-06-02T10:00Z", "end": "2026-06-02T11:00Z"})
	e.ok("sam", "2026-03-02T09:00Z", "createReservation", m{"memberId": "ada", "machineId": "lc1", "start": "2026-06-02T12:00Z", "end": "2026-06-02T13:00Z"})
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	at := "2026-06-02T10:00Z" // more than 30 days after the due date 2026-04-15
	if !blockedAt(e, at) {
		t.Fatal("not blocked")
	}
	e.ok("ada", at, "checkIn", m{"reservationId": "res-1"})
	e.ok("ada", "2026-06-02T10:20Z", "checkOut", m{"reservationId": "res-1"})
	e.ok("ada", "2026-06-02T10:20Z", "cancelReservation", m{"reservationId": "res-2"})
}

func TestBlockEndsOnlyWhenPaid(t *testing.T) {
	e := overdueShop(t)
	at := "2026-06-01T09:00Z"
	// a partial payment does not end the block, nor does credit
	e.ok("ada", at, "recordPayment", m{"invoiceId": "inv-1", "amountCents": 100})
	e.ok("admin", at, "issueCredit", m{"memberId": "ada", "amountCents": 100000, "reason": "x"})
	if !blockedAt(e, at) {
		t.Fatal("block ended before payment")
	}
	if r := tryBook(e, at, "2026-06-02T10:00Z", "2026-06-02T11:00Z"); r["code"] != "account_overdue" {
		t.Fatalf("%v", r)
	}
	e.ok("ada", at, "recordPayment", m{"invoiceId": "inv-1", "amountCents": 9299})
	if blockedAt(e, at) {
		t.Fatal("still blocked after payment")
	}
	if r := tryBook(e, at, "2026-06-02T10:00Z", "2026-06-02T11:00Z"); r["status"] != "ok" {
		t.Fatalf("%v", r)
	}
	if r := tryBuy(e, at); r["status"] != "ok" {
		t.Fatalf("%v", r)
	}
}

func TestOneBlockingInvoiceIsEnough(t *testing.T) {
	e := overdueShop(t)
	e.ok("ada", "2026-04-02T09:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 9399})
	e.ok("sam", "2026-05-01T08:00Z", "generateInvoices", m{"month": "2026-04"}) // inv-2: due 2026-05-15T08:00Z
	// the overdue flag and balance keep their meaning
	acc := obj(e.ok("ada", "2026-05-15T08:00Z", "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["overdueBalanceCents"]) != 4900 || acc["blocked"] != false {
		t.Fatalf("%v", acc)
	}
	if blockedAt(e, "2026-06-14T08:00Z") {
		t.Fatal("blocked before the second invoice is 30 days overdue")
	}
	if !blockedAt(e, "2026-06-14T08:01Z") {
		t.Fatal("not blocked after the second invoice is 30 days overdue")
	}
}
