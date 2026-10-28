package protocol

import (
	"fmt"

	"example.com/smalllibrary/internal/calendar"
	"example.com/smalllibrary/internal/library"
)

// arguments holds the required arguments of one request after they have been
// checked to be present JSON strings.
type arguments map[string]string

// date reads a required argument as a calendar date.
func (a arguments) date(name string) (calendar.Date, error) {
	date, err := calendar.Parse(a[name])
	if err != nil {
		return calendar.Date{}, fmt.Errorf("argument %q: %w", name, err)
	}
	return date, nil
}

// operation describes one operation of the contract: its required argument
// names, which are also the only argument names it accepts, and the function
// that performs it. A returned error is a refusal.
type operation struct {
	fields []string
	run    func(lib *library.Library, args arguments) (map[string]any, error)
}

var operations = map[string]operation{
	"registerMember": {
		fields: []string{"memberId", "name"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			member, err := lib.RegisterMember(a["memberId"], a["name"])
			return memberResult(member), err
		},
	},
	"setMemberStatus": {
		fields: []string{"memberId", "status"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			member, err := lib.SetMemberStatus(a["memberId"], a["status"])
			return memberResult(member), err
		},
	},
	"inspectMember": {
		fields: []string{"memberId"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			member, err := lib.InspectMember(a["memberId"])
			return memberResult(member), err
		},
	},
	"registerCopy": {
		fields: []string{"copyId", "title"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			copy, err := lib.RegisterCopy(a["copyId"], a["title"])
			return copyResult(copy), err
		},
	},
	"inspectCopy": {
		fields: []string{"copyId"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			copy, err := lib.InspectCopy(a["copyId"])
			return copyResult(copy), err
		},
	},
	"borrow": {
		fields: []string{"memberId", "copyId", "borrowDate"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			borrowDate, err := a.date("borrowDate")
			if err != nil {
				return nil, err
			}
			loan, err := lib.Borrow(a["memberId"], a["copyId"], borrowDate)
			return loanResult(loan), err
		},
	},
	"return": {
		fields: []string{"loanId", "returnDate"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			returnDate, err := a.date("returnDate")
			if err != nil {
				return nil, err
			}
			loan, outcome, err := lib.Return(a["loanId"], returnDate)
			return map[string]any{"outcome": string(outcome), "loan": newLoanView(loan)}, err
		},
	},
	"inspectLoan": {
		fields: []string{"loanId"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			loan, err := lib.InspectLoan(a["loanId"])
			return loanResult(loan), err
		},
	},
	"inspectOpenLoans": {
		fields: []string{"memberId"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			loans, err := lib.OpenLoans(a["memberId"])
			views := make([]loanView, 0, len(loans))
			for _, loan := range loans {
				views = append(views, newLoanView(loan))
			}
			return map[string]any{"memberId": a["memberId"], "count": len(views), "loans": views}, err
		},
	},
	"assessOverdue": {
		fields: []string{"assessmentDate"},
		run: func(lib *library.Library, a arguments) (map[string]any, error) {
			assessmentDate, err := a.date("assessmentDate")
			if err != nil {
				return nil, err
			}
			notices := make([]noticeView, 0)
			for _, notice := range lib.AssessOverdue(assessmentDate) {
				notices = append(notices, newNoticeView(notice))
			}
			return map[string]any{"notices": notices}, nil
		},
	},
}
