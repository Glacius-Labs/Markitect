package buildinfo

import "testing"

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		latest, current string
		want            bool
	}{
		{"0.15.0", "0.14.1", true},
		{"0.15.0", "0.15.0", false},
		{"0.15.0", "0.15.1", false},
		{"1.0.0", "0.99.99", true},
		{"0.10.0", "0.9.0", true}, // numbers, not text
		{"v0.15.0", "0.14.0", true},
		{"0.15.0+abc", "0.15.0", false},
		{"0.15.0", "0.15.0-dev", true}, // a release comes after its development version
		{"0.15.0", "0.16.0-dev", false},
		{"0.15.0", "0.0.0-dev", true},
		{"0.16.0-rc.1", "0.15.0", true},
		{"0.16.0", "0.16.0-rc.1", true},
	} {
		got, err := Newer(tc.latest, tc.current)
		if err != nil || got != tc.want {
			t.Errorf("Newer(%q, %q) = %v, %v; want %v", tc.latest, tc.current, got, err, tc.want)
		}
	}
}

func TestNewerRefusesWhatIsNoVersion(t *testing.T) {
	for _, bad := range []string{"", "1", "1.2", "1.2.3.4", "a.b.c", "1.2.x", "01.2.3", "1.2.3-", "-1.2.3"} {
		if _, err := Newer(bad, "1.0.0"); err == nil {
			t.Errorf("Newer(%q, ...) accepted a bad version", bad)
		}
		if _, err := Newer("1.0.0", bad); err == nil {
			t.Errorf("Newer(..., %q) accepted a bad version", bad)
		}
	}
}

func TestIsDev(t *testing.T) {
	for v, want := range map[string]bool{
		"0.0.0-dev":      true,
		"0.16.0-dev":     true,
		"0.16.0-dev+abc": true,
		"0.16.0":         false,
		"0.16.0-rc.1":    false,
		"nonsense":       false,
	} {
		if got := IsDev(v); got != want {
			t.Errorf("IsDev(%q) = %v, want %v", v, got, want)
		}
	}
}
