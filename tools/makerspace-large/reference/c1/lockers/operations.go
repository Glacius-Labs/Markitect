package lockers

import (
	"sort"

	"makerspace/billing"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addLocker", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.ID("lockerId"), policies.Integer("monthlyCents", 1, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("lockerId")
				if m.lockers[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A locker with identifier %s already exists.", id)
				}
				l := &Locker{ID: id, Monthly: c.Num("monthlyCents")}
				m.lockers[id] = l
				return policies.Rec{"locker": lockerView(l)}, id, nil
			}},
		{Name: "rentLocker", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("lockerId", m.LockerExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, lid := c.Str("memberId"), c.Str("lockerId")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				held := 0
				for _, r := range m.list {
					if !r.Active {
						continue
					}
					if r.LockerID == lid {
						return nil, "", policies.Refuse("locker_taken", "The locker has an active rental.")
					}
					if r.MemberID == mid {
						held++
					}
				}
				if held >= 2 {
					return nil, "", policies.Refuse("rental_limit", "The member already has 2 active rentals.")
				}
				r := &Rental{ID: m.ids.Next("rnt"), LockerID: lid, MemberID: mid, Active: true, StartedAt: c.At}
				m.byID[r.ID] = r
				m.list = append(m.list, r)
				return policies.Rec{"rental": rentalView(r)}, r.ID, nil
			}},
		{Name: "endLockerRental", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("rentalId", m.RentalExists, m.RentalOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := m.byID[c.Str("rentalId")]
				if !r.Active {
					return nil, "", policies.Refuse("invalid_state", "The rental has ended.")
				}
				at := c.At
				r.Active, r.EndedAt = false, &at
				return policies.Rec{"rental": rentalView(r)}, r.ID, nil
			}},
		{Name: "billLockers", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				month := c.Num("month")
				if month != policies.MonthOf(c.At) {
					return nil, "", policies.Refuse("not_current_month", "The month is not the month of the request time.")
				}
				if m.billed[month] {
					return nil, "", policies.Refuse("already_billed", "The lockers of the month have already been billed.")
				}
				var active []*Rental
				for _, r := range m.list {
					if r.Active {
						active = append(active, r)
					}
				}
				sort.SliceStable(active, func(i, j int) bool { return active[i].LockerID < active[j].LockerID })
				charges := []policies.Rec{}
				for _, r := range active {
					ch := m.billing.AddCharge(r.MemberID, "locker", r.ID, 1, m.lockers[r.LockerID].Monthly, 0, c.At)
					charges = append(charges, billing.ChargeView(ch))
				}
				m.billed[month] = true
				return policies.Rec{"month": c.Str("month"), "charges": charges}, c.Str("month"), nil
			}},
	}
}
