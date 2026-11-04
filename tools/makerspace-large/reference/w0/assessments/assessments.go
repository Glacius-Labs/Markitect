// Package assessments holds the assessment of a course and the attempts of
// members at it, and the operations on them.
package assessments

import (
	"makerspace/members"
	"makerspace/policies"
	"makerspace/training"
)

type definition struct {
	courseID            string
	passMark, maxTrials int
}

// attempt is one attempt of a member; period is the number of the member's
// enrollment in the course in which it was recorded.
type attempt struct {
	courseID, memberID string
	period, number     int
	score              int
	passed             bool
	at                 int
}

type pairKey struct{ courseID, memberID string }

// Module is the assessments module: it owns every assessment and attempt.
type Module struct {
	members  *members.Module
	training *training.Module
	defs     map[string]*definition
	attempts map[pairKey][]*attempt
}

// New returns the module in the initial state.
func New(mem *members.Module, tr *training.Module) *Module {
	return &Module{members: mem, training: tr, defs: map[string]*definition{}, attempts: map[pairKey][]*attempt{}}
}

// AssessmentMet reports whether the member has met the assessment requirement
// of the course: the course has no assessment, or an attempt of the member's
// current enrollment period has passed.
func (m *Module) AssessmentMet(courseID, memberID string) bool {
	if m.defs[courseID] == nil {
		return true
	}
	e, ok := m.training.EnrollmentOf(courseID, memberID)
	if !ok {
		return false
	}
	for _, a := range m.attempts[pairKey{courseID, memberID}] {
		if a.period == e.Number && a.passed {
			return true
		}
	}
	return false
}

func assessmentView(d *definition) policies.Rec {
	return policies.Rec{"courseId": d.courseID, "passMark": d.passMark, "maxAttempts": d.maxTrials}
}

func attemptView(a *attempt) policies.Rec {
	return policies.Rec{"courseId": a.courseID, "memberId": a.memberID, "number": a.number, "score": a.score,
		"passed": a.passed, "at": policies.FormatTime(a.at)}
}
