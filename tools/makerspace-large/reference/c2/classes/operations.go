package classes

import (
	"sort"

	"makerspace/policies"
	"makerspace/rooms"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	userExists := func(id string) bool { return m.members.User(id) != nil }
	return []*policies.Operation{
		{Name: "scheduleClass", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists), policies.IDRef("instructorId", userExists),
				policies.IDRef("roomId", m.rooms.RoomExists), policies.Stamp("start"), policies.Stamp("end")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				course, uid, room := c.Str("courseId"), c.Str("instructorId"), c.Str("roomId")
				start, end := c.Num("start"), c.Num("end")
				if end <= start {
					return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.At {
					return nil, "", policies.Refuse("starts_in_past", "The class starts before the request time.")
				}
				if !m.instructors.Qualified(uid, m.training.FamilyOf(course)) {
					return nil, "", policies.Refuse("not_instructor", "The instructor holds no qualification for the family of the course.")
				}
				if !m.rooms.Free(room, start, end) {
					return nil, "", policies.Refuse("room_taken", "The room is not free for the interval.")
				}
				for _, o := range m.list {
					if o.InstructorID == uid && !o.Cancelled && policies.Overlaps(start, end, o.Start, o.End) {
						return nil, "", policies.Refuse("instructor_busy", "Another class of the instructor overlaps.")
					}
				}
				cl := &Class{ID: m.ids.Next("cls"), CourseID: course, InstructorID: uid, RoomID: room, Start: start, End: end}
				cl.holdID = m.rooms.HoldFor(room, start, end, "class "+cl.ID, rooms.OriginClass).ID
				m.byID[cl.ID] = cl
				m.list = append(m.list, cl)
				return policies.Rec{"class": classView(cl)}, cl.ID, nil
			}},
		{Name: "cancelClass", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("classId", m.ClassExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cl := m.byID[c.Str("classId")]
				if cl.Cancelled || cl.Start <= c.At {
					return nil, "", policies.Refuse("invalid_state", "The class is cancelled or has started.")
				}
				cl.Cancelled = true
				m.rooms.ReleaseHold(cl.holdID)
				return policies.Rec{"class": classView(cl)}, cl.ID, nil
			}},
		{Name: "markAttendance", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("classId", m.ClassExists), policies.MemberRef(m.members.Exists),
				policies.Flag("present")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cl := m.byID[c.Str("classId")]
				mid := c.Str("memberId")
				if cl.Cancelled {
					return nil, "", policies.Refuse("invalid_state", "The class is cancelled.")
				}
				if c.At < cl.Start {
					return nil, "", policies.Refuse("class_not_started", "The class has not started.")
				}
				if _, ok := m.training.EnrollmentOf(cl.CourseID, mid); !ok {
					return nil, "", policies.Refuse("not_enrolled", "The member has no enrollment in the course.")
				}
				m.marks[markKey{cl.ID, mid}] = c.Flag("present")
				return policies.Rec{"attendance": attendanceView(cl.ID, mid, c.Flag("present"))}, cl.ID + "/" + mid, nil
			}},
		{Name: "listCourseClasses", Perms: "aaa",
			Args: []policies.ArgSpec{policies.IDRef("courseId", m.training.CourseExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				course := c.Str("courseId")
				var list []*Class
				for _, cl := range m.list {
					if cl.CourseID == course {
						list = append(list, cl)
					}
				}
				sort.SliceStable(list, func(i, j int) bool { return list[i].Start < list[j].Start })
				out := make([]policies.Rec, 0, len(list))
				for _, cl := range list {
					out = append(out, classView(cl))
				}
				return policies.Rec{"courseId": course, "classes": out}, "", nil
			}},
		{Name: "listClassAttendance", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("classId", m.ClassExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("classId")
				var ids []string
				for k := range m.marks {
					if k.classID == id {
						ids = append(ids, k.memberID)
					}
				}
				sort.Strings(ids)
				out := make([]policies.Rec, 0, len(ids))
				for _, mid := range ids {
					out = append(out, attendanceView(id, mid, m.marks[markKey{id, mid}]))
				}
				return policies.Rec{"classId": id, "attendance": out}, "", nil
			}},
	}
}
