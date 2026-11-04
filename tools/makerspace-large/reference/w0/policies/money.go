package policies

// ChargeAmount is the amount of one charge line in cents: the exact value
// rounded half up to a whole cent, once per line.
func ChargeAmount(quantity, unit, discount int) int {
	return (quantity*unit*(100-discount) + 50) / 100
}
