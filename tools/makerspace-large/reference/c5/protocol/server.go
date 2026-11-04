// Package protocol is the outer layer of the program: it reads requests, runs
// the checks every operation shares, keeps the audit log and calls the area
// modules, which it wires together.
package protocol

import (
	"makerspace/access"
	"makerspace/assessments"
	"makerspace/billing"
	"makerspace/bookings"
	"makerspace/classes"
	"makerspace/compliance"
	"makerspace/consumables"
	"makerspace/deposits"
	"makerspace/dunning"
	"makerspace/equipment"
	"makerspace/escalations"
	"makerspace/events"
	"makerspace/finance"
	"makerspace/guardians"
	"makerspace/guests"
	"makerspace/hazmat"
	"makerspace/households"
	"makerspace/inspections"
	"makerspace/instructors"
	"makerspace/jobs"
	"makerspace/ledger"
	"makerspace/loans"
	"makerspace/lockers"
	"makerspace/lockouts"
	"makerspace/materials"
	"makerspace/members"
	"makerspace/notices"
	"makerspace/parts"
	"makerspace/passes"
	"makerspace/payouts"
	"makerspace/policies"
	"makerspace/ppe"
	"makerspace/projects"
	"makerspace/promotions"
	"makerspace/purchasing"
	"makerspace/quotes"
	"makerspace/refunds"
	"makerspace/reporting"
	"makerspace/rooms"
	"makerspace/safety"
	"makerspace/scholarships"
	"makerspace/staffing"
	"makerspace/stocktake"
	"makerspace/training"
	"makerspace/volunteering"
	"makerspace/waitlists"
	"makerspace/waivers"
	"makerspace/workorders"
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
	led := ledger.New()
	ntc := notices.New(mem)
	hh := households.New(mem)
	gd := guardians.New(mem, eq)
	wv := waivers.New(mem, eq, gd)
	pp := ppe.New(mem, eq)
	lk := lockouts.New(eq)
	ins := inspections.New(eq, lk)
	// Staffing needs the active reservations of all machines of a family.
	stf := staffing.New(staffing.Needs{
		ActiveReservations: func(familyID string, at int) []staffing.Reservation {
			var out []staffing.Reservation
			for _, r := range bk.ActiveReservationsOfFamily(familyID, at) {
				out = append(out, staffing.Reservation{MachineID: r.MachineID, Start: r.Start, End: r.End})
			}
			return out
		},
	}, mem, eq)
	pt := parts.New(eq)
	rm := rooms.New()
	bill := billing.New(mem, led, hh)
	dep := deposits.New(mem, bill, led)
	pas := passes.New(mem, bill)
	prm := promotions.New(mem)
	sch := scholarships.New(mem)
	rfd := refunds.New(mem, bill, led)
	dun := dunning.New(mem, bill, ntc)
	lkr := lockers.New(mem, bill)
	gst := guests.New(mem, bill)
	vol := volunteering.New(mem, bill, stf)
	// Training needs the attendance gate of classes and the assessment gate of
	// assessments; classes and assessments depend on training, so the server
	// supplies the answers.
	var cls *classes.Module
	var asm *assessments.Module
	tr := training.New(training.Needs{
		AttendanceOwed: func(courseID, memberID string, enrolledAt, at int) bool {
			return cls.AttendanceOwed(courseID, memberID, enrolledAt, at)
		},
		AssessmentMet: func(courseID, memberID string) bool { return asm.AssessmentMet(courseID, memberID) },
	}, mem, eq, bill, gd, prm, sch)
	// Materials needs the hazard gate of hazmat and the freeze of stocktake;
	// both depend on materials, so the server supplies the answers.
	var hz *hazmat.Module
	var stk *stocktake.Module
	mat := materials.New(materials.Needs{
		HazardRefusal: func(memberID, materialID string, quantity, at int) string {
			return hz.HazardRefusal(memberID, materialID, quantity, at)
		},
		Counted: func(materialID string) bool { return stk.Counted(materialID) },
	}, mem, bill, prm)
	hz = hazmat.New(eq, mat, tr)
	stk = stocktake.New(mat, led)
	pur := purchasing.New(mat, led)
	cons := consumables.New(eq, mat, bill)
	lon := loans.New(mem, eq, tr, gd, wv, stf, bill, ntc)
	// Bookings needs the premises gate of access.
	var acc *access.Module
	bk = bookings.New(bookings.Needs{
		BadgeOutside: func(memberID string) bool { return acc.BadgeOutside(memberID) },
	}, mem, tr, eq, bill, gd, wv, pp, lk, ins, stf, pas, cons, lon)
	acc = access.New(mem, bill, gst, bk)
	// Instructors need the scheduled classes of classes, which depends on
	// instructors.
	inst := instructors.New(instructors.Needs{
		HasClasses: func(userID, familyID string, at int) bool { return cls.InstructorHasClasses(userID, familyID, at) },
	}, mem, eq, tr)
	cls = classes.New(mem, tr, inst, rm)
	asm = assessments.New(mem, tr)
	wl := waitlists.New(mem, tr, ntc)
	pyt := payouts.New(mem, led, inst, cls)
	evt := events.New(mem, rm, bill)
	sf := safety.New(eq)
	rep := reporting.New(eq, bk, bill)
	wo := workorders.New(mem, eq, sf, pt, lk)
	esc := escalations.New(eq, sf, bk, wo, ntc)
	prj := projects.New(mem, bk, mat)
	qt := quotes.New(eq, mat)
	jb := jobs.New(mem, eq, bill, led, tr, qt)
	cmp := compliance.New(mem, eq, gd, wv, pp, tr, lk, ins)
	fin := finance.New(mem, led, bill, hh, rfd)
	sv.members = mem

	for _, list := range [][]*policies.Operation{mem.Operations(), tr.Operations(), eq.Operations(), bk.Operations(),
		mat.Operations(), bill.Operations(), sf.Operations(), rep.Operations(), sv.auditOperations(),
		led.Operations(), ntc.Operations(), hh.Operations(), gd.Operations(), wv.Operations(), pp.Operations(),
		lk.Operations(), ins.Operations(), stf.Operations(), pt.Operations(), rm.Operations(),
		dep.Operations(), pas.Operations(), prm.Operations(), sch.Operations(), rfd.Operations(),
		dun.Operations(), lkr.Operations(), gst.Operations(), vol.Operations(), cons.Operations(),
		acc.Operations(), inst.Operations(), cls.Operations(), asm.Operations(), wl.Operations(), pyt.Operations(),
		evt.Operations(), hz.Operations(), pur.Operations(), stk.Operations(), wo.Operations(),
		esc.Operations(), lon.Operations(), prj.Operations(), qt.Operations(), jb.Operations(), cmp.Operations(),
		fin.Operations()} {
		for _, op := range list {
			sv.ops[op.Name] = op
		}
	}
	return sv
}
