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
	Agents   []Agent   `json:"agents"`
	// Protected are path patterns no push may change: the model, the
	// manifest, the lock, execution.yaml and the rest of .markitect/. A
	// Manager's registry page is the one exception, owned by its Manager.
	Protected []string `json:"protected"`
	// Checks are the required checks of execution.yaml, run in order.
	Checks []Check `json:"checks"`
}

// Manager is one Package of the model.
type Manager struct {
	// ID is the Package's path in the model; Root's ID is ".".
	ID     string `json:"id"`
	Parent string `json:"parent,omitempty"`
	// Patterns are the path patterns the Package owns at the base.
	Patterns []string `json:"patterns,omitempty"`
	// Registry is the repository path of the Package's registry page.
	Registry string `json:"registry"`
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

// Check is one required check of execution.yaml.
type Check struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	// Directory is relative to the repository root; empty means the root.
	Directory      string            `json:"directory,omitempty"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
	Environment    map[string]string `json:"environment,omitempty"`
}

// RootID is Root's Package ID.
const RootID = "."

// Branch names a Manager's branch in the Run's remote: root for Root, and
// managers/ plus the Package path with every "/" written as "." for the
// others, so that nested Packages never collide as Git refs do.
func Branch(id string) string {
	if id == RootID {
		return RootBranch
	}
	return "managers/" + strings.ReplaceAll(id, "/", ".")
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

// ByBranch returns the working Manager whose branch that is.
func (p *Plan) ByBranch(branch string) (Manager, bool) {
	for _, m := range p.Managers {
		if m.Working && Branch(m.ID) == branch {
			return m, true
		}
	}
	return Manager{}, false
}

// Within reports whether Package id lies in the subtree of Package top,
// itself included.
func (p *Plan) Within(id, top string) bool {
	for id != "" {
		if id == top {
			return true
		}
		m, ok := p.Manager(id)
		if !ok {
			return false
		}
		id = m.Parent
	}
	return false
}

// ChildToward returns the direct child of top whose subtree holds id, which
// lies strictly below top.
func (p *Plan) ChildToward(top, id string) string {
	for {
		m, _ := p.Manager(id)
		if m.Parent == top {
			return id
		}
		id = m.Parent
	}
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
	if !fullCommit(p.Base) {
		report("base %q: must be a full commit id", p.Base)
	}
	ids, branches, pages := map[string]bool{}, map[string]string{}, map[string]string{}
	for _, m := range p.Managers {
		if ids[m.ID] {
			report("manager %q: duplicate", m.ID)
		}
		ids[m.ID] = true
	}
	if !ids[RootID] {
		report("manager %q: Root is missing", RootID)
	}
	for _, m := range p.Managers {
		switch {
		case m.ID == RootID && m.Parent != "":
			report("manager %q: Root has no parent", m.ID)
		case m.ID != RootID && !ids[m.Parent]:
			report("manager %q: parent %q is not a Manager", m.ID, m.Parent)
		case m.ID != RootID && !validPackagePath(m.ID):
			report("manager %q: not a valid Package path", m.ID)
		}
		if !validRepoPath(m.Registry) {
			report("manager %q: registry page %q is not a clean repository path", m.ID, m.Registry)
		} else if other, dup := pages[m.Registry]; dup {
			report("manager %q: registry page %q is also %q's", m.ID, m.Registry, other)
		}
		pages[m.Registry] = m.ID
		if !m.Working {
			if len(m.Checklist) > 0 {
				report("manager %q: a Manager that does not work has no checklist", m.ID)
			}
			continue
		}
		b := Branch(m.ID)
		if !validBranch(b) {
			report("manager %q: branch %q is not a valid Git branch name", m.ID, b)
		} else if other, dup := branches[b]; dup {
			report("manager %q: branch %q is also %q's", m.ID, b, other)
		}
		branches[b] = m.ID
		if parent, ok := p.Manager(m.Parent); m.ID != RootID && ok && !parent.Working {
			report("manager %q: works, but its parent %q does not", m.ID, m.Parent)
		}
		if len(m.Checklist) == 0 {
			report("manager %q: a working Manager needs a checklist", m.ID)
		}
		seen := map[string]bool{}
		for _, item := range m.Checklist {
			if !validItem(item) {
				report("manager %q: checklist item %q is empty or holds '|' or a line break", m.ID, item)
			}
			if seen[item] {
				report("manager %q: checklist item %q is listed twice", m.ID, item)
			}
			seen[item] = true
		}
	}
	if p.cyclic() {
		report("managers: the parent links form a cycle")
	}
	users := map[string]bool{}
	roles := map[string]bool{}
	for _, a := range p.Agents {
		if a.User == "" || strings.ContainsAny(a.User, ":@/ \t\r\n") {
			report("agent %q: user must be non-empty without ':', '@', '/' or white space", a.User)
		}
		if users[a.User] {
			report("agent %q: duplicate", a.User)
		}
		users[a.User] = true
		m, ok := p.Manager(a.Manager)
		switch {
		case !ok || !m.Working:
			report("agent %q: Manager %q does not work in this Run", a.User, a.Manager)
		case a.Role != Maker && a.Role != Checker:
			report("agent %q: role %q must be %q or %q", a.User, a.Role, Maker, Checker)
		case p.Mode == Verify && a.Role == Maker:
			report("agent %q: verify has no makers", a.User)
		}
		key := a.Manager + "\x00" + string(a.Role)
		if roles[key] {
			report("agent %q: Manager %q already has a %s", a.User, a.Manager, a.Role)
		}
		roles[key] = true
		if len(a.TokenSHA256) != 64 || strings.Trim(strings.ToLower(a.TokenSHA256), "0123456789abcdef") != "" {
			report("agent %q: tokenSha256 must be 64 hex digits", a.User)
		}
	}
	for _, pattern := range p.Protected {
		if !validPattern(pattern) {
			report("protected pattern %q is not a clean relative pattern", pattern)
		}
	}
	for _, m := range p.Managers {
		for _, pattern := range m.Patterns {
			if !validPattern(pattern) {
				report("manager %q: pattern %q is not a clean relative pattern", m.ID, pattern)
			}
		}
	}
	names := map[string]bool{}
	for _, c := range p.Checks {
		if c.Name == "" || names[c.Name] {
			report("check %q: name missing or duplicate", c.Name)
		}
		names[c.Name] = true
		if strings.TrimSpace(c.Command) == "" {
			report("check %q: command missing", c.Name)
		}
		if c.Directory != "" && !validRepoPath(c.Directory) {
			report("check %q: directory %q is not a clean relative path", c.Name, c.Directory)
		}
		if c.TimeoutSeconds <= 0 {
			report("check %q: timeoutSeconds must be positive", c.Name)
		}
		keys := make([]string, 0, len(c.Environment))
		for k := range c.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if k == "" || strings.ContainsAny(k, "=\x00") || strings.Contains(c.Environment[k], "\x00") {
				report("check %q: environment entry %q is not valid", c.Name, k)
			}
		}
	}
	if len(problems) > 0 {
		return errors.New("invalid Run plan:\n- " + strings.Join(problems, "\n- "))
	}
	return nil
}

func (p *Plan) cyclic() bool {
	for _, m := range p.Managers {
		seen := map[string]bool{}
		for id := m.ID; id != ""; {
			if seen[id] {
				return true
			}
			seen[id] = true
			next, ok := p.Manager(id)
			if !ok {
				break
			}
			id = next.Parent
		}
	}
	return false
}

func fullCommit(s string) bool {
	return (len(s) == 40 || len(s) == 64) && strings.Trim(s, "0123456789abcdef") == ""
}

func validItem(s string) bool {
	return strings.TrimSpace(s) == s && s != "" && !strings.ContainsAny(s, "|\r\n")
}

func validRepoPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.HasSuffix(p, "/") || strings.Contains(p, "\\") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

func validPackagePath(p string) bool {
	return validRepoPath(p) && !strings.ContainsAny(p, "\x00\r\n")
}

func validPattern(p string) bool {
	return validRepoPath(p) && !strings.ContainsAny(p, "\x00\r\n")
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
