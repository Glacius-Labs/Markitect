package inventory

import (
	"errors"
	"math"
)

var ErrOverflow = errors.New("available stock would overflow")

// Release adds released stock to available inventory without wrapping.
func Release(available, amount int64) (int64, error) {
	if amount < 0 {
		return available, errors.New("release amount cannot be negative")
	}
	if amount > 0 && available > math.MaxInt64-amount {
		return available, ErrOverflow
	}
	return available + amount, nil
}
