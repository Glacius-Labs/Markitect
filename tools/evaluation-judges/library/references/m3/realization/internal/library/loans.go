package library

import (
	"fmt"

	"example.com/smalllibrary/internal/calendar"
)

// Borrow admits a new loan of a copy to a member. It is admitted only when
// the member is registered and active, the copy is registered and available,
// the member holds fewer than MaxOpenLoans open loans and the due date does
// not fall after 9999-12-31. The due date is the borrowing date plus the loan
// period of the copy's loan class.
func (l *Library) Borrow(memberID, copyID string, borrowDate calendar.Date) (Loan, error) {
	if err := requireID("member", memberID); err != nil {
		return Loan{}, err
	}
	if err := requireID("copy", copyID); err != nil {
		return Loan{}, err
	}
	member, ok := l.members[memberID]
	if !ok {
		return Loan{}, fmt.Errorf("member %q is not registered", memberID)
	}
	if member.Status != Active {
		return Loan{}, fmt.Errorf("member %q is suspended and may not borrow", memberID)
	}
	info, ok := l.copies[copyID]
	if !ok {
		return Loan{}, fmt.Errorf("copy %q is not registered", copyID)
	}
	if _, onLoan := l.openByCopy[copyID]; onLoan {
		return Loan{}, fmt.Errorf("copy %q is already on loan", copyID)
	}
	if l.openLoanCount(memberID) >= MaxOpenLoans {
		return Loan{}, fmt.Errorf("member %q already holds the maximum of %d open loans", memberID, MaxOpenLoans)
	}
	dueDate, err := borrowDate.AddDays(info.class.periodDays())
	if err != nil {
		return Loan{}, fmt.Errorf("the due date of a loan borrowed on %s would be after %s", borrowDate, calendar.LastDate)
	}

	loan := &Loan{
		ID:         fmt.Sprintf("loan-%d", len(l.loanOrder)+1),
		MemberID:   memberID,
		CopyID:     copyID,
		BorrowDate: borrowDate,
		DueDate:    dueDate,
		Status:     Open,
	}
	l.loans[loan.ID] = loan
	l.loanOrder = append(l.loanOrder, loan)
	l.byMember[memberID] = append(l.byMember[memberID], loan)
	l.openByCopy[copyID] = loan
	return *loan, nil
}

// Return closes an open loan on the given date. Returning a loan that is
// already closed succeeds with AlreadyReturned and changes nothing. In both
// cases the date must not be before the borrowing date. Neither suspension
// nor overdue state nor the borrowing limit prevents a return.
func (l *Library) Return(loanID string, returnDate calendar.Date) (Loan, ReturnOutcome, error) {
	if err := requireID("loan", loanID); err != nil {
		return Loan{}, "", err
	}
	loan, ok := l.loans[loanID]
	if !ok {
		return Loan{}, "", fmt.Errorf("loan %q does not exist", loanID)
	}
	if returnDate.Before(loan.BorrowDate) {
		return Loan{}, "", fmt.Errorf("the return date %s is before the borrowing date %s", returnDate, loan.BorrowDate)
	}
	if loan.Status == Closed {
		return *loan, AlreadyReturned, nil
	}
	loan.Status = Closed
	loan.ReturnDate = returnDate
	// Only this loan can be the open loan of its copy, so the copy becomes
	// available again.
	delete(l.openByCopy, loan.CopyID)
	return *loan, Returned, nil
}

// InspectLoan returns a loan, open or closed.
func (l *Library) InspectLoan(loanID string) (Loan, error) {
	if err := requireID("loan", loanID); err != nil {
		return Loan{}, err
	}
	loan, ok := l.loans[loanID]
	if !ok {
		return Loan{}, fmt.Errorf("loan %q does not exist", loanID)
	}
	return *loan, nil
}

// OpenLoans returns the open loans of a registered member, oldest first.
func (l *Library) OpenLoans(memberID string) ([]Loan, error) {
	if _, err := l.InspectMember(memberID); err != nil {
		return nil, err
	}
	open := []Loan{}
	for _, loan := range l.byMember[memberID] {
		if loan.Status == Open {
			open = append(open, *loan)
		}
	}
	return open, nil
}

// AssessOverdue returns one notice for every open loan whose due date is
// strictly earlier than the assessment date. It changes no state.
func (l *Library) AssessOverdue(assessmentDate calendar.Date) []Notice {
	notices := []Notice{}
	for _, loan := range l.loanOrder {
		if loan.Status == Open && loan.DueDate.Before(assessmentDate) {
			notices = append(notices, Notice{
				LoanID:   loan.ID,
				MemberID: loan.MemberID,
				CopyID:   loan.CopyID,
				DueDate:  loan.DueDate,
			})
		}
	}
	return notices
}

func (l *Library) openLoanCount(memberID string) int {
	count := 0
	for _, loan := range l.byMember[memberID] {
		if loan.Status == Open {
			count++
		}
	}
	return count
}
