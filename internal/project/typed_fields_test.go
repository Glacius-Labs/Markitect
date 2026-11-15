package project

import "testing"

// typedFields is a project that uses the fields std 2.3.0 adds (DEC-117): a
// Role as a part of the roles Rule, linked from Who and For, an Outcome that
// says the State it leaves in, a Page with its Path, a Connector with its
// Mode and a Task with its Schedule.
func typedFields(t *testing.T) memRepo {
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	d := SpecDir + "/"
	return memRepo{files: map[string]string{
		ManifestFile:                       "libraries:\n  std: 2.3.0\n",
		LockFile:                           stdLock(t),
		d + "project.md":                   def("std/Project", "Library", "A lending library."),
		d + "purpose/goals/lend.md":        def("std/Goal", "Lend", "**Result:** Books are lent.\n\n**Changes:** Fewer cards.\n\n**Obstacle:** Paper."),
		d + "rules/roles/roles.md":         def("std/Rule", "Roles", "Who may call what is said with these roles."),
		d + "rules/roles/member.md":        def("std/Role", "Member", "A signed-in member.\n\n**Granted by:** signing in."),
		d + "concepts/copy/copy.md":        def("std/Concept", "Copy", "One copy of a book."),
		d + "concepts/copy/states/lent.md": def("std/State", "Lent", "A member holds the [[concepts/copy]]."),
		d + "operations/lend/lend.md": def("std/Operation", "Lend", "Lends a copy.\n\n**Who:** [[rules/roles/member]].\n\n"+
			"**Checked in this order:**\n\n1. [[operations/lend/conditions/free]]: `taken`.\n\n**Satisfies:** [[purpose/goals/lend]]."),
		d + "operations/lend/conditions/free.md": def("std/Condition", "Free", "Nobody holds the copy."),
		d + "operations/lend/outcomes/lent.md":   def("std/Outcome", "Lent", "The copy is lent.\n\n**Leaves in:** [[concepts/copy/states/lent]]."),
		d + "site/site.md":                       "---\nkind: Package\n---\n# Site\n\nThe site.\n",
		d + "site/pages/home.md":                 def("std/Page", "Home", "The first page.\n\n**For:** [[rules/roles/member]].\n\n**Path:** `/`.\n\n**Shows:** the copies."),
		d + "site/pages/loans.md":                def("std/Page", "Loans", "A member's loans.\n\n**For:** [[rules/roles/member]].\n\n**Path:** `/loans`.\n\n**Shows:** the loans."),
		d + "architecture/web.md":                def("std/Component", "Web", "Serves the site.\n\n**Instances:** one.\n\n**Implemented by:** [[site]]."),
		d + "architecture/mail-provider.md":      def("std/ExternalSystem", "Mail provider", "Sends mail.\n\n**Uses:** sending mail.\n\n**When it fails:** mail waits."),
		d + "architecture/mail.md": def("std/Connector", "Mail", "How mail goes out.\n\n**Between:** [[architecture/web]], [[architecture/mail-provider]].\n\n"+
			"**Protocol:** the web hands over each mail over SMTP.\n\n**Limits:** none.\n\n**Mode:** `async`."),
		d + "administration/tasks/back-up.md": def("std/Task", "Back up", "Keeps a copy.\n\n**Actor:** [[purpose/stakeholders/staff]].\n\n"+
			"**Steps:**\n\n1. Copy the data.\n\n**Schedule:** daily at `03:00`.\n\n**Satisfies:** [[purpose/goals/lend]]."),
		d + "purpose/stakeholders/staff.md": def("std/Stakeholder", "Staff", "The people who run the library."),
	}}
}

func TestTypedFieldsCompile(t *testing.T) {
	p := load(t, typedFields(t))
	for _, e := range p.Errors {
		if e.Code != "definition.unused" {
			t.Errorf("error: %v", e)
		}
	}
}

// check holds the new fields to their form at once: no spec had them before.
func TestTypedFieldsAreCheckedAtOnce(t *testing.T) {
	r := typedFields(t)
	d := SpecDir + "/"
	r.files[d+"site/pages/loans.md"] = "---\nkind: std/Page\n---\n# Loans\n\nA member's loans.\n\n**For:** `member`.\n\n**Path:** `/`, also.\n\n**Shows:** the loans.\n"
	r.files[d+"architecture/mail.md"] = "---\nkind: std/Connector\n---\n# Mail\n\nHow mail goes out.\n\n**Between:** [[architecture/web]].\n\n**Protocol:** SMTP.\n\n**Limits:** none.\n\n**Mode:** fire and forget.\n"
	r.files[d+"operations/lend/outcomes/lent.md"] = "---\nkind: std/Outcome\n---\n# Lent\n\nThe copy is lent.\n\n**Leaves in:** [[concepts/copy]].\n"
	p := load(t, r)
	expect(t, p.Errors, "field.unique", `site/pages/loans gives the "Path" /, as site/pages/home does; two std/Pages of site give it apart`)
	expect(t, p.Errors, "field.value", `"Mode" of a std/Connector is one of `+"`sync`, `async`")
	expect(t, p.Errors, "field.target", `"Leaves in" of a std/Outcome links a std/State; concepts/copy/copy is a std/Concept`)
}
