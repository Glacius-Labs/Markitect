package protocol

import "example.com/smalllibrary/internal/library"

// The views give the domain records their wire shape.

type memberView struct {
	MemberID string `json:"memberId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

type copyView struct {
	CopyID    string `json:"copyId"`
	Title     string `json:"title"`
	LoanClass string `json:"loanClass"`
	Available bool   `json:"available"`
}

type loanView struct {
	LoanID     string  `json:"loanId"`
	MemberID   string  `json:"memberId"`
	CopyID     string  `json:"copyId"`
	BorrowDate string  `json:"borrowDate"`
	DueDate    string  `json:"dueDate"`
	Status     string  `json:"status"`
	ReturnDate *string `json:"returnDate"` // null while the loan is open
	Renewals   int     `json:"renewals"`
}

type noticeView struct {
	LoanID   string `json:"loanId"`
	MemberID string `json:"memberId"`
	CopyID   string `json:"copyId"`
	DueDate  string `json:"dueDate"`
}

func newLoanView(loan library.Loan) loanView {
	view := loanView{
		LoanID:     loan.ID,
		MemberID:   loan.MemberID,
		CopyID:     loan.CopyID,
		BorrowDate: loan.BorrowDate.String(),
		DueDate:    loan.DueDate.String(),
		Status:     string(loan.Status),
		Renewals:   loan.Renewals,
	}
	if loan.Status == library.Closed {
		returnDate := loan.ReturnDate.String()
		view.ReturnDate = &returnDate
	}
	return view
}

func newNoticeView(notice library.Notice) noticeView {
	return noticeView{
		LoanID:   notice.LoanID,
		MemberID: notice.MemberID,
		CopyID:   notice.CopyID,
		DueDate:  notice.DueDate.String(),
	}
}

func memberResult(member library.Member) map[string]any {
	return map[string]any{"member": memberView{
		MemberID: member.ID,
		Name:     member.Name,
		Status:   string(member.Status),
	}}
}

func copyResult(copy library.Copy) map[string]any {
	return map[string]any{"copy": copyView{
		CopyID:    copy.ID,
		Title:     copy.Title,
		LoanClass: string(copy.LoanClass),
		Available: copy.Available,
	}}
}

func loanResult(loan library.Loan) map[string]any {
	return map[string]any{"loan": newLoanView(loan)}
}
