package inventory

import "context"

type ReserveRequest struct {
	OrderID  string
	SKU      string
	Quantity int
}

type Reservation struct {
	ID       string
	OrderID  string
	SKU      string
	Quantity int
}

type Reserver interface {
	Reserve(context.Context, ReserveRequest) (Reservation, error)
}
