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

// PausedFee is the monthly fee of a paused member in cents, whatever the tier.
const PausedFee = 500

// MonthlyFee is the monthly fee of the member in cents: the fee of the tier, or
// the fixed fee while the membership is paused.
func MonthlyFee(m *Member) int {
	if m.Standing == "paused" {
		return PausedFee
	}
	return TierFee(m.Tier)
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

// Tiers lists the tiers in ascending order.
func Tiers() []string { return append([]string(nil), tiers...) }
