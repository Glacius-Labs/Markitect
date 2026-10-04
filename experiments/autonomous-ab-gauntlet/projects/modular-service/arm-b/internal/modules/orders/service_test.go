package orders

import (
	"context"
	"errors"
	billingContract "example.com/acme/modular-service/internal/contracts/billing"
	inventoryContract "example.com/acme/modular-service/internal/contracts/inventory"
	"testing"
	"time"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC) }

type store struct{ orders []Order }

func (s *store) Save(_ context.Context, order Order) error {
	s.orders = append(s.orders, order)
	return nil
}

type stock struct {
	got inventoryContract.ReserveRequest
	err error
}

func (s *stock) Reserve(_ context.Context, req inventoryContract.ReserveRequest) (inventoryContract.Reservation, error) {
	s.got = req
	return inventoryContract.Reservation{ID: "r1"}, s.err
}

type invoices struct{ got billingContract.IssueRequest }

func (i *invoices) Issue(_ context.Context, req billingContract.IssueRequest) (billingContract.Invoice, error) {
	i.got = req
	return billingContract.Invoice{ID: "i1"}, nil
}
func TestCreateCoordinatesContractsAndPersistsOrder(t *testing.T) {
	st, inv, bill := &store{}, &stock{}, &invoices{}
	s := New(st, inv, bill, fixedClock{})
	order, err := s.Create(context.Background(), CreateRequest{CustomerID: "c1", SKU: "sku", Quantity: 2, UnitPriceCents: 450})
	if err != nil {
		t.Fatal(err)
	}
	if order.ID != "order-1" || !order.CreatedAt.Equal(fixedClock{}.Now()) {
		t.Fatalf("unexpected order: %+v", order)
	}
	if inv.got.OrderID != order.ID || inv.got.Quantity != 2 {
		t.Fatalf("reservation request: %+v", inv.got)
	}
	if bill.got.OrderID != order.ID || bill.got.AmountCents != 900 {
		t.Fatalf("invoice request: %+v", bill.got)
	}
	if len(st.orders) != 1 {
		t.Fatalf("saved %d orders, want 1", len(st.orders))
	}
}
func TestCreateRejectsInvalidAndPropagatesReservationFailure(t *testing.T) {
	s := New(&store{}, &stock{}, &invoices{}, fixedClock{})
	if _, err := s.Create(context.Background(), CreateRequest{CustomerID: "c1", Quantity: 1, UnitPriceCents: 10}); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("got %v, want ErrInvalidOrder", err)
	}
	want := errors.New("offline")
	failing := &stock{err: want}
	_, err := New(&store{}, failing, &invoices{}, fixedClock{}).Create(context.Background(), CreateRequest{CustomerID: "c1", SKU: "sku", Quantity: 1, UnitPriceCents: 10})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want wrapped dependency error", err)
	}
}
