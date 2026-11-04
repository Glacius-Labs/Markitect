package reporting

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
