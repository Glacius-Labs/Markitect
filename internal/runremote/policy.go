package runremote

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
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
	// Files lists every path of a commit's tree.
	Files(commit string) ([]string, error)
	// Tips lists the commits a branch has pointed to in this Run, oldest
	// first; a branch of the Run without that record is an error.
	Tips(branch string) ([]string, error)
	// State is a commit's full state as a project repository, for the
	// compiler.
	State(commit string) project.Repository
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
	Run(commit Commit, checks []project.Check) ([]CheckResult, error)
}

// Update is one ref update of a push.
type Update struct{ Old, New, Ref string }

// Decision is the hooks' answer to one push.
type Decision struct {
	Accepted bool `json:"accepted"`
	// Reason says why a push was refused and what to do next.
	Reason string `json:"reason,omitempty"`
	// Note says what an accepted push should know, such as a required
	// check that failed without refusing it.
	Note    string        `json:"note,omitempty"`
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
	dec.Manager, dec.Role = Name(m.ID), agent.Role
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
	if len(updates) == 0 {
		return refuse("the push updates nothing. Next: push your branch %s.", own), nil
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
		case t.Kind != Plain && t.Manager != Name(d.m.ID):
			return refuse("commit %s names Manager %q in %s, but you work for %q. Next: name your own Manager.", short(id), t.Manager, KeyManager, Name(d.m.ID)), nil
		case t.Kind == Review && d.plan.Mode == Verify:
			return refuse("commit %s carries %s lines; in verify a checker gives its own verdicts with %s lines. Next: write verdicts instead.", short(id), KeyReview, KeyVerdict), nil
		case t.Kind == Review && d.agent.Role != Checker:
			return refuse("commit %s carries %s lines; only your checker writes a review, as one commit of its own. Next: remove them from your commit message.", short(id), KeyReview), nil
		case t.Kind == Submission && d.plan.Mode == Apply && d.agent.Role == Checker:
			return refuse("commit %s is a submission (%s without %s); a checker answers with a review. Next: add %s and %s lines, and no %s lines.", short(id), KeyManager, KeyReview, KeyReview, KeyCheck, KeyVerdict), nil
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
// area, and a submission at the tip has a complete checklist and compiles.
// The required checks run on every submission; they refuse only a merged
// state, one that holds a child's approved work (DEC-087). On any other
// submission their results go to the checker as evidence.
func (d *decider) work(u Update) (Decision, error) {
	childRef, err := d.childApprovals(u.New)
	if err != nil {
		return Decision{}, err
	}
	if problems, err := d.content(u.New, childRef); err != nil {
		return Decision{}, err
	} else if len(problems) > 0 {
		return refuse("%s", strings.Join(problems, "\n")), nil
	}
	tip, err := d.repo.Commit(u.New)
	if err != nil {
		return Decision{}, err
	}
	t, perr := ParseMessage(tip.Message)
	if perr != nil || t.Kind != Submission || t.Manager != Name(d.m.ID) {
		return Decision{Accepted: true, Kind: "work"}, nil
	}
	if problem := d.checklist(t.Verdicts); problem != "" {
		return refuse("submission %s: %s", short(u.New), problem), nil
	}
	if problem, err := d.compile(u.New); err != nil {
		return Decision{}, err
	} else if problem != "" {
		return refuse("submission %s does not compile: %s", short(u.New), problem), nil
	}
	results, err := d.checks.Run(tip, d.plan.Checks)
	if err != nil {
		return refuse("the required checks could not run on %s: %v. Nothing was decided about your work. Next: push again; if this repeats, report it to your parent.", short(u.New), err), nil
	}
	merged := false
	for _, ref := range childRef {
		merged = merged || ref != d.plan.Base
	}
	dec := Decision{Accepted: true, Kind: "submission", Checks: results}
	for _, r := range results {
		if r.Passed {
			continue
		}
		why := fmt.Sprintf("exited with %d", r.Exit)
		if r.TimedOut {
			why = "ran out of time"
		}
		if merged {
			dec.Accepted = false
			dec.Reason = fmt.Sprintf("submission %s holds your children's approved work, and the required check %q %s on its full state. Next: find the cause, fix it in your own area or tell the child whose work causes it, run the check yourself and submit again.\n%s",
				short(u.New), r.Name, why, r.Output)
		} else {
			dec.Note = fmt.Sprintf("the required check %q %s on the branch's full state. This submission holds no child's approved work, so the check does not refuse it: your checker gets the result as evidence, and the merged state that integrates this work must pass it.",
				r.Name, why)
		}
		break
	}
	return dec, nil
}

// childApprovals returns, for each child Manager, the newest approved commit
// of its branch that tip contains, or the base when it contains none or the
// child does not work.
func (d *decider) childApprovals(tip string) (map[string]string, error) {
	refs := map[string]string{}
	for _, c := range d.plan.Managers {
		if c.ID == model.RootPackage || model.ParentOf(c.ID) != d.m.ID {
			continue
		}
		ref := d.plan.Base
		if c.Working {
			var err error
			ref, err = d.newest(Branch(c.ID), tip, func(id string) (bool, error) { return d.approved(id, c.ID) })
			if err != nil {
				return nil, err
			}
		}
		refs[c.ID] = ref
	}
	return refs, nil
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
	if u.Old == d.plan.Base || perr != nil || st.Kind != Submission || st.Manager != Name(d.m.ID) {
		return refuse("the tip %s of your branch is no submission of %q, so there is nothing to review. Next: wait for your maker's submission.", short(u.Old), Name(d.m.ID)), nil
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
	case t.Review == Rejected && disputed == 0 && len(submitted) > 0:
		return refuse("review %s rejects without disputing an item. Next: dispute the items you found defective, with the defect as evidence.", short(u.New)), nil
	}
	return Decision{Accepted: true, Kind: "review"}, nil
}

// verdicts decides a checker's push in verify: one commit without file
// changes that gives every checklist item a verdict. Every such commit holds
// the Run's starting commit's tree, so the required checks run once, on
// Root's verdicts, as in every Run; Root always works. It needs no compile:
// the host compiled that tree before the Run started. A failing check does
// not refuse the push, since verify changes nothing and the checker cannot
// fix it; the Result reports it.
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
	dec := Decision{Accepted: true, Kind: "verdicts"}
	if d.m.ID != model.RootPackage {
		return dec, nil
	}
	results, err := d.checks.Run(c, d.plan.Checks)
	if err != nil {
		return refuse("the required checks could not run on %s: %v. Nothing was decided about your verdicts. Next: push again.", short(u.New), err), nil
	}
	dec.Checks = results
	return dec, nil
}

// maxCompileErrors bounds the compile errors a refusal lists.
const maxCompileErrors = 20

// compile says why a commit's full state does not compile, as `markitect
// check` reports it, or "" when it compiles.
func (d *decider) compile(commit string) (string, error) {
	p, err := project.Load(d.repo.State(commit))
	if err != nil {
		return "", err
	}
	if len(p.Errors) == 0 {
		return "", nil
	}
	errs := p.Errors.Sorted()
	var b strings.Builder
	fmt.Fprintf(&b, "`markitect check` finds %d compile errors:", len(errs))
	for i, e := range errs {
		if i == maxCompileErrors {
			fmt.Fprintf(&b, "\n- and %d more", len(errs)-i)
			break
		}
		b.WriteString("\n- " + e.String())
	}
	b.WriteString("\nNext: fix them, run `markitect check` in your clone and submit again.")
	return b.String(), nil
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
		case v.Word == Deviates && d.plan.Mode == Apply:
			return fmt.Sprintf("%q is %s: in apply a maker makes the realization conform and says %s, or reports %s. Next: fix it and submit again.", v.Item, Deviates, Changed, ModelProblem)
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
// newest state the tip has taken in, else as at the base. A registry page
// belongs to its own Package; everything else below .markitect/ stays as at
// the base. A file two Packages claim equally, or a path that differs from
// another only in case, is refused.
func (d *decider) content(tip string, childRef map[string]string) ([]string, error) {
	base := d.plan.Base
	parentRef := base
	if d.m.ID != model.RootPackage {
		ref, err := d.newest(Branch(model.ParentOf(d.m.ID)), tip, func(string) (bool, error) { return true, nil })
		if err != nil {
			return nil, err
		}
		parentRef = ref
	}
	me, parent := Name(d.m.ID), Name(model.ParentOf(d.m.ID))
	// required returns the commit whose content the file must keep, or ""
	// when the Manager may change it.
	required := func(file string) (string, string) {
		owner, page := d.plan.Pages[file]
		what := "is the registry page of"
		if !page {
			if strings.HasPrefix(file, project.Dir+"/") {
				return base, "is protected: the model, the manifest, the lock, execution.yaml and the rest of " + project.Dir + "/ change only through the owner, never in a Run"
			}
			var tied []string
			owner, tied = d.own.Owner(file)
			if len(tied) > 0 {
				return base, fmt.Sprintf("is claimed equally by %s, which the registry does not allow", quoteAll(tied))
			}
			what = "belongs to"
		}
		switch {
		case owner == d.m.ID:
			return "", ""
		case model.Within(owner, d.m.ID):
			child := ChildToward(d.m.ID, owner)
			ref := childRef[child]
			if ref == base {
				return ref, fmt.Sprintf("%s %q below you; it must stay as at the base until you merge %q's approved work", what, Name(owner), child)
			}
			return ref, fmt.Sprintf("%s %q below you; it must be exactly as in %q's approved commit %s that you merged", what, Name(owner), child, short(ref))
		case parentRef == base:
			return base, fmt.Sprintf("%s %q outside your area; it must stay as at the base. Ask your parent %q for a change there", what, Name(owner), parent)
		default:
			return parentRef, fmt.Sprintf("%s %q outside your area; it must be exactly as in your parent's state %s that you merged. Ask your parent %q for a change there", what, Name(owner), short(parentRef), parent)
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
	var problems, fromBase []string
	seen := map[string]bool{}
	for _, ref := range ordered {
		changed, err := d.repo.Changed(ref, tip)
		if err != nil {
			return nil, err
		}
		if ref == base {
			fromBase = changed
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
	collisions, err := d.caseCollisions(tip, fromBase)
	if err != nil {
		return nil, err
	}
	problems = append(problems, collisions...)
	sort.Strings(problems)
	if len(problems) > 0 {
		problems = append([]string{fmt.Sprintf("the tip %s of %s changes files outside %q's area:", short(tip), d.branch, me)}, problems...)
	}
	return problems, nil
}

// caseCollisions refuses a changed path that differs only in case from
// another path of the tip, file or folder: a checkout on Windows or macOS
// holds both as one, so one Manager could overwrite another's file.
func (d *decider) caseCollisions(tip string, changed []string) ([]string, error) {
	if len(changed) == 0 {
		return nil, nil
	}
	files, err := d.repo.Files(tip)
	if err != nil {
		return nil, err
	}
	spellings := map[string]map[string]bool{}
	for _, f := range files {
		for _, p := range prefixes(f) {
			folded := strings.ToLower(p)
			if spellings[folded] == nil {
				spellings[folded] = map[string]bool{}
			}
			spellings[folded][p] = true
		}
	}
	var problems []string
	reported := map[string]bool{}
	for _, f := range changed {
		for _, p := range prefixes(f) {
			folded := strings.ToLower(p)
			if len(spellings[folded]) < 2 || reported[folded] {
				continue
			}
			reported[folded] = true
			var all []string
			for s := range spellings[folded] {
				all = append(all, s)
			}
			sort.Strings(all)
			problems = append(problems, fmt.Sprintf("%s differ only in case, and a checkout on Windows or macOS holds them as one. Next: keep one spelling.", strings.Join(all, " and ")))
		}
	}
	return problems, nil
}

// prefixes returns a path's folders and the path itself.
func prefixes(file string) []string {
	var out []string
	for i, r := range file {
		if r == '/' {
			out = append(out, file[:i])
		}
	}
	return append(out, file)
}

func quoteAll(ids []string) string {
	q := make([]string, len(ids))
	for i, id := range ids {
		q[i] = fmt.Sprintf("%q", Name(id))
	}
	return strings.Join(q, " and ")
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
	return perr == nil && t.Kind == Review && t.Review == Approved && t.Manager == Name(manager), nil
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
