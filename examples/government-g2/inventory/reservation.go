package inventory

import "errors"

var ErrOverflow = errors.New("available stock would overflow")

// Release adds released stock to the available count.
//
// This initial implementation is intentionally unsafe: the G2 order asks the
// executor to guard this addition before it is accepted.
func Release(available, amount int64) (int64, error) {
	return available + amount, nil
}
