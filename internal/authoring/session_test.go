package authoring

import (
	"testing"
	"time"
)

func TestShortDurationKeepsTheSettingsNotation(t *testing.T) {
	for d, want := range map[time.Duration]string{10 * time.Minute: "10m", time.Hour: "1h", 90 * time.Minute: "1h30m", 45 * time.Second: "45s"} {
		if got := shortDuration(d); got != want {
			t.Errorf("shortDuration(%s) = %s, want %s", d, got, want)
		}
	}
}
