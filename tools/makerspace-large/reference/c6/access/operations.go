package access

import "makerspace/policies"

// inOpeningPeriod reports whether the minute lies in the opening period of its
// date: the opening time is in, the closing time is out.
func (m *Module) inOpeningPeriod(at int) bool {
	day := policies.DayOf(at)
	h, ok := m.bookings.OpeningHoursOf(day)
	if !ok {
		return false
	}
	minute := at - day*policies.MinPerDay
	return h.OpenMinute <= minute && minute < h.CloseMinute
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	badgeArg := func() policies.ArgSpec {
		return policies.OwnedIDRef("badgeId", m.BadgeExists, m.BadgeOwner)
	}
	return []*policies.Operation{
		{Name: "issueBadge", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				if m.activeBadge(mid) != nil {
					return nil, "", policies.Refuse("badge_active", "The member has an active badge.")
				}
				b := &Badge{ID: m.ids.Next("bdg"), MemberID: mid}
				m.byID[b.ID] = b
				m.badges = append(m.badges, b)
				return policies.Rec{"badge": badgeView(b)}, b.ID, nil
			}},
		{Name: "revokeBadge", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("badgeId", m.BadgeExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				b := m.byID[c.Str("badgeId")]
				if b.Revoked {
					return nil, "", policies.Refuse("invalid_state", "The badge is already revoked.")
				}
				b.Revoked, b.Inside = true, false
				return policies.Rec{"badge": badgeView(b)}, b.ID, nil
			}},
		{Name: "enterDoor", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{badgeArg()},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				b := m.byID[c.Str("badgeId")]
				if b.Revoked {
					return nil, "", policies.Refuse("invalid_state", "The badge is revoked.")
				}
				if b.Inside {
					return nil, "", policies.Refuse("already_inside", "The badge is already inside.")
				}
				if !m.members.InGoodStanding(b.MemberID) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				if !m.inOpeningPeriod(c.At) {
					return nil, "", policies.Refuse("outside_opening_hours", "The request time is outside the opening hours.")
				}
				if m.billing.AccountOf(b.MemberID, c.At).Overdue > 0 {
					return nil, "", policies.Refuse("account_overdue", "The member has an overdue balance.")
				}
				b.Inside = true
				e := m.addEvent("enter", b.ID, nil, c.At)
				return policies.Rec{"badge": badgeView(b), "event": eventView(e)}, b.ID, nil
			}},
		{Name: "leaveDoor", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{badgeArg()},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				b := m.byID[c.Str("badgeId")]
				if b.Revoked {
					return nil, "", policies.Refuse("invalid_state", "The badge is revoked.")
				}
				if !b.Inside {
					return nil, "", policies.Refuse("not_inside", "The badge is not inside.")
				}
				b.Inside = false
				e := m.addEvent("leave", b.ID, nil, c.At)
				return policies.Rec{"badge": badgeView(b), "event": eventView(e)}, b.ID, nil
			}},
		{Name: "admitGuest", Mutation: true, Perms: "oaa",
			// Ownership is judged on badgeId alone, so guestPassId carries no owner.
			Args: []policies.ArgSpec{policies.IDRef("guestPassId", m.guests.GuestPassExists), badgeArg()},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				pid := c.Str("guestPassId")
				b := m.byID[c.Str("badgeId")]
				member, date, status, _ := m.guests.GuestPassOf(pid)
				if status == "revoked" || b.Revoked {
					return nil, "", policies.Refuse("invalid_state", "The guest pass or the badge is revoked.")
				}
				if date != policies.DayOf(c.At) {
					return nil, "", policies.Refuse("pass_not_today", "The guest pass is for another date.")
				}
				if member != b.MemberID {
					return nil, "", policies.Refuse("pass_of_another_member", "The guest pass belongs to another member.")
				}
				if !b.Inside {
					return nil, "", policies.Refuse("sponsor_not_inside", "The badge is not inside.")
				}
				for _, e := range m.events {
					if e.Kind == "guest" && e.GuestPassID != nil && *e.GuestPassID == pid {
						return nil, "", policies.Refuse("guest_already_admitted", "The guest pass admitted its guest already.")
					}
				}
				e := m.addEvent("guest", b.ID, &pid, c.At)
				return policies.Rec{"event": eventView(e)}, pid, nil
			}},
		{Name: "listDoorEvents", Perms: "naa",
			Args: []policies.ArgSpec{policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				day := c.Num("date")
				out := []policies.Rec{}
				for _, e := range m.events {
					if policies.DayOf(e.At) == day {
						out = append(out, eventView(e))
					}
				}
				return policies.Rec{"date": c.Str("date"), "events": out}, "", nil
			}},
	}
}
