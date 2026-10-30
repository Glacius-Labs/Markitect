// Package runremote holds the rules of a Run's Git remote (concept 2.0 §6.3):
// the Run's plan, the commit trailers that carry verdicts and reviews, and the
// decision the server-side hook makes on every push. It runs no Git and no
// process; the remote server supplies the repository facts and the checks.
package runremote

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// Mode says what the agents of a Run may push.
type Mode string

const (
	// Apply: makers push work and submissions, checkers push reviews, parents merge.
	Apply Mode = "apply"
	// Verify: checkers only, each pushing verdicts as a commit without file changes.
	Verify Mode = "verify"
)

// Role is an agent's part in its Manager.
type Role string

const (
	Maker   Role = "maker"
	Checker Role = "checker"
)

// RootBranch is Root's branch and BaseBranch the read-only branch at the
// Run's starting commit. Every other Manager's branch is under managers/.
const (
	RootBranch = "root"
	BaseBranch = "base"
)

// Plan is everything the hooks need, fixed by the host before the Run starts
// and never changed while it runs. Ownership, checklists and checks come from
// the base commit: the model, its registry and execution.yaml cannot change in
// a Run, so the base states them for the whole Run.
type Plan struct {
	Run  string `json:"run"`
	Mode Mode   `json:"mode"`
	// Base is the full commit the Run starts from.
	Base string `json:"base"`
	// Managers lists every Package of the model, working or not.
	Managers []Manager `json:"managers"`
	// Pages maps every registry page a Run may change to the Package whose
	// Manager changes it: a live Package's page that exists at the base, and
	// Root's in any case, to that Package, and the page of a Package that no
	// longer exists to Root, which cleans it up. A Package's first page, with
	// its first patterns, is the owner's (DEC-081), so no Manager creates
	// one. Everything else below .markitect/ is protected: the model, the
	// manifest, the lock, execution.yaml, the ignore page, Libraries and
	// Results.
	Pages  map[string]string `json:"pages"`
	Agents []Agent           `json:"agents"`
	// Checks are the required checks of execution.yaml, run in order.
	Checks []project.Check `json:"checks"`
}

// Manager is one Package of the model.
type Manager struct {
	// ID is the Package's path in the model; Root's is "".
	ID string `json:"id"`
	// Patterns are the path patterns the Package owns at the base.
	Patterns []string `json:"patterns,omitempty"`
	// Working Managers have a branch, agents and a checklist in this Run.
	Working   bool     `json:"working,omitempty"`
	Checklist []string `json:"checklist,omitempty"`
}

// Agent is one credential of the remote.
type Agent struct {
	User    string `json:"user"`
	Manager string `json:"manager"`
	Role    Role   `json:"role"`
	// TokenSHA256 is the hex SHA-256 of the agent's token; the remote never
	// holds the token itself.
	TokenSHA256 string `json:"tokenSha256"`
}

// Branch names a Manager's branch in the Run's remote: root for Root, and
// managers/ plus the Package path with every "/" written as "." for the
// others, so that nested Packages never collide as Git refs do. Package
// names hold no ".", so no two Packages share a branch.
func Branch(id string) string {
	if id == model.RootPackage {
		return RootBranch
	}
	return "managers/" + strings.ReplaceAll(id, "/", ".")
}

// Name is how trailers and messages write a Package ID: "." for Root, which
// no Package path can be.
func Name(id string) string {
	if id == model.RootPackage {
		return "."
	}
	return id
}

// Manager returns the Manager with that ID.
func (p *Plan) Manager(id string) (Manager, bool) {
	for _, m := range p.Managers {
		if m.ID == id {
			return m, true
		}
	}
	return Manager{}, false
}

// Agent returns the agent with that user name.
func (p *Plan) Agent(user string) (Agent, bool) {
	for _, a := range p.Agents {
		if a.User == user {
			return a, true
		}
	}
	return Agent{}, false
}

// Image is the container image the required checks run in, and so the Run
// remote's image; empty when the project has no checks.
func (p *Plan) Image() string {
	if len(p.Checks) == 0 {
		return ""
	}
	return p.Checks[0].Environment
}

// ChildToward returns the direct child of top whose subtree holds id, which
// lies strictly below top.
func ChildToward(top, id string) string {
	for model.ParentOf(id) != top {
		id = model.ParentOf(id)
	}
	return id
}

// PlanFor builds the plan of a Run from the project at the base commit and
// the impact of the change: every Package with its patterns, the working
// Managers with their checklists, the registry pages and the required checks.
// The host adds the agents.
func PlanFor(run string, mode Mode, base string, p *project.Project, imp analysis.Impact) (*Plan, error) {
	if len(p.Errors) > 0 {
		return nil, fmt.Errorf("the project at the base has %d compile errors; no Run starts", len(p.Errors))
	}
	plan := &Plan{Run: run, Mode: mode, Base: base, Pages: map[string]string{}, Checks: p.Checks}
	lists := map[string]analysis.Checklist{}
	for _, c := range imp.Checklists {
		lists[c.Package] = c
	}
	working := map[string]bool{}
	for _, pkg := range imp.Packages {
		working[pkg] = true
	}
	for _, id := range p.Model.SortedPackages() {
		m := Manager{ID: id, Working: working[id]}
		page, ok := p.Registry.Pages[id]
		if ok {
			for _, pattern := range page.Patterns {
				m.Patterns = append(m.Patterns, pattern.Text)
			}
		}
		if m.Working {
			m.Checklist = Items(lists[id])
		}
		plan.Managers = append(plan.Managers, m)
		if ok || id == model.RootPackage {
			plan.Pages[PageFile(id)] = id
		}
	}
	for _, page := range p.Registry.Pages {
		if page.Orphaned {
			plan.Pages[page.File] = model.RootPackage
		}
	}
	return plan, nil
}

// PageFile is the registry page of a Package.
func PageFile(id string) string {
	if id == model.RootPackage {
		return project.RootPage
	}
	return project.RegistryDir + "/" + id + ".md"
}

// Items are the checklist items of one Manager as verdict trailers name
// them: a Definition by its ID, the n-th Finding as "finding n on ID", and
// stale realization by its code, subject and place.
func Items(c analysis.Checklist) []string {
	var items []string
	for _, i := range c.Own {
		items = append(items, i.Definition)
	}
	for _, i := range c.Inherited {
		items = append(items, i.Definition)
	}
	for n, f := range c.Findings {
		items = append(items, fmt.Sprintf("finding %d on %s", n+1, f.Definition))
	}
	for _, s := range c.Stale {
		subject := s.File
		if s.Code == "stale.definition" {
			subject = s.Target
		}
		place := s.PageFile
		if s.Line > 0 {
			place = fmt.Sprintf("%s:%d", place, s.Line)
		}
		if subject == "" {
			items = append(items, s.Code+" "+place)
		} else {
			items = append(items, s.Code+" "+subject+" at "+place)
		}
	}
	return items
}

// Validate reports every inconsistency of the plan. A Run never starts on an
// invalid plan.
func (p *Plan) Validate() error {
	var problems []string
	report := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	if p.Run == "" {
		report("run: missing")
	}
	if p.Mode != Apply && p.Mode != Verify {
		report("mode %q: must be %q or %q", p.Mode, Apply, Verify)
	}
	if len(p.Base) != 40 || strings.Trim(p.Base, "0123456789abcdef") != "" {
		report("base %q: must be a full SHA-1 commit id", p.Base)
	}
	ids := map[string]bool{}
	for _, m := range p.Managers {
		if ids[m.ID] {
			report("manager %q: duplicate", m.ID)
		}
		ids[m.ID] = true
	}
	if !ids[model.RootPackage] {
		report("managers: Root is missing")
	}
	branches := map[string]string{}
	for _, m := range p.Managers {
		if m.ID != model.RootPackage && (!model.ValidPath(m.ID) || !ids[model.ParentOf(m.ID)]) {
			report("manager %q: not a Package path below another Manager", m.ID)
		}
		for _, pattern := range m.Patterns {
			if err := realization.ValidatePattern(pattern); err != nil {
				report("manager %q: pattern %q: %v", Name(m.ID), pattern, err)
			}
		}
		if !m.Working {
			if len(m.Checklist) > 0 {
				report("manager %q: a Manager that does not work has no checklist", Name(m.ID))
			}
			continue
		}
		b := Branch(m.ID)
		if !validBranch(b) {
			report("manager %q: branch %q is not a valid Git branch name", Name(m.ID), b)
		} else if other, dup := branches[b]; dup {
			report("manager %q: branch %q is also %q's", Name(m.ID), b, other)
		}
		branches[b] = Name(m.ID)
		if parent, ok := p.Manager(model.ParentOf(m.ID)); m.ID != model.RootPackage && ok && !parent.Working {
			report("manager %q: works, but its parent %q does not", Name(m.ID), Name(parent.ID))
		}
		seen := map[string]bool{}
		for _, item := range m.Checklist {
			if strings.TrimSpace(item) != item || item == "" || strings.ContainsAny(item, "|\r\n") {
				report("manager %q: checklist item %q is empty, padded or holds '|' or a line break", Name(m.ID), item)
			}
			if seen[item] {
				report("manager %q: checklist item %q is listed twice", Name(m.ID), item)
			}
			seen[item] = true
		}
	}
	pages := make([]string, 0, len(p.Pages))
	for f := range p.Pages {
		pages = append(pages, f)
	}
	sort.Strings(pages)
	for _, f := range pages {
		if f != project.RootPage && !(strings.HasPrefix(f, project.RegistryDir+"/") && strings.HasSuffix(f, ".md") && realization.ValidFile(f)) {
			report("page %q: not a registry page", f)
		}
		if !ids[p.Pages[f]] {
			report("page %q: Package %q is not a Manager", f, p.Pages[f])
		}
	}
	users, roles := map[string]bool{}, map[string]bool{}
	for _, a := range p.Agents {
		if a.User == "" || strings.ContainsAny(a.User, ":@/ \t\r\n") {
			report("agent %q: user must be non-empty without ':', '@', '/' or white space", a.User)
		}
		if a.User == HostUser {
			report("agent %q: the name is the host's", a.User)
		}
		if users[a.User] {
			report("agent %q: duplicate", a.User)
		}
		users[a.User] = true
		m, ok := p.Manager(a.Manager)
		switch {
		case !ok || !m.Working:
			report("agent %q: Manager %q does not work in this Run", a.User, Name(a.Manager))
		case a.Role != Maker && a.Role != Checker:
			report("agent %q: role %q must be %q or %q", a.User, a.Role, Maker, Checker)
		case p.Mode == Verify && a.Role == Maker:
			report("agent %q: verify has no makers", a.User)
		}
		key := a.Manager + "\x00" + string(a.Role)
		if roles[key] {
			report("agent %q: Manager %q already has a %s", a.User, Name(a.Manager), a.Role)
		}
		roles[key] = true
		if len(a.TokenSHA256) != 64 || strings.Trim(a.TokenSHA256, "0123456789abcdef") != "" {
			report("agent %q: tokenSha256 must be 64 lower-case hex digits", a.User)
		}
	}
	names := map[string]bool{}
	for _, c := range p.Checks {
		if c.Name == "" || names[c.Name] {
			report("check %q: name missing or duplicate", c.Name)
		}
		names[c.Name] = true
		switch {
		case strings.TrimSpace(c.Command) == "":
			report("check %q: command missing", c.Name)
		case c.Directory != "" && c.Directory != "." && !realization.ValidFile(c.Directory):
			report("check %q: directory %q is not a clean relative path", c.Name, c.Directory)
		case c.Timeout <= 0:
			report("check %q: timeout must be positive", c.Name)
		case c.Environment == "":
			report("check %q: names no image", c.Name)
		case c.Environment != p.Image():
			report("check %q: runs in %q, but the Run's remote runs every check in one image, %q", c.Name, c.Environment, p.Image())
		}
	}
	if len(problems) > 0 {
		return errors.New("invalid Run plan:\n- " + strings.Join(problems, "\n- "))
	}
	return nil
}

// validBranch applies Git's ref name rules to refs/heads/<b>.
func validBranch(b string) bool {
	if b == "" || strings.HasSuffix(b, "/") || strings.Contains(b, "..") || strings.Contains(b, "@{") || strings.HasSuffix(b, ".") {
		return false
	}
	for _, r := range b {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	for _, seg := range strings.Split(b, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") || strings.HasSuffix(seg, ".lock") {
			return false
		}
	}
	return true
}
