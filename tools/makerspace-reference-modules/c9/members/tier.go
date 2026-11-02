package members

var tiers = []string{"basic", "plus", "pro"}

// TierFee is the monthly fee of a tier in cents.
func TierFee(tier string) int {
	switch tier {
	case "basic":
		return 2500
	case "plus":
		return 4900
	}
	return 9900
}

// TierDiscount is the usage discount of a tier in percent.
func TierDiscount(tier string) int {
	switch tier {
	case "basic":
		return 0
	case "plus":
		return 10
	}
	return 25
}
