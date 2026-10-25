package ms

import "sort"

type user struct {
	id, name, role string
}

type member struct {
	user         *user
	tier         string
	standing     string
	registeredAt int
	credit       int
}

type family struct{ id, name string }

type course struct {
	id, familyID, title string
	fee, capacity       int
}

type enrollment struct {
	courseID, memberID string
	status             string
	enrolledAt         int
	completedAt        *int
}

type certification struct {
	memberID, familyID string
	issuedOn           int // day
	revoked            bool
}

type machine struct {
	id, name, familyID string
	rate               int
}

type window struct {
	id, machineID string
	start, end    int
	reason        string
	cancelled     bool
}

type reservation struct {
	id, memberID, machineID string
	start, end              int
	status                  string // stored: reserved, checkedIn, completed, cancelled
	checkedInAt             *int
	checkedOutAt            *int
	rate                    *int
	metered                 *int
	amount                  *int
}

// charge is one invoice line; billed is false until it is on an invoice.
type charge struct {
	memberID, kind, ref string
	quantity, unit      int
	discount, amount    int
	at                  int
	billed              bool
}

type material struct {
	id, name    string
	unit, stock int
}

type purchase struct {
	id, memberID, materialID string
	quantity, unit, discount int
	amount, at               int
}

type invoice struct {
	id, memberID  string
	month         int
	issuedAt      int
	dueAt         int
	lines         []charge
	total         int
	creditApplied int
	paid, balance int
}

func (i *invoice) status() string {
	if i.balance == 0 {
		return "paid"
	}
	return "open"
}

func (i *invoice) overdue(at int) bool { return i.status() == "open" && at >= i.dueAt }

type payment struct {
	id, invoiceID, memberID string
	amount, applied, credit int
	at                      int
}

type incident struct {
	id, machineID string
	severity      string
	description   string
	reportedBy    string
	reportedAt    int
	resolved      bool
	resolution    string
	resolvedAt    int
}

type auditEntry struct {
	seq       int
	at        int
	actor     string
	actorRole string
	op        string
	target    string
	outcome   string // accepted or refused
}

// creditEvent is credit issued by an admin, counted in the revenue report.
type creditEvent struct {
	at, amount int
}

type state struct {
	users        map[string]*user
	members      map[string]*member
	families     map[string]*family
	courses      map[string]*course
	enrollments  map[string]*enrollment // key courseID + "\x00" + memberID
	certs        map[string]*certification
	machines     map[string]*machine
	windows      map[string]*window
	windowList   []*window
	reservations map[string]*reservation
	resList      []*reservation
	charges      []*charge
	materials    map[string]*material
	purchases    []*purchase
	invoices     map[string]*invoice
	invoiceList  []*invoice
	payments     []*payment
	incidents    map[string]*incident
	incidentList []*incident
	audit        []*auditEntry
	creditEvents []creditEvent
	invoiced     map[int]bool

	lastOK int // request time of the latest request answered ok, -1 before
	n      map[string]int
}

func newState() *state {
	s := &state{
		users:        map[string]*user{},
		members:      map[string]*member{},
		families:     map[string]*family{},
		courses:      map[string]*course{},
		enrollments:  map[string]*enrollment{},
		certs:        map[string]*certification{},
		machines:     map[string]*machine{},
		windows:      map[string]*window{},
		reservations: map[string]*reservation{},
		materials:    map[string]*material{},
		invoices:     map[string]*invoice{},
		incidents:    map[string]*incident{},
		invoiced:     map[int]bool{},
		lastOK:       -1,
		n:            map[string]int{},
	}
	s.users["admin"] = &user{id: "admin", name: "Administrator", role: "admin"}
	return s
}

// peekID returns the next identifier of a kind without consuming it.
func (s *state) peekID(prefix string) string {
	return prefix + "-" + itoa(s.n[prefix]+1)
}

// nextID consumes and returns the next identifier of a kind.
func (s *state) nextID(prefix string) string {
	s.n[prefix]++
	return prefix + "-" + itoa(s.n[prefix])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// reportedStatus applies the time-dependent missed state.
func (r *reservation) reportedStatus(at int) string {
	if r.status == "reserved" && r.end <= at {
		return "missed"
	}
	return r.status
}

func (r *reservation) active(at int) bool {
	st := r.reportedStatus(at)
	return st == "reserved" || st == "checkedIn"
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func enrollmentKey(courseID, memberID string) string { return courseID + "\x00" + memberID }

func certKey(memberID, familyID string) string { return memberID + "\x00" + familyID }

func (s *state) memberCharges(memberID string) []*charge {
	var out []*charge
	for _, c := range s.charges {
		if c.memberID == memberID {
			out = append(out, c)
		}
	}
	return out
}

// addCharge records a charge and returns it. The amount follows section 4.3.
func (s *state) addCharge(memberID, kind, ref string, quantity, unit, discount, at int) *charge {
	c := &charge{memberID: memberID, kind: kind, ref: ref, quantity: quantity, unit: unit,
		discount: discount, amount: chargeAmount(quantity, unit, discount), at: at}
	s.charges = append(s.charges, c)
	return c
}

func chargeAmount(quantity, unit, discount int) int {
	return (quantity*unit*(100-discount) + 50) / 100
}

func tierFee(tier string) int {
	switch tier {
	case "basic":
		return 2500
	case "plus":
		return 4900
	}
	return 9900
}

func tierDiscount(tier string) int {
	switch tier {
	case "basic":
		return 0
	case "plus":
		return 10
	}
	return 25
}
