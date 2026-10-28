// Package harness renders what one maker or checker of a Manager receives
// (concept 2.0 §5, §6.2): its role and repository, the model it knows, its
// task with the checklist, and the answer format, together with the tools it
// answers through. It is pure: it reads a loaded project and writes text.
// Prompts are a task and an answer format; process lives in host state and
// the tools.
package harness

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Version names the harness text. It changes whenever the rendered text does.
const Version = 1

// MaxInlineDiff bounds the diff a checker receives inline. A larger diff is
// listed by file, and the checker reads it in its clone.
const MaxInlineDiff = 256 << 10

// Input is everything one agent's documents are rendered from.
type Input struct {
	Run  string
	Mode Mode
	Role Role
	// Package is the Manager's Package path; "" is Root.
	Package string
	// Project is the project at the Run's base, without compile errors.
	Project *project.Project
	// Items is the Manager's checklist (Items of its analysis checklist).
	Items []Item
	// Working are the Packages that work in this Run.
	Working []string
	// Submission is what a checker in apply reviews.
	Submission *Submission
}

// Submission is one maker submission as its checker sees it: the commit, its
// verdicts and the diff from the Run's base.
type Submission struct {
	Commit   string
	Verdicts []Verdict
	// Diff is `git diff base COMMIT`; Files lists the changed files with
	// their change, for a diff above MaxInlineDiff.
	Diff  string
	Files []string
}

// Verdict is one submitted verdict.
type Verdict struct {
	Item, Verdict, Evidence string
}

// Document is the rendered text of one agent. Base states the role and the
// repository; Context is the model the Manager knows; Task is the first
// message, with the checklist and the answer format.
type Document struct {
	Base, Context, Task string
}

// Render writes the documents of one agent. An empty checklist is a parent
// that only integrates its children's work.
func Render(in Input) (Document, error) {
	if in.Project == nil || in.Project.Model == nil {
		return Document{}, fmt.Errorf("harness: no loaded project")
	}
	if len(in.Project.Errors) > 0 {
		return Document{}, fmt.Errorf("harness: the project has compile errors; no agent starts")
	}
	if _, ok := in.Project.Model.Packages[in.Package]; !ok {
		return Document{}, fmt.Errorf("harness: no Package %q in the model", in.Package)
	}
	switch {
	case in.Mode != Apply && in.Mode != Verify:
		return Document{}, fmt.Errorf("harness: unknown mode %q", in.Mode)
	case in.Role != Maker && in.Role != Checker:
		return Document{}, fmt.Errorf("harness: unknown role %q", in.Role)
	case in.Mode == Verify && in.Role == Maker:
		return Document{}, fmt.Errorf("harness: verify has no makers")
	case in.Mode == Apply && in.Role == Checker && in.Submission == nil:
		return Document{}, fmt.Errorf("harness: a checker in apply needs the submission it reviews")
	}
	for _, it := range in.Items {
		if it.Key == "" || strings.ContainsAny(it.Key, "|\r\n") {
			return Document{}, fmt.Errorf("harness: checklist item %q is no plan key", it.Key)
		}
	}
	r := renderer{in: in, m: in.Project.Model}
	return Document{Base: r.base(), Context: r.context(), Task: r.task()}, nil
}

type renderer struct {
	in Input
	m  *model.Model
}

func (r renderer) working(pkg string) bool {
	for _, w := range r.in.Working {
		if w == pkg {
			return true
		}
	}
	return false
}

// children are the Manager's direct child Packages that work in this Run.
func (r renderer) children() []string {
	var out []string
	for _, c := range r.m.Packages[r.in.Package].Children {
		if r.working(c) {
			out = append(out, c)
		}
	}
	return out
}

func (r renderer) base() string {
	var b strings.Builder
	in := r.in
	fmt.Fprintf(&b, "# Role\n\nYou are the %s of the Manager of %s in Markitect Run `%s` (%s).\n\n", in.Role, packageName(in.Package), in.Run, in.Mode)
	b.WriteString("Markitect keeps one canonical model of this project, written as Markdown Definitions under `.markitect/model/`, " +
		"and an organization of Managers, one per Package, realizes it. Each Manager owns the files its registry page claims and answers for their conformance to the model. " +
		"The model is authoritative and does not change in a Run.\n\n")
	switch {
	case in.Role == Maker:
		b.WriteString("As maker you change the files of your area so that they conform, and give each checklist item a verdict with evidence. " +
			"A fresh checker who sees only the model and your diff then re-checks every verdict.\n\n")
	case in.Mode == Apply:
		b.WriteString("As checker you re-check a maker's verdicts. You did not see the maker's work: you see the model, the repository at the submission and its diff from the base. You never edit files.\n\n")
	default:
		b.WriteString("As checker you judge whether the repository at the Run's base conforms to each checklist item. You never edit files.\n\n")
	}
	b.WriteString("# Repository\n\n")
	branch := runremote.Branch(in.Package)
	fmt.Fprintf(&b, "- `/work` is your clone of the Run's remote (`origin`), on your Manager's branch `%s`. The branch `%s` is the Run's starting commit.\n", branch, runremote.BaseBranch)
	if page := r.page(in.Package); page != "" {
		fmt.Fprintf(&b, "- Your Package's registry page is `%s`.", page)
	} else {
		fmt.Fprintf(&b, "- Your Package has no registry page yet; it belongs at `%s`.", registryPath(in.Package))
	}
	if pats := r.patterns(in.Package); len(pats) > 0 {
		fmt.Fprintf(&b, " Your area is the paths it claims: %s", code(pats))
		if in.Package == "" {
			b.WriteString(", and every path no other Package claims")
		}
		b.WriteString(".\n")
	} else if in.Package == "" {
		b.WriteString(" Your area is every path no other Package claims.\n")
	} else {
		b.WriteString(" It claims no paths.\n")
	}
	b.WriteString("- Every repository file outside `.markitect/` is listed on its owner's registry page, with the Definitions it realizes, or ignored.\n")
	if in.Package != "" {
		parent := model.ParentOf(in.Package)
		fmt.Fprintf(&b, "- Your parent is %s, on branch `%s`.\n", packageName(parent), runremote.Branch(parent))
	}
	if kids := r.children(); len(kids) > 0 {
		var names []string
		for _, c := range kids {
			names = append(names, fmt.Sprintf("`%s` (branch `%s`)", c, runremote.Branch(c)))
		}
		fmt.Fprintf(&b, "- Your children in this Run: %s. Their work reaches your branch when you merge their approved commits.\n", strings.Join(names, ", "))
	}
	if in.Role == Maker {
		b.WriteString("- The remote accepts a push to your branch only when it changes files of your area and your registry page, or merges your children's approved commits unchanged. " +
			"Nothing else under `.markitect/` changes. A refused push names its reason.\n")
	} else {
		b.WriteString("- The remote accepts from you only a commit that changes no file.\n")
	}
	b.WriteString("- `markitect check`, `markitect refs ID`, `markitect graph`, `markitect impact --base base` and `markitect lint` read the model in `/work`.\n")
	return b.String()
}

// context renders the model the Manager knows (concept 2.0 §5): its own
// Definitions, everything its parents define, the Definitions they link to in
// other Packages with the Kinds in use, and a table of contents of its
// children's public Definitions.
func (r renderer) context() string {
	var b strings.Builder
	shown := map[string]bool{}
	b.WriteString("# Model\n\n")
	section := func(title string, ids []string) {
		var fresh []string
		for _, id := range ids {
			if !shown[id] {
				shown[id] = true
				fresh = append(fresh, id)
			}
		}
		if len(fresh) == 0 {
			return
		}
		fmt.Fprintf(&b, "## %s\n\n", title)
		for _, id := range fresh {
			r.definition(&b, id)
		}
	}
	pkg := r.m.Packages[r.in.Package]
	section("Your Package's Definitions", pkg.Definitions)
	var inherited []string
	above := model.Ancestors(r.in.Package)
	for i := len(above) - 1; i >= 0; i-- {
		inherited = append(inherited, r.m.Packages[above[i]].Definitions...)
	}
	section("Inherited from your parents", inherited)
	// Links point up or sideways (concept 2.0 §2.3); what is not shown yet
	// lies in another Package or a Library.
	var linked []string
	for _, id := range append(append([]string(nil), pkg.Definitions...), inherited...) {
		for _, l := range r.m.Definitions[id].Links {
			if _, ok := r.m.Definitions[l.Target]; ok && !l.Package && !l.Kind {
				linked = append(linked, l.Target)
			}
		}
	}
	sort.Strings(linked)
	section("Linked from these", linked)
	var kinds []string
	for id := range shown {
		if k := r.m.Definitions[id].Kind; k != "" {
			kinds = append(kinds, k)
		}
	}
	sort.Strings(kinds)
	section("Kinds of the Definitions above", kinds)
	for _, c := range pkg.Children {
		fmt.Fprintf(&b, "## Child %s\n\nPublic Definitions in its subtree, by title and first sentence; read the file for the full text.\n\n", c)
		n := 0
		for _, p := range r.m.Subtree(c) {
			for _, id := range r.m.Packages[p].Definitions {
				d := r.m.Definitions[id]
				if d.Internal {
					continue
				}
				fmt.Fprintf(&b, "- `%s` (%s): %s", id, d.File, d.Title)
				if s := firstSentence(d.Data); s != "" {
					fmt.Fprintf(&b, ": %s", s)
				}
				b.WriteString("\n")
				n++
			}
		}
		if n == 0 {
			b.WriteString("- none\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (r renderer) definition(b *strings.Builder, id string) {
	d := r.m.Definitions[id]
	if d == nil {
		return
	}
	where := d.File
	if !d.InModel() {
		where = "Library " + d.Library
	}
	fmt.Fprintf(b, "### `%s` (%s)\n\n", id, where)
	fence := fenceFor(d.Data)
	fmt.Fprintf(b, "%smarkdown\n%s", fence, bytes.TrimRight(d.Data, "\n"))
	fmt.Fprintf(b, "\n%s\n\n", fence)
}

func (r renderer) task() string {
	var b strings.Builder
	in := r.in
	b.WriteString("# Task\n\n")
	switch {
	case in.Role == Maker && len(in.Items) == 0:
		b.WriteString("Your checklist is empty: integrate your children's approved work into your branch and submit it, with no verdict.\n\n")
	case in.Role == Maker:
		b.WriteString("Make the repository conform to every item on your checklist, and give each item a verdict. " +
			"A verdict means: conforms to this item and to the Rules above it, so a change that breaks an unchanged inherited Rule does not conform.\n\n")
	case in.Mode == Apply:
		fmt.Fprintf(&b, "Re-check every verdict of submission `%s` on branch `%s` against the model and the repository at that commit.\n\n", in.Submission.Commit, runremote.Branch(in.Package))
	default:
		b.WriteString("Judge whether the repository at the Run's base conforms to every item on your checklist and to the Rules above it.\n\n")
	}
	if len(in.Items) > 0 {
		b.WriteString("## Checklist\n\n| Item | Part | Why |\n|---|---|---|\n")
		for _, it := range in.Items {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", it.Key, it.Part, cell(it.Detail))
		}
		b.WriteString("\n")
	}
	if in.Role == Checker && in.Mode == Apply {
		s := in.Submission
		b.WriteString("## Submitted verdicts\n\n| Item | Verdict | Evidence |\n|---|---|---|\n")
		for _, v := range s.Verdicts {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", v.Item, v.Verdict, cell(v.Evidence))
		}
		fmt.Fprintf(&b, "\n## Diff from `%s` to `%s`\n\n", runremote.BaseBranch, s.Commit)
		if len(s.Diff) <= MaxInlineDiff {
			fence := fenceFor([]byte(s.Diff))
			fmt.Fprintf(&b, "%sdiff\n%s\n%s\n\n", fence, strings.TrimRight(s.Diff, "\n"), fence)
		} else {
			fmt.Fprintf(&b, "The diff has %d bytes, more than is shown here; `git diff %s %s` in `/work` shows it. Changed files:\n\n", len(s.Diff), runremote.BaseBranch, s.Commit)
			for _, f := range s.Files {
				fmt.Fprintf(&b, "- %s\n", f)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("# Answer\n\n")
	switch {
	case in.Role == Maker:
		b.WriteString("Call `submit` with exactly one verdict per checklist item:\n\n" +
			"- `conforms`: the item already conformed; you changed nothing for it.\n" +
			"- `changed`: you changed files so that it conforms.\n" +
			"- `model-problem`: the model contradicts itself or leaves open behavior at the project boundary for this item, so it cannot be settled within the model; also `escalate` it.\n\n" +
			"A gap without effect at the project boundary is yours to decide and document. Evidence names the files, tests or Definitions that show the verdict holds. " +
			"The remote accepts the submission only when the required checks pass on your branch. Your checker's answer, and every message for you, arrives as a later message.\n")
	case in.Mode == Apply:
		b.WriteString("Call `review` with one answer per submitted verdict, `confirmed` or `disputed`, each with evidence: what you read or ran, and for a dispute what is wrong and where. " +
			"The outcome is `approved` when you confirm every verdict and `rejected` when you dispute at least one.\n")
	default:
		b.WriteString("Call `submit` with exactly one verdict per checklist item, each with evidence: `conforms`; `deviates`, when the repository does not conform, with what you observed and what the model expects; " +
			"or `model-problem`, when the model contradicts itself or leaves open behavior at the project boundary for this item.\n")
	}
	return b.String()
}

func (r renderer) page(pkg string) string {
	if r.in.Project.Registry == nil {
		return ""
	}
	if p, ok := r.in.Project.Registry.Pages[pkg]; ok && !p.Orphaned {
		return p.File
	}
	return ""
}

func (r renderer) patterns(pkg string) []string {
	if r.in.Project.Registry == nil {
		return nil
	}
	p, ok := r.in.Project.Registry.Pages[pkg]
	if !ok {
		return nil
	}
	var out []string
	for _, pat := range p.Patterns {
		out = append(out, pat.Text)
	}
	return out
}

func registryPath(pkg string) string {
	if pkg == "" {
		return ".markitect/registry.md"
	}
	return ".markitect/registry/" + pkg + ".md"
}

func packageName(pkg string) string {
	if pkg == "" {
		return "Root"
	}
	return "Package `" + pkg + "`"
}

func code(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = "`" + s + "`"
	}
	return strings.Join(quoted, ", ")
}

func cell(s string) string {
	return strings.ReplaceAll(oneLine(s), "|", `\|`)
}

// fenceFor returns a backtick fence longer than any backtick run in data.
func fenceFor(data []byte) string {
	longest, run := 0, 0
	for _, c := range data {
		if c == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", max(3, longest+1))
}

// firstSentence is the first sentence of a Definition's first paragraph
// after its front matter and headings.
func firstSentence(data []byte) string {
	page, err := model.ParsePage(data)
	if err != nil {
		return ""
	}
	var para []string
	for _, line := range strings.Split(string(page.Body), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" && len(para) > 0:
			return sentence(strings.Join(para, " "))
		case line == "", strings.HasPrefix(line, "#"):
			continue
		default:
			para = append(para, line)
		}
	}
	return sentence(strings.Join(para, " "))
}

func sentence(p string) string {
	if i := strings.Index(p, ". "); i >= 0 {
		return p[:i+1]
	}
	return p
}
