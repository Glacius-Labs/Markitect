// Package calendar provides the library's date type: a Gregorian calendar
// date without time or time zone, in the years 0001 to 9999.
package calendar

import (
	"errors"
	"fmt"
	"time"
)

// First and last year of the supported date range.
const (
	MinYear = 1
	MaxYear = 9999
)

// LastDate is the last date of the supported range, as text.
const LastDate = "9999-12-31"

// Date is a valid calendar date in the supported range. The zero value is
// 0001-01-01.
type Date struct {
	t time.Time // midnight UTC; used only for calendar arithmetic
}

// Parse reads a date written as YYYY-MM-DD. It refuses anything else:
// other layouts, times, missing zero padding, impossible days and years
// outside 0001 to 9999.
func Parse(text string) (Date, error) {
	if len(text) != 10 || text[4] != '-' || text[7] != '-' {
		return Date{}, fmt.Errorf("%q is not a date in the form YYYY-MM-DD", text)
	}
	year, okYear := digits(text[0:4])
	month, okMonth := digits(text[5:7])
	day, okDay := digits(text[8:10])
	if !okYear || !okMonth || !okDay {
		return Date{}, fmt.Errorf("%q is not a date in the form YYYY-MM-DD", text)
	}
	if year < MinYear || year > MaxYear {
		return Date{}, fmt.Errorf("the year of %q is outside 0001 to 9999", text)
	}
	if month < 1 || month > 12 {
		return Date{}, fmt.Errorf("the month of %q is not between 01 and 12", text)
	}
	if day < 1 || day > daysInMonth(year, time.Month(month)) {
		return Date{}, fmt.Errorf("%q is not a calendar date", text)
	}
	return newDate(year, time.Month(month), day), nil
}

// ErrOutOfRange is returned when date arithmetic leaves the supported range.
var ErrOutOfRange = errors.New("the date is after " + LastDate)

// AddDays returns the date n calendar days later. A result after 9999-12-31
// is an error; dates are never clamped or wrapped.
func (d Date) AddDays(n int) (Date, error) {
	t := d.t.AddDate(0, 0, n)
	if t.Year() > MaxYear {
		return Date{}, ErrOutOfRange
	}
	return Date{t: t}, nil
}

// Before reports whether d is strictly earlier than other.
func (d Date) Before(other Date) bool {
	return d.t.Before(other.t)
}

// String formats the date as YYYY-MM-DD.
func (d Date) String() string {
	return d.t.Format("2006-01-02")
}

func newDate(year int, month time.Month, day int) Date {
	return Date{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func daysInMonth(year int, month time.Month) int {
	// Day zero of the next month is the last day of this month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// digits converts a string of ASCII digits to a number.
func digits(text string) (int, bool) {
	n := 0
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		n = n*10 + int(text[i]-'0')
	}
	return n, true
}
