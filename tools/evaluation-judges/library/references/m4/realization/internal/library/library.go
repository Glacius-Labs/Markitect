// Package library holds the domain of the small library: members, book
// copies and loans, the lending policy and the overdue assessment.
//
// Every error returned by a Library method is an expected refusal: the
// request was understood but cannot be admitted, and the library state is
// exactly as it was before the call.
package library

import (
	"fmt"
	"strings"

	"example.com/smalllibrary/internal/calendar"
)

// The lending policy. All values are positive integers.
const (
	// MaxOpenLoans is the maximum number of open loans a member may hold
	// after a new loan is admitted.
	MaxOpenLoans = 3
	// LoanPeriodDays is the number of calendar days added to the borrowing
	// date to get the due date of a new loan of a standard copy.
	LoanPeriodDays = 14
	// ShortLoanPeriodDays is the same for a short copy.
	ShortLoanPeriodDays = 7
	// MaxRenewals is the number of times one loan can be renewed.
	MaxRenewals = 1
)

// LoanClass selects the loan period of a copy.
type LoanClass string

// The two loan classes.
const (
	Standard LoanClass = "standard"
	Short    LoanClass = "short"
)

// periodDays is the loan period of the class in calendar days.
func (c LoanClass) periodDays() int {
	if c == Short {
		return ShortLoanPeriodDays
	}
	return LoanPeriodDays
}

// MemberStatus is the status of a member.
type MemberStatus string

// The two member statuses.
const (
	Active    MemberStatus = "active"
	Suspended MemberStatus = "suspended"
)

// LoanStatus is the state of a loan.
type LoanStatus string

// The two loan states.
const (
	Open   LoanStatus = "open"
	Closed LoanStatus = "closed"
)

// ReturnOutcome tells whether a return closed a loan or found it closed.
type ReturnOutcome string

// The two outcomes of a successful return.
const (
	Returned        ReturnOutcome = "returned"
	AlreadyReturned ReturnOutcome = "alreadyReturned"
)

// Member is a registered member of the library.
type Member struct {
	ID     string
	Name   string
	Status MemberStatus
}

// Copy is one physical book. Availability is derived from loans.
type Copy struct {
	ID        string
	Title     string
	LoanClass LoanClass
	Available bool
}

// copyInfo is what registration fixes about a copy.
type copyInfo struct {
	title string
	class LoanClass
}

// Loan lends one copy to one member. DueDate is the current due date, which
// each renewal moves forward; Renewals counts the renewals. ReturnDate is
// meaningful only when the loan is closed.
type Loan struct {
	ID         string
	MemberID   string
	CopyID     string
	BorrowDate calendar.Date
	DueDate    calendar.Date
	Renewals   int
	Status     LoanStatus
	ReturnDate calendar.Date
}

// Notice reports one overdue loan.
type Notice struct {
	LoanID   string
	MemberID string
	CopyID   string
	DueDate  calendar.Date
}

// Library is one initially empty library. It is not safe for concurrent use;
// operations are processed one after the other.
type Library struct {
	members    map[string]Member
	copies     map[string]copyInfo
	loans      map[string]*Loan
	loanOrder  []*Loan            // all loans in creation order
	byMember   map[string][]*Loan // all loans of a member in creation order
	openByCopy map[string]*Loan   // the open loan of a copy, if any
}

// New returns an empty library.
func New() *Library {
	return &Library{
		members:    make(map[string]Member),
		copies:     make(map[string]copyInfo),
		loans:      make(map[string]*Loan),
		byMember:   make(map[string][]*Loan),
		openByCopy: make(map[string]*Loan),
	}
}

// RegisterMember registers a new active member.
func (l *Library) RegisterMember(id, name string) (Member, error) {
	if err := requireID("member", id); err != nil {
		return Member{}, err
	}
	if err := requireText("name", name); err != nil {
		return Member{}, err
	}
	if _, exists := l.members[id]; exists {
		return Member{}, fmt.Errorf("member %q is already registered", id)
	}
	member := Member{ID: id, Name: name, Status: Active}
	l.members[id] = member
	return member, nil
}

// SetMemberStatus sets the status of a registered member to "active" or
// "suspended". Setting the current status again succeeds. Existing loans are
// not touched.
func (l *Library) SetMemberStatus(id, status string) (Member, error) {
	if err := requireID("member", id); err != nil {
		return Member{}, err
	}
	newStatus := MemberStatus(status)
	if newStatus != Active && newStatus != Suspended {
		return Member{}, fmt.Errorf("status %q is neither %q nor %q", status, Active, Suspended)
	}
	member, ok := l.members[id]
	if !ok {
		return Member{}, fmt.Errorf("member %q is not registered", id)
	}
	member.Status = newStatus
	l.members[id] = member
	return member, nil
}

// InspectMember returns a registered member.
func (l *Library) InspectMember(id string) (Member, error) {
	if err := requireID("member", id); err != nil {
		return Member{}, err
	}
	member, ok := l.members[id]
	if !ok {
		return Member{}, fmt.Errorf("member %q is not registered", id)
	}
	return member, nil
}

// RegisterCopy registers a new available copy of the given loan class, which
// must be Standard or Short.
func (l *Library) RegisterCopy(id, title string, class LoanClass) (Copy, error) {
	if err := requireID("copy", id); err != nil {
		return Copy{}, err
	}
	if err := requireText("title", title); err != nil {
		return Copy{}, err
	}
	if class != Standard && class != Short {
		return Copy{}, fmt.Errorf("loan class %q is neither %q nor %q", class, Standard, Short)
	}
	if _, exists := l.copies[id]; exists {
		return Copy{}, fmt.Errorf("copy %q is already registered", id)
	}
	l.copies[id] = copyInfo{title: title, class: class}
	return l.copyRecord(id), nil
}

// InspectCopy returns a registered copy with its current availability.
func (l *Library) InspectCopy(id string) (Copy, error) {
	if err := requireID("copy", id); err != nil {
		return Copy{}, err
	}
	if _, ok := l.copies[id]; !ok {
		return Copy{}, fmt.Errorf("copy %q is not registered", id)
	}
	return l.copyRecord(id), nil
}

// copyRecord builds the record of a registered copy. A copy is available
// exactly when it has no open loan.
func (l *Library) copyRecord(id string) Copy {
	_, onLoan := l.openByCopy[id]
	info := l.copies[id]
	return Copy{ID: id, Title: info.title, LoanClass: info.class, Available: !onLoan}
}

func requireID(kind, id string) error {
	if id == "" {
		return fmt.Errorf("the %s identifier must not be empty", kind)
	}
	return nil
}

func requireText(what, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("the %s must contain a non-whitespace character", what)
	}
	return nil
}
