// Package training holds the courses, the enrollments and the certifications
// of members, and the operations on them.
package training

import (
	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/guardians"
	"makerspace/members"
	"makerspace/policies"
	"makerspace/promotions"
	"makerspace/scholarships"
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
	number             int // how many times the member has enrolled in the course
}

type certification struct {
	memberID, familyID string
	issuedOn           int // day
	revoked            bool
}

// Needs is what training asks of the program without depending on the modules
// that know the answer: the protocol module supplies it from classes and
// assessments.
type Needs struct {
	// AttendanceOwed reports whether the member still owes attendance for the
	// course, given the enrolledAt of the member's enrollment and the request
	// time at (attendance gate).
	AttendanceOwed func(courseID, memberID string, enrolledAt, at int) bool
	// AssessmentMet reports whether the member has met the assessment
	// requirement of the course (assessment gate).
	AssessmentMet func(courseID, memberID string) bool
}

// Enrollment is a copy of the enrollment of one member in one course. Number
// counts the enrollments of the member in the course so far, the first being 1.
type Enrollment struct {
	CourseID, MemberID, Status string
	EnrolledAt                 int
	CompletedAt                *int
	Number                     int
}

// Module is the training module: it owns every course, enrollment and
// certification.
type Module struct {
	needs        Needs
	members      *members.Module
	equipment    *equipment.Module
	billing      *billing.Module
	guardians    *guardians.Module
	promotions   *promotions.Module
	scholarships *scholarships.Module
	courses      map[string]*course
	enrollments  map[string]*enrollment // key courseID + "\x00" + memberID
	certs        map[string]*certification
}

// New returns the module in the initial state.
func New(needs Needs, mem *members.Module, eq *equipment.Module, bill *billing.Module, gd *guardians.Module,
	prm *promotions.Module, sch *scholarships.Module) *Module {
	return &Module{
		needs:        needs,
		members:      mem,
		equipment:    eq,
		billing:      bill,
		guardians:    gd,
		promotions:   prm,
		scholarships: sch,
		courses:      map[string]*course{},
		enrollments:  map[string]*enrollment{},
		certs:        map[string]*certification{},
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

// FamilyOf is the family of the course.
func (t *Module) FamilyOf(courseID string) string { return t.courses[courseID].familyID }

// EnrollmentOf is the enrollment of the member in the course, when one exists.
func (t *Module) EnrollmentOf(courseID, memberID string) (Enrollment, bool) {
	e := t.enrollments[enrollmentKey(courseID, memberID)]
	if e == nil {
		return Enrollment{}, false
	}
	return Enrollment{CourseID: e.courseID, MemberID: e.memberID, Status: e.status,
		EnrolledAt: e.enrolledAt, CompletedAt: e.completedAt, Number: e.number}, true
}

// FreePlaces is the capacity of the course less its enrolled count.
func (t *Module) FreePlaces(courseID string) int {
	co := t.courses[courseID]
	return co.capacity - t.courseView(co)["enrolledCount"].(int)
}

// Enroll enrolls the member in the course at the request time exactly as
// enrollInCourse does: it makes the refusals of that operation in their order
// and, when none applies, every effect of it, the charge and the discounts
// included. A refusal changes nothing. It checks no role and writes no audit
// entry.
func (t *Module) Enroll(courseID, memberID string, at int) (Enrollment, *policies.Refusal) {
	if _, ref := t.enroll(courseID, memberID, at); ref != nil {
		return Enrollment{}, ref
	}
	e, _ := t.EnrollmentOf(courseID, memberID)
	return e, nil
}

// enroll is the enrollment of a member in a course: the refusals in their
// order, then the enrollment, the course charge and the discount's effects.
func (t *Module) enroll(courseID, memberID string, at int) (*enrollment, *policies.Refusal) {
	co := t.courses[courseID]
	m := t.members.Member(memberID)
	if m.Standing != "good" {
		return nil, policies.Refuse("member_suspended", "The member is suspended.")
	}
	if t.guardians.ConsentMissing(m.User.ID, co.familyID, policies.DayOf(at)) {
		return nil, policies.Refuse("guardian_consent_missing", "The member is a minor without consent for the family.")
	}
	key := enrollmentKey(co.id, m.User.ID)
	e := t.enrollments[key]
	if e != nil && e.status == "enrolled" {
		return nil, policies.Refuse("already_enrolled", "The member is already enrolled.")
	}
	if t.FreePlaces(co.id) <= 0 {
		return nil, policies.Refuse("course_full", "The course has no free place.")
	}
	percent, apply := t.courseDiscount(m.User.ID, co, at)
	if e == nil {
		e = &enrollment{courseID: co.id, memberID: m.User.ID}
		t.enrollments[key] = e
	}
	e.status, e.enrolledAt, e.completedAt = "enrolled", at, nil
	e.number++
	t.billing.AddCharge(m.User.ID, "course", co.id, 1, co.fee, percent, at)
	apply()
	return e, nil
}

// courseDiscount decides the discountPercent of a course charge from the
// member's covered scholarship award and oldest unused redeemed promotion
// that covers courses: the higher percent wins, the award on equal percent.
// apply uses only the winner.
func (t *Module) courseDiscount(memberID string, co *course, at int) (percent int, apply func()) {
	if co.fee == 0 {
		return 0, func() {}
	}
	var award *scholarships.Award
	awardDiscount := 0
	if a, remaining, ok := t.scholarships.ActiveAward(memberID); ok {
		if d := co.fee - policies.ChargeAmount(1, co.fee, a.Percent); d <= remaining {
			award, awardDiscount, percent = a, d, a.Percent
		}
	}
	red := t.promotions.OldestUnused(memberID, "course")
	if red != nil && (award == nil || red.Percent > percent) {
		return red.Percent, func() { t.promotions.Use(red, co.id, at) }
	}
	if award != nil {
		return percent, func() { t.scholarships.Spend(award.FundID, awardDiscount) }
	}
	return 0, func() {}
}
