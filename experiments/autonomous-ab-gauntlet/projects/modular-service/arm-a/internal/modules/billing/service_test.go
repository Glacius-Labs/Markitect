package billing

import (
	"context"
	"errors"
	contract "example.com/acme/modular-service/internal/contracts/billing"
	"testing"
)

func TestIssueIsIdempotentByOrder(t *testing.T) {
	s := New()
	req := contract.IssueRequest{OrderID: "o1", AmountCents: 1200}
	first, err := s.Issue(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Issue(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("duplicate issue changed invoice: first=%+v second=%+v", first, second)
	}
	if _, err := s.Issue(context.Background(), contract.IssueRequest{OrderID: "o2", AmountCents: 0}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("got %v, want ErrInvalidAmount", err)
	}
}
