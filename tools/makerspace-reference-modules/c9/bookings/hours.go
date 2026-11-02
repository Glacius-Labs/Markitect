package bookings

import (
	"time"

	"makerspace/policies"
)

// Booking rules of section 4.4.
const (
	granularity    = 15  // minutes
	maxReservation = 240 // minutes
)

// OpeningHours returns the opening period of a day as minutes of the day.
func OpeningHours(day int) (open, close int, ok bool) {
	switch policies.Epoch.AddDate(0, 0, day).Weekday() {
	case time.Sunday:
		return 12 * 60, 16 * 60, true
	case time.Saturday:
		return 9 * 60, 17 * 60, true
	case time.Friday:
		return 8 * 60, 20 * 60, true
	}
	return 8 * 60, 22 * 60, true
}

func clock(minutes int) string {
	return policies.TwoDigits(minutes/60) + ":" + policies.TwoDigits(minutes%60)
}

// insideOpeningHours reports whether [start,end) lies in the opening period of
// the date of start.
func insideOpeningHours(start, end int) bool {
	day := policies.DayOf(start)
	open, close, ok := OpeningHours(day)
	if !ok {
		return false
	}
	return start >= day*policies.MinPerDay+open && end <= day*policies.MinPerDay+close
}
