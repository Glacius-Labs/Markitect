// Package waivers holds the current waiver of each tool family and the
// signatures of members, and the operations on them.
package waivers

import (
	"makerspace/equipment"
	"makerspace/guardians"
	"makerspace/members"
	"makerspace/policies"
)

type waiver struct {
	familyID, text string
	version        int
}

// signature is one signature of one version of a family's waiver.
type signature struct {
	memberID, familyID string
	version, signedAt  int
}

// Module is the waivers module: it owns every waiver and signature.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	guardians *guardians.Module
	waivers   map[string]*waiver
	latest    map[string]*signature // latest signature, key memberID + "\x00" + familyID
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module, gd *guardians.Module) *Module {
	return &Module{members: mem, equipment: eq, guardians: gd,
		waivers: map[string]*waiver{}, latest: map[string]*signature{}}
}

func sigKey(memberID, familyID string) string { return memberID + "\x00" + familyID }

// HasWaiver reports whether the family has a waiver.
func (m *Module) HasWaiver(familyID string) bool { return m.waivers[familyID] != nil }

// Signed reports whether the member has signed the current version of the
// family's waiver with a signature that has not lapsed on the day. A signature
// lapses 365 days after the day it was made. It is false when the family has no
// waiver.
func (m *Module) Signed(memberID, familyID string, day int) bool {
	w, s := m.waivers[familyID], m.latest[sigKey(memberID, familyID)]
	return w != nil && s != nil && s.version == w.version && day < policies.DayOf(s.signedAt)+365
}

// Missing reports whether the member is refused waiver_missing for the
// family on the day: the family has a waiver and the member has not signed its
// current version with a signature that has not lapsed.
func (m *Module) Missing(memberID, familyID string, day int) bool {
	return m.HasWaiver(familyID) && !m.Signed(memberID, familyID, day)
}

func waiverView(w *waiver) policies.Rec {
	return policies.Rec{"familyId": w.familyID, "version": w.version, "text": w.text}
}

func signatureView(s *signature) policies.Rec {
	return policies.Rec{"memberId": s.memberID, "familyId": s.familyID, "version": s.version,
		"signedAt": policies.FormatTime(s.signedAt)}
}
