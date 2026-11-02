package materials

// bulkDiscount is the discount percent of one purchase by its quantity.
func bulkDiscount(quantity int) int {
	switch {
	case quantity >= 25:
		return 12
	case quantity >= 10:
		return 5
	}
	return 0
}
