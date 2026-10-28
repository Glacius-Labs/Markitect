package library

import (
	"fmt"
	"testing"

	"example.com/smalllibrary/internal/calendar"
)

func date(t *testing.T, text string) calendar.Date {
	t.Helper()
	d, err := calendar.Parse(text)
	if err != nil {
		t.Fatalf("bad test date %q: %v", text, err)
	}
	return d
}

func registerMember(t *testing.T, l *Library, id string) {
	t.Helper()
	if _, err := l.RegisterMember(id, "Name of "+id); err != nil {
		t.Fatalf("RegisterMember(%q): %v", id, err)
	}
}

func registerCopy(t *testing.T, l *Library, id string) {
	t.Helper()
	registerCopyOfClass(t, l, id, Standard)
}

func registerCopyOfClass(t *testing.T, l *Library, id string, class LoanClass) {
	t.Helper()
	if _, err := l.RegisterCopy(id, "Title of "+id, class); err != nil {
		t.Fatalf("RegisterCopy(%q, %q): %v", id, class, err)
	}
}

func borrow(t *testing.T, l *Library, memberID, copyID, borrowDate string) Loan {
	t.Helper()
	loan, err := l.Borrow(memberID, copyID, date(t, borrowDate))
	if err != nil {
		t.Fatalf("Borrow(%q, %q, %s): %v", memberID, copyID, borrowDate, err)
	}
	return loan
}

func mustReturn(t *testing.T, l *Library, loanID, returnDate string) (Loan, ReturnOutcome) {
	t.Helper()
	loan, outcome, err := l.Return(loanID, date(t, returnDate))
	if err != nil {
		t.Fatalf("Return(%q, %s): %v", loanID, returnDate, err)
	}
	return loan, outcome
}

func openLoanCount(t *testing.T, l *Library, memberID string) int {
	t.Helper()
	loans, err := l.OpenLoans(memberID)
	if err != nil {
		t.Fatalf("OpenLoans(%q): %v", memberID, err)
	}
	return len(loans)
}

func copyAvailable(t *testing.T, l *Library, copyID string) bool {
	t.Helper()
	c, err := l.InspectCopy(copyID)
	if err != nil {
		t.Fatalf("InspectCopy(%q): %v", copyID, err)
	}
	return c.Available
}

// --- Members ---

func TestRegisterMemberCreatesAnActiveMember(t *testing.T) {
	l := New()
	member, err := l.RegisterMember("member-1", "Ada")
	if err != nil {
		t.Fatal(err)
	}
	want := Member{ID: "member-1", Name: "Ada", Status: Active}
	if member != want {
		t.Errorf("registered %+v, want %+v", member, want)
	}
	inspected, err := l.InspectMember("member-1")
	if err != nil || inspected != want {
		t.Errorf("InspectMember = %+v, %v; want %+v", inspected, err, want)
	}
}

func TestRegisterMemberRefusals(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")

	if _, err := l.RegisterMember("member-1", "Changed"); err == nil {
		t.Error("a duplicate member identifier was accepted")
	}
	if member, _ := l.InspectMember("member-1"); member.Name != "Name of member-1" {
		t.Errorf("the duplicate registration replaced the member: %+v", member)
	}
	if _, err := l.RegisterMember("", "Ada"); err == nil {
		t.Error("an empty member identifier was accepted")
	}
	for _, name := range []string{"", " ", " \t\n"} {
		if _, err := l.RegisterMember("member-2", name); err == nil {
			t.Errorf("the blank name %q was accepted", name)
		}
	}
	if _, err := l.InspectMember("member-2"); err == nil {
		t.Error("a refused registration created a member")
	}
}

func TestMembersAreComparedExactlyAndMayShareNames(t *testing.T) {
	l := New()
	for _, id := range []string{"Case", "case", " case", "case ", "café", "café"} {
		if _, err := l.RegisterMember(id, "Same Name"); err != nil {
			t.Errorf("RegisterMember(%q): %v", id, err)
		}
	}
	member, err := l.InspectMember(" case")
	if err != nil || member.ID != " case" {
		t.Errorf("InspectMember(\" case\") = %+v, %v", member, err)
	}
}

func TestNamesAreReturnedAsSupplied(t *testing.T) {
	l := New()
	member, err := l.RegisterMember("member-1", "  Ada Lovelace \t")
	if err != nil || member.Name != "  Ada Lovelace \t" {
		t.Errorf("RegisterMember = %+v, %v", member, err)
	}
}

func TestSetMemberStatus(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")

	for _, status := range []string{"suspended", "suspended", "active", "active"} {
		member, err := l.SetMemberStatus("member-1", status)
		if err != nil {
			t.Fatalf("SetMemberStatus(%q): %v", status, err)
		}
		if string(member.Status) != status || member.Name != "Name of member-1" {
			t.Errorf("SetMemberStatus(%q) = %+v", status, member)
		}
	}
}

func TestSetMemberStatusRefusals(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")

	if _, err := l.SetMemberStatus("absent", "active"); err == nil {
		t.Error("the status of an unknown member was set")
	}
	if _, err := l.InspectMember("absent"); err == nil {
		t.Error("setting a status registered a missing member")
	}
	for _, status := range []string{"inactive", "", "Active", "SUSPENDED"} {
		if _, err := l.SetMemberStatus("member-1", status); err == nil {
			t.Errorf("the status %q was accepted", status)
		}
	}
	if member, _ := l.InspectMember("member-1"); member.Status != Active {
		t.Errorf("a refused status change altered the member: %+v", member)
	}
}

func TestInspectMemberRefusals(t *testing.T) {
	l := New()
	if _, err := l.InspectMember("absent"); err == nil {
		t.Error("an unknown member was inspected")
	}
	if _, err := l.InspectMember(""); err == nil {
		t.Error("an empty member identifier was inspected")
	}
}

// --- Copies ---

func TestRegisterAndInspectCopy(t *testing.T) {
	l := New()
	c, err := l.RegisterCopy("copy-1", "Dune", Standard)
	if err != nil {
		t.Fatal(err)
	}
	want := Copy{ID: "copy-1", Title: "Dune", LoanClass: Standard, Available: true}
	if c != want {
		t.Errorf("registered %+v, want %+v", c, want)
	}
	// Two copies may share a title.
	if _, err := l.RegisterCopy("copy-2", "Dune", Standard); err != nil {
		t.Errorf("a second copy with the same title was refused: %v", err)
	}
	inspected, err := l.InspectCopy("copy-2")
	if err != nil || inspected.Title != "Dune" || !inspected.Available {
		t.Errorf("InspectCopy = %+v, %v", inspected, err)
	}
}

func TestCopyRefusals(t *testing.T) {
	l := New()
	registerCopy(t, l, "copy-1")

	if _, err := l.RegisterCopy("copy-1", "Overwrite", Short); err == nil {
		t.Error("a duplicate copy identifier was accepted")
	}
	if c, _ := l.InspectCopy("copy-1"); c.Title != "Title of copy-1" || c.LoanClass != Standard {
		t.Errorf("the duplicate registration replaced the copy: %+v", c)
	}
	if _, err := l.RegisterCopy("", "Title", Standard); err == nil {
		t.Error("an empty copy identifier was accepted")
	}
	if _, err := l.RegisterCopy("copy-2", " \t", Standard); err == nil {
		t.Error("a blank title was accepted")
	}
	if _, err := l.InspectCopy("copy-2"); err == nil {
		t.Error("a refused registration created a copy")
	}
	if _, err := l.InspectCopy(""); err == nil {
		t.Error("an empty copy identifier was inspected")
	}
}

func TestRegisterCopyWithALoanClass(t *testing.T) {
	l := New()
	c, err := l.RegisterCopy("copy-1", "Dune", Short)
	want := Copy{ID: "copy-1", Title: "Dune", LoanClass: Short, Available: true}
	if err != nil || c != want {
		t.Errorf("RegisterCopy = %+v, %v; want %+v", c, err, want)
	}
	if inspected, err := l.InspectCopy("copy-1"); err != nil || inspected != want {
		t.Errorf("InspectCopy = %+v, %v; want %+v", inspected, err, want)
	}
}

func TestUnknownLoanClassesAreRefusedWithoutRegistering(t *testing.T) {
	l := New()
	for _, class := range []LoanClass{"", "Short", "STANDARD", "reference", " short"} {
		if _, err := l.RegisterCopy("copy-1", "Dune", class); err == nil {
			t.Errorf("the loan class %q was accepted", class)
		}
		if _, err := l.InspectCopy("copy-1"); err == nil {
			t.Errorf("the refused registration with class %q created a copy", class)
		}
	}
}

// --- Borrowing ---

func TestBorrowCreatesAnOpenLoan(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	registerCopy(t, l, "copy-2")

	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

	if loan.ID == "" || loan.MemberID != "member-1" || loan.CopyID != "copy-1" ||
		loan.BorrowDate.String() != "2026-10-01" || loan.DueDate.String() != "2026-10-15" ||
		loan.Status != Open {
		t.Errorf("unexpected loan %+v", loan)
	}
	if copyAvailable(t, l, "copy-1") {
		t.Error("the borrowed copy is still available")
	}
	if !copyAvailable(t, l, "copy-2") {
		t.Error("another copy became unavailable")
	}
	if n := openLoanCount(t, l, "member-1"); n != 1 {
		t.Errorf("open loan count = %d, want 1", n)
	}
	inspected, err := l.InspectLoan(loan.ID)
	if err != nil || inspected != loan {
		t.Errorf("InspectLoan = %+v, %v; want %+v", inspected, err, loan)
	}
}

func TestBorrowRefusalsChangeNothing(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerMember(t, l, "member-2")
	registerCopy(t, l, "copy-1")
	registerCopy(t, l, "copy-2")
	borrow(t, l, "member-1", "copy-1", "2026-10-01")

	tests := []struct {
		name     string
		memberID string
		copyID   string
		date     string
	}{
		{"unknown member", "absent", "copy-2", "2026-10-02"},
		{"unknown copy", "member-1", "absent", "2026-10-02"},
		{"empty member identifier", "", "copy-2", "2026-10-02"},
		{"empty copy identifier", "member-1", "", "2026-10-02"},
		{"copy already on loan to the same member", "member-1", "copy-1", "2026-10-02"},
		{"copy already on loan to another member", "member-2", "copy-1", "2026-10-02"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := l.Borrow(test.memberID, test.copyID, date(t, test.date)); err == nil {
				t.Fatal("the borrowing was admitted")
			}
			if n := openLoanCount(t, l, "member-1"); n != 1 {
				t.Errorf("member-1 has %d open loans, want 1", n)
			}
			if n := openLoanCount(t, l, "member-2"); n != 0 {
				t.Errorf("member-2 has %d open loans, want 0", n)
			}
			if !copyAvailable(t, l, "copy-2") {
				t.Error("copy-2 became unavailable")
			}
		})
	}
}

func TestBorrowingLimit(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerMember(t, l, "member-2")
	for i := 1; i <= MaxOpenLoans+2; i++ {
		registerCopy(t, l, fmt.Sprintf("copy-%d", i))
	}

	// Up to the limit, every loan is admitted; the last one reaches it.
	loans := make([]Loan, 0, MaxOpenLoans)
	for i := 1; i <= MaxOpenLoans; i++ {
		loans = append(loans, borrow(t, l, "member-1", fmt.Sprintf("copy-%d", i), "2026-10-01"))
	}
	if n := openLoanCount(t, l, "member-1"); n != MaxOpenLoans {
		t.Fatalf("open loan count = %d, want %d", n, MaxOpenLoans)
	}

	// The next loan is refused, even though every held loan is overdue by now.
	next := fmt.Sprintf("copy-%d", MaxOpenLoans+1)
	if _, err := l.Borrow("member-1", next, date(t, "2026-11-01")); err == nil {
		t.Fatal("a loan beyond the limit was admitted")
	}
	if !copyAvailable(t, l, next) {
		t.Error("the refused loan made the copy unavailable")
	}
	if n := openLoanCount(t, l, "member-1"); n != MaxOpenLoans {
		t.Errorf("open loan count = %d after the refusal, want %d", n, MaxOpenLoans)
	}

	// Another member is not affected by the first member's limit.
	borrow(t, l, "member-2", next, "2026-11-01")

	// A return frees exactly one place; closed loans are not counted.
	mustReturn(t, l, loans[0].ID, "2026-11-01")
	last := fmt.Sprintf("copy-%d", MaxOpenLoans+2)
	borrow(t, l, "member-1", last, "2026-11-01")
	if _, err := l.Borrow("member-1", loans[0].CopyID, date(t, "2026-11-01")); err == nil {
		t.Error("the limit was exceeded after one return and one new loan")
	}
	if n := openLoanCount(t, l, "member-1"); n != MaxOpenLoans {
		t.Errorf("open loan count = %d, want %d", n, MaxOpenLoans)
	}
}

func TestDueDateIsTheLoanPeriodOfTheLoanClassAfterBorrowing(t *testing.T) {
	tests := []struct {
		class               LoanClass
		borrowDate, dueDate string
	}{
		{Standard, "2026-10-01", "2026-10-15"},
		{Standard, "2026-12-25", "2027-01-08"}, // across a year boundary
		{Standard, "2028-02-20", "2028-03-05"}, // across a leap day
		{Standard, "2026-02-20", "2026-03-06"}, // no leap day
		{Standard, "0001-01-01", "0001-01-15"},
		{Standard, "9999-12-17", "9999-12-31"}, // the last allowed due date
		{Short, "2026-10-01", "2026-10-08"},
		{Short, "2026-12-28", "2027-01-04"}, // across a year boundary
		{Short, "2028-02-25", "2028-03-03"}, // across a leap day
		{Short, "2026-02-25", "2026-03-04"}, // no leap day
		{Short, "0001-01-01", "0001-01-08"},
		{Short, "9999-12-24", "9999-12-31"}, // the last allowed due date
	}
	for _, test := range tests {
		l := New()
		registerMember(t, l, "member-1")
		registerCopyOfClass(t, l, "copy-1", test.class)
		loan := borrow(t, l, "member-1", "copy-1", test.borrowDate)
		if loan.DueDate.String() != test.dueDate {
			t.Errorf("%s copy borrowed on %s: due %s, want %s", test.class, test.borrowDate, loan.DueDate, test.dueDate)
		}
	}
}

func TestDueDateAfterTheLastDateIsRefused(t *testing.T) {
	tests := []struct {
		class        LoanClass
		refusedDates []string
		lastAdmitted string
	}{
		{Standard, []string{"9999-12-18", "9999-12-31"}, "9999-12-17"},
		{Short, []string{"9999-12-25", "9999-12-31"}, "9999-12-24"},
	}
	for _, test := range tests {
		l := New()
		registerMember(t, l, "member-1")
		registerCopyOfClass(t, l, "copy-1", test.class)

		for _, borrowDate := range test.refusedDates {
			if _, err := l.Borrow("member-1", "copy-1", date(t, borrowDate)); err == nil {
				t.Errorf("a %s loan borrowed on %s was admitted", test.class, borrowDate)
			}
		}
		if !copyAvailable(t, l, "copy-1") || openLoanCount(t, l, "member-1") != 0 {
			t.Errorf("a refused %s loan changed the library", test.class)
		}
		// The boundary itself is still admitted.
		borrow(t, l, "member-1", "copy-1", test.lastAdmitted)
	}
}

func TestTheLimitCountsLoansOfBothClasses(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	for i := 1; i <= MaxOpenLoans+1; i++ {
		class := Standard
		if i%2 == 0 {
			class = Short
		}
		registerCopyOfClass(t, l, fmt.Sprintf("copy-%d", i), class)
	}
	for i := 1; i <= MaxOpenLoans; i++ {
		borrow(t, l, "member-1", fmt.Sprintf("copy-%d", i), "2026-10-01")
	}
	if _, err := l.Borrow("member-1", fmt.Sprintf("copy-%d", MaxOpenLoans+1), date(t, "2026-10-02")); err == nil {
		t.Error("a loan beyond the limit was admitted")
	}
}

func TestLoanIdentifiersAreUnique(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")

	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
		if loan.ID == "" || seen[loan.ID] {
			t.Fatalf("loan identifier %q is empty or reused", loan.ID)
		}
		seen[loan.ID] = true
		mustReturn(t, l, loan.ID, "2026-10-02")
	}
}

// --- Renewals ---

func renew(t *testing.T, l *Library, loanID, renewalDate string) Loan {
	t.Helper()
	loan, err := l.Renew(loanID, date(t, renewalDate))
	if err != nil {
		t.Fatalf("Renew(%q, %s): %v", loanID, renewalDate, err)
	}
	return loan
}

func TestNewLoansAreNotRenewed(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	if loan := borrow(t, l, "member-1", "copy-1", "2026-10-01"); loan.Renewals != 0 {
		t.Errorf("a new loan has %d renewals, want 0", loan.Renewals)
	}
}

func TestRenewMovesTheDueDateByTheLoanPeriodOfTheCopy(t *testing.T) {
	tests := []struct {
		class                       LoanClass
		borrowDate, due, renewedDue string
	}{
		{Standard, "2026-10-01", "2026-10-15", "2026-10-29"},
		{Standard, "2026-12-20", "2027-01-03", "2027-01-17"}, // across a year boundary
		{Standard, "2028-02-10", "2028-02-24", "2028-03-09"}, // across a leap day
		{Short, "2026-10-01", "2026-10-08", "2026-10-15"},
		{Short, "2026-12-25", "2027-01-01", "2027-01-08"},
		{Short, "2028-02-20", "2028-02-27", "2028-03-05"},
	}
	for _, test := range tests {
		l := New()
		registerMember(t, l, "member-1")
		registerCopyOfClass(t, l, "copy-1", test.class)
		loan := borrow(t, l, "member-1", "copy-1", test.borrowDate)
		if loan.DueDate.String() != test.due {
			t.Fatalf("%s loan due %s, want %s", test.class, loan.DueDate, test.due)
		}

		renewed := renew(t, l, loan.ID, test.borrowDate)
		if renewed.DueDate.String() != test.renewedDue || renewed.Renewals != 1 || renewed.Status != Open ||
			renewed.BorrowDate != loan.BorrowDate || renewed.ID != loan.ID {
			t.Errorf("%s loan renewed: %+v, want due %s and one renewal", test.class, renewed, test.renewedDue)
		}
		// The renewed loan is what later inspections show; the copy stays on loan.
		if inspected, err := l.InspectLoan(loan.ID); err != nil || inspected != renewed {
			t.Errorf("InspectLoan = %+v, %v; want %+v", inspected, err, renewed)
		}
		if open, _ := l.OpenLoans("member-1"); len(open) != 1 || open[0] != renewed {
			t.Errorf("OpenLoans = %+v, want only %+v", open, renewed)
		}
		if copyAvailable(t, l, "copy-1") {
			t.Error("renewing released the copy")
		}
	}
}

func TestRenewalDateBoundaries(t *testing.T) {
	// Borrowed 2026-10-01, due 2026-10-15: the renewal date must lie in [borrowing date, due date].
	tests := []struct {
		renewalDate string
		admitted    bool
	}{
		{"2026-09-30", false},
		{"2026-10-01", true},
		{"2026-10-10", true},
		{"2026-10-15", true},
		{"2026-10-16", false},
		{"2027-01-01", false},
	}
	for _, test := range tests {
		l := New()
		registerMember(t, l, "member-1")
		registerCopy(t, l, "copy-1")
		loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

		_, err := l.Renew(loan.ID, date(t, test.renewalDate))
		if (err == nil) != test.admitted {
			t.Errorf("renewal on %s: error %v, admitted %v", test.renewalDate, err, test.admitted)
		}
		inspected, _ := l.InspectLoan(loan.ID)
		if !test.admitted && inspected != loan {
			t.Errorf("a refused renewal on %s changed the loan: %+v", test.renewalDate, inspected)
		}
	}
}

func TestRenewalsAreLimited(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	var renewed Loan
	for i := 1; i <= MaxRenewals; i++ {
		renewed = renew(t, l, loan.ID, "2026-10-02")
		if renewed.Renewals != i {
			t.Fatalf("renewal %d: renewal count %d", i, renewed.Renewals)
		}
	}
	if _, err := l.Renew(loan.ID, date(t, "2026-10-02")); err == nil {
		t.Error("a loan was renewed more often than the policy allows")
	}
	if inspected, _ := l.InspectLoan(loan.ID); inspected != renewed {
		t.Errorf("a refused renewal changed the loan: %+v, want %+v", inspected, renewed)
	}
}

func TestRenewalRefusalsChangeNothing(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	registerCopy(t, l, "copy-2")
	open := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	closed := borrow(t, l, "member-1", "copy-2", "2026-10-01")
	mustReturn(t, l, closed.ID, "2026-10-02")
	closed, _ = l.InspectLoan(closed.ID)

	for name, loanID := range map[string]string{"unknown loan": "absent", "empty loan identifier": "", "copy identifier": "copy-1", "closed loan": closed.ID} {
		if _, err := l.Renew(loanID, date(t, "2026-10-03")); err == nil {
			t.Errorf("%s: the renewal was admitted", name)
		}
	}
	if inspected, _ := l.InspectLoan(open.ID); inspected != open {
		t.Errorf("a refused renewal changed the open loan: %+v", inspected)
	}
	if inspected, _ := l.InspectLoan(closed.ID); inspected != closed {
		t.Errorf("a refused renewal changed the closed loan: %+v", inspected)
	}
}

func TestRenewalDoesNotChangeTheOpenLoanCountAndIsNotBlockedByTheLimit(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	var first Loan
	for i := 1; i <= MaxOpenLoans; i++ {
		id := fmt.Sprintf("copy-%d", i)
		registerCopy(t, l, id)
		loan := borrow(t, l, "member-1", id, "2026-10-01")
		if i == 1 {
			first = loan
		}
	}
	renew(t, l, first.ID, "2026-10-05")
	if n := openLoanCount(t, l, "member-1"); n != MaxOpenLoans {
		t.Errorf("open loan count = %d after a renewal, want %d", n, MaxOpenLoans)
	}
}

func TestRenewalIsRefusedWhenTheNewDueDateIsAfterTheLastDate(t *testing.T) {
	tests := []struct {
		class                   LoanClass
		borrowDate, renewalDate string
		admitted                bool
	}{
		{Standard, "9999-12-17", "9999-12-20", false}, // due 9999-12-31, a renewal would pass the last date
		{Short, "9999-12-17", "9999-12-20", true},     // due 9999-12-24, a renewal ends exactly on the last date
		{Short, "9999-12-20", "9999-12-27", false},    // due 9999-12-27, a renewal would pass the last date
	}
	for _, test := range tests {
		l := New()
		registerMember(t, l, "member-1")
		registerCopyOfClass(t, l, "copy-1", test.class)
		loan := borrow(t, l, "member-1", "copy-1", test.borrowDate)

		renewed, err := l.Renew(loan.ID, date(t, test.renewalDate))
		if (err == nil) != test.admitted {
			t.Errorf("%s loan borrowed %s renewed %s: error %v, admitted %v", test.class, test.borrowDate, test.renewalDate, err, test.admitted)
		}
		if test.admitted && renewed.DueDate.String() != "9999-12-31" {
			t.Errorf("renewed due date %s, want 9999-12-31", renewed.DueDate)
		}
		if inspected, _ := l.InspectLoan(loan.ID); !test.admitted && inspected != loan {
			t.Errorf("a refused renewal changed the loan: %+v", inspected)
		}
	}
}

func TestOverdueAssessmentUsesTheRenewedDueDate(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01") // due 2026-10-15
	renewed := renew(t, l, loan.ID, "2026-10-10")            // due 2026-10-29

	if notices := l.AssessOverdue(date(t, "2026-10-16")); len(notices) != 0 {
		t.Errorf("the renewed loan is overdue before its new due date: %+v", notices)
	}
	if notices := l.AssessOverdue(date(t, "2026-10-29")); len(notices) != 0 {
		t.Errorf("a loan due on the assessment date is not yet overdue: %+v", notices)
	}
	notices := l.AssessOverdue(date(t, "2026-10-30"))
	if len(notices) != 1 || notices[0].LoanID != loan.ID || notices[0].DueDate != renewed.DueDate {
		t.Errorf("notices = %+v, want one for the renewed due date %s", notices, renewed.DueDate)
	}
}

func TestAnOverdueLoanCannotBeRenewed(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01") // due 2026-10-15
	if _, err := l.Renew(loan.ID, date(t, "2026-10-16")); err == nil {
		t.Error("an overdue loan was renewed")
	}
}

func TestReturnKeepsTheRenewalCount(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	renewed := renew(t, l, loan.ID, "2026-10-05")

	closed, _ := mustReturn(t, l, loan.ID, "2026-10-06")
	if closed.Renewals != 1 || closed.DueDate != renewed.DueDate {
		t.Errorf("the closed loan lost its renewal: %+v", closed)
	}
	again, outcome := mustReturn(t, l, loan.ID, "2026-10-07")
	if outcome != AlreadyReturned || again != closed {
		t.Errorf("repeated return: %v %+v, want %v %+v", outcome, again, AlreadyReturned, closed)
	}
}

// --- Suspension ---

func TestSuspensionBlocksBorrowingButNotExistingLoansOrReturns(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	registerCopy(t, l, "copy-2")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

	if _, err := l.SetMemberStatus("member-1", "suspended"); err != nil {
		t.Fatal(err)
	}

	// New borrowing is refused and changes nothing.
	if _, err := l.Borrow("member-1", "copy-2", date(t, "2026-10-02")); err == nil {
		t.Fatal("a suspended member borrowed a copy")
	}
	if !copyAvailable(t, l, "copy-2") {
		t.Error("the refused loan made the copy unavailable")
	}

	// The existing loan keeps its due date and open state.
	existing, err := l.InspectLoan(loan.ID)
	if err != nil || existing != loan {
		t.Errorf("the existing loan changed: %+v, %v; want %+v", existing, err, loan)
	}
	if n := openLoanCount(t, l, "member-1"); n != 1 {
		t.Errorf("open loan count = %d, want 1", n)
	}

	// Returning is still possible.
	returned, outcome := mustReturn(t, l, loan.ID, "2026-10-03")
	if outcome != Returned || returned.Status != Closed {
		t.Errorf("return by a suspended member: %v %+v", outcome, returned)
	}

	// Restoring the active status allows borrowing again.
	if _, err := l.SetMemberStatus("member-1", "active"); err != nil {
		t.Fatal(err)
	}
	borrow(t, l, "member-1", "copy-2", "2026-10-04")
}

func TestSuspensionBlocksRenewalUntilTheStatusIsRestored(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

	if _, err := l.SetMemberStatus("member-1", "suspended"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Renew(loan.ID, date(t, "2026-10-05")); err == nil {
		t.Fatal("a suspended member renewed a loan")
	}
	if inspected, _ := l.InspectLoan(loan.ID); inspected != loan {
		t.Errorf("a refused renewal changed the loan: %+v", inspected)
	}
	if _, err := l.SetMemberStatus("member-1", "active"); err != nil {
		t.Fatal(err)
	}
	renew(t, l, loan.ID, "2026-10-05")
}

func TestRestoredMemberIsStillSubjectToTheLimit(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	for i := 1; i <= MaxOpenLoans+1; i++ {
		registerCopy(t, l, fmt.Sprintf("copy-%d", i))
	}
	for i := 1; i <= MaxOpenLoans; i++ {
		borrow(t, l, "member-1", fmt.Sprintf("copy-%d", i), "2026-10-01")
	}
	l.SetMemberStatus("member-1", "suspended")
	l.SetMemberStatus("member-1", "active")

	if _, err := l.Borrow("member-1", fmt.Sprintf("copy-%d", MaxOpenLoans+1), date(t, "2026-10-02")); err == nil {
		t.Error("restoring the active status lifted the borrowing limit")
	}
}

// --- Returns ---

func TestReturnClosesTheLoanAndReleasesTheCopy(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

	returned, outcome := mustReturn(t, l, loan.ID, "2026-10-20") // overdue returns are fine
	if outcome != Returned {
		t.Errorf("outcome = %q, want %q", outcome, Returned)
	}
	if returned.Status != Closed || returned.ReturnDate.String() != "2026-10-20" ||
		returned.BorrowDate != loan.BorrowDate || returned.DueDate != loan.DueDate {
		t.Errorf("unexpected closed loan %+v", returned)
	}
	if !copyAvailable(t, l, "copy-1") {
		t.Error("the copy is not available after the return")
	}
	if n := openLoanCount(t, l, "member-1"); n != 0 {
		t.Errorf("open loan count = %d, want 0", n)
	}
	inspected, err := l.InspectLoan(loan.ID)
	if err != nil || inspected != returned {
		t.Errorf("InspectLoan = %+v, %v; want %+v", inspected, err, returned)
	}
}

func TestReturnOnTheBorrowingDateIsAccepted(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	mustReturn(t, l, loan.ID, "2026-10-01")
}

func TestRepeatedReturnReportsAlreadyReturnedAndKeepsTheOriginalDate(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	first, _ := mustReturn(t, l, loan.ID, "2026-10-02")

	for _, returnDate := range []string{"2026-10-02", "2026-10-09", "2026-10-01"} {
		again, outcome := mustReturn(t, l, loan.ID, returnDate)
		if outcome != AlreadyReturned {
			t.Errorf("repeated return on %s: outcome %q, want %q", returnDate, outcome, AlreadyReturned)
		}
		if again != first {
			t.Errorf("repeated return on %s changed the loan: %+v, want %+v", returnDate, again, first)
		}
	}
}

func TestRepeatedReturnStillNeedsAValidDate(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	first, _ := mustReturn(t, l, loan.ID, "2026-10-02")

	if _, _, err := l.Return(loan.ID, date(t, "2026-09-30")); err == nil {
		t.Error("a repeated return before the borrowing date was accepted")
	}
	if inspected, _ := l.InspectLoan(loan.ID); inspected != first {
		t.Errorf("a refused repeated return changed the loan: %+v", inspected)
	}
}

func TestReturningAnOldLoanDoesNotReleaseANewerLoanOfTheCopy(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	oldLoan := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	mustReturn(t, l, oldLoan.ID, "2026-10-02")
	newLoan := borrow(t, l, "member-1", "copy-1", "2026-10-03")

	_, outcome := mustReturn(t, l, oldLoan.ID, "2026-10-04")
	if outcome != AlreadyReturned {
		t.Errorf("outcome = %q, want %q", outcome, AlreadyReturned)
	}
	if copyAvailable(t, l, "copy-1") {
		t.Error("the repeated return of the old loan released the copy")
	}
	if inspected, _ := l.InspectLoan(newLoan.ID); inspected.Status != Open {
		t.Errorf("the newer loan was closed: %+v", inspected)
	}
	if _, err := l.Borrow("member-1", "copy-1", date(t, "2026-10-05")); err == nil {
		t.Error("the copy was lent twice")
	}
}

func TestReturnRefusals(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopy(t, l, "copy-1")
	loan := borrow(t, l, "member-1", "copy-1", "2026-10-01")

	if _, _, err := l.Return("absent", date(t, "2026-10-02")); err == nil {
		t.Error("an unknown loan was returned")
	}
	if _, _, err := l.Return("", date(t, "2026-10-02")); err == nil {
		t.Error("an empty loan identifier was returned")
	}
	if _, _, err := l.Return("copy-1", date(t, "2026-10-02")); err == nil {
		t.Error("a loan was returned by its copy identifier")
	}
	if _, _, err := l.Return(loan.ID, date(t, "2026-09-30")); err == nil {
		t.Error("a return before the borrowing date was accepted")
	}
	if inspected, _ := l.InspectLoan(loan.ID); inspected != loan {
		t.Errorf("a refused return changed the loan: %+v", inspected)
	}
	if copyAvailable(t, l, "copy-1") {
		t.Error("a refused return released the copy")
	}
}

func TestReturnIsNotBlockedByTheLimit(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	var loans []Loan
	for i := 1; i <= MaxOpenLoans; i++ {
		id := fmt.Sprintf("copy-%d", i)
		registerCopy(t, l, id)
		loans = append(loans, borrow(t, l, "member-1", id, "2026-10-01"))
	}
	for _, loan := range loans {
		mustReturn(t, l, loan.ID, "2027-01-01")
	}
	if n := openLoanCount(t, l, "member-1"); n != 0 {
		t.Errorf("open loan count = %d, want 0", n)
	}
}

// --- Loan inspection ---

func TestInspectOpenLoans(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerMember(t, l, "member-2")
	registerCopy(t, l, "copy-1")
	registerCopy(t, l, "copy-2")

	if loans, err := l.OpenLoans("member-1"); err != nil || len(loans) != 0 || loans == nil {
		t.Errorf("a member without loans: %v, %v; want an empty list", loans, err)
	}

	first := borrow(t, l, "member-1", "copy-1", "2026-10-01")
	second := borrow(t, l, "member-1", "copy-2", "2026-10-02")
	mustReturn(t, l, first.ID, "2026-10-03")

	loans, err := l.OpenLoans("member-1")
	if err != nil || len(loans) != 1 || loans[0] != second {
		t.Errorf("OpenLoans = %+v, %v; want only %+v", loans, err, second)
	}
	if n := openLoanCount(t, l, "member-2"); n != 0 {
		t.Errorf("another member has %d open loans", n)
	}
	if _, err := l.OpenLoans("absent"); err == nil {
		t.Error("the loans of an unknown member were listed")
	}
	if _, err := l.OpenLoans(""); err == nil {
		t.Error("the loans of an empty member identifier were listed")
	}
}

func TestInspectLoanRefusals(t *testing.T) {
	l := New()
	if _, err := l.InspectLoan("absent"); err == nil {
		t.Error("an unknown loan was inspected")
	}
	if _, err := l.InspectLoan(""); err == nil {
		t.Error("an empty loan identifier was inspected")
	}
}

// --- Overdue assessment ---

func TestAssessOverdue(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerMember(t, l, "member-2")
	for _, id := range []string{"copy-1", "copy-2", "copy-3"} {
		registerCopy(t, l, id)
	}
	overdue1 := borrow(t, l, "member-1", "copy-1", "2026-10-01") // due 2026-10-15
	overdue2 := borrow(t, l, "member-2", "copy-2", "2026-10-01")
	returned := borrow(t, l, "member-1", "copy-3", "2026-10-01")
	l.SetMemberStatus("member-2", "suspended")
	mustReturn(t, l, returned.ID, "2026-10-20")

	if notices := l.AssessOverdue(date(t, "2026-10-14")); len(notices) != 0 {
		t.Errorf("notices before the due date: %+v", notices)
	}
	if notices := l.AssessOverdue(date(t, "2026-10-15")); len(notices) != 0 {
		t.Errorf("a loan due on the assessment date is not yet overdue: %+v", notices)
	}

	want := map[string]Notice{
		overdue1.ID: {LoanID: overdue1.ID, MemberID: "member-1", CopyID: "copy-1", DueDate: overdue1.DueDate},
		overdue2.ID: {LoanID: overdue2.ID, MemberID: "member-2", CopyID: "copy-2", DueDate: overdue2.DueDate},
	}
	for i := 0; i < 2; i++ { // the same state and date give the same notices
		notices := l.AssessOverdue(date(t, "2026-10-16"))
		if len(notices) != len(want) {
			t.Fatalf("got %d notices, want %d: %+v", len(notices), len(want), notices)
		}
		for _, notice := range notices {
			if want[notice.LoanID] != notice {
				t.Errorf("unexpected notice %+v", notice)
			}
		}
	}

	// Assessing changes no state.
	if n := openLoanCount(t, l, "member-1"); n != 1 {
		t.Errorf("open loan count = %d, want 1", n)
	}
	// The returned loan is excluded whatever the assessment date.
	for _, notice := range l.AssessOverdue(date(t, "2027-01-01")) {
		if notice.LoanID == returned.ID {
			t.Errorf("a returned loan was reported: %+v", notice)
		}
	}
}

func TestAssessOverdueUsesTheDueDateOfTheLoanClass(t *testing.T) {
	l := New()
	registerMember(t, l, "member-1")
	registerCopyOfClass(t, l, "short-1", Short)
	registerCopyOfClass(t, l, "standard-1", Standard)
	short := borrow(t, l, "member-1", "short-1", "2026-10-01") // due 2026-10-08
	borrow(t, l, "member-1", "standard-1", "2026-10-01")       // due 2026-10-15

	if notices := l.AssessOverdue(date(t, "2026-10-08")); len(notices) != 0 {
		t.Errorf("a loan due on the assessment date is not yet overdue: %+v", notices)
	}
	notices := l.AssessOverdue(date(t, "2026-10-09"))
	if len(notices) != 1 || notices[0].LoanID != short.ID || notices[0].DueDate != short.DueDate {
		t.Errorf("notices = %+v, want only the short loan", notices)
	}
	if notices := l.AssessOverdue(date(t, "2026-10-16")); len(notices) != 2 {
		t.Errorf("notices = %+v, want both loans", notices)
	}
}

func TestAssessOverdueWithoutLoans(t *testing.T) {
	notices := New().AssessOverdue(date(t, "2026-10-16"))
	if notices == nil || len(notices) != 0 {
		t.Errorf("notices = %#v, want an empty list", notices)
	}
}
