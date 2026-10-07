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
