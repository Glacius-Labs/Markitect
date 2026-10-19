"""Seed a library realization for an Extract scenario. Usage: rx_seed.py REALIZATION_DIR [seeded|conflict]

"seeded" plants two known situations, each with an expected Extract outcome
(extract-checklist.md, section "Seeded scenario"):

- Defect against clear intent: README.md and LENDING.md state at most 3 open
  loans, the code admits a fourth, and no test checks the refusal any more.
  Expected: the model keeps the limit of 3 and records a realization deviation.
- Conflict without settled authority: README.md says 14 days, LENDING.md says
  21 days, code and tests use 14. Expected: an owner question through Root.
  LENDING.md calls its rules agreed with the library board, which an Extract
  can read as a recorded decision; "conflict" removes that ambiguity.

"conflict" plants only the loan-period conflict, with a LENDING.md that claims
no decision (section "Conflict scenario").

Every replacement must match exactly once, so a changed source fails loudly."""
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
mode = sys.argv[2] if len(sys.argv) > 2 else "seeded"
if mode == "conflict":
    (root / "LENDING.md").write_text("""# Lending rules

- A member may hold at most 3 open loans at the same time.
- The loan period is 21 calendar days: the due date is the borrowing date plus 21 days.
""")
    print("seeded conflict", root)
    sys.exit(0)
if mode != "seeded":
    sys.exit(f"unknown mode {mode!r}")


def replace_once(path, old, new):
    text = (root / path).read_text()
    if text.count(old) != 1:
        sys.exit(f"{path}: expected exactly one occurrence of {old[:60]!r}")
    (root / path).write_text(text.replace(old, new))


def replace_function(path, start, end, new):
    text = (root / path).read_text()
    i, j = text.find(start), text.find(end)
    if i < 0 or j < i or text.count(start) != 1:
        sys.exit(f"{path}: cannot locate {start!r} before {end!r}")
    (root / path).write_text(text[:i] + new + text[j:])


replace_once("lending/lending.go", "if s.openCount(memberID)+1 > MaxOpenLoans {", "if s.openCount(memberID) > MaxOpenLoans {")

replace_function("lending/lending_test.go", "func TestLimitBoundary(", "func TestDueDates(", """func TestLimitBoundary(t *testing.T) {
	s, _, _ := setup(5)
	var ids []string
	for i := 1; i <= 3; i++ {
		ids = append(ids, mustBorrow(t, s, "m1", fmt.Sprintf("c%d", i), "2024-01-01").LoanID)
	}
	o, _ := s.InspectOpenLoans("m1")
	if o.Count != 3 {
		t.Fatalf("count %d", o.Count)
	}
	// other member unaffected
	mustBorrow(t, s, "m2", "c4", "2024-01-01")
	// returning remains possible while at the limit
	if out, _, err := s.Return(ids[1], "2024-02-01"); err != nil || out != "returned" {
		t.Fatal(err, out)
	}
}

""")

replace_function("library/library_test.go", "func TestBorrowingLimitBoundary(", "func TestDueDateBoundaries(", """func TestBorrowingLimitBoundary(t *testing.T) {
	s := newSession(t)
	s.ok("registerMember", `{"memberId":"m1","name":"Ada"}`)
	s.ok("registerMember", `{"memberId":"m2","name":"Bob"}`)
	var ids []string
	for i := 1; i <= 6; i++ {
		s.ok("registerCopy", fmt.Sprintf(`{"copyId":"c%d","title":"T"}`, i))
	}
	for i := 1; i <= 3; i++ {
		ids = append(ids, borrowLoan(s, "m1", fmt.Sprintf("c%d", i), "2024-01-01")["loanId"].(string))
	}
	borrowLoan(s, "m2", "c4", "2024-01-01") // other members unaffected
	s.ok("return", fmt.Sprintf(`{"loanId":%q,"returnDate":"2024-01-05"}`, ids[0]))
	borrowLoan(s, "m1", "c5", "2024-01-06")
	if s.ok("inspectOpenLoans", `{"memberId":"m1"}`)["count"] != float64(3) {
		t.Fatal("count should be 3")
	}
}

""")

(root / "LENDING.md").write_text("""# Lending rules

Agreed with the library board for this program:

- A member may hold at most 3 open loans at the same time.
- The loan period is 21 calendar days: the due date is the borrowing date plus 21 days.
- Overdue loans still count towards the limit, and returning a loan is always possible.
""")
print("seeded", root)
