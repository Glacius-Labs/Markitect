package runremote

import (
	"fmt"
	"sort"
	"strings"
)

// ZeroID is Git's id of a missing ref in a push update.
const ZeroID = "0000000000000000000000000000000000000000"

// Commit is what the hooks read of one commit.
type Commit struct {
	ID      string
	Parents []string
	Tree    string
	Message string
}

// Repo answers the hooks' questions about the Run's repository as it stands
// before the push is applied, with the pushed objects visible.
type Repo interface {
	// Type returns the object type of id: commit, tag, tree or blob.
	Type(id string) (string, error)
	Commit(id string) (Commit, error)
	// New lists the commits reachable from tip that no ref reaches yet.
	New(tip string) ([]string, error)
	IsAncestor(a, b string) (bool, error)
	// Changed lists the paths whose content or mode differ between two commits.
	Changed(a, b string) ([]string, error)
	// Tips lists the commits a branch has pointed to in this Run, oldest first.
	Tips(branch string) ([]string, error)
}

// CheckResult is the outcome of one required check.
type CheckResult struct {
	Name     string  `json:"name"`
	Passed   bool    `json:"passed"`
	Exit     int     `json:"exit"`
	TimedOut bool    `json:"timedOut,omitempty"`
	Seconds  float64 `json:"seconds"`
	// Output is the tail of the check's combined output.
	Output string `json:"output,omitempty"`
}

// CheckRunner runs the required checks on a commit's full repository state.
// An error means the checks could not run; it is never a pass.
type CheckRunner interface {
	Run(commit Commit, checks []Check) ([]CheckResult, error)
}

// Update is one ref update of a push.
type Update struct{ Old, New, Ref string }

// Decision is the hooks' answer to one push.
type Decision struct {
	Accepted bool `json:"accepted"`
	// Reason says why a push was refused and what to do next.
	Reason  string        `json:"reason,omitempty"`
	Manager string        `json:"manager,omitempty"`
	Role    Role          `json:"role,omitempty"`
	Kind    string        `json:"kind,omitempty"` // work, submission, review or verdicts
	Checks  []CheckResult `json:"checks,omitempty"`
}

// Decide is the server-side hook: it accepts or refuses one push by user. The
// guarantees of concept 2.0 §6.3 hold for every branch tip it accepts.
func Decide(plan *Plan, own Ownership, repo Repo, checks CheckRunner, user string, updates []Update) Decision {
	agent, ok := plan.Agent(user)
	if !ok {
		return refuse("%q is no agent of this Run. Next: push with the credentials the Run gave you.", user)
	}
	m, _ := plan.Manager(agent.Manager)
	d := &decider{plan: plan, own: own, repo: repo, checks: checks, agent: agent, m: m, branch: Branch(m.ID)}
	dec, err := d.decide(updates)
	if err != nil {
		dec = refuse("the remote could not decide: %v. Next: push again; if this repeats, report it to your parent.", err)
	}
	dec.Manager, dec.Role = m.ID, agent.Role
	return dec
}

type decider struct {
	plan   *Plan
	own    Ownership
	repo   Repo
	checks CheckRunner
	agent  Agent
	m      Manager
	branch string
}

func refuse(format string, args ...any) Decision {
	return Decision{Reason: fmt.Sprintf(format, args...)}
}

func (d *decider) decide(updates []Update) (Decision, error) {
	own := "refs/heads/" + d.branch
	for _, u := range updates {
		if u.Ref != own {
			return refuse("%s is not your branch; you may push only to %s. Next: push with `git push origin HEAD:%s`.", u.Ref, own, own), nil
		}
	}
	if len(updates) != 1 {
		return refuse("push your branch %s once, not %d updates. Next: push it alone.", own, len(updates)), nil
	}
	u := updates[0]
	switch {
	case u.New == ZeroID:
		return refuse("deleting %s is not allowed. Next: add commits on top of it instead.", own), nil
	case u.Old == ZeroID:
		return refuse("%s cannot be created; the Run created it. Next: fetch it and push on top of it.", own), nil
	}
	if t, err := d.repo.Type(u.New); err != nil {
		return Decision{}, err
	} else if t != "commit" {
		return refuse("%s must point to a commit, not a %s. Next: push a commit.", own, t), nil
	}
	if ok, err := d.repo.IsAncestor(u.Old, u.New); err != nil {
		return Decision{}, err
	} else if !ok {
		return refuse("%s moved from %s to %s, which does not contain it: history on the remote is never rewritten (no force push, rebase or amend of pushed commits). Next: `git fetch origin %s`, put your work on top of it and push again.",
			own, short(u.Old), short(u.New), d.branch), nil
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
		if perr != nil {
			return refuse("commit %s: %v. Next: rewrite that commit's message (it is not on the remote yet) and push again.", short(id), perr), nil
		}
		switch {
		case t.Kind != Plain && t.Manager != d.m.ID:
			return refuse("commit %s names Manager %q in %s, but you work for %q. Next: name your own Manager.", short(id), t.Manager, KeyManager, d.m.ID), nil
		case t.Kind == Review && d.plan.Mode == Verify:
			return refuse("commit %s carries %s lines; in verify a checker gives its own verdicts with %s lines. Next: write verdicts instead.", short(id), KeyReview, KeyVerdict), nil
		case t.Kind == Review && d.agent.Role != Checker:
			return refuse("commit %s carries %s lines; only your checker writes a review, as one commit of its own. Next: remove them from your commit message.", short(id), KeyReview), nil
		case t.Kind == Submission && d.plan.Mode == Apply && d.agent.Role == Checker:
			return refuse("commit %s carries %s lines; a checker confirms or disputes verdicts with %s lines. Next: write a review instead.", short(id), KeyVerdict, KeyCheck), nil
		}
	}
	switch {
	case d.plan.Mode == Verify:
		return d.verdicts(u, fresh)
	case d.agent.Role == Checker:
		return d.review(u, fresh)
	}
	return d.work(u)
}

// work decides a maker's push: its branch tip stays within the Manager's
// area, and a submission at the tip has a complete checklist and passes the
// required checks.
func (d *decider) work(u Update) (Decision, error) {
	if problems, err := d.content(u.New); err != nil {
		return Decision{}, err
	} else if len(problems) > 0 {
		return refuse("%s", strings.Join(problems, "\n")), nil
	}
	tip, err := d.repo.Commit(u.New)
	if err != nil {
		return Decision{}, err
	}
	t, perr := ParseMessage(tip.Message)
	if perr != nil || t.Kind != Submission || t.Manager != d.m.ID {
		return Decision{Accepted: true, Kind: "work"}, nil
	}
	if problem := d.checklist(t.Verdicts); problem != "" {
		return refuse("submission %s: %s", short(u.New), problem), nil
	}
	results, err := d.checks.Run(tip, d.plan.Checks)
	if err != nil {
		return refuse("the required checks could not run on %s: %v. Nothing was decided about your work. Next: push again; if this repeats, report it to your parent.", short(u.New), err), nil
	}
	dec := Decision{Accepted: true, Kind: "submission", Checks: results}
	for _, r := range results {
		if !r.Passed {
			why := fmt.Sprintf("exited with %d", r.Exit)
			if r.TimedOut {
				why = "ran out of time"
			}
			dec.Accepted = false
			dec.Reason = fmt.Sprintf("submission %s: the required check %q %s on the branch's full state. Next: fix it, run the check yourself and submit again.\n%s",
				short(u.New), r.Name, why, r.Output)
			break
		}
	}
	return dec, nil
}

// review decides a checker's push: one commit on the submission at the
// branch tip that changes no file and confirms or disputes its verdicts.
func (d *decider) review(u Update, fresh []string) (Decision, error) {
	next := "Next: fetch the branch, write `git commit --allow-empty` on its tip with the review trailers and push only that commit."
	if len(fresh) != 1 || fresh[0] != u.New {
		return refuse("a checker pushes exactly one new commit, its review; this push brings %d. %s", len(fresh), next), nil
	}
	c, err := d.repo.Commit(u.New)
	if err != nil {
		return Decision{}, err
	}
	if len(c.Parents) != 1 || c.Parents[0] != u.Old {
		return refuse("review %s must have the branch tip %s as its only parent. %s", short(u.New), short(u.Old), next), nil
	}
	sub, err := d.repo.Commit(u.Old)
	if err != nil {
		return Decision{}, err
	}
	if c.Tree != sub.Tree {
		return refuse("review %s changes files; a checker never edits. Send defects to your maker in the review instead. %s", short(u.New), next), nil
	}
	st, perr := ParseMessage(sub.Message)
	if u.Old == d.plan.Base || perr != nil || st.Kind != Submission || st.Manager != d.m.ID {
		return refuse("the tip %s of your branch is no submission of %q, so there is nothing to review. Next: wait for your maker's submission.", short(u.Old), d.m.ID), nil
	}
	t, _ := ParseMessage(c.Message)
	if t.Kind != Review {
		return refuse("review %s needs %s and %s lines. %s", short(u.New), KeyReview, KeyCheck, next), nil
	}
	submitted := map[string]bool{}
	for _, v := range st.Verdicts {
		submitted[v.Item] = true
	}
	answered, disputed := map[string]bool{}, 0
	for _, l := range t.Checks {
		switch {
		case !submitted[l.Item]:
			return refuse("review %s answers %q, which the submission gives no verdict. Next: answer only the submitted items.", short(u.New), l.Item), nil
		case answered[l.Item]:
			return refuse("review %s answers %q twice. Next: answer each item once.", short(u.New), l.Item), nil
		}
		answered[l.Item] = true
		if l.Word == Disputed {
			disputed++
		}
	}
	switch {
	case t.Review == Approved && disputed > 0:
		return refuse("review %s approves but disputes %d items. Next: reject, or confirm every item.", short(u.New), disputed), nil
	case t.Review == Approved && len(answered) != len(submitted):
		var missing []string
		for item := range submitted {
			if !answered[item] {
				missing = append(missing, item)
			}
		}
		sort.Strings(missing)
		return refuse("review %s approves without confirming %s. Next: re-check and confirm every verdict.", short(u.New), strings.Join(missing, ", ")), nil
	case t.Review == Rejected && disputed == 0:
		return refuse("review %s rejects without disputing an item. Next: dispute the items you found defective, with the defect as evidence.", short(u.New)), nil
	}
	return Decision{Accepted: true, Kind: "review"}, nil
}

// verdicts decides a checker's push in verify: one commit without file
// changes that gives every checklist item a verdict.
func (d *decider) verdicts(u Update, fresh []string) (Decision, error) {
	next := "Next: write `git commit --allow-empty` on your branch tip with your verdicts and push only that commit."
	if len(fresh) != 1 || fresh[0] != u.New {
		return refuse("in verify you push exactly one new commit, your verdicts; this push brings %d. %s", len(fresh), next), nil
	}
	c, err := d.repo.Commit(u.New)
	if err != nil {
		return Decision{}, err
	}
	base, err := d.repo.Commit(d.plan.Base)
	if err != nil {
		return Decision{}, err
	}
	if len(c.Parents) != 1 || c.Parents[0] != u.Old {
		return refuse("commit %s must have the branch tip %s as its only parent. %s", short(u.New), short(u.Old), next), nil
	}
	if c.Tree != base.Tree {
		return refuse("commit %s changes files; verify changes nothing. %s", short(u.New), next), nil
	}
	t, _ := ParseMessage(c.Message)
	if t.Kind != Submission {
		return refuse("commit %s carries no %s lines. %s", short(u.New), KeyVerdict, next), nil
	}
	if problem := d.checklist(t.Verdicts); problem != "" {
		return refuse("commit %s: %s", short(u.New), problem), nil
	}
	return Decision{Accepted: true, Kind: "verdicts"}, nil
}

// checklist reports the first gap between the verdicts and the Manager's
// checklist: every item exactly once, nothing else.
func (d *decider) checklist(verdicts []Line) string {
	want := map[string]bool{}
	for _, item := range d.m.Checklist {
		want[item] = true
	}
	seen := map[string]bool{}
	for _, v := range verdicts {
		switch {
		case !want[v.Item]:
			return fmt.Sprintf("%q is not on your checklist. Next: give verdicts only for your checklist items: %s.", v.Item, strings.Join(d.m.Checklist, ", "))
		case seen[v.Item]:
			return fmt.Sprintf("%q has two verdicts. Next: give each item one verdict.", v.Item)
		}
		seen[v.Item] = true
	}
	var missing []string
	for _, item := range d.m.Checklist {
		if !seen[item] {
			missing = append(missing, item)
		}
	}
	if len(missing) > 0 {
		return fmt.Sprintf("no verdict for %s. Next: give every checklist item a %s line.", strings.Join(missing, ", "), KeyVerdict)
	}
	return ""
}

// content checks the files of a tip against the Manager's area. A file the
// Manager owns is free. A file below it must be as the child toward its owner
// last had it approved, as far as the tip has taken it in, else as at the
// base. A file outside its subtree must be as on its parent's branch, in the
// newest state the tip has taken in, else as at the base. Protected files
// stay as at the base. A registry page belongs to its own Package.
func (d *decider) content(tip string) ([]string, error) {
	pages := map[string]string{}
	for _, m := range d.plan.Managers {
		pages[m.Registry] = m.ID
	}
	base := d.plan.Base
	parentRef := base
	if d.m.Parent != "" {
		ref, err := d.newest(Branch(d.m.Parent), tip, func(string) (bool, error) { return true, nil })
		if err != nil {
			return nil, err
		}
		parentRef = ref
	}
	childRef := map[string]string{}
	for _, c := range d.plan.Managers {
		if c.Parent != d.m.ID || c.ID == d.m.ID {
			continue
		}
		ref := base
		if c.Working {
			var err error
			ref, err = d.newest(Branch(c.ID), tip, func(id string) (bool, error) { return d.approved(id, c.ID) })
			if err != nil {
				return nil, err
			}
		}
		childRef[c.ID] = ref
	}
	// required returns the commit whose content the file must keep, or ""
	// when the Manager may change it.
	required := func(file string) (string, string) {
		owner, page := pages[file]
		if !page {
			for _, pattern := range d.plan.Protected {
				if Match(pattern, file) {
					return base, "is protected: the model, the manifest, the lock and execution.yaml change only through the owner, never in a Run"
				}
			}
			owner = d.own.Owner(file)
		}
		what := "belongs to"
		if page {
			what = "is the registry page of"
		}
		switch {
		case owner == d.m.ID:
			return "", ""
		case d.plan.Within(owner, d.m.ID):
			child := d.plan.ChildToward(d.m.ID, owner)
			ref := childRef[child]
			if ref == base {
				return ref, fmt.Sprintf("%s %q below you; it must stay as at the base until you merge %q's approved work", what, owner, child)
			}
			return ref, fmt.Sprintf("%s %q below you; it must be exactly as in %q's approved commit %s that you merged", what, owner, child, short(ref))
		case d.m.Parent == "":
			return base, fmt.Sprintf("%s %q", what, owner)
		case parentRef == base:
			return base, fmt.Sprintf("%s %q outside your area; it must stay as at the base. Ask your parent %q for a change there", what, owner, d.m.Parent)
		default:
			return parentRef, fmt.Sprintf("%s %q outside your area; it must be exactly as in your parent's state %s that you merged. Ask your parent %q for a change there", what, owner, short(parentRef), d.m.Parent)
		}
	}
	refs := map[string]bool{base: true, parentRef: true}
	for _, r := range childRef {
		refs[r] = true
	}
	ordered := make([]string, 0, len(refs))
	for r := range refs {
		ordered = append(ordered, r)
	}
	sort.Strings(ordered)
	var problems []string
	seen := map[string]bool{}
	for _, ref := range ordered {
		changed, err := d.repo.Changed(ref, tip)
		if err != nil {
			return nil, err
		}
		for _, file := range changed {
			want, why := required(file)
			if want != ref || seen[file] {
				continue
			}
			seen[file] = true
			problems = append(problems, fmt.Sprintf("%s %s. Next: restore it with `git checkout %s -- %s`, commit and push again.", file, why, short(want), file))
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		problems = append([]string{fmt.Sprintf("the tip %s of %s changes files outside %q's area:", short(tip), d.branch, d.m.ID)}, problems...)
	}
	return problems, nil
}

// newest returns the newest commit the branch has pointed to in this Run that
// passes keep and that tip contains, or the base.
func (d *decider) newest(branch, tip string, keep func(string) (bool, error)) (string, error) {
	tips, err := d.repo.Tips(branch)
	if err != nil {
		return "", err
	}
	for i := len(tips) - 1; i >= 0; i-- {
		id := tips[i]
		if id == d.plan.Base {
			continue
		}
		if ok, err := keep(id); err != nil || !ok {
			if err != nil {
				return "", err
			}
			continue
		}
		if ok, err := d.repo.IsAncestor(id, tip); err != nil {
			return "", err
		} else if ok {
			return id, nil
		}
	}
	return d.plan.Base, nil
}

func (d *decider) approved(id, manager string) (bool, error) {
	c, err := d.repo.Commit(id)
	if err != nil {
		return false, err
	}
	t, perr := ParseMessage(c.Message)
	return perr == nil && t.Kind == Review && t.Review == Approved && t.Manager == manager, nil
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
