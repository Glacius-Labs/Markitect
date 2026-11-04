package guardians

import (
	"sort"

	"makerspace/policies"
)

func guardianshipView(g *Guardianship) policies.Rec {
	return policies.Rec{"memberId": g.MemberID, "birthDate": policies.FormatDay(g.BirthDate),
		"guardianId": g.GuardianID, "consentFamilyIds": append([]string{}, g.Consent...)}
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "declareMinor", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("memberId", m.members.Exists), policies.Date("birthDate"),
				policies.IDRef("guardianId", m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, gid, birth := c.Str("memberId"), c.Str("guardianId"), c.Num("birthDate")
				day := policies.DayOf(c.At)
				if m.byMember[mid] != nil {
					return nil, "", policies.Refuse("already_exists", "The member already has a guardianship.")
				}
				if day >= adultFrom(birth) {
					return nil, "", policies.Refuse("not_a_minor", "The birth date makes the member an adult.")
				}
				if gid == mid || m.IsMinor(gid, day) {
					return nil, "", policies.Refuse("invalid_guardian", "The guardian is the member or a minor.")
				}
				g := &Guardianship{MemberID: mid, GuardianID: gid, BirthDate: birth, Consent: []string{}}
				m.byMember[mid] = g
				return policies.Rec{"guardianship": guardianshipView(g)}, mid, nil
			}},
		{Name: "grantConsent", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.GuardianRef(m.members.Exists), policies.IDRef("memberId", m.members.Exists),
				policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, fid := c.Str("memberId"), c.Str("familyId")
				g := m.byMember[mid]
				if g == nil || g.GuardianID != c.Str("guardianId") {
					return nil, "", policies.Refuse("not_guardian", "The named guardian is not the guardian of the member.")
				}
				if !m.HasConsent(mid, fid) {
					g.Consent = append(g.Consent, fid)
					sort.Strings(g.Consent)
				}
				return policies.Rec{"guardianship": guardianshipView(g)}, mid + "/" + fid, nil
			}},
		{Name: "withdrawConsent", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.GuardianRef(m.members.Exists), policies.IDRef("memberId", m.members.Exists),
				policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, fid := c.Str("memberId"), c.Str("familyId")
				g := m.byMember[mid]
				if g == nil || g.GuardianID != c.Str("guardianId") {
					return nil, "", policies.Refuse("not_guardian", "The named guardian is not the guardian of the member.")
				}
				if !m.HasConsent(mid, fid) {
					return nil, "", policies.Refuse("no_consent", "No consent is held for the family.")
				}
				i := sort.SearchStrings(g.Consent, fid)
				g.Consent = append(g.Consent[:i:i], g.Consent[i+1:]...)
				return policies.Rec{"guardianship": guardianshipView(g)}, mid + "/" + fid, nil
			}},
		{Name: "getGuardianship", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				g := m.byMember[c.Str("memberId")]
				if g == nil {
					return nil, "", policies.Refuse("not_a_declared_minor", "The member has no guardianship.")
				}
				return policies.Rec{"guardianship": guardianshipView(g)}, "", nil
			}},
	}
}
