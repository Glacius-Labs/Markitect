// Package reporting holds the utilization and revenue reports. It reads the
// records of equipment, bookings and billing and keeps none of its own.
package reporting

import (
	"sort"

	"makerspace/billing"
	"makerspace/bookings"
	"makerspace/equipment"
)

// Module is the reporting module.
type Module struct {
	equipment *equipment.Module
	bookings  *bookings.Module
	billing   *billing.Module
}

// New returns the module, which reads the given modules.
func New(eq *equipment.Module, bk *bookings.Module, bill *billing.Module) *Module {
	return &Module{equipment: eq, bookings: bk, billing: bill}
}

type interval struct{ from, to int }

// unionMinutes is the number of minutes covered by at least one interval,
// counting only the part inside [lo,hi).
func unionMinutes(ivs []interval, lo, hi int) int {
	var clipped []interval
	for _, iv := range ivs {
		a, b := max(iv.from, lo), min(iv.to, hi)
		if a < b {
			clipped = append(clipped, interval{a, b})
		}
	}
	sort.Slice(clipped, func(i, j int) bool { return clipped[i].from < clipped[j].from })
	total, end := 0, lo
	for _, iv := range clipped {
		a := max(iv.from, end)
		if iv.to > a {
			total += iv.to - a
			end = iv.to
		}
	}
	return total
}

func utilizationPercent(used, available int) int {
	if available == 0 {
		return 0
	}
	return (used*100*2 + available) / (available * 2)
}
