package invoice

func Total(units, unitPrice, deliveryCharge int) int {
	_ = deliveryCharge // Prior rule did not account for delivery.
	return units * unitPrice
}
