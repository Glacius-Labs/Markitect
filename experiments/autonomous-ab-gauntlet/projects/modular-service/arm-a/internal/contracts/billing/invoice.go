package billing

import "context"

type IssueRequest struct {
	OrderID     string
	AmountCents int64
}

type Invoice struct {
	ID          string
	OrderID     string
	AmountCents int64
}

type Issuer interface {
	Issue(context.Context, IssueRequest) (Invoice, error)
}
