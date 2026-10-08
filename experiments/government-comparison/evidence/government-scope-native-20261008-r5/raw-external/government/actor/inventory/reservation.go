package inventory

import "errors"

var ErrOverflow = errors.New("available stock would overflow")

// Release adds released stock to the available count.
//
// This initial implementation is intentionally unsafe: the Order requires
// overflow protection before the candidate can pass its fixed check.
func Release(available, amount int64) (int64, error) {
	return available + amount, nil
}
