package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// session feeds request lines to one server, as the program would see them
// on standard input, and decodes the responses.
type session struct {
	t      *testing.T
	server *Server
}

func newSession(t *testing.T) *session {
	return &session{t: t, server: NewServer()}
}

// send sends one raw line and returns the decoded response.
func (s *session) send(line string) map[string]any {
	s.t.Helper()
	reply, err := s.server.Handle([]byte(line))
	if err != nil {
		s.t.Fatalf("Handle(%s): %v", line, err)
	}
	if !bytes.HasSuffix(reply, []byte("\n")) || bytes.Count(reply, []byte("\n")) != 1 {
		s.t.Fatalf("Handle(%s) = %q, want exactly one line", line, reply)
	}
	var decoded map[string]any
	if err := json.Unmarshal(reply, &decoded); err != nil {
		s.t.Fatalf("Handle(%s) = %q, not a JSON object: %v", line, reply, err)
	}
	return decoded
}

// call sends a well-formed request with the given arguments.
func (s *session) call(op string, args map[string]any) map[string]any {
	s.t.Helper()
	line, err := json.Marshal(map[string]any{"id": "req", "op": op, "args": args})
	if err != nil {
		s.t.Fatal(err)
	}
	response := s.send(string(line))
	if response["id"] != "req" {
		s.t.Errorf("%s: response id = %v, want req", op, response["id"])
	}
	return response
}

// ok sends a request that must succeed and returns its result.
func (s *session) ok(op string, args map[string]any) map[string]any {
	s.t.Helper()
	response := s.call(op, args)
	if response["status"] != "ok" {
		s.t.Fatalf("%s %v: %v, want ok", op, args, response)
	}
	result, _ := response["result"].(map[string]any)
	if result == nil {
		s.t.Fatalf("%s %v: no result in %v", op, args, response)
	}
	return result
}

// refused sends a request that must be refused.
func (s *session) refused(op string, args map[string]any) {
	s.t.Helper()
	response := s.call(op, args)
	if response["status"] != "refused" {
		s.t.Fatalf("%s %v: %v, want refused", op, args, response)
	}
	if reason, _ := response["reason"].(string); strings.TrimSpace(reason) == "" {
		s.t.Errorf("%s %v: refusal without a reason: %v", op, args, response)
	}
	if _, has := response["result"]; has {
		s.t.Errorf("%s %v: refusal with a result: %v", op, args, response)
	}
}

func (s *session) invalid(line string, wantID any) {
	s.t.Helper()
	response := s.send(line)
	if response["status"] != "invalidRequest" {
		s.t.Fatalf("%s: %v, want invalidRequest", line, response)
	}
	if reason, _ := response["reason"].(string); strings.TrimSpace(reason) == "" {
		s.t.Errorf("%s: invalidRequest without a reason", line)
	}
	if id, has := response["id"]; !has || id != wantID {
		s.t.Errorf("%s: id = %v, want %v", line, id, wantID)
	}
	if _, has := response["result"]; has {
		s.t.Errorf("%s: invalidRequest with a result", line)
	}
}

type obj = map[string]any

func (s *session) registerMember(id string) {
	s.t.Helper()
	s.ok("registerMember", obj{"memberId": id, "name": "Name of " + id})
}

func (s *session) registerCopy(id string) {
	s.t.Helper()
	s.ok("registerCopy", obj{"copyId": id, "title": "Title of " + id})
}

// borrow admits a loan and returns the loan record.
func (s *session) borrow(memberID, copyID, borrowDate string) obj {
	s.t.Helper()
	result := s.ok("borrow", obj{"memberId": memberID, "copyId": copyID, "borrowDate": borrowDate})
	return result["loan"].(map[string]any)
}

func (s *session) copyAvailable(copyID string) bool {
	s.t.Helper()
	copy := s.ok("inspectCopy", obj{"copyId": copyID})["copy"].(map[string]any)
	return copy["available"].(bool)
}

func (s *session) openLoanCount(memberID string) int {
	s.t.Helper()
	result := s.ok("inspectOpenLoans", obj{"memberId": memberID})
	count := int(result["count"].(float64))
	if loans := result["loans"].([]any); len(loans) != count {
		s.t.Errorf("count %d does not match %d listed loans", count, len(loans))
	}
	return count
}

// --- Successful operations ---

func TestMemberCopyAndLoanLifecycle(t *testing.T) {
	s := newSession(t)

	member := s.ok("registerMember", obj{"memberId": "member-1", "name": "Ada"})
	wantMember := obj{"member": obj{"memberId": "member-1", "name": "Ada", "status": "active"}}
	if !reflect.DeepEqual(member, wantMember) {
		t.Errorf("registerMember result = %v, want %v", member, wantMember)
	}

	copyResult := s.ok("registerCopy", obj{"copyId": "copy-1", "title": "Dune"})
	wantCopy := obj{"copy": obj{"copyId": "copy-1", "title": "Dune", "available": true}}
	if !reflect.DeepEqual(copyResult, wantCopy) {
		t.Errorf("registerCopy result = %v, want %v", copyResult, wantCopy)
	}

	loan := s.borrow("member-1", "copy-1", "2026-10-01")
	loanID, _ := loan["loanId"].(string)
	wantLoan := obj{"loanId": loanID, "memberId": "member-1", "copyId": "copy-1",
		"borrowDate": "2026-10-01", "dueDate": "2026-10-15", "status": "open", "returnDate": nil}
	if loanID == "" || !reflect.DeepEqual(loan, wantLoan) {
		t.Errorf("borrow loan = %v, want %v", loan, wantLoan)
	}
	if s.copyAvailable("copy-1") {
		t.Error("the borrowed copy is available")
	}
	if got := s.ok("inspectLoan", obj{"loanId": loanID})["loan"]; !reflect.DeepEqual(got, wantLoan) {
		t.Errorf("inspectLoan = %v, want %v", got, wantLoan)
	}
	open := s.ok("inspectOpenLoans", obj{"memberId": "member-1"})
	if !reflect.DeepEqual(open, obj{"memberId": "member-1", "count": 1.0, "loans": []any{wantLoan}}) {
		t.Errorf("inspectOpenLoans = %v", open)
	}

	returned := s.ok("return", obj{"loanId": loanID, "returnDate": "2026-10-05"})
	wantClosed := obj{"loanId": loanID, "memberId": "member-1", "copyId": "copy-1",
		"borrowDate": "2026-10-01", "dueDate": "2026-10-15", "status": "closed", "returnDate": "2026-10-05"}
	if !reflect.DeepEqual(returned, obj{"outcome": "returned", "loan": wantClosed}) {
		t.Errorf("return result = %v", returned)
	}
	if !s.copyAvailable("copy-1") {
		t.Error("the returned copy is not available")
	}
	open = s.ok("inspectOpenLoans", obj{"memberId": "member-1"})
	if !reflect.DeepEqual(open, obj{"memberId": "member-1", "count": 0.0, "loans": []any{}}) {
		t.Errorf("inspectOpenLoans after the return = %v", open)
	}
}

func TestMemberStatusAndInspection(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")

	for _, status := range []string{"suspended", "suspended", "active"} {
		result := s.ok("setMemberStatus", obj{"memberId": "member-1", "status": status})
		if member := result["member"].(obj); member["status"] != status || member["name"] != "Name of member-1" {
			t.Errorf("setMemberStatus(%s) = %v", status, member)
		}
	}
	result := s.ok("inspectMember", obj{"memberId": "member-1"})
	if member := result["member"].(obj); member["status"] != "active" {
		t.Errorf("inspectMember = %v", member)
	}
}

func TestAssessOverdue(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")
	s.registerCopy("copy-2")
	loan := s.borrow("member-1", "copy-1", "2026-10-01")
	returned := s.borrow("member-1", "copy-2", "2026-10-01")
	s.ok("return", obj{"loanId": returned["loanId"], "returnDate": "2026-10-02"})

	empty := obj{"notices": []any{}}
	for _, date := range []string{"2026-10-14", "2026-10-15"} {
		if got := s.ok("assessOverdue", obj{"assessmentDate": date}); !reflect.DeepEqual(got, empty) {
			t.Errorf("assessOverdue(%s) = %v, want no notices", date, got)
		}
	}
	want := obj{"notices": []any{obj{
		"loanId": loan["loanId"], "memberId": "member-1", "copyId": "copy-1", "dueDate": "2026-10-15"}}}
	if got := s.ok("assessOverdue", obj{"assessmentDate": "2026-10-16"}); !reflect.DeepEqual(got, want) {
		t.Errorf("assessOverdue = %v, want %v", got, want)
	}
}

// --- Expected refusals ---

func TestRefusalsLeaveTheStateUnchanged(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")
	s.registerCopy("copy-2")
	loan := s.borrow("member-1", "copy-1", "2026-10-01")
	loanID := loan["loanId"]

	s.refused("registerMember", obj{"memberId": "member-1", "name": "Changed"})
	s.refused("registerMember", obj{"memberId": "", "name": "Empty"})
	s.refused("registerMember", obj{"memberId": "member-2", "name": "  "})
	s.refused("registerCopy", obj{"copyId": "copy-1", "title": "Changed"})
	s.refused("registerCopy", obj{"copyId": "", "title": "Empty"})
	s.refused("registerCopy", obj{"copyId": "copy-3", "title": "\t"})
	s.refused("setMemberStatus", obj{"memberId": "absent", "status": "active"})
	s.refused("setMemberStatus", obj{"memberId": "member-1", "status": "inactive"})
	s.refused("inspectMember", obj{"memberId": "absent"})
	s.refused("inspectMember", obj{"memberId": ""})
	s.refused("inspectCopy", obj{"copyId": "absent"})
	s.refused("inspectLoan", obj{"loanId": "absent"})
	s.refused("inspectOpenLoans", obj{"memberId": "absent"})
	s.refused("borrow", obj{"memberId": "absent", "copyId": "copy-2", "borrowDate": "2026-10-02"})
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "absent", "borrowDate": "2026-10-02"})
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-1", "borrowDate": "2026-10-02"}) // already on loan
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-2", "borrowDate": "2026-02-29"})
	s.refused("return", obj{"loanId": "absent", "returnDate": "2026-10-02"})
	s.refused("return", obj{"loanId": "copy-1", "returnDate": "2026-10-02"}) // a copy id is no loan id
	s.refused("return", obj{"loanId": loanID, "returnDate": "2026-09-30"})
	s.refused("assessOverdue", obj{"assessmentDate": "2026-13-01"})

	if got := s.ok("inspectMember", obj{"memberId": "member-1"})["member"].(obj)["name"]; got != "Name of member-1" {
		t.Errorf("a refusal changed the member name to %v", got)
	}
	if got := s.ok("inspectCopy", obj{"copyId": "copy-1"})["copy"].(obj)["title"]; got != "Title of copy-1" {
		t.Errorf("a refusal changed the copy title to %v", got)
	}
	if !s.copyAvailable("copy-2") {
		t.Error("a refused borrowing made copy-2 unavailable")
	}
	if s.copyAvailable("copy-1") {
		t.Error("a refused return released copy-1")
	}
	if got := s.ok("inspectLoan", obj{"loanId": loanID})["loan"]; !reflect.DeepEqual(got, obj(loan)) {
		t.Errorf("a refusal changed the loan: %v", got)
	}
	if s.openLoanCount("member-1") != 1 {
		t.Error("a refusal changed the open loan count")
	}
	s.refused("inspectMember", obj{"memberId": "member-2"})
	s.refused("inspectCopy", obj{"copyId": "copy-3"})
}

func TestMissingNullAndMistypedArgumentsAreRefused(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")
	loan := s.borrow("member-1", "copy-1", "2026-10-01")

	valid := map[string]obj{
		"registerMember":   {"memberId": "member-2", "name": "Bea"},
		"setMemberStatus":  {"memberId": "member-1", "status": "active"},
		"inspectMember":    {"memberId": "member-1"},
		"registerCopy":     {"copyId": "copy-2", "title": "Book"},
		"inspectCopy":      {"copyId": "copy-1"},
		"borrow":           {"memberId": "member-1", "copyId": "copy-2", "borrowDate": "2026-10-02"},
		"return":           {"loanId": loan["loanId"], "returnDate": "2026-10-02"},
		"inspectLoan":      {"loanId": loan["loanId"]},
		"inspectOpenLoans": {"memberId": "member-1"},
		"assessOverdue":    {"assessmentDate": "2026-10-02"},
	}
	for op, args := range valid {
		for field := range args {
			missing := obj{}
			nullValue := obj{}
			number := obj{}
			list := obj{}
			for k, v := range args {
				missing[k], nullValue[k], number[k], list[k] = v, v, v, v
			}
			delete(missing, field)
			nullValue[field] = nil
			number[field] = 17
			list[field] = []string{"x"}
			for _, variant := range []obj{missing, nullValue, number, list} {
				s.refused(op, variant)
			}
		}
	}
	if s.openLoanCount("member-1") != 1 || s.copyAvailable("copy-1") {
		t.Error("refused requests changed the state")
	}
	s.refused("inspectMember", obj{"memberId": "member-2"})
	s.refused("inspectCopy", obj{"copyId": "copy-2"})
}

func TestInvalidDatesAreRefused(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")

	for _, date := range []string{"", "not-a-date", "2026-13-01", "2026-02-29", "2026-1-5", "2026-10-01T00:00:00Z",
		"0000-12-31", "10000-01-01", " 2026-10-01"} {
		s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-1", "borrowDate": date})
		s.refused("assessOverdue", obj{"assessmentDate": date})
	}
	if !s.copyAvailable("copy-1") {
		t.Error("a refused borrowing made the copy unavailable")
	}
	s.ok("assessOverdue", obj{"assessmentDate": "0001-01-01"})
	s.ok("assessOverdue", obj{"assessmentDate": "9999-12-31"})
}

func TestIdentifiersAreComparedExactly(t *testing.T) {
	s := newSession(t)
	for _, id := range []string{"Case", "case", " padded ", "padded", "caf\u00e9", "cafe\u0301"} {
		s.ok("registerMember", obj{"memberId": id, "name": " Name\t"})
	}
	member := s.ok("inspectMember", obj{"memberId": " padded "})["member"].(obj)
	if member["memberId"] != " padded " || member["name"] != " Name\t" {
		t.Errorf("inspectMember = %v", member)
	}
	s.refused("inspectMember", obj{"memberId": "CASE"})
}

// --- Borrowing limit and due dates ---

func TestBorrowingLimitAtTheLimitValue(t *testing.T) {
	const limit = 5 // the lending policy: at most five open loans per member
	s := newSession(t)
	s.registerMember("member-1")
	s.registerMember("member-2")
	for i := 1; i <= limit+1; i++ {
		s.registerCopy(fmt.Sprintf("copy-%d", i))
	}

	var loans []obj
	for i := 1; i <= limit; i++ { // the last admitted loan is the fifth
		loans = append(loans, s.borrow("member-1", fmt.Sprintf("copy-%d", i), "2026-10-01"))
	}
	if got := s.openLoanCount("member-1"); got != limit {
		t.Fatalf("open loan count = %d, want %d", got, limit)
	}

	// The next loan is refused, although all five loans are overdue.
	next := fmt.Sprintf("copy-%d", limit+1)
	s.refused("borrow", obj{"memberId": "member-1", "copyId": next, "borrowDate": "2026-12-01"})
	if !s.copyAvailable(next) || s.openLoanCount("member-1") != limit {
		t.Error("the refused loan changed the state")
	}

	// Another member has a limit of their own.
	s.borrow("member-2", next, "2026-12-01")

	// Returning one loan frees one place.
	s.ok("return", obj{"loanId": loans[0]["loanId"], "returnDate": "2026-12-01"})
	s.borrow("member-1", "copy-1", "2026-12-01")
	s.registerCopy("copy-extra")
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-extra", "borrowDate": "2026-12-01"})
}

func TestDueDateCalculation(t *testing.T) {
	tests := []struct{ borrowDate, dueDate string }{
		{"2026-10-01", "2026-10-15"},
		{"2026-12-25", "2027-01-08"},
		{"2028-02-20", "2028-03-05"},
		{"9999-12-17", "9999-12-31"},
	}
	for _, test := range tests {
		s := newSession(t)
		s.registerMember("member-1")
		s.registerCopy("copy-1")
		if loan := s.borrow("member-1", "copy-1", test.borrowDate); loan["dueDate"] != test.dueDate {
			t.Errorf("borrowed on %s: due %v, want %s", test.borrowDate, loan["dueDate"], test.dueDate)
		}
	}
}

func TestDueDateAfterTheLastDateIsRefused(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")

	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-1", "borrowDate": "9999-12-18"})
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-1", "borrowDate": "9999-12-31"})
	if !s.copyAvailable("copy-1") || s.openLoanCount("member-1") != 0 {
		t.Error("a refused loan changed the state")
	}
	s.borrow("member-1", "copy-1", "9999-12-17")
}

// --- Suspension and returns ---

func TestSuspensionAcrossMembershipAndLending(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")
	s.registerCopy("copy-2")
	loan := s.borrow("member-1", "copy-1", "2026-10-01")

	s.ok("setMemberStatus", obj{"memberId": "member-1", "status": "suspended"})
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-2", "borrowDate": "2026-10-02"})
	if got := s.ok("inspectLoan", obj{"loanId": loan["loanId"]})["loan"]; !reflect.DeepEqual(got, obj(loan)) {
		t.Errorf("suspension changed the existing loan: %v", got)
	}
	// The suspended member's overdue loan is still reported.
	notices := s.ok("assessOverdue", obj{"assessmentDate": "2026-10-16"})["notices"].([]any)
	if len(notices) != 1 {
		t.Errorf("notices = %v, want the suspended member's loan", notices)
	}
	if outcome := s.ok("return", obj{"loanId": loan["loanId"], "returnDate": "2026-10-16"})["outcome"]; outcome != "returned" {
		t.Errorf("outcome = %v, want returned", outcome)
	}
	s.ok("setMemberStatus", obj{"memberId": "member-1", "status": "active"})
	s.borrow("member-1", "copy-2", "2026-10-17")
}

func TestRepeatedReturns(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")
	s.registerCopy("copy-1")
	old := s.borrow("member-1", "copy-1", "2026-10-01")
	s.ok("return", obj{"loanId": old["loanId"], "returnDate": "2026-10-02"})
	newer := s.borrow("member-1", "copy-1", "2026-10-03")

	// Repeating the return keeps the original return date ...
	again := s.ok("return", obj{"loanId": old["loanId"], "returnDate": "2026-10-04"})
	if again["outcome"] != "alreadyReturned" || again["loan"].(obj)["returnDate"] != "2026-10-02" {
		t.Errorf("repeated return = %v", again)
	}
	// ... still needs a valid date ...
	s.refused("return", obj{"loanId": old["loanId"], "returnDate": "2026-09-30"})
	s.refused("return", obj{"loanId": old["loanId"], "returnDate": "2026-02-30"})
	// ... and does not release the copy held by the newer loan.
	if s.copyAvailable("copy-1") {
		t.Error("the repeated return released the copy of the newer loan")
	}
	if got := s.ok("inspectLoan", obj{"loanId": newer["loanId"]})["loan"].(obj)["status"]; got != "open" {
		t.Errorf("the newer loan is %v, want open", got)
	}
	s.refused("borrow", obj{"memberId": "member-1", "copyId": "copy-1", "borrowDate": "2026-10-05"})
}

// --- Envelope and transport rules ---

func TestInvalidRequests(t *testing.T) {
	s := newSession(t)
	s.registerMember("member-1")

	tests := []struct {
		name   string
		line   string
		wantID any
	}{
		{"malformed JSON", `{malformed json`, nil},
		{"empty line", ``, nil},
		{"not an object", `["inspectMember"]`, nil},
		{"null", `null`, nil},
		{"trailing text", `{"id":"a","op":"inspectMember","args":{"memberId":"member-1"}} x`, nil},
		{"missing id", `{"op":"inspectMember","args":{"memberId":"member-1"}}`, nil},
		{"numeric id", `{"id":7,"op":"inspectMember","args":{"memberId":"member-1"}}`, nil},
		{"null id", `{"id":null,"op":"inspectMember","args":{"memberId":"member-1"}}`, nil},
		{"missing op", `{"id":"a","args":{}}`, "a"},
		{"numeric op", `{"id":"a","op":1,"args":{}}`, "a"},
		{"unknown op", `{"id":"a","op":"reset","args":{}}`, "a"},
		{"missing args", `{"id":"a","op":"inspectMember"}`, "a"},
		{"null args", `{"id":"a","op":"inspectMember","args":null}`, "a"},
		{"array args", `{"id":"a","op":"inspectMember","args":[]}`, "a"},
		{"unknown envelope field", `{"id":"a","op":"inspectMember","args":{"memberId":"member-1"},"extra":true}`, "a"},
		{"envelope field in other case", `{"id":"a","Op":"inspectMember","op":"inspectMember","args":{"memberId":"member-1"}}`, "a"},
		{"unknown argument", `{"id":"a","op":"inspectMember","args":{"memberId":"member-1","extra":true}}`, "a"},
		{"argument in other case", `{"id":"a","op":"inspectMember","args":{"memberId":"member-1","MemberId":"member-1"}}`, "a"},
		{"operation in other case", `{"id":"a","op":"InspectMember","args":{"memberId":"member-1"}}`, "a"},
		{"invalid UTF-8", "{\"id\":\"a\",\"op\":\"inspectMember\",\"args\":{\"memberId\":\"\xff\"}}", nil},
	}
	for _, test := range tests {
		t.Logf("case: %s", test.name)
		s.invalid(test.line, test.wantID)
	}
	// Processing continues and the state is unchanged.
	s.registerMember("member-2")
	s.ok("inspectMember", obj{"memberId": "member-1"})
}

func TestRequestIdIsEchoedExactly(t *testing.T) {
	s := newSession(t)
	response := s.send(`{"id":" Ünï \"x\" ","op":"registerMember","args":{"memberId":"m","name":"n"}}`)
	if response["id"] != ` Ünï "x" ` || response["status"] != "ok" {
		t.Errorf("response = %v", response)
	}
	// The same id may be used again; it is not an idempotency key.
	s.send(`{"id":"same","op":"registerMember","args":{"memberId":"m2","name":"n"}}`)
	response = s.send(`{"id":"same","op":"registerMember","args":{"memberId":"m2","name":"n"}}`)
	if response["id"] != "same" || response["status"] != "refused" {
		t.Errorf("the repeated request = %v, want a refusal", response)
	}
}

func TestServeAnswersEveryLineInOrderAndEndsAtEOF(t *testing.T) {
	input := strings.Join([]string{
		`{"id":"q1","op":"registerMember","args":{"memberId":"member-1","name":"Ada"}}`,
		`{"id":"q2","op":"inspectMember","args":{"memberId":"member-1"}}` + "\r",
		`garbage`,
		``,
		`{"id":"q3","op":"inspectMember","args":{"memberId":"absent"}}`,
		`{"id":"q4","op":"inspectMember","args":{"memberId":"member-1"}}`, // no final newline
	}, "\n")

	var out bytes.Buffer
	if err := Serve(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	wantStatus := []string{"ok", "ok", "invalidRequest", "invalidRequest", "refused", "ok"}
	wantID := []any{"q1", "q2", nil, nil, "q3", "q4"}
	if len(lines) != len(wantStatus) {
		t.Fatalf("got %d response lines, want %d:\n%s", len(lines), len(wantStatus), out.String())
	}
	for i, line := range lines {
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("response %d is not JSON: %q", i, line)
		}
		if response["status"] != wantStatus[i] || response["id"] != wantID[i] {
			t.Errorf("response %d = %s, want status %s and id %v", i, line, wantStatus[i], wantID[i])
		}
	}
}

func TestServeWithoutInputWritesNothing(t *testing.T) {
	var out bytes.Buffer
	if err := Serve(strings.NewReader(""), &out); err != nil || out.Len() != 0 {
		t.Errorf("Serve(empty) = %v, output %q", err, out.String())
	}
}

func TestEveryOperationOfTheContractIsOffered(t *testing.T) {
	want := []string{"registerMember", "setMemberStatus", "inspectMember", "registerCopy", "inspectCopy",
		"borrow", "return", "inspectLoan", "inspectOpenLoans", "assessOverdue"}
	if len(operations) != len(want) {
		t.Errorf("%d operations offered, want exactly the %d of the contract", len(operations), len(want))
	}
	for _, name := range want {
		if _, ok := operations[name]; !ok {
			t.Errorf("operation %q is missing", name)
		}
	}
}
