package orders

import (
	"context"
	"errors"
	billingContract "example.com/acme/modular-service/internal/contracts/billing"
	inventoryContract "example.com/acme/modular-service/internal/contracts/inventory"
	"example.com/acme/modular-service/internal/core"
	"fmt"
	"sync"
	"time"
)

var ErrInvalidOrder = errors.New("order requires customer, SKU, positive quantity, and positive unit price")

type CreateRequest struct {
	CustomerID, SKU string
	Quantity        int
	UnitPriceCents  int64
}
type Order struct {
	ID, CustomerID, SKU string
	Quantity            int
	UnitPriceCents      int64
	CreatedAt           time.Time
}
type Store interface {
	Save(context.Context, Order) error
}
type Service struct {
	store   Store
	stock   inventoryContract.Reserver
	billing billingContract.Issuer
	clock   core.Clock
	mu      sync.Mutex
	nextID  int
}

func New(store Store, stock inventoryContract.Reserver, billing billingContract.Issuer, clock core.Clock) *Service {
	return &Service{store: store, stock: stock, billing: billing, clock: clock}
}
func (s *Service) Create(ctx context.Context, request CreateRequest) (Order, error) {
	if request.CustomerID == "" || request.SKU == "" || request.Quantity <= 0 || request.UnitPriceCents <= 0 {
		return Order{}, ErrInvalidOrder
	}
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("order-%d", s.nextID)
	s.mu.Unlock()
	order := Order{ID: id, CustomerID: request.CustomerID, SKU: request.SKU, Quantity: request.Quantity, UnitPriceCents: request.UnitPriceCents, CreatedAt: s.clock.Now().UTC()}
	if _, err := s.stock.Reserve(ctx, inventoryContract.ReserveRequest{OrderID: id, SKU: request.SKU, Quantity: request.Quantity}); err != nil {
		return Order{}, fmt.Errorf("reserve stock: %w", err)
	}
	if _, err := s.billing.Issue(ctx, billingContract.IssueRequest{OrderID: id, AmountCents: int64(request.Quantity) * request.UnitPriceCents}); err != nil {
		return Order{}, fmt.Errorf("issue invoice: %w", err)
	}
	if err := s.store.Save(ctx, order); err != nil {
		return Order{}, fmt.Errorf("save order: %w", err)
	}
	return order, nil
}
