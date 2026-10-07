package inventory

// Release returns the updated available count when the requested reservation
// is valid. The example is intentionally small; Government models the obligation
// and ownership without inspecting this Go implementation.
func Release(available, reserved, amount int) (int, bool) {
	maxInt := int(^uint(0) >> 1)
	if amount <= 0 || amount > reserved || available < 0 || available > maxInt-amount {
		return available, false
	}
	return available + amount, true
}
