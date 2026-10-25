package ms_test

import "testing"

func TestMaterials(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	mat := obj(e.ok("sam", now, "addMaterial", m{"materialId": "ply", "name": "Plywood", "unitPriceCents": 199})["material"])
	assertKeys(t, mat, "materialId", "name", "unitPriceCents", "stock")
	if num(mat["stock"]) != 0 {
		t.Fatalf("%v", mat)
	}
	e.refused("sam", now, "addMaterial", m{"materialId": "ply", "name": "x", "unitPriceCents": 1}, "already_exists")
	e.refused("sam", now, "addMaterial", m{"materialId": "x", "name": "x", "unitPriceCents": 0}, "invalid_argument")
	e.refused("sam", now, "restockMaterial", m{"materialId": "ply", "quantity": 0}, "invalid_argument")
	e.refused("sam", now, "restockMaterial", m{"materialId": "none", "quantity": 1}, "not_found")
	if s := obj(e.ok("sam", now, "restockMaterial", m{"materialId": "ply", "quantity": 10})["material"]); num(s["stock"]) != 10 {
		t.Fatalf("%v", s)
	}
	e.refused("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 11}, "insufficient_stock")
	e.refused("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 10001}, "invalid_argument")
	out := e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 3})
	p := obj(out["purchase"])
	assertKeys(t, p, "purchaseId", "memberId", "materialId", "quantity", "unitPriceCents", "discountPercent", "amountCents", "at")
	if p["purchaseId"] != "pur-1" || num(p["amountCents"]) != 597 || num(p["discountPercent"]) != 0 || num(obj(out["material"])["stock"]) != 7 {
		t.Fatalf("%v", out)
	}
	// the member's tier plays no part in material prices; the purchase can take the entire stock
	e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 7})
	e.refused("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 1}, "insufficient_stock")
	e.ok("sam", now, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	e.refused("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 1}, "member_suspended")
	e.ok("sam", now, "addMaterial", m{"materialId": "Acrylic", "name": "A", "unitPriceCents": 1})
	ml := list(e.ok("ada", now, "listMaterials", m{})["materials"])
	if len(ml) != 2 || obj(ml[0])["materialId"] != "Acrylic" || obj(ml[1])["materialId"] != "ply" {
		t.Fatalf("%v", ml)
	}
}

func TestMaterialChargeRounding(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "addMaterial", m{"materialId": "a", "name": "A", "unitPriceCents": 1})
	e.ok("sam", now, "restockMaterial", m{"materialId": "a", "quantity": 100})
	e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 1})
	acc := obj(e.ok("ada", now, "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["unbilledCents"]) != 3001 { // course 3000 + material 1
		t.Fatalf("%v", acc)
	}
}

// usageMonth gives ada one usage charge of 1499 cents in March 2026.
func usageMonth(t *testing.T) *env {
	e := shop(t)
	book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
	e.ok("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"})
	e.ok("ada", "2026-03-03T10:37Z", "checkOut", m{"reservationId": "res-1"})
	return e
}

func TestInvoiceGeneration(t *testing.T) {
	e := usageMonth(t)
	e.refused("sam", "2026-03-31T23:59Z", "generateInvoices", m{"month": "2026-03"}, "month_not_ended")
	e.refused("sam", "2026-03-31T23:59Z", "generateInvoices", m{"month": "2026-04"}, "month_not_ended")
	out := e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	if out["month"] != "2026-03" || len(list(out["invoices"])) != 1 {
		t.Fatalf("%v", out)
	}
	inv := obj(list(out["invoices"])[0])
	assertKeys(t, inv, "invoiceId", "memberId", "month", "issuedAt", "dueAt", "lines", "totalCents",
		"creditAppliedCents", "paidCents", "balanceCents", "status", "overdue")
	// membership 4900 + course 3000 + usage 1499
	if inv["invoiceId"] != "inv-1" || inv["issuedAt"] != "2026-04-01T08:00Z" || inv["dueAt"] != "2026-04-15T08:00Z" ||
		num(inv["totalCents"]) != 9399 || num(inv["balanceCents"]) != 9399 || inv["status"] != "open" || inv["overdue"] != false {
		t.Fatalf("%v", inv)
	}
	lines := list(inv["lines"])
	if len(lines) != 3 {
		t.Fatalf("%v", lines)
	}
	first := obj(lines[0])
	if first["kind"] != "membership" || first["ref"] != "plus" || num(first["quantity"]) != 1 ||
		num(first["unitPriceCents"]) != 4900 || num(first["discountPercent"]) != 0 || first["at"] != "2026-04-01T08:00Z" {
		t.Fatalf("%v", first)
	}
	if obj(lines[1])["kind"] != "course" || obj(lines[2])["kind"] != "usage" || num(obj(lines[2])["amountCents"]) != 1499 {
		t.Fatalf("%v", lines)
	}
	e.refused("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"}, "already_invoiced")
	acc := obj(e.ok("ada", "2026-04-01T08:00Z", "getAccount", m{"memberId": "ada"})["account"])
	assertKeys(t, acc, "memberId", "creditBalanceCents", "openBalanceCents", "overdueBalanceCents", "unbilledCents", "blocked")
	if num(acc["openBalanceCents"]) != 9399 || num(acc["unbilledCents"]) != 0 {
		t.Fatalf("%v", acc)
	}
}

func TestInvoicesForAllMembersAndEmptyMonths(t *testing.T) {
	e := newEnv(t)
	e.ok("admin", "2026-03-10T09:00Z", "registerMember", m{"memberId": "zed", "name": "Z", "tier": "pro"})
	e.ok("admin", "2026-03-10T09:00Z", "registerMember", m{"memberId": "abe", "name": "A", "tier": "basic"})
	e.ok("admin", "2026-03-10T09:00Z", "setMemberStanding", m{"memberId": "zed", "standing": "suspended"})
	// no member before February: February is invoiced with no invoices
	out := e.ok("admin", "2026-03-31T23:59Z", "generateInvoices", m{"month": "2026-02"})
	if len(list(out["invoices"])) != 0 {
		t.Fatalf("%v", out)
	}
	e.refused("admin", "2026-03-31T23:59Z", "generateInvoices", m{"month": "2026-02"}, "already_invoiced")
	// a member registered in March gets March's invoice, suspended or not, in memberId order
	out = e.ok("admin", "2026-04-01T00:00Z", "generateInvoices", m{"month": "2026-03"})
	inv := list(out["invoices"])
	if len(inv) != 2 || obj(inv[0])["memberId"] != "abe" || obj(inv[0])["invoiceId"] != "inv-1" ||
		obj(inv[1])["memberId"] != "zed" || num(obj(inv[1])["totalCents"]) != 9900 {
		t.Fatalf("%v", inv)
	}
	// a member registered later is not invoiced for March
	e.ok("admin", "2026-04-02T00:00Z", "registerMember", m{"memberId": "new", "name": "N", "tier": "basic"})
	// months can be generated in any order; the member list is that of the month's end
	out = e.ok("admin", "2026-04-02T00:00Z", "generateInvoices", m{"month": "2026-01"})
	if len(list(out["invoices"])) != 0 {
		t.Fatalf("%v", out)
	}
}

func TestChargesBelongToTheirMonth(t *testing.T) {
	e := usageMonth(t)
	// a charge from April is not on the March invoice
	e.ok("sam", "2026-04-02T09:00Z", "addMaterial", m{"materialId": "a", "name": "A", "unitPriceCents": 100})
	e.ok("sam", "2026-04-02T09:00Z", "restockMaterial", m{"materialId": "a", "quantity": 5})
	e.ok("ada", "2026-04-02T09:00Z", "purchaseMaterial", m{"memberId": "ada", "materialId": "a", "quantity": 2})
	out := e.ok("sam", "2026-04-02T10:00Z", "generateInvoices", m{"month": "2026-03"})
	if n := len(list(obj(list(out["invoices"])[0])["lines"])); n != 3 {
		t.Fatalf("lines: %d", n)
	}
	acc := obj(e.ok("ada", "2026-04-02T10:00Z", "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["unbilledCents"]) != 200 {
		t.Fatalf("%v", acc)
	}
	out = e.ok("sam", "2026-05-01T00:00Z", "generateInvoices", m{"month": "2026-04"})
	lines := list(obj(list(out["invoices"])[0])["lines"])
	if len(lines) != 2 || obj(lines[1])["kind"] != "material" || obj(lines[1])["ref"] != "pur-1" {
		t.Fatalf("%v", lines)
	}
}

func TestPaymentsAndCredit(t *testing.T) {
	e := usageMonth(t)
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	out := e.ok("ada", "2026-04-02T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 4000})
	pay := obj(out["payment"])
	assertKeys(t, pay, "paymentId", "invoiceId", "memberId", "amountCents", "appliedCents", "creditedCents", "at")
	inv := obj(out["invoice"])
	if pay["paymentId"] != "pay-1" || num(pay["appliedCents"]) != 4000 || num(pay["creditedCents"]) != 0 ||
		num(inv["paidCents"]) != 4000 || num(inv["balanceCents"]) != 5399 || inv["status"] != "open" || num(out["creditBalanceCents"]) != 0 {
		t.Fatalf("%v", out)
	}
	assertKeys(t, out, "payment", "invoice", "creditBalanceCents")
	// an overpayment, made by staff, credits the invoice's member
	out = e.ok("sam", "2026-04-03T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 6000})
	pay, inv = obj(out["payment"]), obj(out["invoice"])
	if pay["memberId"] != "ada" || num(pay["appliedCents"]) != 5399 || num(pay["creditedCents"]) != 601 ||
		inv["status"] != "paid" || num(inv["balanceCents"]) != 0 || num(out["creditBalanceCents"]) != 601 {
		t.Fatalf("%v", out)
	}
	e.refused("ada", "2026-04-03T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 1}, "invoice_paid")
	e.refused("ada", "2026-04-03T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 0}, "invalid_argument")
	// credit is not applied to an existing invoice; it reduces the next invoice
	cr := e.ok("admin", "2026-04-04T08:00Z", "issueCredit", m{"memberId": "ada", "amountCents": 10000, "reason": "goodwill"})
	assertKeys(t, cr, "memberId", "amountCents", "creditBalanceCents")
	if num(cr["creditBalanceCents"]) != 10601 {
		t.Fatalf("%v", cr)
	}
	e.refused("sam", "2026-04-04T08:00Z", "issueCredit", m{"memberId": "ada", "amountCents": 1, "reason": "x"}, "forbidden")
	next := obj(list(e.ok("sam", "2026-05-01T08:00Z", "generateInvoices", m{"month": "2026-04"})["invoices"])[0])
	if num(next["totalCents"]) != 4900 || num(next["creditAppliedCents"]) != 4900 || num(next["balanceCents"]) != 0 || next["status"] != "paid" {
		t.Fatalf("%v", next)
	}
	acc := obj(e.ok("ada", "2026-05-01T08:00Z", "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["creditBalanceCents"]) != 10601-4900 {
		t.Fatalf("%v", acc)
	}
	// a paid invoice takes no further payment
	e.refused("ada", "2026-05-01T08:00Z", "recordPayment", m{"invoiceId": "inv-2", "amountCents": 1}, "invoice_paid")
}

func TestPartialCreditAndPaymentLimits(t *testing.T) {
	e := usageMonth(t)
	e.ok("admin", "2026-03-20T08:00Z", "issueCredit", m{"memberId": "ada", "amountCents": 1000, "reason": "x"})
	inv := obj(list(e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})["invoices"])[0])
	if num(inv["creditAppliedCents"]) != 1000 || num(inv["balanceCents"]) != 8399 || num(inv["paidCents"]) != 0 {
		t.Fatalf("%v", inv)
	}
	e.refused("ada", "2026-04-01T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 100000001}, "invalid_argument")
	e.ok("ada", "2026-04-01T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 100000000})
}

func TestOverdueFlagAndBalances(t *testing.T) {
	e := usageMonth(t)
	e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})
	overdue := func(at string) (any, int) {
		inv := obj(list(e.ok("ada", at, "listInvoices", m{"memberId": "ada"})["invoices"])[0])
		acc := obj(e.ok("ada", at, "getAccount", m{"memberId": "ada"})["account"])
		if inv["overdue"] != (num(acc["overdueBalanceCents"]) > 0) {
			t.Fatalf("flag and balance disagree: %v %v", inv, acc)
		}
		return inv["overdue"], num(acc["openBalanceCents"])
	}
	if o, open := overdue("2026-04-15T07:59Z"); o != false || open != 9399 {
		t.Fatal("overdue one minute early")
	}
	if o, _ := overdue("2026-04-15T08:00Z"); o != true {
		t.Fatal("overdue at the due time")
	}
	// paying after the due date is allowed and clears the flag
	e.ok("ada", "2026-04-20T08:00Z", "recordPayment", m{"invoiceId": "inv-1", "amountCents": 9399})
	if o, _ := overdue("2026-04-20T08:00Z"); o != false {
		t.Fatal("paid invoice is not overdue")
	}
}

func TestListInvoicesOrderedByMonth(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-06-01T00:00Z", "generateInvoices", m{"month": "2026-05"})
	e.ok("sam", "2026-06-01T00:00Z", "generateInvoices", m{"month": "2026-03"})
	e.ok("sam", "2026-06-01T00:00Z", "generateInvoices", m{"month": "2026-04"})
	inv := list(e.ok("ada", "2026-06-01T00:00Z", "listInvoices", m{"memberId": "ada"})["invoices"])
	if len(inv) != 3 || obj(inv[0])["month"] != "2026-03" || obj(inv[2])["month"] != "2026-05" {
		t.Fatalf("%v", inv)
	}
}

func TestUsageDiscountByTier(t *testing.T) {
	for tier, want := range map[string]int{"basic": 1665, "plus": 1499, "pro": 1249} {
		e := shop(t)
		e.ok("sam", "2026-03-02T09:00Z", "changeTier", m{"memberId": "ada", "tier": tier})
		book(e, "ada", "2026-03-02T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z")
		e.ok("ada", "2026-03-03T10:00Z", "checkIn", m{"reservationId": "res-1"})
		out := e.ok("ada", "2026-03-03T10:37Z", "checkOut", m{"reservationId": "res-1"})
		if got := num(obj(out["charge"])["amountCents"]); got != want {
			t.Errorf("%s: %d, want %d", tier, got, want)
		}
	}
}

func TestTierChangeAppliesToLaterInvoices(t *testing.T) {
	e := usageMonth(t)
	e.ok("sam", "2026-03-31T10:00Z", "changeTier", m{"memberId": "ada", "tier": "pro"})
	inv := obj(list(e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})["invoices"])[0])
	first := obj(list(inv["lines"])[0])
	if first["ref"] != "pro" || num(first["amountCents"]) != 9900 || num(obj(list(inv["lines"])[2])["amountCents"]) != 1499 {
		t.Fatalf("%v", inv)
	}
}
