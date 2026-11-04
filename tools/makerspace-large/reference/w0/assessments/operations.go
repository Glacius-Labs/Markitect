package assessments

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "defineAssessment", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.Integer("passMark", 0, 100),
				policies.Integer("maxAttempts", 1, 10)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid := c.Str("courseId")
				if m.defs[cid] != nil {
					return nil, "", policies.Refuse("already_exists", "The course has an assessment already.")
				}
				d := &definition{courseID: cid, passMark: c.Num("passMark"), maxTrials: c.Num("maxAttempts")}
				m.defs[cid] = d
				return policies.Rec{"assessment": assessmentView(d)}, cid, nil
			}},
		{Name: "recordAttempt", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.MemberRef(m.members.Exists),
				policies.Integer("score", 0, 100)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid, mid := c.Str("courseId"), c.Str("memberId")
				d := m.defs[cid]
				if d == nil {
					return nil, "", policies.Refuse("no_assessment", "The course has no assessment.")
				}
				e, ok := m.training.EnrollmentOf(cid, mid)
				if !ok || e.Status != "enrolled" {
					return nil, "", policies.Refuse("not_enrolled", "The member is not enrolled in the course.")
				}
				key := pairKey{cid, mid}
				inPeriod := 0
				for _, a := range m.attempts[key] {
					if a.period == e.Number {
						inPeriod++
					}
				}
				if inPeriod >= d.maxTrials {
					return nil, "", policies.Refuse("attempts_exhausted", "The member has used all attempts of the enrollment period.")
				}
				a := &attempt{courseID: cid, memberID: mid, period: e.Number, number: inPeriod + 1,
					score: c.Num("score"), passed: c.Num("score") >= d.passMark, at: c.At}
				m.attempts[key] = append(m.attempts[key], a)
				return policies.Rec{"attempt": attemptView(a)}, cid + "/" + mid, nil
			}},
		{Name: "listAttempts", Perms: "oaa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cid, mid := c.Str("courseId"), c.Str("memberId")
				out := []policies.Rec{}
				for _, a := range m.attempts[pairKey{cid, mid}] {
					out = append(out, attemptView(a))
				}
				return policies.Rec{"courseId": cid, "memberId": mid, "attempts": out}, "", nil
			}},
	}
}
