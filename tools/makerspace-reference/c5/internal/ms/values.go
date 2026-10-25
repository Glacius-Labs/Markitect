package ms

import (
	"encoding/json"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

// Times are minutes since 2000-01-01T00:00Z; dates are days since that day;
// months are months since 2000-01.

const minPerDay = 1440

var epoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

type valueKind int

const (
	kindIdentifier valueKind = iota
	kindText
	kindInteger
	kindTimestamp
	kindDate
	kindMonth
	kindEnum
)

// argSpec describes one argument of an operation.
type argSpec struct {
	name     string
	kind     valueKind
	min, max int
	enum     []string
	ref      refKind
}

// value is a validated argument: a string for every kind except integers.
type value struct {
	s string
	n int
}

func isIdentifier(s string) bool {
	if len(s) < 1 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		alnum := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		if i == 0 && !alnum {
			return false
		}
		if !alnum && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func isText(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.White_Space, r) {
			return true
		}
	}
	return false
}

func digits(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}

// parseDay parses YYYY-MM-DD within 2000-01-01 to 2099-12-31.
func parseDay(s string) (int, bool) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return 0, false
	}
	y, ok1 := digits(s[0:4])
	m, ok2 := digits(s[5:7])
	d, ok3 := digits(s[8:10])
	if !ok1 || !ok2 || !ok3 || y < 2000 || y > 2099 || m < 1 || m > 12 || d < 1 {
		return 0, false
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d || int(t.Month()) != m {
		return 0, false
	}
	return int(t.Sub(epoch) / (24 * time.Hour)), true
}

func formatDay(day int) string {
	return epoch.AddDate(0, 0, day).Format("2006-01-02")
}

// parseTime parses YYYY-MM-DDTHH:MMZ.
func parseTime(s string) (int, bool) {
	if len(s) != 17 || s[10] != 'T' || s[13] != ':' || s[16] != 'Z' {
		return 0, false
	}
	day, ok := parseDay(s[0:10])
	h, ok1 := digits(s[11:13])
	m, ok2 := digits(s[14:16])
	if !ok || !ok1 || !ok2 || h > 23 || m > 59 {
		return 0, false
	}
	return day*minPerDay + h*60 + m, true
}

func formatTime(m int) string {
	return epoch.Add(time.Duration(m)*time.Minute).Format("2006-01-02T15:04") + "Z"
}

func parseMonth(s string) (int, bool) {
	if len(s) != 7 || s[4] != '-' {
		return 0, false
	}
	y, ok1 := digits(s[0:4])
	m, ok2 := digits(s[5:7])
	if !ok1 || !ok2 || y < 2000 || y > 2099 || m < 1 || m > 12 {
		return 0, false
	}
	return (y-2000)*12 + m - 1, true
}

func formatMonth(idx int) string {
	return strconv.Itoa(2000+idx/12) + "-" + twoDigits(idx%12+1)
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// monthStart is the first minute of the month with the given index.
func monthStart(idx int) int {
	t := time.Date(2000+idx/12, time.Month(idx%12+1), 1, 0, 0, 0, 0, time.UTC)
	return int(t.Sub(epoch) / time.Minute)
}

// monthOf is the index of the month containing minute m.
func monthOf(m int) int {
	t := epoch.Add(time.Duration(m) * time.Minute)
	return (t.Year()-2000)*12 + int(t.Month()) - 1
}

func dayOf(m int) int { return m / minPerDay }

// overlaps reports whether the half-open intervals [a,b) and [c,d) overlap.
func overlaps(a, b, c, d int) bool { return a < d && c < b }

// validate checks one raw argument against its spec.
func validate(spec argSpec, raw json.RawMessage, present bool) (value, bool) {
	if !present {
		return value{}, false
	}
	if spec.kind == kindInteger {
		lit := string(raw)
		if len(lit) == 0 || lit[0] == '-' || len(lit) > 18 {
			return value{}, false
		}
		n, ok := digits(lit)
		if !ok || n < spec.min || n > spec.max {
			return value{}, false
		}
		return value{n: n}, true
	}
	if len(raw) == 0 || raw[0] != '"' {
		return value{}, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || !utf8.ValidString(s) {
		return value{}, false
	}
	switch spec.kind {
	case kindIdentifier:
		return value{s: s}, isIdentifier(s)
	case kindText:
		return value{s: s}, isText(s)
	case kindTimestamp:
		n, ok := parseTime(s)
		return value{s: s, n: n}, ok
	case kindDate:
		n, ok := parseDay(s)
		return value{s: s, n: n}, ok
	case kindMonth:
		n, ok := parseMonth(s)
		return value{s: s, n: n}, ok
	case kindEnum:
		for _, e := range spec.enum {
			if e == s {
				return value{s: s}, true
			}
		}
	}
	return value{}, false
}
