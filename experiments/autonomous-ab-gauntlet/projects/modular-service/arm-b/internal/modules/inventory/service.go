package inventory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	contract "example.com/acme/modular-service/internal/contracts/inventory"
)

var ErrUnavailable = errors.New("insufficient stock")

type Service struct {
	mu     sync.Mutex
	stock  map[string]int
	nextID int
}

func New(stock map[string]int) *Service {
	copy := make(map[string]int, len(stock))
	for sku, quantity := range stock {
		copy[sku] = quantity
	}
	return &Service{stock: copy}
}

func (s *Service) Reserve(_ context.Context, request contract.ReserveRequest) (contract.Reservation, error) {
	if request.OrderID == "" || request.SKU == "" || request.Quantity <= 0 {
		return contract.Reservation{}, errors.New("order, SKU, and positive quantity are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stock[request.SKU] < request.Quantity {
		return contract.Reservation{}, ErrUnavailable
	}
	s.stock[request.SKU] -= request.Quantity
	s.nextID++
	return contract.Reservation{ID: fmt.Sprintf("reservation-%d", s.nextID), OrderID: request.OrderID, SKU: request.SKU, Quantity: request.Quantity}, nil
}

func (s *Service) Available(sku string) int { s.mu.Lock(); defer s.mu.Unlock(); return s.stock[sku] }
