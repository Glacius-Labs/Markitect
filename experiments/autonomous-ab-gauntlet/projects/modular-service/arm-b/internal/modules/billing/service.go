package billing

import (
	"context"
	"errors"
	contract "example.com/acme/modular-service/internal/contracts/billing"
	"fmt"
	"sync"
)

var ErrInvalidAmount = errors.New("amount must be positive")

type Service struct {
	mu       sync.Mutex
	invoices map[string]contract.Invoice
	nextID   int
}

func New() *Service { return &Service{invoices: map[string]contract.Invoice{}} }
func (s *Service) Issue(_ context.Context, request contract.IssueRequest) (contract.Invoice, error) {
	if request.OrderID == "" || request.AmountCents <= 0 {
		return contract.Invoice{}, ErrInvalidAmount
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.invoices[request.OrderID]; ok {
		return old, nil
	}
	s.nextID++
	result := contract.Invoice{ID: fmt.Sprintf("invoice-%d", s.nextID), OrderID: request.OrderID, AmountCents: request.AmountCents}
	s.invoices[request.OrderID] = result
	return result, nil
}
