package project

import "testing"

// businessCase is a project that states its business case in the Kinds of
// std (DEC-115): a Stakeholder with what it does today, gains and gives, a
// Problem a Goal removes, a Conflict between two Stakeholders, and an
// Alternative to building the system that nothing links.
func businessCase(t *testing.T) memRepo {
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	staff := SpecDir + "/purpose/stakeholders/staff/"
	return memRepo{files: map[string]string{
		ManifestFile:            "libraries:\n  std: 2.3.0\n",
		LockFile:                stdLock(t),
		SpecDir + "/project.md": def("std/Project", "Library", "A lending library."),
		staff + "staff.md":      def("std/Stakeholder", "Staff", "The people who lend the books."),
		staff + "today/lend-on-a-card.md": def("std/Practice", "Lend on a card", "They write each loan on a card.\n\n"+
			"**Steps:**\n\n1. Find the card.\n2. Write the book on it.\n\n**Problems:** [[purpose/problems/searching-the-cards]]"),
		staff + "concerns/quick.md": def("std/Concern", "Quick", "A loan is quick."),
		staff + "wishes/lend.md":    def("std/Wish", "Lend", "Lend a book."),
		staff + "benefits/find-a-book.md": def("std/Benefit", "Find a book", "They see who holds a book.\n\n"+
			"**Instead of:** [[purpose/stakeholders/staff/today/lend-on-a-card]]\n\n**Serves:** [[purpose/stakeholders/staff/concerns/quick]]"),
		staff + "drawbacks/a-screen.md": def("std/Drawback", "A screen", "They learn to lend at a screen.\n\n"+
			"**Justified by:** [[purpose/stakeholders/staff/benefits/find-a-book]]\n\n**Eased by:** [[quality/fast]]"),
		SpecDir + "/purpose/stakeholders/members/members.md":            def("std/Stakeholder", "Members", "The people who borrow the books."),
		SpecDir + "/purpose/stakeholders/members/concerns/keep-long.md": def("std/Concern", "Keep long", "A book is kept long."),
		SpecDir + "/purpose/problems/searching-the-cards.md":            def("std/Problem", "Searching the cards", "A missing book is searched for on every card.\n\n**Affects:** [[purpose/stakeholders/staff]]\n\n**Cost:** an hour a week."),
		SpecDir + "/purpose/goals/know-who-holds-a-book.md":             def("std/Goal", "Know who holds a book", "**Result:** The holder is known.\n\n**Changes:** No search.\n\n**Obstacle:** Cards by member.\n\n**Removes:** [[purpose/problems/searching-the-cards]]"),
		SpecDir + "/purpose/conflicts/loan-period.md":                   def("std/Conflict", "Loan period", "Members give way: a loan ends after four weeks.\n\n**Between:** [[purpose/stakeholders/members/concerns/keep-long]], [[purpose/stakeholders/staff/concerns/quick]]\n\n**Settled by:** [[purpose/goals/know-who-holds-a-book]]"),
		SpecDir + "/purpose/alternatives/sort-the-cards-by-book.md":     def("std/Alternative", "Sort the cards by book", "Not chosen: a member's own loans could no longer be found."),
		SpecDir + "/quality/fast.md":                                    def("std/Quality", "Fast", "A loan takes a second.\n\n**Measure:** within `1` second."),
	}}
}

// The business case compiles, nothing of it is unused, and an Alternative
// under purpose/ needs no link: their Kinds declare that no file realizes them.
func TestBusinessCaseCompilesAndNeedsNoLinkOrRow(t *testing.T) {
	if p := load(t, businessCase(t)); len(p.Errors) != 0 {
		t.Fatalf("errors:\n%v", p.Errors)
	}
}

// purpose/ lies in Root, so what eases a Drawback or settles a Conflict may not
// be a Definition inside a child Package.
func TestBusinessCaseLinksNothingInAChildPackage(t *testing.T) {
	r := businessCase(t)
	r.files[SpecDir+"/desk/desk.md"] = "---\nkind: Package\n---\n# Desk\n\nThe desk.\n"
	r.files[SpecDir+"/desk/quality/quick-loan.md"] = "---\nkind: std/Quality\n---\n# Quick loan\n\nA loan is quick.\n\n**Measure:** within `1` second.\n"
	r.files[SpecDir+"/desk/decisions/four-weeks.md"] = "---\nkind: std/Decision\n---\n# Four weeks\n\nA loan ends after four weeks.\n"
	r.files[SpecDir+"/purpose/stakeholders/staff/drawbacks/a-screen.md"] = "---\nkind: std/Drawback\n---\n# A screen\n\nThey learn a screen.\n\n" +
		"**Justified by:** [[purpose/goals/know-who-holds-a-book]]\n\n**Eased by:** [[desk/quality/quick-loan]]\n"
	r.files[SpecDir+"/purpose/conflicts/loan-period.md"] = "---\nkind: std/Conflict\n---\n# Loan period\n\nMembers give way.\n\n" +
		"**Between:** [[purpose/stakeholders/members/concerns/keep-long]], [[purpose/stakeholders/staff/concerns/quick]]\n\n**Settled by:** [[desk/decisions/four-weeks]]\n"
	p := load(t, r)
	expect(t, p.Errors, "link.downward", "desk/quality/quick-loan")
	expect(t, p.Errors, "link.downward", "desk/decisions/four-weeks")
}
