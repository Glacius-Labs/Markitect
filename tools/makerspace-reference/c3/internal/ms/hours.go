package ms

import "time"

// Booking rules of section 4.4.
const (
	granularity    = 30  // minutes
	maxReservation = 240 // minutes
)

// openingHours returns the opening period of a day as minutes of the day.
func openingHours(day int) (open, close int, ok bool) {
	switch epoch.AddDate(0, 0, day).Weekday() {
	case time.Sunday:
		return 0, 0, false
	case time.Saturday:
		return 10 * 60, 18 * 60, true
	}
	return 9 * 60, 21 * 60, true
}

func clock(minutes int) string {
	return twoDigits(minutes/60) + ":" + twoDigits(minutes%60)
}

// insideOpeningHours reports whether [start,end) lies in the opening period of
// the date of start.
func insideOpeningHours(start, end int) bool {
	day := dayOf(start)
	open, close, ok := openingHours(day)
	if !ok {
		return false
	}
	return start >= day*minPerDay+open && end <= day*minPerDay+close
}
