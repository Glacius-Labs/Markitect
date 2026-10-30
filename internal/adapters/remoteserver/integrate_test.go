package remoteserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// integrate pushes HEAD of the clone to branch as the host's integration,
// from a bundle thin against the Run's branches, as the host does.
func (r *remote) integrate(from *agent, branch string) runremote.Decision {
	r.t.Helper()
	bundle := filepath.Join(r.t.TempDir(), "integration.bundle")
	from.git("bundle", "create", "-q", bundle, "HEAD", "--not", "origin/"+branch)
	f, err := os.Open(bundle)
	if err != nil {
		r.t.Fatal(err)
	}
	defer f.Close()
	dec, err := Integrate(r.l, r.git, []string{"MARKITECT_REMOTE_TEST_HOOK=1"}, f, branch, from.git("rev-parse", "HEAD"))
	if err != nil {
		r.t.Fatal(err)
	}
	return dec
}

// approveChildren has members and loans submit approved work.
func approveChildren(r *remote) {
	r.t.Helper()
	for _, child := range []struct{ name, file, item string }{{"members", "src/members/member.go", "members/member"}, {"loans", "src/loans/loan.go", "loans/loan"}} {
		a := r.agent(child.name, maker)
		a.write(child.file, "package x // v2\n")
		a.commit(submission(child.name, child.item))
		a.mustPush()
		r.approve(child.name, child.item)
	}
}

// The host integrates a parent without items of its own: its merge of every
// working child's approved work is a submission with the Manager line alone,
// which the hook compiles and checks as a merged state and the parent's
// checker then reviews.
func TestTheHostIntegratesAParentWithoutItems(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) { p.Managers[0].Checklist = nil })
	approveChildren(r)
	host := r.agent(".", maker)
	host.sync()
	host.git("merge", "-q", "--no-edit", "origin/managers/members")
	host.git("merge", "-q", "--no-ff", "-m", "Integrate\n\nMarkitect-Manager: .\n", "origin/managers/loans")
	if dec := r.integrate(host, "root"); !dec.Accepted || dec.Kind != "integration" || len(dec.Checks) != 1 || !dec.Checks[0].Passed {
		t.Fatalf("decision %+v", dec)
	}
	e := r.log()
	if last := e[len(e)-1]; last.User != runremote.HostUser || last.Kind != "integration" || last.Manager != "." {
		t.Fatalf("last log entry %+v", last)
	}
	if got := run(t, "", "--git-dir="+r.l.Repo(), "rev-parse", "root"); got != host.git("rev-parse", "HEAD") {
		t.Fatalf("root is at %s", got)
	}
	r.approve(".")
}

// The host adds nothing of its own and integrates every working child, only
// on a parent without items, and a merged state that fails a required check
// is refused as any parent's is.
func TestTheHooksHoldTheHostsIntegrationToAParentsPush(t *testing.T) {
	r := newRemote(t, func(p *runremote.Plan) { p.Managers[0].Checklist = nil })
	approveChildren(r)
	refused := func(branch, want string, prepare func(a *agent)) {
		t.Helper()
		a := r.agent(".", maker)
		if branch != "root" {
			a = r.agent("members", maker)
		}
		a.sync()
		prepare(a)
		if dec := r.integrate(a, branch); dec.Accepted || !strings.Contains(dec.Reason, want) {
			t.Fatalf("decision %+v, want a refusal with %q", dec, want)
		}
	}
	submit := "Integrate\n\nMarkitect-Manager: .\n"
	refused("root", `holds no approved work of "loans"`, func(a *agent) {
		a.git("merge", "-q", "--no-ff", "-m", submit, "origin/managers/members")
	})
	refused("root", "README.md, which lie outside", func(a *agent) {
		a.git("merge", "-q", "--no-edit", "origin/managers/members")
		a.git("merge", "-q", "--no-edit", "origin/managers/loans")
		a.write("README.md", "# Library by the host\n")
		a.commit(submit)
	})
	refused("root", "is no submission", func(a *agent) {
		a.git("merge", "-q", "--no-edit", "origin/managers/members")
		a.git("merge", "-q", "--no-edit", "origin/managers/loans")
	})
	refused("managers/members", "has checklist items of its own", func(a *agent) {
		a.commit("Integrate\n\nMarkitect-Manager: members\n")
	})

	// A child's approved work that fails the check on the merged state.
	loans := r.agent("loans", maker)
	loans.write("src/loans/FAIL", "x\n")
	loans.write(".markitect/registry/loans.md", "---\npaths: [src/loans/**]\n---\n"+registryTable+"| src/loans/loan.go | [[loans/loan]] | Type |\n| src/loans/FAIL | [[loans/loan]] | Marker |\n")
	loans.commit(submission("loans", "loans/loan"))
	loans.mustPush()
	r.approve("loans", "loans/loan")
	refused("root", `the required check "no-fail-marker" exited with 1`, func(a *agent) {
		a.git("merge", "-q", "--no-edit", "origin/managers/members")
		a.git("merge", "-q", "--no-ff", "-m", submit, "origin/managers/loans")
	})
	if got := run(t, "", "--git-dir="+r.l.Repo(), "rev-parse", "root"); got != r.plan.Base {
		t.Fatalf("a refused integration moved root to %s", got)
	}
}
