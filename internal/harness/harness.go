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

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Version names the harness text. It changes whenever the rendered text does.
const Version = 5

// MaxInlineDiff bounds the diff a checker receives inline. A larger diff is
// listed by file, and the checker reads it in its clone.
const MaxInlineDiff = 256 << 10

// Input is everything one agent's documents are rendered from: the engine's
// start, the project at the Run's base and, for a checker in apply, the
// submission it reviews.
type Input struct {
	Start agentport.Start
	// Project is the project at the Run's base, without compile errors.
	Project *project.Project
	// Submission is what a checker in apply reviews.
	Submission *Submission
}

// Package is the Package path of a Manager ID: "" for Root's ".".
func Package(id string) string {
	if id == runremote.Name(model.RootPackage) {
		return model.RootPackage
	}
	return id
}

// Submission is one maker submission as its checker sees it: the commit, its
// verdicts and its diff.
type Submission struct {
	Commit   string
	Verdicts []Verdict
	// From is the commit the diff starts from: empty for the commit the
	// Manager's work started from, which Render shows; for Turn, the
	// submission the checker's previous turn received.
	From string
	// Diff is `git diff FROM COMMIT`; Files lists the changed files with
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
	s := in.Start
	pkg := Package(s.Manager.ID)
	if _, ok := in.Project.Model.Packages[pkg]; !ok || s.Manager.ID == "" {
		return Document{}, fmt.Errorf("harness: no Package %q in the model", s.Manager.ID)
	}
	switch {
	case s.Mode != Apply && s.Mode != Verify:
		return Document{}, fmt.Errorf("harness: unknown mode %q", s.Mode)
	case s.Role != Maker && s.Role != Checker:
		return Document{}, fmt.Errorf("harness: unknown role %q", s.Role)
	case s.Mode == Verify && s.Role == Maker:
		return Document{}, fmt.Errorf("harness: verify has no makers")
	case s.Mode == Apply && s.Role == Checker && (in.Submission == nil || in.Submission.From != ""):
		return Document{}, fmt.Errorf("harness: a checker in apply needs the submission it reviews, with its diff from its Manager's start")
	}
	for _, it := range s.Checklist {
		if it.Key == "" || strings.ContainsAny(it.Key, "|\r\n") {
			return Document{}, fmt.Errorf("harness: checklist item %q is no plan key", it.Key)
		}
	}
	for _, c := range s.Manager.Children {
		if model.ParentOf(c) != pkg || in.Project.Model.Packages[c] == nil {
			return Document{}, fmt.Errorf("harness: %q is no child Package of %s", c, s.Manager.ID)
		}
	}
	r := renderer{in: in, s: s, pkg: pkg, m: in.Project.Model}
	return Document{Base: r.base(), Context: r.context(), Task: r.task()}, nil
}

type renderer struct {
	in  Input
	s   agentport.Start
	pkg string
	m   *model.Model
}

func (r renderer) base() string {
	var b strings.Builder
	s := r.s
	fmt.Fprintf(&b, "# Role\n\nYou are the %s of the Manager of %s in Markitect Run `%s` (%s).\n\n", s.Role, packageName(r.pkg), s.Run, s.Mode)
	b.WriteString("Markitect keeps one canonical model of this project, written as Markdown Definitions under `.markitect/model/`, " +
		"and an organization of Managers, one per Package, realizes it. Each Manager owns the files its registry page claims and answers for their conformance to the model. " +
		"The model is authoritative and does not change in a Run.\n\n")
	switch {
	case s.Role == Maker:
		b.WriteString("As maker you change the files of your area so that they conform, and give each checklist item a verdict with evidence. " +
			"Your checker then re-checks every verdict in a conversation of its own; it sees the model, the repository at your submission and its diff, and nothing you write besides the commit, not even your Notes. " +
			"You keep this conversation for the Run: a later turn brings your checker's answer, a message or your children's approved work.\n\n")
	case s.Mode == Apply:
		b.WriteString("As checker you re-check a maker's verdicts. You did not see the maker's work: you see the model, the repository at the submission and its diff from the commit your Manager's work started from, and nothing the maker wrote besides the commit. You never edit files. " +
			"You keep this conversation for the Run: each new submission comes in a later turn.\n\n")
	default:
		b.WriteString("As checker you judge whether the repository at the Run's base conforms to each checklist item. You never edit files.\n\n")
	}
	b.WriteString("# Repository\n\n")
	branch := runremote.Branch(r.pkg)
	fmt.Fprintf(&b, "- `/work` is your clone of the Run's remote (`origin`), on your Manager's branch `%s`. The branch `%s` is the Run's starting commit.\n", branch, runremote.BaseBranch)
	switch page := r.page(r.pkg); {
	case page != "":
		fmt.Fprintf(&b, "- Your Package's registry page is `%s`.", page)
	case r.pkg == "":
		fmt.Fprintf(&b, "- Your Package has no registry page yet; it belongs at `%s`.", runremote.PageFile(r.pkg))
	default:
		fmt.Fprintf(&b, "- Your Package has no registry page at the Run's start, and only the owner writes its first one (it would be `%s`): "+
			"items that cannot conform without files of your own get `model-problem`.", runremote.PageFile(r.pkg))
	}
	if pats := r.patterns(r.pkg); len(pats) > 0 {
		fmt.Fprintf(&b, " Your area is the paths it claims: %s", code(pats))
		if r.pkg == "" {
			b.WriteString(", and every path no other Package claims")
		}
		b.WriteString(".\n")
	} else if r.pkg == "" {
		b.WriteString(" Your area is every path no other Package claims.\n")
	} else {
		b.WriteString(" It claims no paths.\n")
	}
	b.WriteString("- Every repository file outside `.markitect/` is listed on its owner's registry page, with the Definitions it realizes, or ignored.\n")
	if r.pkg != "" {
		parent := model.ParentOf(r.pkg)
		fmt.Fprintf(&b, "- Your parent is %s, on branch `%s`.\n", packageName(parent), runremote.Branch(parent))
	}
	if kids := s.Manager.Children; len(kids) > 0 {
		var names []string
		for _, c := range kids {
			names = append(names, fmt.Sprintf("`%s` (branch `%s`)", c, runremote.Branch(c)))
		}
		fmt.Fprintf(&b, "- Your children in this Run: %s. Their work reaches your branch when you merge their approved commits.\n", strings.Join(names, ", "))
	}
	if s.Role == Maker {
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
	pkg := r.m.Packages[r.pkg]
	section("Your Package's Definitions", pkg.Definitions)
	var inherited []string
	above := model.Ancestors(r.pkg)
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
	in, s := r.in, r.s
	b.WriteString("# Task\n\n")
	switch {
	case s.Task != "":
		b.WriteString(strings.TrimSpace(s.Task) + "\n\n")
	case s.Role == Maker && len(s.Checklist) == 0:
		b.WriteString("Your checklist is empty: integrate your children's approved work into your branch and submit it, with no verdict.\n\n")
	case s.Role == Maker:
		b.WriteString("Make the repository conform to every item on your checklist, and give each item a verdict. " +
			"A verdict means: conforms to this item and to the Rules above it, so a change that breaks an unchanged inherited Rule does not conform.\n\n")
	case s.Mode == Apply:
		fmt.Fprintf(&b, "Re-check every verdict of submission `%s` on branch `%s` against the model and the repository at that commit.\n\n", in.Submission.Commit, runremote.Branch(r.pkg))
	default:
		b.WriteString("Judge whether the repository at the Run's base conforms to every item on your checklist and to the Rules above it.\n\n")
	}
	if s.Role == Checker && s.Mode == Apply && s.Round > 1 {
		b.WriteString(recheck)
	}
	if len(s.Checklist) > 0 {
		b.WriteString("## Checklist\n\n| Item | Part | Why |\n|---|---|---|\n")
		for _, it := range s.Checklist {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", it.Key, it.Part, cell(it.Why))
		}
		b.WriteString("\n")
	}
	r.covers(&b)
	if s.Role == Checker && s.Mode == Apply {
		s := in.Submission
		b.WriteString("## Submitted verdicts\n\n| Item | Verdict | Evidence |\n|---|---|---|\n")
		for _, v := range s.Verdicts {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", v.Item, v.Verdict, cell(v.Evidence))
		}
		b.WriteString("\n")
		diff(&b, runremote.BaseBranch, s)
	}
	messages(&b, s.Inbox)
	b.WriteString("# Answer\n\n")
	switch {
	case s.Role == Maker:
		b.WriteString("Call `submit` with exactly one verdict per checklist item:\n\n" +
			"- `conforms`: the item already conformed; you changed nothing for it.\n" +
			"- `changed`: you changed files so that it conforms.\n" +
			"- `model-problem`: the model contradicts itself or leaves open behavior at the project boundary for this item, so it cannot be settled within the model; also `escalate` it.\n\n" +
			"A gap without effect at the project boundary is yours to decide and document. Evidence names the files, tests or Definitions that show the verdict holds. " +
			"The remote runs the required checks on your branch's full state: it refuses a submission that merges your children's work and fails them, and your checker sees their results either way. " +
			"Your checker's answer and every message for you arrive in a later turn.\n")
	case s.Mode == Apply:
		b.WriteString("Call `review` with one answer per submitted verdict, `confirmed` or `disputed`, each with evidence: what you read or ran, and for a dispute what is wrong and where. " +
			"The outcome is `approved` when you confirm every verdict and `rejected` when you dispute at least one.\n")
	default:
		b.WriteString("Call `submit` with exactly one verdict per checklist item, each with evidence: `conforms`; `deviates`, when the repository does not conform, with what you observed and what the model expects; " +
			"or `model-problem`, when the model contradicts itself or leaves open behavior at the project boundary for this item. " +
			"`changed` is no answer here: verify changes nothing.\n")
	}
	return b.String()
}

// recheck tells a checker after its first turn that it reviews afresh.
const recheck = "Your earlier reviews in this Run do not carry over: re-check every verdict of this submission in full against the model and the repository at its commit, " +
	"those you confirmed before included, as for a submission you see for the first time.\n\n"

// Turn writes the message of a later turn of an agent that continues its
// conversation (long-lived Managers): only what brought the turn about,
// since the conversation holds the role, the model, the checklist and the
// answer format. A maker gets the host's event and the messages delivered
// with the turn. A checker in apply gets the new submission: each checklist
// item's verdict and how it changed since last, the submission its previous
// turn received, and the diff from last, which in.Submission.From names.
func Turn(in Input, last *Submission) (string, error) {
	s := in.Start
	switch {
	case s.Role != Maker && s.Role != Checker:
		return "", fmt.Errorf("harness: unknown role %q", s.Role)
	case s.Mode == Apply && s.Role == Checker && (in.Submission == nil || last == nil || in.Submission.From != last.Commit):
		return "", fmt.Errorf("harness: a checker's later turn needs the new submission with its diff from the one its previous turn received")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Turn %d\n\n", s.Round)
	event := s.Event
	if event == "" {
		event = s.Task
	}
	if event == "" && s.Role == Checker && s.Mode == Apply {
		event = fmt.Sprintf("Re-check every verdict of submission `%s` on branch `%s` against the model and the repository at that commit.", in.Submission.Commit, s.Git.Branch)
	}
	if event = strings.TrimSpace(event); event != "" {
		b.WriteString(event + "\n\n")
	}
	if s.Role == Checker && s.Mode == Apply {
		sub := in.Submission
		b.WriteString(recheck)
		if len(s.Checklist) > 0 {
			verdicts := func(sub *Submission) map[string]Verdict {
				m := map[string]Verdict{}
				for _, v := range sub.Verdicts {
					m[v.Item] = v
				}
				return m
			}
			now, before := verdicts(sub), verdicts(last)
			fmt.Fprintf(&b, "## Checklist\n\n| Item | Verdict | Since `%s` | Evidence |\n|---|---|---|---|\n", short(last.Commit))
			for _, it := range s.Checklist {
				v, ok := now[it.Key]
				w, had := before[it.Key]
				verdict, since := v.Verdict, "unchanged"
				switch {
				case !ok:
					verdict, since = "none", "none"
				case !had:
					since = "new"
				case v.Verdict != w.Verdict:
					since = "was `" + w.Verdict + "`"
				case v.Evidence != w.Evidence:
					since = "new evidence"
				}
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", it.Key, verdict, since, cell(v.Evidence))
			}
			b.WriteString("\n")
		}
		if sub.Commit == last.Commit {
			fmt.Fprintf(&b, "Submission `%s` is the one your previous turn received; its diff is in this conversation, and `git diff %s %s` in `/work` shows it again.\n\n", sub.Commit, s.Git.Base, sub.Commit)
		} else {
			fmt.Fprintf(&b, "Below is the diff from `%s`, the submission your previous turn received; `git diff %s %s` in `/work` shows the whole diff from the commit the Manager's work started from.\n\n", last.Commit, s.Git.Base, sub.Commit)
			diff(&b, last.Commit, sub)
		}
	}
	messages(&b, s.Inbox)
	b.WriteString("# Answer\n\n")
	switch {
	case s.Role == Maker:
		b.WriteString("Answer as your first turn says: a submission calls `submit` with exactly one verdict per checklist item.\n")
	case s.Mode == Apply:
		b.WriteString("Answer as your first turn says: call `review` with one answer per submitted verdict.\n")
	default:
		b.WriteString("Answer as your first turn says: call `submit` with exactly one verdict per checklist item.\n")
	}
	return b.String(), nil
}

// diff writes a submission's diff from the commit from, or lists the
// changed files when the diff is larger than is shown.
func diff(b *strings.Builder, from string, s *Submission) {
	fmt.Fprintf(b, "## Diff from `%s` to `%s`\n\n", from, s.Commit)
	if len(s.Diff) <= MaxInlineDiff {
		fence := fenceFor([]byte(s.Diff))
		fmt.Fprintf(b, "%sdiff\n%s\n%s\n\n", fence, strings.TrimRight(s.Diff, "\n"), fence)
		return
	}
	fmt.Fprintf(b, "The diff is larger than %d KiB, more than is shown here; `git diff %s %s` in `/work` shows it. Changed files:\n\n", MaxInlineDiff>>10, from, s.Commit)
	for _, f := range s.Files {
		fmt.Fprintf(b, "- %s\n", f)
	}
	b.WriteString("\n")
}

// messages writes the messages delivered with a turn.
func messages(b *strings.Builder, inbox []agentport.Message) {
	if len(inbox) == 0 {
		return
	}
	b.WriteString("## Messages for you\n\n")
	for _, m := range inbox {
		from := "the host"
		if m.From != "" {
			from = "`" + m.From + "`"
		}
		fmt.Fprintf(b, "### `%s`: %s from %s", m.ID, m.Kind, from)
		if m.ReplyTo != "" {
			fmt.Fprintf(b, ", answering `%s`", m.ReplyTo)
		}
		fmt.Fprintf(b, "\n\n%s\n\n", strings.TrimSpace(m.Text))
	}
}

func short(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

// covers states what a verdict covers (DEC-088): the model in full for each
// item, the code of the Manager's own Package only, and integration by use.
func (r renderer) covers(b *strings.Builder) {
	s := r.s
	maker, verify, parent := s.Role == Maker, s.Mode == Verify, len(s.Manager.Children) > 0
	if len(s.Checklist) == 0 && (!parent || verify) {
		return
	}
	b.WriteString("## What a verdict covers\n\n")
	if len(s.Checklist) > 0 {
		b.WriteString("- **The model, in full.** A verdict covers the item's whole Definition, its examples included, and the Definitions it links to, as the model above shows them. ")
		if maker || verify {
			b.WriteString("Where they contradict each other, nothing can conform, and the only right verdict is `model-problem`. ")
		} else {
			b.WriteString("Where they contradict each other, nothing can conform: the maker's only right verdict is `model-problem`, so dispute a verdict that claims the item conforms. ")
		}
		b.WriteString("It is no review of the whole model: stay with each item and its links.\n")
		switch {
		case maker:
			b.WriteString("- **Evidence.** For `conforms` and `changed`, the evidence names how each example in the item's Definition is met, for example by which test.\n")
		case verify:
			b.WriteString("- **Evidence.** For `conforms`, the evidence names how each example in the item's Definition is met, for example by which test.\n")
		default:
			b.WriteString("- **Evidence.** Re-check each example in the item's Definition yourself. Dispute a `conforms` or `changed` verdict whose evidence does not show how each example is met, " +
				"and one whose realization matches part of the Definition but not its example.\n")
		}
		// Root has no parent, and verify integrates nothing: there the
		// verdicts judge the Package's files and nothing is left to a parent.
		switch {
		case r.pkg == model.RootPackage || verify:
			b.WriteString("- **The code, your own Package's.** The verdicts judge your Package's files against your items.\n")
		case maker:
			b.WriteString("- **The code, your own Package's.** Your verdicts judge your Package's files against your items. Whether the rest of the program uses them yet is your parent's integration.\n")
		default:
			b.WriteString("- **The code, your own Package's.** Judge your Package's files against your items. Whether the rest of the program uses them yet is your parent's integration, " +
				"which its checker judges and the required checks gate on the merged state; it is no reason to dispute a verdict.\n")
		}
	}
	switch {
	case parent && maker:
		b.WriteString("- **Your children's work.** Integrate it by using it, for example by calling a child's code, not by building a child's behavior a second time in your own files.\n")
	case parent && !verify:
		b.WriteString("- **Your children's work.** The Manager integrates it by using it. Dispute the submission when one of its own files builds a child's behavior a second time instead.\n")
	}
	b.WriteString("\n")
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
