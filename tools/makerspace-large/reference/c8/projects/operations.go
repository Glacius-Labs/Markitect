package projects

import (
	"sort"

	"makerspace/policies"
)

type share struct{ usage, material int }

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	owned := func(name string) policies.ArgSpec {
		return policies.OwnedIDRef(name, m.ProjectExists, m.ProjectOwner)
	}
	return []*policies.Operation{
		{Name: "createProject", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.ID("projectId"),
				policies.OwnedIDRef("ownerId", m.members.Exists, func(key string) string { return key }),
				policies.Text("title")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("projectId")
				if m.byID[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A project with identifier %s already exists.", id)
				}
				p := &project{id: id, ownerID: c.Str("ownerId"), title: c.Str("title")}
				m.byID[id] = p
				return policies.Rec{"project": projectView(p)}, id, nil
			}},
		{Name: "addCollaborator", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{owned("projectId"), policies.IDRef("memberId", m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p, mid := m.byID[c.Str("projectId")], c.Str("memberId")
				if p.closed {
					return nil, "", policies.Refuse("project_closed", "The project is closed.")
				}
				if p.isMember(mid) {
					return nil, "", policies.Refuse("already_collaborator", "The member is the owner or already a collaborator.")
				}
				p.collaborators = append(p.collaborators, mid)
				sort.Strings(p.collaborators)
				return policies.Rec{"project": projectView(p)}, p.id + "/" + mid, nil
			}},
		{Name: "attachReservation", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{owned("projectId"), policies.IDRef("reservationId", m.bookings.ReservationExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p, rid := m.byID[c.Str("projectId")], c.Str("reservationId")
				if p.closed {
					return nil, "", policies.Refuse("project_closed", "The project is closed.")
				}
				r, _ := m.bookings.ReservationOf(rid, c.At)
				if !p.isMember(r.MemberID) {
					return nil, "", policies.Refuse("not_project_member", "The member of the reservation is not in the project.")
				}
				if _, taken := m.reservationOf[rid]; taken {
					return nil, "", policies.Refuse("already_attached", "The reservation is already attached to a project.")
				}
				m.reservationOf[rid] = p.id
				p.reservationIDs = append(p.reservationIDs, rid)
				return policies.Rec{"project": projectView(p)}, p.id + "/" + rid, nil
			}},
		{Name: "attachPurchase", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{owned("projectId"), policies.IDRef("purchaseId", m.materials.PurchaseExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p, pid := m.byID[c.Str("projectId")], c.Str("purchaseId")
				if p.closed {
					return nil, "", policies.Refuse("project_closed", "The project is closed.")
				}
				member, _, _ := m.materials.PurchaseOf(pid)
				if !p.isMember(member) {
					return nil, "", policies.Refuse("not_project_member", "The member of the purchase is not in the project.")
				}
				if _, taken := m.purchaseOf[pid]; taken {
					return nil, "", policies.Refuse("already_attached", "The purchase is already attached to a project.")
				}
				m.purchaseOf[pid] = p.id
				p.purchaseIDs = append(p.purchaseIDs, pid)
				return policies.Rec{"project": projectView(p)}, p.id + "/" + pid, nil
			}},
		{Name: "closeProject", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{owned("projectId")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p := m.byID[c.Str("projectId")]
				if p.closed {
					return nil, "", policies.Refuse("invalid_state", "The project is already closed.")
				}
				p.closed = true
				return policies.Rec{"project": projectView(p)}, p.id, nil
			}},
		{Name: "getProjectCost", Perms: "oaa",
			Args: []policies.ArgSpec{owned("projectId")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				return m.cost(m.byID[c.Str("projectId")], c.At), "", nil
			}},
	}
}

// cost reads the cost of the project from the records attached to it.
func (m *Module) cost(p *project, at int) policies.Rec {
	shares := map[string]*share{}
	of := func(member string) *share {
		if shares[member] == nil {
			shares[member] = &share{}
		}
		return shares[member]
	}
	usage, material := 0, 0
	for _, rid := range p.reservationIDs {
		r, _ := m.bookings.ReservationOf(rid, at)
		if r.Status == "completed" && r.AmountCents != nil {
			usage += *r.AmountCents
			of(r.MemberID).usage += *r.AmountCents
		}
	}
	for _, pid := range p.purchaseIDs {
		member, amount, _ := m.materials.PurchaseOf(pid)
		material += amount
		of(member).material += amount
	}
	by := []policies.Rec{}
	for _, k := range policies.SortedKeys(shares) {
		by = append(by, policies.Rec{"memberId": k, "usageCents": shares[k].usage, "materialCents": shares[k].material})
	}
	return policies.Rec{"projectId": p.id, "usageCents": usage, "materialCents": material,
		"totalCents": usage + material, "byMember": by}
}
