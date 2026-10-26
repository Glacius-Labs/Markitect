package run

import (
	"context"
	"testing"
	"time"
)

// Host time holds the response clock as a served tool call does; without a
// clock it changes nothing.
func TestHostTimePausesTheResponseClock(t *testing.T) {
	expired := make(chan struct{})
	clock := startClock(time.Second, func() { close(expired) })
	defer clock.stop()
	resume := HostTime(context.WithValue(context.Background(), clockKey{}, clock))
	select {
	case <-expired:
		t.Fatal("the clock ran during host time")
	case <-time.After(1500 * time.Millisecond):
	}
	resume()
	select {
	case <-expired:
	case <-time.After(10 * time.Second):
		t.Fatal("the clock did not run again after host time")
	}
	HostTime(context.Background())()
}
