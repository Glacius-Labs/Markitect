// Package guardians holds the declarations that members are minors, their
// guardians and the consent of the guardians per tool family, and the
// operations on them.
package guardians

import (
	"sort"
	"time"

	"makerspace/equipment"
	"makerspace/members"
	"makerspace/policies"
)

// Guardianship declares a member a minor, with the guardian and the families
// the guardian consented to.
type Guardianship struct {
	MemberID, GuardianID string
	BirthDate            int // day
	Consent              []string
}

// Module is the guardians module: it owns every guardianship.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	byMember  map[string]*Guardianship
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module) *Module {
	return &Module{members: mem, equipment: eq, byMember: map[string]*Guardianship{}}
}

// adultFrom is the first day on which a member born on the day is an adult: the
// 18th anniversary of the birth date, 1 March for a birth date of 29 February
// when that year has no 29 February.
func adultFrom(birthDay int) int {
	b := policies.Epoch.AddDate(0, 0, birthDay)
	t := time.Date(b.Year()+18, b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(t.Sub(policies.Epoch) / (24 * time.Hour))
}

// IsMinor reports whether the member is a minor on the day: the member has a
// guardianship and the day is before the 18th anniversary of the birth date.
func (m *Module) IsMinor(memberID string, day int) bool {
	g := m.byMember[memberID]
	return g != nil && day < adultFrom(g.BirthDate)
}

// HasConsent reports whether a guardian's consent for the family is held.
func (m *Module) HasConsent(memberID, familyID string) bool {
	g := m.byMember[memberID]
	if g == nil {
		return false
	}
	i := sort.SearchStrings(g.Consent, familyID)
	return i < len(g.Consent) && g.Consent[i] == familyID
}

// ConsentMissing reports whether the member is refused guardian_consent_missing
// for the family on the day: a minor on the day without consent for the family.
func (m *Module) ConsentMissing(memberID, familyID string, day int) bool {
	return m.IsMinor(memberID, day) && !m.HasConsent(memberID, familyID)
}
