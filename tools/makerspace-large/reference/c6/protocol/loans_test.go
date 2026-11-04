package protocol_test

import "testing"

// toolShop is the shop with a waiver for the laser family, signed by ada, a
// tool drill of that family with a fee of 200 cents a day, and staff sam on
// shift for the rest of 2 March.
func toolShop(t *testing.T) *env {
	e := shop(t)
	const at = "2026-03-02T09:00Z"
	e.ok("admin", at, "publishWaiver", m{"familyId": "laser", "text": "Safety"})
	e.ok("sam", at, "signWaiver", m{"memberId": "ada", "familyId": "laser"})
	e.ok("admin", at, "addShift", m{"userId": "sam", "start": at, "end": "2026-03-02T21:00Z"})
	e.ok("admin", at, "addTool", m{"toolId": "drill", "name": "Drill", "familyId": "laser", "lateFeeCentsPerDay": 200})
	return e
}

func TestToolCatalog(t *testing.T) {
	e := newEnv(t)
	at := "2026-03-02T09:00Z"
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "Laser"})
	tool := obj(e.ok("admin", at, "addTool", m{"toolId": "drill", "name": "Drill", "familyId": "laser", "lateFeeCentsPerDay": 200})["tool"])
	assertKeys(t, tool, "toolId", "name", "familyId", "lateFeeCentsPerDay", "onLoan")
	if tool["onLoan"] != false || num(tool["lateFeeCentsPerDay"]) != 200 {
		t.Fatalf("%v", tool)
	}
	e.refused("admin", at, "addTool", m{"toolId": "drill", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 1}, "already_exists")
	e.refused("admin", at, "addTool", m{"toolId": "x", "name": "D", "familyId": "nofam", "lateFeeCentsPerDay": 1}, "not_found")
	e.refused("admin", at, "addTool", m{"toolId": "x", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 0}, "invalid_argument")
	e.refused("admin", at, "addTool", m{"toolId": "x", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 100001}, "invalid_argument")
	e.ok("admin", at, "addTool", m{"toolId": "T10", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 100000})
	e.ok("admin", at, "addTool", m{"toolId": "T9", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 1})
	e.ok("admin", at, "registerMember", m{"memberId": "ada", "name": "Ada", "tier": "basic"})
	e.refused("ada", at, "addTool", m{"toolId": "y", "name": "D", "familyId": "laser", "lateFeeCentsPerDay": 1}, "forbidden")
	var ids []any
	for _, tl := range list(e.ok("ada", at, "listTools", m{})["tools"]) {
		ids = append(ids, obj(tl)["toolId"])
	}
	if len(ids) != 3 || ids[0] != "T10" || ids[1] != "T9" || ids[2] != "drill" {
		t.Fatalf("%v", ids)
	}
}

func TestBorrowAndReturnTool(t *testing.T) {
	e := toolShop(t)
	at := "2026-03-02T10:00Z"
	e.refused("ada", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "forbidden")
	out := e.ok("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	loan, tool := obj(out["loan"]), obj(out["tool"])
	assertKeys(t, loan, "loanId", "toolId", "memberId", "borrowedAt", "dueOn", "returnedAt", "status", "overdue", "chargedDays", "renewals")
	if loan["loanId"] != "lon-1" || loan["borrowedAt"] != at || loan["dueOn"] != "2026-03-16" || loan["returnedAt"] != nil ||
		loan["status"] != "active" || loan["overdue"] != false || num(loan["chargedDays"]) != 0 || num(loan["renewals"]) != 0 || tool["onLoan"] != true {
		t.Fatalf("%v", out)
	}
	e.refused("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "tool_on_loan")
	e.refused("ada", "2026-03-02T11:00Z", "returnTool", m{"loanId": "lon-1"}, "forbidden")
	ret := e.ok("sam", "2026-03-16T23:59Z", "returnTool", m{"loanId": "lon-1"})
	loan, tool = obj(ret["loan"]), obj(ret["tool"])
	if loan["status"] != "returned" || loan["returnedAt"] != "2026-03-16T23:59Z" || loan["overdue"] != false ||
		tool["onLoan"] != false || ret["charge"] != nil {
		t.Fatalf("%v", ret)
	}
	e.refused("sam", "2026-03-16T23:59Z", "returnTool", m{"loanId": "lon-1"}, "invalid_state")
	e.refused("sam", "2026-03-16T23:59Z", "returnTool", m{"loanId": "lon-9"}, "not_found")
	// the tool can be lent again and the counter goes on
	if again := obj(e.ok("admin", "2026-03-17T08:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})["loan"]); again["loanId"] != "lon-2" {
		t.Fatalf("%v", again)
	}
}

func TestBorrowToolRuleOrder(t *testing.T) {
	e := toolShop(t)
	at := "2026-03-02T10:00Z"
	borrow := func(actor, tool string) resp {
		return e.send(actor, at, "borrowTool", m{"memberId": "ada", "toolId": tool})
	}
	e.ok("admin", at, "registerUser", m{"userId": "kim", "name": "Kim", "role": "staff"})
	if r := borrow("kim", "drill"); r["code"] != "approver_not_on_shift" {
		t.Fatalf("%v", r)
	}
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	e.refused("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "member_suspended")
	e.refused("kim", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "approver_not_on_shift")
	e.ok("sam", at, "setMemberStanding", m{"memberId": "ada", "standing": "good"})
	// an admin needs no shift; certification and waiver are judged for the family of the tool
	e.ok("sam", at, "revokeCertification", m{"memberId": "ada", "familyId": "laser"})
	e.refused("admin", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "not_certified")
	e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("sam", at, "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("admin", at, "publishWaiver", m{"familyId": "laser", "text": "Safety, new version"})
	e.refused("admin", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "waiver_missing")
	e.ok("sam", at, "signWaiver", m{"memberId": "ada", "familyId": "laser"})
	e.ok("admin", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	// no PPE, lockout or maintenance rule of machines applies to a tool
	e.ok("admin", at, "applyLockout", m{"machineId": "lc1", "reason": "jam"})
	e.ok("admin", at, "addTool", m{"toolId": "saw", "name": "Saw", "familyId": "laser", "lateFeeCentsPerDay": 50})
	e.ok("admin", at, "borrowTool", m{"memberId": "ada", "toolId": "saw"})
}

func TestBorrowToolLimitAndOverdueOrder(t *testing.T) {
	e := toolShop(t)
	at := "2026-03-02T10:00Z"
	for _, id := range []string{"t2", "t3", "t4"} {
		e.ok("admin", at, "addTool", m{"toolId": id, "name": "T", "familyId": "laser", "lateFeeCentsPerDay": 100})
	}
	for _, id := range []string{"drill", "t2", "t3"} {
		e.ok("sam", at, "borrowTool", m{"memberId": "ada", "toolId": id})
	}
	e.refused("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "t4"}, "loan_limit")
	// a returned loan frees a place, and a tool on loan is refused before the limit
	e.ok("sam", at, "returnTool", m{"loanId": "lon-1"})
	e.refused("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "t3"}, "tool_on_loan")
	e.ok("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "t4"})
	// from 2026-03-17 lon-2 and lon-3 are overdue: that refusal comes before the limit
	late := "2026-03-17T00:00Z"
	e.ok("admin", late, "addShift", m{"userId": "sam", "start": "2026-03-17T00:00Z", "end": "2026-03-17T12:00Z"})
	e.refused("sam", late, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "loan_overdue")
	// a missing certification is refused before the overdue rule
	e.ok("sam", late, "revokeCertification", m{"memberId": "ada", "familyId": "laser"})
	e.refused("sam", late, "borrowTool", m{"memberId": "ada", "toolId": "drill"}, "not_certified")
}

func TestOverdueLoansAndLateFeeCharges(t *testing.T) {
	e := toolShop(t)
	e.ok("sam", "2026-03-02T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	loanOn := func(at string) map[string]any {
		return obj(list(e.ok("ada", at, "listMemberLoans", m{"memberId": "ada"})["loans"])[0])
	}
	if got := list(e.ok("sam", "2026-03-16T12:00Z", "chargeOverdueLoans", m{})["charges"]); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	if l := loanOn("2026-03-16T23:59Z"); l["overdue"] != false {
		t.Fatalf("the loan is not overdue on its due date: %v", l)
	}
	if l := loanOn("2026-03-17T00:00Z"); l["overdue"] != true {
		t.Fatalf("%v", l)
	}
	e.refused("ada", "2026-03-19T09:00Z", "chargeOverdueLoans", m{}, "forbidden")
	got := list(e.ok("sam", "2026-03-19T09:00Z", "chargeOverdueLoans", m{})["charges"])
	if len(got) != 1 {
		t.Fatalf("%v", got)
	}
	ch := obj(got[0])
	assertKeys(t, ch, "kind", "ref", "quantity", "unitPriceCents", "discountPercent", "amountCents", "at")
	if ch["kind"] != "lateFee" || ch["ref"] != "lon-1" || num(ch["quantity"]) != 3 || num(ch["unitPriceCents"]) != 200 ||
		num(ch["discountPercent"]) != 0 || num(ch["amountCents"]) != 600 || ch["at"] != "2026-03-19T09:00Z" {
		t.Fatalf("%v", ch)
	}
	if l := loanOn("2026-03-19T09:00Z"); num(l["chargedDays"]) != 3 {
		t.Fatalf("%v", l)
	}
	// the same day charges nothing more, the next day one day
	if got := list(e.ok("sam", "2026-03-19T18:00Z", "chargeOverdueLoans", m{})["charges"]); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	got = list(e.ok("sam", "2026-03-20T09:00Z", "chargeOverdueLoans", m{})["charges"])
	if len(got) != 1 || num(obj(got[0])["quantity"]) != 1 || num(obj(got[0])["amountCents"]) != 200 {
		t.Fatalf("%v", got)
	}
	// the return charges the remaining two days
	ret := e.ok("sam", "2026-03-22T09:00Z", "returnTool", m{"loanId": "lon-1"})
	ch = obj(ret["charge"])
	if ch["kind"] != "lateFee" || ch["ref"] != "lon-1" || num(ch["quantity"]) != 2 || num(ch["amountCents"]) != 400 {
		t.Fatalf("%v", ret)
	}
	if l := obj(ret["loan"]); num(l["chargedDays"]) != 6 || l["overdue"] != false {
		t.Fatalf("%v", l)
	}
	if got := list(e.ok("sam", "2026-03-23T09:00Z", "chargeOverdueLoans", m{})["charges"]); len(got) != 0 {
		t.Fatalf("a returned loan is not charged: %v", got)
	}
	// the charges are unbilled charges of the member: the course fee and 600 + 200 + 400 cents
	acc := obj(e.ok("ada", "2026-03-23T09:00Z", "getAccount", m{"memberId": "ada"})["account"])
	if num(acc["unbilledCents"]) != 3000+1200 {
		t.Fatalf("%v", acc)
	}
}

func TestChargeOverdueLoansOrderAndNotice(t *testing.T) {
	e := toolShop(t)
	const at = "2026-03-02T10:00Z"
	e.ok("admin", at, "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("admin", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.ok("admin", at, "completeCourse", m{"courseId": "laser-101", "memberId": "bob"})
	e.ok("sam", at, "signWaiver", m{"memberId": "bob", "familyId": "laser"})
	for _, id := range []string{"t2", "t3"} {
		e.ok("admin", at, "addTool", m{"toolId": id, "name": "T", "familyId": "laser", "lateFeeCentsPerDay": 100})
	}
	// ten loans are not needed to see the numeric order: lon-2 is charged before lon-10 would be
	e.ok("sam", at, "borrowTool", m{"memberId": "bob", "toolId": "t3"})
	e.ok("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	e.ok("sam", at, "borrowTool", m{"memberId": "ada", "toolId": "t2"})
	got := list(e.ok("sam", "2026-03-18T09:00Z", "chargeOverdueLoans", m{})["charges"])
	if len(got) != 3 || obj(got[0])["ref"] != "lon-1" || obj(got[1])["ref"] != "lon-2" || obj(got[2])["ref"] != "lon-3" {
		t.Fatalf("%v", got)
	}
	notices := list(e.ok("ada", "2026-03-18T09:00Z", "listNotices", m{"memberId": "ada"})["notices"])
	if len(notices) != 2 {
		t.Fatalf("%v", notices)
	}
	n := obj(notices[0])
	if n["kind"] != "loan" || n["ref"] != "lon-2" || n["subject"] != "Tool loan lon-2 is overdue" || n["at"] != "2026-03-18T09:00Z" {
		t.Fatalf("%v", n)
	}
	// only the first charge of a loan posts a notice, and returning posts none
	e.ok("sam", "2026-03-19T09:00Z", "chargeOverdueLoans", m{})
	e.ok("sam", "2026-03-19T09:00Z", "returnTool", m{"loanId": "lon-2"})
	if notices = list(e.ok("ada", "2026-03-19T09:00Z", "listNotices", m{"memberId": "ada"})["notices"]); len(notices) != 2 {
		t.Fatalf("%v", notices)
	}
	// standing is not checked
	e.ok("sam", "2026-03-19T09:00Z", "setMemberStanding", m{"memberId": "bob", "standing": "suspended"})
	if got = list(e.ok("sam", "2026-03-20T09:00Z", "chargeOverdueLoans", m{})["charges"]); len(got) != 2 {
		t.Fatalf("%v", got)
	}
}

func TestRenewLoan(t *testing.T) {
	e := toolShop(t)
	e.ok("admin", "2026-03-02T09:00Z", "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", "2026-03-02T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	renew := func(actor, at string) resp { return e.send(actor, at, "renewLoan", m{"loanId": "lon-1"}) }
	if r := e.send("bob", "2026-03-10T10:00Z", "renewLoan", m{"loanId": "lon-1"}); r["code"] != "forbidden" {
		t.Fatalf("%v", r)
	}
	e.refused("ada", "2026-03-10T10:00Z", "renewLoan", m{"loanId": "lon-9"}, "not_found")
	// on the due date the loan is not overdue; the renewal counts from the old due date
	out := e.ok("ada", "2026-03-16T23:59Z", "renewLoan", m{"loanId": "lon-1"})
	loan := obj(out["loan"])
	assertKeys(t, out, "loan")
	if loan["dueOn"] != "2026-03-30" || num(loan["renewals"]) != 1 || loan["status"] != "active" || loan["overdue"] != false ||
		num(loan["chargedDays"]) != 0 {
		t.Fatalf("%v", loan)
	}
	if got := list(e.ok("sam", "2026-03-20T09:00Z", "chargeOverdueLoans", m{})["charges"]); len(got) != 0 {
		t.Fatalf("a renewed loan is judged by its new due date: %v", got)
	}
	// staff and admin may renew, no shift is needed (sam has none on 21 March), and standing is judged on the loan's member
	e.ok("admin", "2026-03-21T10:00Z", "setMemberStanding", m{"memberId": "ada", "standing": "suspended"})
	if r := renew("sam", "2026-03-21T10:00Z"); r["code"] != "member_suspended" {
		t.Fatalf("%v", r)
	}
	e.ok("admin", "2026-03-21T10:00Z", "setMemberStanding", m{"memberId": "ada", "standing": "good"})
	if l := obj(e.ok("admin", "2026-03-30T08:00Z", "renewLoan", m{"loanId": "lon-1"})["loan"]); l["dueOn"] != "2026-04-13" || num(l["renewals"]) != 2 {
		t.Fatalf("%v", l)
	}
	if r := renew("ada", "2026-03-30T09:00Z"); r["code"] != "renewal_limit" {
		t.Fatalf("%v", r)
	}
	// an overdue loan is refused before the limit
	if r := renew("ada", "2026-04-14T00:00Z"); r["code"] != "loan_overdue" {
		t.Fatalf("%v", r)
	}
	e.ok("sam", "2026-04-14T00:00Z", "returnTool", m{"loanId": "lon-1"})
	if r := renew("ada", "2026-04-14T01:00Z"); r["code"] != "invalid_state" {
		t.Fatalf("%v", r)
	}
}

func TestRenewLoanOverdueRules(t *testing.T) {
	e := toolShop(t)
	e.ok("admin", "2026-03-02T09:00Z", "addTool", m{"toolId": "saw", "name": "Saw", "familyId": "laser", "lateFeeCentsPerDay": 100})
	e.ok("sam", "2026-03-02T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	e.ok("admin", "2026-03-10T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "saw"})
	// lon-1 is overdue on 17 March, lon-2 is not: only the overdue loan itself blocks its renewal
	if r := e.send("ada", "2026-03-17T00:00Z", "renewLoan", m{"loanId": "lon-1"}); r["code"] != "loan_overdue" {
		t.Fatalf("%v", r)
	}
	if l := obj(e.ok("ada", "2026-03-17T00:00Z", "renewLoan", m{"loanId": "lon-2"})["loan"]); l["dueOn"] != "2026-04-07" {
		t.Fatalf("%v", l)
	}
}

func TestOverdueLoanBlocksCheckInOnly(t *testing.T) {
	e := toolShop(t)
	e.ok("sam", "2026-03-02T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	book(e, "ada", "2026-03-02T10:00Z", "2026-03-16T10:00Z", "2026-03-16T11:00Z")
	e.ok("ada", "2026-03-16T10:00Z", "checkIn", m{"reservationId": "res-1"})
	e.ok("ada", "2026-03-16T10:30Z", "checkOut", m{"reservationId": "res-1"})
	book(e, "ada", "2026-03-16T11:00Z", "2026-03-17T10:00Z", "2026-03-17T11:00Z")
	// on the day after the due date the reservation can be made, checking in is refused
	e.refused("ada", "2026-03-17T10:00Z", "checkIn", m{"reservationId": "res-2"}, "loan_overdue")
	book(e, "ada", "2026-03-17T10:00Z", "2026-03-18T10:00Z", "2026-03-18T11:00Z")
	e.ok("ada", "2026-03-17T10:00Z", "cancelReservation", m{"reservationId": "res-3"})
	// the tool back, checking in works again
	e.ok("sam", "2026-03-17T10:30Z", "returnTool", m{"loanId": "lon-1"})
	e.ok("ada", "2026-03-17T10:31Z", "checkIn", m{"reservationId": "res-2"})
	e.ok("ada", "2026-03-17T10:40Z", "checkOut", m{"reservationId": "res-2"})
}

func TestMemberLoansListAndOwnership(t *testing.T) {
	e := toolShop(t)
	e.ok("admin", "2026-03-02T09:00Z", "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.ok("sam", "2026-03-02T10:00Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	e.refused("bob", "2026-03-02T10:00Z", "listMemberLoans", m{"memberId": "ada"}, "forbidden")
	e.ok("sam", "2026-03-02T10:00Z", "returnTool", m{"loanId": "lon-1"})
	e.ok("sam", "2026-03-02T10:30Z", "borrowTool", m{"memberId": "ada", "toolId": "drill"})
	out := e.ok("ada", "2026-03-02T11:00Z", "listMemberLoans", m{"memberId": "ada"})
	loans := list(out["loans"])
	if out["memberId"] != "ada" || len(loans) != 2 || obj(loans[0])["status"] != "returned" || obj(loans[1])["loanId"] != "lon-2" {
		t.Fatalf("%v", out)
	}
	if empty := list(e.ok("bob", "2026-03-02T11:00Z", "listMemberLoans", m{"memberId": "bob"})["loans"]); len(empty) != 0 {
		t.Fatalf("%v", empty)
	}
}
