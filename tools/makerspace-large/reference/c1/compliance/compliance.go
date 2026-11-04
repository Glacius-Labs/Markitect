// Package compliance answers where a member stands towards the rules that gate
// the use of a machine. It only reads: it changes nothing.
package compliance

import (
	"makerspace/equipment"
	"makerspace/guardians"
	"makerspace/inspections"
	"makerspace/lockouts"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/ppe"
	"makerspace/training"
	"makerspace/waivers"
)

// Module is the compliance module: it owns no record.
type Module struct {
	members     *members.Module
	equipment   *equipment.Module
	guardians   *guardians.Module
	waivers     *waivers.Module
	ppe         *ppe.Module
	training    *training.Module
	lockouts    *lockouts.Module
	inspections *inspections.Module
}

// New returns the module.
func New(mem *members.Module, eq *equipment.Module, gd *guardians.Module, wv *waivers.Module, pp *ppe.Module,
	tr *training.Module, lk *lockouts.Module, ins *inspections.Module) *Module {
	return &Module{members: mem, equipment: eq, guardians: gd, waivers: wv, ppe: pp, training: tr,
		lockouts: lk, inspections: ins}
}

// Blockers lists the codes that stop the member from reserving the machine for
// a slot that starts on the day, in the order of the refusals of
// createReservation.
func (m *Module) Blockers(memberID, machineID string, day int) []string {
	family := m.equipment.Machine(machineID).FamilyID
	out := []string{}
	if !m.members.InGoodStanding(memberID) {
		out = append(out, "member_suspended")
	}
	if m.guardians.ConsentMissing(memberID, family, day) {
		out = append(out, "guardian_consent_missing")
	}
	if !m.training.Certified(memberID, family, day) {
		out = append(out, "not_certified")
	}
	if m.waivers.Missing(memberID, family) {
		out = append(out, "waiver_missing")
	}
	if m.lockouts.LockedOut(machineID) {
		out = append(out, "machine_locked_out")
	}
	if m.inspections.DueOn(machineID, day) {
		out = append(out, "inspection_due")
	}
	return out
}

func (m *Module) familyRow(memberID, familyID string, day int) policies.Rec {
	return policies.Rec{"familyId": familyID,
		"certified":    m.training.Certified(memberID, familyID, day),
		"consent":      !m.guardians.IsMinor(memberID, day) || m.guardians.HasConsent(memberID, familyID),
		"waiverSigned": !m.waivers.HasWaiver(familyID) || m.waivers.Signed(memberID, familyID),
		"ppeHeld":      !m.ppe.Missing(memberID, familyID)}
}
