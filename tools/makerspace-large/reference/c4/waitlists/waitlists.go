// Package waitlists holds the waiting lists of full courses, and the
// operations on them.
package waitlists

import (
	"makerspace/members"
	"makerspace/notices"
	"makerspace/policies"
	"makerspace/training"
)

// Entry is the place of one member on the waiting list of one course.
type Entry struct {
	CourseID, MemberID string
	JoinedAt           int
}

// Module is the waitlists module: it owns every waiting list.
type Module struct {
	members  *members.Module
	training *training.Module
	notices  *notices.Module
	lists    map[string][]*Entry // by course, in join order
}

// New returns the module in the initial state.
func New(mem *members.Module, tr *training.Module, ntc *notices.Module) *Module {
	return &Module{members: mem, training: tr, notices: ntc, lists: map[string][]*Entry{}}
}

func (m *Module) indexOf(courseID, memberID string) int {
	for i, e := range m.lists[courseID] {
		if e.MemberID == memberID {
			return i
		}
	}
	return -1
}

func entryView(e *Entry) policies.Rec {
	return policies.Rec{"courseId": e.CourseID, "memberId": e.MemberID, "joinedAt": policies.FormatTime(e.JoinedAt)}
}

func enrollmentView(e training.Enrollment) policies.Rec {
	return policies.Rec{"courseId": e.CourseID, "memberId": e.MemberID, "status": e.Status,
		"enrolledAt": policies.FormatTime(e.EnrolledAt), "completedAt": policies.OptTime(e.CompletedAt)}
}
