package calendar

import (
	"errors"
	"testing"
)

func TestParseAcceptsValidDates(t *testing.T) {
	for _, text := range []string{"2026-10-01", "0001-01-01", "9999-12-31", "2024-02-29", "2000-02-29"} {
		date, err := Parse(text)
		if err != nil {
			t.Errorf("Parse(%q) refused: %v", text, err)
			continue
		}
		if date.String() != text {
			t.Errorf("Parse(%q).String() = %q", text, date.String())
		}
	}
}

func TestParseRefusesInvalidDates(t *testing.T) {
	for _, text := range []string{
		"", "2026-10", "2026-10-1", "26-10-01", "2026/10/01", "2026-10-01T00:00:00Z",
		" 2026-10-01", "2026-10-01 ", "not-a-date", "2026-13-01", "2026-00-10",
		"2026-10-00", "2026-10-32", "2026-02-29", "1900-02-29", "2026-04-31",
		"0000-06-01", "+026-10-01", "２０２６-10-01",
	} {
		if date, err := Parse(text); err == nil {
			t.Errorf("Parse(%q) = %s, want a refusal", text, date)
		}
	}
}

func TestAddDays(t *testing.T) {
	tests := []struct {
		from string
		days int
		want string
	}{
		{"2026-10-01", 14, "2026-10-15"},
		{"2026-12-25", 14, "2027-01-08"}, // across a year boundary
		{"2028-02-20", 14, "2028-03-05"}, // across a leap day
		{"2027-02-20", 14, "2027-03-06"}, // no leap day
		{"2026-10-01", 0, "2026-10-01"},  // zero days
		{"9999-12-17", 14, "9999-12-31"}, // the last date is allowed
		{"0001-01-01", 14, "0001-01-15"}, // the first year
		{"9999-12-31", 0, "9999-12-31"},  // staying on the last date
	}
	for _, test := range tests {
		from, err := Parse(test.from)
		if err != nil {
			t.Fatal(err)
		}
		got, err := from.AddDays(test.days)
		if err != nil {
			t.Errorf("%s + %d days refused: %v", test.from, test.days, err)
			continue
		}
		if got.String() != test.want {
			t.Errorf("%s + %d days = %s, want %s", test.from, test.days, got, test.want)
		}
	}
}

func TestAddDaysRefusesDatesAfterTheLastDate(t *testing.T) {
	for _, text := range []string{"9999-12-18", "9999-12-31"} {
		from, err := Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := from.AddDays(14); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("%s + 14 days = %s, err %v; want ErrOutOfRange", text, got, err)
		}
	}
}

func TestBefore(t *testing.T) {
	early, _ := Parse("2026-10-14")
	late, _ := Parse("2026-10-15")
	if !early.Before(late) {
		t.Error("2026-10-14 should be before 2026-10-15")
	}
	if late.Before(early) || late.Before(late) {
		t.Error("Before must be strict")
	}
}
