// Package classes holds the classes of courses and the attendance marks of
// members, and the operations on them.
package classes

import (
	"makerspace/instructors"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/rooms"
	"makerspace/training"
)

// Class is one session of a course.
type Class struct {
	ID, CourseID, InstructorID, RoomID string
	Start, End                         int
	Cancelled                          bool
	holdID                             string
}

// Status is scheduled or cancelled.
func (c *Class) Status() string {
	if c.Cancelled {
		return "cancelled"
	}
	return "scheduled"
}

type markKey struct{ classID, memberID string }

// Module is the classes module: it owns every class and attendance mark.
type Module struct {
	members     *members.Module
	training    *training.Module
	instructors *instructors.Module
	rooms       *rooms.Module
	ids         policies.IDs
	byID        map[string]*Class
	list        []*Class
	marks       map[markKey]bool
}

// New returns the module in the initial state.
func New(mem *members.Module, tr *training.Module, inst *instructors.Module, rm *rooms.Module) *Module {
	return &Module{members: mem, training: tr, instructors: inst, rooms: rm, byID: map[string]*Class{},
		marks: map[markKey]bool{}}
}

// ClassExists reports whether the identifier names a class.
func (m *Module) ClassExists(id string) bool { _, ok := m.byID[id]; return ok }

// AttendanceOwed reports whether the member still owes attendance for the
// course: a class of the course that is scheduled, has started at the request
// time at and began at or after enrolledAt has no present mark of the member.
func (m *Module) AttendanceOwed(courseID, memberID string, enrolledAt, at int) bool {
	for _, c := range m.list {
		if c.CourseID == courseID && !c.Cancelled && c.Start <= at && c.Start >= enrolledAt &&
			!m.marks[markKey{c.ID, memberID}] {
			return true
		}
	}
	return false
}

// Taught is a scheduled class of an instructor with its length in minutes.
type Taught struct {
	ClassID string
	Minutes int
}

// ScheduledInMonth lists the scheduled classes of the user as instructor whose
// start lies in the month, in creation order, each with its length.
func (m *Module) ScheduledInMonth(userID string, month int) []Taught {
	var out []Taught
	for _, c := range m.list {
		if c.InstructorID == userID && !c.Cancelled && policies.MonthOf(c.Start) == month {
			out = append(out, Taught{ClassID: c.ID, Minutes: c.End - c.Start})
		}
	}
	return out
}

// InstructorHasClasses reports whether the user is the instructor of a
// scheduled class whose course belongs to the family and whose start is after
// the time at.
func (m *Module) InstructorHasClasses(userID, familyID string, at int) bool {
	for _, c := range m.list {
		if c.InstructorID == userID && !c.Cancelled && c.Start > at && m.training.FamilyOf(c.CourseID) == familyID {
			return true
		}
	}
	return false
}

func classView(c *Class) policies.Rec {
	return policies.Rec{"classId": c.ID, "courseId": c.CourseID, "instructorId": c.InstructorID,
		"roomId": c.RoomID, "start": policies.FormatTime(c.Start), "end": policies.FormatTime(c.End),
		"status": c.Status()}
}

func attendanceView(classID, memberID string, present bool) policies.Rec {
	return policies.Rec{"classId": classID, "memberId": memberID, "present": present}
}
