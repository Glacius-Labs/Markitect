// Package policies holds what every module of the program shares: the formats
// of values, the refusal codes, the roles, the rules of time and money, the
// identifier counters and the vocabulary in which a module describes its
// operations. It depends on no other module of the project.
package policies

import "time"

// Times are minutes since 2000-01-01T00:00Z; dates are days since that day;
// months are months since 2000-01.

// MinPerDay is the length of a day in minutes.
const MinPerDay = 1440

// Epoch is the start of the time scale.
var Epoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// MonthStart is the first minute of the month with the given index.
func MonthStart(idx int) int {
	t := time.Date(2000+idx/12, time.Month(idx%12+1), 1, 0, 0, 0, 0, time.UTC)
	return int(t.Sub(Epoch) / time.Minute)
}

// MonthOf is the index of the month containing minute m.
func MonthOf(m int) int {
	t := Epoch.Add(time.Duration(m) * time.Minute)
	return (t.Year()-2000)*12 + int(t.Month()) - 1
}

// DayOf is the day containing minute m.
func DayOf(m int) int { return m / MinPerDay }

// Overlaps reports whether the half-open intervals [a,b) and [c,d) overlap.
func Overlaps(a, b, c, d int) bool { return a < d && c < b }
