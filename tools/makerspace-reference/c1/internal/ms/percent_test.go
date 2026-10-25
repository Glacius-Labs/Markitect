package ms

import "testing"

func TestUtilizationPercentRoundsHalfUp(t *testing.T) {
	cases := []struct{ used, avail, want int }{
		{0, 0, 0}, {5, 0, 0}, {0, 100, 0}, {1, 8, 13}, {1, 3, 33}, {2, 3, 67}, {1, 2, 50},
		{3, 8, 38}, {1, 200, 1}, {1, 201, 0}, {150, 100, 150}, {120, 720, 17},
	}
	for _, c := range cases {
		if got := utilizationPercent(c.used, c.avail); got != c.want {
			t.Errorf("%d/%d: %d, want %d", c.used, c.avail, got, c.want)
		}
	}
}

func TestChargeAmountRoundsHalfUpOncePerLine(t *testing.T) {
	cases := []struct{ qty, unit, disc, want int }{
		{37, 45, 10, 1499}, {9, 199, 0, 1791}, {1, 1, 0, 1}, {0, 45, 10, 0}, {1, 5, 10, 5},
		{1, 1, 50, 1}, {1, 1, 49, 1}, {1, 1, 51, 0}, {1, 3, 25, 2},
	}
	for _, c := range cases {
		if got := chargeAmount(c.qty, c.unit, c.disc); got != c.want {
			t.Errorf("%d x %d at %d%%: %d, want %d", c.qty, c.unit, c.disc, got, c.want)
		}
	}
}

func TestTimeArithmetic(t *testing.T) {
	if d, ok := parseDay("2027-03-01"); !ok || formatDay(d+365) != "2028-02-29" {
		t.Fatal("2027-03-01 plus 365 days")
	}
	if _, ok := parseDay("2026-02-29"); ok {
		t.Fatal("2026-02-29 is not valid")
	}
	if _, ok := parseDay("2028-02-29"); !ok {
		t.Fatal("2028-02-29 is valid")
	}
	if got := formatMonth(monthOf(mustTime(t, "2026-12-31T23:59Z"))); got != "2026-12" {
		t.Fatal(got)
	}
	if got := formatTime(monthStart(monthOf(mustTime(t, "2026-12-31T23:59Z")) + 1)); got != "2027-01-01T00:00Z" {
		t.Fatal(got)
	}
	if !overlaps(0, 60, 59, 120) || overlaps(0, 60, 60, 120) {
		t.Fatal("half-open overlap")
	}
}

func mustTime(t *testing.T, s string) int {
	n, ok := parseTime(s)
	if !ok {
		t.Fatal(s)
	}
	return n
}
