// Package instructors holds the qualifications of users to teach classes for a
// tool family, and the operations on them.
package instructors

import (
	"sort"

	"makerspace/equipment"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/training"
)

type key struct{ userID, familyID string }

// Needs is what instructors asks of the program without depending on the
// module that knows the answer: the protocol module supplies it from classes.
type Needs struct {
	// HasClasses reports whether the user is the instructor of a scheduled
	// class whose course belongs to the family and whose start is after at.
	HasClasses func(userID, familyID string, at int) bool
}

// Module is the instructors module: it owns every qualification.
type Module struct {
	needs     Needs
	members   *members.Module
	equipment *equipment.Module
	training  *training.Module
	quals     map[key]bool
}

// New returns the module in the initial state.
func New(needs Needs, mem *members.Module, eq *equipment.Module, tr *training.Module) *Module {
	return &Module{needs: needs, members: mem, equipment: eq, training: tr, quals: map[key]bool{}}
}

// Qualified reports whether the user holds a qualification for the family.
func (m *Module) Qualified(userID, familyID string) bool { return m.quals[key{userID, familyID}] }

// HoldsAny reports whether the user holds a qualification for any family.
func (m *Module) HoldsAny(userID string) bool {
	for k := range m.quals {
		if k.userID == userID {
			return true
		}
	}
	return false
}

func qualificationView(userID, familyID string) policies.Rec {
	return policies.Rec{"userId": userID, "familyId": familyID}
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	userExists := func(id string) bool { return m.members.User(id) != nil }
	return []*policies.Operation{
		{Name: "qualifyInstructor", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("userId", userExists), policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid, fid := c.Str("userId"), c.Str("familyId")
				switch m.members.User(uid).Role {
				case "admin":
					return nil, "", policies.Refuse("not_eligible", "An admin cannot be qualified.")
				case "member":
					if !m.training.Certified(uid, fid, policies.DayOf(c.At)) {
						return nil, "", policies.Refuse("not_eligible", "The member holds no valid certification for the family.")
					}
				}
				if m.Qualified(uid, fid) {
					return nil, "", policies.Refuse("already_exists", "The user is qualified for the family already.")
				}
				m.quals[key{uid, fid}] = true
				return policies.Rec{"qualification": qualificationView(uid, fid)}, uid + "/" + fid, nil
			}},
		{Name: "withdrawInstructor", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("userId", userExists), policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				uid, fid := c.Str("userId"), c.Str("familyId")
				if !m.Qualified(uid, fid) {
					return nil, "", policies.Refuse("not_instructor", "The user holds no qualification for the family.")
				}
				if m.needs.HasClasses(uid, fid, c.At) {
					return nil, "", policies.Refuse("instructor_has_classes", "The user has a scheduled class of the family.")
				}
				delete(m.quals, key{uid, fid})
				return policies.Rec{"qualification": qualificationView(uid, fid)}, uid + "/" + fid, nil
			}},
		{Name: "listInstructors", Perms: "aaa",
			Args: []policies.ArgSpec{policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid := c.Str("familyId")
				var users []string
				for k := range m.quals {
					if k.familyID == fid {
						users = append(users, k.userID)
					}
				}
				sort.Strings(users)
				out := make([]policies.Rec, 0, len(users))
				for _, u := range users {
					out = append(out, qualificationView(u, fid))
				}
				return policies.Rec{"familyId": fid, "instructors": out}, "", nil
			}},
	}
}
