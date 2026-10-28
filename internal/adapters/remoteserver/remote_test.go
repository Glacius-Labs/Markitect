package remoteserver

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
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
		for _, file := range []string{".markitect/model/members/member.md", ".markitect/manifest.yaml", ".markitect/lock.yaml", ".markitect/execution.yaml",
			".markitect/ignore.md", ".markitect/model/new.md", ".markitect/results/x.md", ".markitect/registry/nobody.md", ".markitect/libraries/x/k.md"} {
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
		submission("members", "members/member"):                                                                                                      "no verdict for members/rules",
		submission("members", "members/member", "members/rules", "loans/loan"):                                                                       "\"loans/loan\" is not on your checklist",
		submission("members", "members/member", "members/rules", "members/rules"):                                                                    "has two verdicts",
		submission("loans", "loans/loan"):                                                                                                            "but you work for \"members\"",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | changed |\n":                                                      "gives no evidence",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | fine | ok\n":                                                      "must be one of conforms, changed, model-problem",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member changed\n":                                                          "must read",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | deviates | x\nMarkitect-Verdict: members/rules | conforms | ok\n": "in apply a maker makes the realization conform",
		"Submit\n\nMarkitect-Verdict: members/member | changed | ok\n\nMore text.\n":                                                                 "must form the message's last paragraph",
		"Submit\n\nMarkitect-Verdict: members/member | changed | ok\n":                                                                               "need a Markitect-Manager line",
		"Submit\n\nMarkitect-Manager: members\nMarkitect-Verdikt: x | changed | ok\n":                                                                "unknown Markitect trailer",
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
		p.Checks = append(p.Checks, project.Check{Name: "slow", Command: "sleep 30", Timeout: time.Second, Environment: "test-image"})
	})
	m := r.agent("members", maker)
	m.commit(submission("members", "members/member"))
	m.mustRefuse(`the required check "slow" ran out of time`)
}

func TestTheChecksSeeTheBranchsFullStateInTheImagesEnvironment(t *testing.T) {
	// The image's environment, as the remote records it at seeding; a
	// variable inside the image's home moves into the check's fresh home.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("T3_WANT", "package members // v2")
	t.Setenv("T3_CACHE", home+"/cache")
	r := newRemote(t, func(p *runremote.Plan) {
		p.Checks = []project.Check{{Name: "state", Directory: "src", Timeout: time.Minute, Environment: "test-image",
			Command: `test "$(cat members/member.go)" = "$T3_WANT" && test -f ../go.work && test -z "$(git status --porcelain)" &&
test "$T3_CACHE" = "$HOME/cache" && test "$HOME" != "` + filepath.ToSlash(home) + `" && test -w "$TMPDIR" && touch built.txt`}}
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
	// A defect a verify checker finds is a deviation, which becomes a Finding.
	c.commit("Verify\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | deviates | member.go accepts an empty name\n")
	c.mustPush()
	if e := r.log(); e[len(e)-1].Kind != "verdicts" {
		t.Fatalf("last log entry %+v", e[len(e)-1])
	}
	// One commit, on the tip alone.
	c.commit("a")
	c.commit(submission("members", "members/member"))
	c.mustRefuse("exactly one new commit")
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	merge := c.git("commit-tree", "HEAD^{tree}", "-p", "HEAD", "-p", r.plan.Base, "-m", submission("members", "members/member"))
	c.git("reset", "-q", "--hard", merge)
	c.mustRefuse("as its only parent")
}

func TestAReviewAnswersEachItemOnceOnTheTipAlone(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	m.commit(submission("members", "members/member"))
	m.mustPush()
	c := r.agent("members", checker)
	c.commit("Review\n\nMarkitect-Manager: members\nMarkitect-Review: approved\n" +
		"Markitect-Check: members/member | confirmed | read\nMarkitect-Check: members/member | confirmed | read again\n")
	c.mustRefuse(`answers "members/member" twice`)
	c.git("reset", "-q", "--hard", "origin/"+c.branch)
	// A review that also merges another commit is no review of the tip.
	merge := c.git("commit-tree", "HEAD^{tree}", "-p", "HEAD", "-p", r.plan.Base, "-m",
		review("members", runremote.Approved, map[string]string{"members/member": runremote.Confirmed}))
	c.git("reset", "-q", "--hard", merge)
	c.mustRefuse("as its only parent")
}

func TestChecksThatCannotRunRefuseTheSubmission(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	// A gitlink cannot be checked out, so the checks' copy cannot hold the state.
	m.git("update-index", "--add", "--cacheinfo", "160000,"+r.plan.Base+",src/members/vendor")
	m.git("commit", "-q", "-m", submission("members", "members/member"))
	m.mustRefuse("the required checks could not run")
}

func TestAParentIntegratesSeveralChildrenAndMaySubmitWithoutItems(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) { p.Managers[0].Checklist = nil })
	for _, child := range []struct{ name, file, item string }{{"members", "src/members/member.go", "members/member"}, {"loans", "src/loans/loan.go", "loans/loan"}} {
		a := r.agent(child.name, maker)
		a.write(child.file, "package x // v2\n")
		a.commit(submission(child.name, child.item))
		a.mustPush()
		r.approve(child.name, child.item)
	}
	root := r.agent(".", maker)
	root.sync()
	root.git("merge", "-q", "--no-edit", "origin/managers/members")
	root.git("merge", "-q", "--no-edit", "origin/managers/loans")
	// Root has no checklist items: its Manager line alone submits.
	root.commit("Integrate\n\nMarkitect-Manager: .\n")
	root.mustPush()
	if e := r.log(); e[len(e)-1].Kind != "submission" || len(e[len(e)-1].Checks) != 1 {
		t.Fatalf("last log entry %+v", e[len(e)-1])
	}
	// Its checker may reject it without an item to dispute, or approve it.
	c := r.agent(".", checker)
	c.commit("Review\n\nThe merge of loans drops the members change.\n\nMarkitect-Manager: .\nMarkitect-Review: rejected\n")
	c.mustPush()
	root.sync()
	root.git("merge", "-q", "--ff-only", "origin/root")
	root.commit("Integrate\n\nMarkitect-Manager: .\n")
	root.mustPush()
	r.approve(".")
}

func TestPathsThatDifferOnlyInCaseAreRefused(t *testing.T) {
	r := newRemote(t, nil)
	root := r.agent(".", maker)
	// Staged without touching the clone, which may not tell the two apart.
	blob := root.git("hash-object", "-w", "README.md")
	root.git("update-index", "--add", "--cacheinfo", "100644,"+blob+",src/Members/member.go")
	root.git("commit", "-q", "-m", "shadow a child's file")
	root.mustRefuse("src/Members and src/members differ only in case")
}

func TestTheHostsLandingCheckChangesNoBranch(t *testing.T) {
	r := newRemote(t, nil)
	m := r.agent("members", maker)
	refs := func() string { return run(t, "", "--git-dir="+r.l.Repo(), "for-each-ref") }
	before := refs()
	check := func(commit string) ([]runremote.CheckResult, error) {
		bundle := filepath.Join(t.TempDir(), "landing.bundle")
		m.git("bundle", "create", "-q", bundle, "HEAD", "--not", "origin/"+m.branch)
		f, err := os.Open(bundle)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		return LandingCheck(r.l, r.git, f, commit, io.Discard)
	}
	// A merged state only the host holds, thin against the Run's branches.
	m.write("src/members/FAIL", "x\n")
	failing := m.commit("merged state")
	if res, err := check(failing); err != nil || len(res) != 1 || res[0].Passed {
		t.Fatalf("landing check of a failing state: %+v %v", res, err)
	}
	m.git("rm", "-q", "src/members/FAIL")
	passing := m.commit("fixed merged state")
	if res, err := check(passing); err != nil || len(res) != 1 || !res[0].Passed {
		t.Fatalf("landing check of a passing state: %+v %v", res, err)
	}
	if _, err := check(strings.Repeat("e", 40)); err == nil || !strings.Contains(err.Error(), "does not hold the commit") {
		t.Fatalf("landing check of a missing commit: %v", err)
	}
	if after := refs(); after != before {
		t.Fatalf("the landing check changed refs:\n%s\n%s", before, after)
	}
	if e := r.log(); e[len(e)-1].Kind != "landing-check" || e[len(e)-1].User != "host" {
		t.Fatalf("last log entry %+v", e[len(e)-1])
	}
}

func TestAFileTwoPackagesClaimEquallyIsRefused(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) {
		p.Managers[1].Patterns = append(p.Managers[1].Patterns, "src/x?/**")
		p.Managers[2].Patterns = append(p.Managers[2].Patterns, "src/?y/**")
	})
	root := r.agent(".", maker)
	root.write("src/xy/a.go", "package xy\n")
	root.commit("new folder")
	root.mustRefuse(`is claimed equally by "loans" and "members"`)
}
