package bookings

// ActiveInterval is the reserved interval of an active reservation.
type ActiveInterval struct{ Start, End int }

// ActiveIntervals lists the reserved intervals of the machine's active
// reservations at the request time at, in creation order.
func (b *Module) ActiveIntervals(machineID string, at int) []ActiveInterval {
	var out []ActiveInterval
	for _, r := range b.resList {
		if r.machineID == machineID && r.active(at) {
			out = append(out, ActiveInterval{Start: r.start, End: r.end})
		}
	}
	return out
}
