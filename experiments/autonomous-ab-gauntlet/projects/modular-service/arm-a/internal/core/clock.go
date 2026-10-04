package core

import "time"

// Clock makes time-dependent use cases deterministic in tests.
type Clock interface {
	Now() time.Time
}

// SystemClock reads the current wall clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
