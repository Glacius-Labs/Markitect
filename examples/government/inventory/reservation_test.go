package inventory

import "testing"

func TestReleaseRejectsAmountAboveReservation(t *testing.T) {
	if got, ok := Release(3, 2, 3); ok || got != 3 {
		t.Fatalf("Release(3, 2, 3) = (%d, %t), want (3, false)", got, ok)
	}
}

func TestReleaseRejectsOverflowThatWouldMakeAvailableStockNegative(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	if got, ok := Release(maxInt-1, 2, 2); ok || got != maxInt-1 {
		t.Fatalf("Release(maxInt-1, 2, 2) = (%d, %t), want (maxInt-1, false)", got, ok)
	}
}
