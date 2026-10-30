package runremote

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// decideHost decides the host's integration of a parent without items of its
// own (long-lived Managers): instead of a maker turn that only merges, the
// host merges the working children's approved commits on the parent's
// branch, and the parent's checker reviews the merged state. The push is
// held to what a parent's push is held to, and the host adds nothing of its
// own: every file it changes lies in a child's area and is as that child's
// approved commit has it, the tip holds every working child's approval, it
// compiles, and the required checks pass on it, as on any merged state.
func decideHost(plan *Plan, own Ownership, repo Repo, checks CheckRunner, updates []Update) Decision {
	if len(updates) != 1 {
		return refuse("the host integrates one branch per push, not %d.", len(updates))
	}
	branch := strings.TrimPrefix(updates[0].Ref, "refs/heads/")
	var m Manager
	found := false
	for _, c := range plan.Managers {
		if c.Working && Branch(c.ID) == branch && "refs/heads/"+branch == updates[0].Ref {
			m, found = c, true
		}
	}
	switch {
	case plan.Mode != Apply:
		return refuse("the host integrates only in apply.")
	case !found:
		return refuse("%s is no working Manager's branch; the host integrates only those.", updates[0].Ref)
	case len(m.Checklist) > 0:
		return refuse("%q has checklist items of its own, so its maker integrates its children's work.", Name(m.ID))
	}
	var children []string
	for _, c := range plan.Managers {
		if c.Working && c.ID != model.RootPackage && model.ParentOf(c.ID) == m.ID {
			children = append(children, c.ID)
		}
	}
	if len(children) == 0 {
		return refuse("%q has no working children to integrate.", Name(m.ID))
	}
	d := &decider{plan: plan, own: own, repo: repo, checks: checks, agent: Agent{User: HostUser, Manager: m.ID}, m: m, branch: branch}
	dec, err := d.integration(updates[0], children)
	if err != nil {
		dec = refuse("the remote could not decide the host's integration: %v", err)
	}
	dec.Manager = Name(m.ID)
	return dec
}

func (d *decider) integration(u Update, children []string) (Decision, error) {
	own := "refs/heads/" + d.branch
	switch {
	case u.New == ZeroID || u.Old == ZeroID:
		return refuse("the host neither creates nor deletes %s.", own), nil
	}
	if t, err := d.repo.Type(u.New); err != nil {
		return Decision{}, err
	} else if t != "commit" {
		return refuse("%s must point to a commit, not a %s.", own, t), nil
	}
	if ok, err := d.repo.IsAncestor(u.Old, u.New); err != nil {
		return Decision{}, err
	} else if !ok {
		return refuse("%s moved from %s to %s, which does not contain it; history on the remote is never rewritten.", own, short(u.Old), short(u.New)), nil
	}
	fresh, err := d.repo.New(u.New)
	if err != nil {
		return Decision{}, err
	}
	for _, id := range fresh {
		c, err := d.repo.Commit(id)
		if err != nil {
			return Decision{}, err
		}
		t, perr := ParseMessage(c.Message)
		switch {
		case perr != nil:
			return refuse("commit %s: %v.", short(id), perr), nil
		case t.Kind == Review || len(t.Verdicts) > 0:
			return refuse("commit %s carries verdict or review lines; the host's integration carries neither.", short(id)), nil
		case t.Kind == Submission && t.Manager != Name(d.m.ID):
			return refuse("commit %s names Manager %q, not %q.", short(id), t.Manager, Name(d.m.ID)), nil
		}
	}
	tip, err := d.repo.Commit(u.New)
	if err != nil {
		return Decision{}, err
	}
	if t, _ := ParseMessage(tip.Message); t.Kind != Submission || t.Manager != Name(d.m.ID) || len(t.Verdicts) > 0 {
		return refuse("the tip %s is no submission of %q; the host's integration ends with one, for the parent's checker.", short(u.New), Name(d.m.ID)), nil
	}
	childRef, err := d.childApprovals(u.New)
	if err != nil {
		return Decision{}, err
	}
	var missing []string
	for _, c := range children {
		if childRef[c] == d.plan.Base {
			missing = append(missing, fmt.Sprintf("%q", Name(c)))
		}
	}
	if len(missing) > 0 {
		return refuse("the tip %s holds no approved work of %s; the host integrates every working child's.", short(u.New), strings.Join(missing, ", ")), nil
	}
	// Nothing of the host's own: every file it changes lies below the parent,
	// and content() holds each to the child's approval.
	changed, err := d.repo.Changed(u.Old, u.New)
	if err != nil {
		return Decision{}, err
	}
	var outside []string
	for _, file := range changed {
		owner, page := d.plan.Pages[file]
		if !page {
			owner, _ = d.own.Owner(file)
		}
		if owner == d.m.ID || !model.Within(owner, d.m.ID) || strings.HasPrefix(file, project.Dir+"/") && !page {
			outside = append(outside, file)
		}
	}
	if len(outside) > 0 {
		sort.Strings(outside)
		return refuse("the host's integration changes %s, which lie outside %q's children's areas; it only merges their approved work.", strings.Join(outside, ", "), Name(d.m.ID)), nil
	}
	if problems, err := d.content(u.New, childRef); err != nil {
		return Decision{}, err
	} else if len(problems) > 0 {
		return refuse("%s", strings.Join(problems, "\n")), nil
	}
	if problem, err := d.compile(u.New); err != nil {
		return Decision{}, err
	} else if problem != "" {
		return refuse("the merged state %s does not compile: %s", short(u.New), problem), nil
	}
	results, err := d.checks.Run(tip, d.plan.Checks)
	if err != nil {
		return refuse("the required checks could not run on %s: %v.", short(u.New), err), nil
	}
	for _, r := range results {
		if r.Passed {
			continue
		}
		why := fmt.Sprintf("exited with %d", r.Exit)
		if r.TimedOut {
			why = "ran out of time"
		}
		return Decision{Reason: fmt.Sprintf("the required check %q %s on the merged state %s.\n%s", r.Name, why, short(u.New), r.Output), Checks: results}, nil
	}
	return Decision{Accepted: true, Kind: "integration", Checks: results}, nil
}
