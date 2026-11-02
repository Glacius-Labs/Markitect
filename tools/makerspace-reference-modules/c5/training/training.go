// Package training holds the courses, the enrollments and the certifications
// of members, and the operations on them.
package training

import (
	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/members"
)

type course struct {
	id, familyID, title string
	fee, capacity       int
}

type enrollment struct {
	courseID, memberID string
	status             string
	enrolledAt         int
	completedAt        *int
}

type certification struct {
	memberID, familyID string
	issuedOn           int // day
	revoked            bool
}

// Module is the training module: it owns every course, enrollment and
// certification.
type Module struct {
	members     *members.Module
	equipment   *equipment.Module
	billing     *billing.Module
	courses     map[string]*course
	enrollments map[string]*enrollment // key courseID + "\x00" + memberID
	certs       map[string]*certification
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module, bill *billing.Module) *Module {
	return &Module{
		members:     mem,
		equipment:   eq,
		billing:     bill,
		courses:     map[string]*course{},
		enrollments: map[string]*enrollment{},
		certs:       map[string]*certification{},
	}
}

func enrollmentKey(courseID, memberID string) string { return courseID + "\x00" + memberID }

func certKey(memberID, familyID string) string { return memberID + "\x00" + familyID }

// CourseExists reports whether the identifier names a course.
func (t *Module) CourseExists(id string) bool { _, ok := t.courses[id]; return ok }

// Certified reports whether the member holds a certification for the family
// that is valid on the given day (section 7.2).
func (t *Module) Certified(memberID, familyID string, day int) bool {
	cert := t.certs[certKey(memberID, familyID)]
	return cert != nil && !cert.revoked && cert.issuedOn <= day && day < cert.issuedOn+365
}
