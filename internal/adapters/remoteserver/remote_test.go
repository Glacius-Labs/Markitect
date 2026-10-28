package remoteserver

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

const (
	maker   = runremote.Maker
	checker = runremote.Checker
)

func TestAMakerPushesOnlyItsOwnBranch(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	m.write("src/members/member.go", "package members // v2\n")
	m.commit("work")
	for _, ref := range []string{"refs/heads/managers/loans", "refs/heads/root", "refs/heads/base", "refs/heads/elsewhere", "refs/tags/v1", "refs/notes/x"} {
		out, ok := m.pushTo("HEAD:" + ref)
		if ok || !strings.Contains(out, "is not your branch") {
			t.Fatalf("push to %s: accepted %v, want a refusal:\n%s", ref, ok, out)
		}
	}
	// Two refs in one push are refused as a whole, even if one is its own.
	out, err := gitCmd(m.dir, "push", "--porcelain", "origin", "HEAD:refs/heads/managers/members", "HEAD:refs/heads/managers/loans")
	if err == nil || !strings.Contains(out, "is not your branch") {
		t.Fatalf("push of two refs: %v\n%s", err, out)
	}
	m.mustPush()
}

func TestHistoryIsNeverRewrittenOrDeleted(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	m.write("src/members/member.go", "package members // v2\n")
	m.commit("first")
	m.mustPush()
	m.git("reset", "-q", "--hard", "HEAD~1")
	m.write("src/members/member.go", "package members // other\n")
	m.commit("rewritten")
	if out, ok := m.push("--force"); ok || !strings.Contains(out, "history on the remote is never rewritten") {
		t.Fatalf("force push accepted %v:\n%s", ok, out)
	}
	if out, ok := m.pushTo(":refs/heads/managers/members"); ok || !strings.Contains(out, "deleting") {
		t.Fatalf("delete accepted %v:\n%s", ok, out)
	}
}

func TestAPushStaysInTheManagersArea(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	cases := map[string]string{
		"src/loans/loan.go":            "belongs to \"loans\" outside your area",
		"README.md":                    "belongs to \".\" outside your area",
		"src/new-root-file.txt":        "belongs to \".\" outside your area",
		"src/members/cards/card.go":    "belongs to \"members/cards\" below you",
		".markitect/registry/loans.md": "is the registry page of \"loans\"",
	}
	for file, want := range cases {
		m.write(file, "changed\n")
		m.commit("touch " + file)
		out := m.mustRefuse(want)
		if !strings.Contains(out, "git checkout") {
			t.Fatalf("refusal names no next step:\n%s", out)
		}
		m.git("reset", "-q", "--hard", "origin/"+m.branch)
	}
	// Its own files and its own registry page are free.
	m.write("src/members/member.go", "package members // v2\n")
	m.write("src/members/new.go", "package members\n")
	m.write(".markitect/registry/members.md", "# members\n\n- src/members/new.go\n")
	m.commit("own area")
	m.mustPush()
}

func TestTheModelManifestLockAndExecutionNeverChange(t *testing.T) {
	r := newRemote(t, nil)
	for _, who := range []string{".", "members"} {
		m := r.agent(who, maker)
		for _, file := range []string{".markitect/model/members/member.md", ".markitect/markitect.yaml", ".markitect/markitect.lock", ".markitect/execution.yaml", ".markitect/model/new.md", ".markitect/results/x.md"} {
			m.write(file, "changed\n")
			m.commit("touch " + file)
			m.mustRefuse("is protected")
			m.git("reset", "-q", "--hard", "origin/"+m.branch)
		}
	}
	// Root may not change a child's registry page either.
	root := r.agent(".", maker)
	root.write(".markitect/registry/members.md", "changed\n")
	root.commit("touch")
	root.mustRefuse("is the registry page of \"members\" below you")
}

func TestASubmissionNeedsAVerdictForEveryChecklistItem(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) {
		for i := range p.Managers {
			if p.Managers[i].ID == "members" {
				p.Managers[i].Checklist = []string{"members/member", "members/rules"}
			}
		}
	})
	m := r.agent("members", maker)
	m.write("src/members/member.go", "package members // v2\n")
	cases := map[string]string{
		submission("members", "members/member"):                                                 "no verdict for members/rules",
		submission("members", "members/member", "members/rules", "loans/loan"):                  "\"loans/loan\" is not on your checklist",
		submission("members", "members/member", "members/rules", "members/rules"):               "has two verdicts",
		submission("loans", "loans/loan"):                                                       "but you work for \"members\"",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | changed |\n": "gives no evidence",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | fine | ok\n": "must be one of conforms, changed, model-problem",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member changed\n":     "must read",
		"Submit\n\nMarkitect-Verdict: members/member | changed | ok\n\nMore text.\n":            "must form the message's last paragraph",
		"Submit\n\nMarkitect-Verdict: members/member | changed | ok\n":                          "need a Markitect-Manager line",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdikt: x | changed | ok\n":           "unknown Markitect trailer",
	}
	for message, want := range cases {
		m.git("add", "-A")
		m.git("commit", "-q", "--allow-empty", "-m", message)
		m.mustRefuse(want)
		m.git("reset", "-q", "--soft", "HEAD~1")
	}
	m.git("commit", "-q", "-m", submission("members", "members/member", "members/rules"))
	m.mustPush()
	if e := r.log(); e[len(e)-1].Kind != "submission" || len(e[len(e)-1].Checks) != 1 || !e[len(e)-1].Checks[0].Passed {
		t.Fatalf("last log entry %+v, want a submission whose check passed", e[len(e)-1])
	}
}

func TestASubmissionPassesTheRequiredChecksOnTheBranchState(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	m.write("src/members/FAIL", "x\n")
	m.commit("work in progress")
	m.mustPush() // work without verdicts runs no check
	m.commit(submission("members", "members/member"))
	out := m.mustRefuse(`the required check "no-fail-marker" exited with 1`)
	if !strings.Contains(out, "check no-fail-marker failed") {
		t.Fatalf("the pusher does not see the check result:\n%s", out)
	}
	m.git("rm", "-q", "src/members/FAIL")
	m.commit(submission("members", "members/member"))
	m.mustPush()
}

func TestACheckThatRunsOutOfTimeFails(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) {
		p.Checks = append(p.Checks, runremote.Check{Name: "slow", Command: "sleep 30", TimeoutSeconds: 1})
	})
	m := r.agent("members", maker)
	m.commit(submission("members", "members/member"))
	m.mustRefuse(`the required check "slow" ran out of time`)
}

func TestTheChecksSeeTheBranchsFullStateAsACleanCheckout(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) {
		p.Checks = []runremote.Check{{Name: "state", Directory: "src", TimeoutSeconds: 60,
			Environment: map[string]string{"WANT": "package members // v2"},
			Command:     `test "$(cat members/member.go)" = "$WANT" && test -f ../go.work && test -z "$(git status --porcelain)" && touch built.txt`}}
	})
	m := r.agent("members", maker)
	m.write("src/members/member.go", "package members // v2")
	m.commit(submission("members", "members/member"))
	m.mustPush()
}

func TestAReviewIsOneCommitThatChangesNoFile(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	c := r.agent("members", checker)
	all := map[string]string{"members/member": runremote.Confirmed}

	// Nothing to review before a submission.
	c.commit(review("members", runremote.Approved, all))
	c.mustRefuse("is no submission of \"members\"")

	m.write("src/members/member.go", "package members // v2\n")
	m.commit(submission("members", "members/member"))
	m.mustPush()

	c.git("fetch", "-q", "origin")
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	cases := []struct {
		setup func()
		want  string
	}{
		{func() {
			c.write("src/members/member.go", "fixed by the checker\n")
			c.commit(review("members", runremote.Approved, all))
		}, "a checker never edits"},
		{func() { c.commit(review("members", runremote.Approved, map[string]string{})) }, "approves without confirming members/member"},
		{func() {
			c.commit(review("members", runremote.Approved, map[string]string{"members/member": runremote.Disputed}))
		}, "approves but disputes"},
		{func() { c.commit(review("members", runremote.Rejected, all)) }, "rejects without disputing"},
		{func() {
			c.commit(review("members", runremote.Approved, map[string]string{"members/member": runremote.Confirmed, "loans/loan": runremote.Confirmed}))
		}, "which the submission gives no verdict"},
		{func() { c.commit("note"); c.commit(review("members", runremote.Approved, all)) }, "exactly one new commit"},
		{func() { c.commit(submission("members", "members/member")) }, "a checker confirms or disputes verdicts"},
		{func() { c.commit("plain") }, "needs Markitect-Review and Markitect-Check lines"},
	}
	for _, tc := range cases {
		tc.setup()
		c.mustRefuse(tc.want)
		c.git("reset", "-q", "--hard", "origin/"+c.branch)
	}
	// A maker cannot approve itself.
	m.commit(review("members", runremote.Approved, all))
	m.mustRefuse("only your checker writes a review")
	m.git("reset", "-q", "--hard", "HEAD~1")

	c.commit(review("members", runremote.Rejected, map[string]string{"members/member": runremote.Disputed}))
	c.mustPush()
	// After a rejection the maker works on and submits again.
	m.sync()
	m.git("merge", "-q", "--ff-only", "origin/"+m.branch)
	m.write("src/members/member.go", "package members // v3\n")
	m.commit(submission("members", "members/member"))
	m.mustPush()
	r.approve("members", "members/member")
}

func TestAParentMergesOnlyApprovedChildWork(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	m.write("src/members/member.go", "package members // v2\n")
	m.commit(submission("members", "members/member"))
	m.mustPush()

	root := r.agent(".", maker)
	root.sync()
	// Not approved yet: the child's area must stay as at the base.
	root.git("merge", "-q", "--no-edit", "origin/managers/members")
	root.mustRefuse("must stay as at the base until you merge \"members\"'s approved work")
	root.git("reset", "-q", "--hard", "origin/root")

	approval := r.approve("members", "members/member")
	root.sync()
	root.git("merge", "-q", "--no-edit", "origin/managers/members")
	root.write("README.md", "# Library v2\n")
	root.commit("own change")
	root.mustPush()

	// An evil merge: the parent slips an edit into the child's area.
	m.sync()
	m.git("merge", "-q", "--ff-only", "origin/"+m.branch)
	m.write("src/members/member.go", "package members // v3\n")
	m.commit(submission("members", "members/member"))
	m.mustPush()
	r.approve("members", "members/member")
	root.sync()
	root.git("merge", "-q", "--no-commit", "origin/managers/members")
	root.write("src/members/member.go", "package members // v3, touched by root\n")
	root.git("add", "-A")
	root.git("commit", "-q", "-m", "merge")
	root.mustRefuse("must be exactly as in \"members\"'s approved commit")
	root.git("reset", "-q", "--hard", "origin/root")

	// Altering the child's approved commit: an amended copy is no approved work.
	if got := root.git("show", approval+":src/members/member.go"); got != "package members // v2" {
		t.Fatalf("approved content %q", got)
	}
	root.write("src/members/member.go", "package members // v2 amended\n")
	root.commit("amended copy")
	root.mustRefuse("belongs to \"members\" below you")
	root.git("reset", "-q", "--hard", "origin/root")

	// Merging the child's later work that is not approved.
	m.sync()
	m.git("merge", "-q", "--ff-only", "origin/"+m.branch)
	m.write("src/members/member.go", "package members // v4 unreviewed\n")
	m.commit("more work")
	m.mustPush()
	root.sync()
	root.git("merge", "-q", "--no-edit", "origin/managers/members")
	root.mustRefuse("must be exactly as in \"members\"'s approved commit")
}

func TestApprovedWorkClimbsTheTreeUnchanged(t *testing.T) {
	r := newRemote(t, nil)
	cards := r.agent("members/cards", maker)
	cards.write("src/members/cards/card.go", "package cards // v2\n")
	cards.commit(submission("members/cards", "members/cards/card"))
	cards.mustPush()
	r.approve("members/cards", "members/cards/card")

	m := r.agent("members", maker)
	m.sync()
	m.git("merge", "-q", "--no-edit", "origin/managers/members.cards")
	m.commit(submission("members", "members/member"))
	m.mustPush()
	r.approve("members", "members/member")

	root := r.agent(".", maker)
	root.sync()
	root.git("merge", "-q", "--no-edit", "origin/managers/members")
	root.write("src/members/cards/card.go", "package cards // v2, root's edit\n")
	root.commit("edit a grandchild's file")
	root.mustRefuse("belongs to \"members/cards\" below you; it must be exactly as in \"members\"'s approved commit")
	root.git("reset", "-q", "--hard", "HEAD~1")
	root.commit(submission(".", "project"))
	root.mustPush()
	r.approve(".", "project")
	root.sync()
	if got := root.git("show", "origin/root:src/members/cards/card.go"); got != "package cards // v2" {
		t.Fatalf("Root's approved state holds %q", got)
	}
}

func TestAChildTakesItsParentsStateUnchanged(t *testing.T) {
	r := newRemote(t, nil)
	root := r.agent(".", maker)
	root.write("README.md", "# Library v2\n")
	root.write("go.work", "go 1.27\nuse ./src\n")
	root.commit("shared files")
	root.mustPush()

	m := r.agent("members", maker)
	m.sync()
	m.git("merge", "-q", "--no-edit", "origin/root")
	m.write("src/members/member.go", "package members // v2\n")
	m.commit("work on the parent's state")
	m.mustPush()

	// Undoing one of the parent's changes would reach the parent at its merge.
	m.git("checkout", "-q", r.plan.Base, "--", "go.work")
	m.commit("undo")
	m.mustRefuse("must be exactly as in your parent's state")
	m.git("reset", "-q", "--hard", "HEAD~1")

	// A sibling's work reaches a child only through the common parent.
	l := r.agent("loans", maker)
	l.write("src/loans/loan.go", "package loans // v2\n")
	l.commit(submission("loans", "loans/loan"))
	l.mustPush()
	r.approve("loans", "loans/loan")
	m.sync()
	m.git("merge", "-q", "--no-edit", "origin/managers/loans")
	m.mustRefuse("belongs to \"loans\" outside your area")
}

func TestOnlyTheRunsAgentsReachTheRemote(t *testing.T) {
	r := newRemote(t, nil)
	for _, user := range []*url.Userinfo{url.UserPassword("members-maker", "wrong"), url.UserPassword("stranger", "token-stranger"), nil} {
		u, _ := url.Parse(r.url)
		u.User = user
		if out, err := gitCmd(t.TempDir(), "ls-remote", u.String()); err == nil {
			t.Fatalf("ls-remote as %v succeeded:\n%s", user, out)
		}
	}
}

func TestVerifyTakesVerdictsWithoutFileChanges(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) {
		p.Mode = runremote.Verify
		var agents []runremote.Agent
		for _, a := range p.Agents {
			if a.Role == runremote.Checker {
				agents = append(agents, a)
			}
		}
		p.Agents = agents
	})
	c := r.agent("members", checker)
	c.write("src/members/member.go", "fixed\n")
	c.commit(submission("members", "members/member"))
	c.mustRefuse("verify changes nothing")
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	c.commit(submission("members", "members/rules"))
	c.mustRefuse("is not on your checklist")
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	c.commit(review("members", runremote.Approved, map[string]string{"members/member": runremote.Confirmed}))
	c.mustRefuse("in verify a checker gives its own verdicts")
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	c.commit(submission("members", "members/member"))
	c.mustPush()
	if e := r.log(); e[len(e)-1].Kind != "verdicts" {
		t.Fatalf("last log entry %+v", e[len(e)-1])
	}
}
