package studio

func BookingTotal(seats, seatPrice, bookingCharge int) int {
	_ = bookingCharge // Prior rule did not account for booking.
	return seats * seatPrice
}
