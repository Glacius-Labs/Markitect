package inventory

import (
	"context"
	"errors"
	contract "example.com/acme/modular-service/internal/contracts/inventory"
	"testing"
)

func TestReserveIsAtomicAndRejectsUnavailableStock(t *testing.T) {
	s := New(map[string]int{"SKU-1": 3})
	if _, err := s.Reserve(context.Background(), contract.ReserveRequest{OrderID: "o1", SKU: "SKU-1", Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reserve(context.Background(), contract.ReserveRequest{OrderID: "o2", SKU: "SKU-1", Quantity: 2}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
	if got := s.Available("SKU-1"); got != 1 {
		t.Fatalf("available=%d, want 1", got)
	}
}
