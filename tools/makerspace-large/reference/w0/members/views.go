package members

import "makerspace/policies"

func userView(u *User) policies.Rec {
	return policies.Rec{"userId": u.ID, "name": u.Name, "role": u.Role}
}

func memberView(m *Member) policies.Rec {
	return policies.Rec{"memberId": m.User.ID, "name": m.User.Name, "tier": m.Tier,
		"standing": m.Standing, "registeredAt": policies.FormatTime(m.RegisteredAt)}
}
