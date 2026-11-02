// Package protocol is the outer layer of the program: it reads requests, runs
// the checks every operation shares, keeps the audit log and calls the area
// modules, which it wires together.
package protocol

import (
	"makerspace/billing"
	"makerspace/bookings"
	"makerspace/equipment"
	"makerspace/materials"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/reporting"
	"makerspace/safety"
	"makerspace/training"
)

// Server holds one makerspace and answers request lines.
type Server struct {
	members *members.Module
	ops     map[string]*policies.Operation

	audit  []*auditEntry
	lastOK int // request time of the latest request answered ok, -1 before
}

// NewServer returns a server in the initial state.
func NewServer() *Server {
	sv := &Server{lastOK: -1, ops: map[string]*policies.Operation{}}

	// Equipment needs the reservations of bookings, which depends on
	// equipment: the server supplies the answer.
	var bk *bookings.Module
	eq := equipment.New(equipment.Needs{
		ActiveReservationOverlaps: func(machineID string, start, end, at int) bool {
			return bk.ActiveReservationOverlaps(machineID, start, end, at)
		},
	})
	mem := members.New()
	bill := billing.New(mem)
	tr := training.New(mem, eq, bill)
	sf := safety.New(eq)
	bk = bookings.New(mem, tr, eq, bill, sf)
	mat := materials.New(mem, bill)
	rep := reporting.New(eq, bk, bill)
	sv.members = mem

	for _, list := range [][]*policies.Operation{mem.Operations(), tr.Operations(), eq.Operations(), bk.Operations(),
		mat.Operations(), bill.Operations(), sf.Operations(), rep.Operations(), sv.auditOperations()} {
		for _, op := range list {
			sv.ops[op.Name] = op
		}
	}
	return sv
}
