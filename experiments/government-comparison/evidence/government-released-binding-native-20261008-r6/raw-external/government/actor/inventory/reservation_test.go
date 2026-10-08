package inventory

import (
	"errors"
	"math"
	"testing"
)

func TestReleaseRejectsOverflow(t *testing.T) {
	if _, err := Release(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Release(MaxInt64, 1) error = %v, want ErrOverflow", err)
	}
}

func TestReleaseAddsWithinRange(t *testing.T) {
	got, err := Release(8, 3)
	if err != nil || got != 11 {
		t.Fatalf("Release(8, 3) = (%d, %v), want (11, nil)", got, err)
	}
}
