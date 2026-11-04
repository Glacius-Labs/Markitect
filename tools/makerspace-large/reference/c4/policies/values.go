package policies

import (
	"strconv"
	"time"
	"unicode"
)

// IsIdentifier reports whether s is an identifier: 1 to 64 characters from
// A-Z, a-z, 0-9, '.', '_' and '-', the first a letter or digit.
func IsIdentifier(s string) bool {
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

// IsText reports whether s holds a character without the White_Space property.
func IsText(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.White_Space, r) {
			return true
		}
	}
	return false
}

// Digits parses a nonempty run of decimal digits.
func Digits(s string) (int, bool) {
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

// ParseDay parses YYYY-MM-DD within 2000-01-01 to 2099-12-31.
func ParseDay(s string) (int, bool) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return 0, false
	}
	y, ok1 := Digits(s[0:4])
	m, ok2 := Digits(s[5:7])
	d, ok3 := Digits(s[8:10])
	if !ok1 || !ok2 || !ok3 || y < 2000 || y > 2099 || m < 1 || m > 12 || d < 1 {
		return 0, false
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d || int(t.Month()) != m {
		return 0, false
	}
	return int(t.Sub(Epoch) / (24 * time.Hour)), true
}

// FormatDay formats a day as YYYY-MM-DD.
func FormatDay(day int) string {
	return Epoch.AddDate(0, 0, day).Format("2006-01-02")
}

// ParseTime parses YYYY-MM-DDTHH:MMZ.
func ParseTime(s string) (int, bool) {
	if len(s) != 17 || s[10] != 'T' || s[13] != ':' || s[16] != 'Z' {
		return 0, false
	}
	day, ok := ParseDay(s[0:10])
	h, ok1 := Digits(s[11:13])
	m, ok2 := Digits(s[14:16])
	if !ok || !ok1 || !ok2 || h > 23 || m > 59 {
		return 0, false
	}
	return day*MinPerDay + h*60 + m, true
}

// FormatTime formats a minute as YYYY-MM-DDTHH:MMZ.
func FormatTime(m int) string {
	return Epoch.Add(time.Duration(m)*time.Minute).Format("2006-01-02T15:04") + "Z"
}

// ParseMonth parses YYYY-MM.
func ParseMonth(s string) (int, bool) {
	if len(s) != 7 || s[4] != '-' {
		return 0, false
	}
	y, ok1 := Digits(s[0:4])
	m, ok2 := Digits(s[5:7])
	if !ok1 || !ok2 || y < 2000 || y > 2099 || m < 1 || m > 12 {
		return 0, false
	}
	return (y-2000)*12 + m - 1, true
}

// FormatMonth formats a month index as YYYY-MM.
func FormatMonth(idx int) string {
	return strconv.Itoa(2000+idx/12) + "-" + TwoDigits(idx%12+1)
}

// TwoDigits formats n with at least two digits.
func TwoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// OptTime formats an optional minute, or gives null.
func OptTime(p *int) any {
	if p == nil {
		return nil
	}
	return FormatTime(*p)
}
