package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Work without a Run (DEC-096, DEC-097): checklist, answer, gate and status read
// the project folder itself, without Git, and keep what they learn about the work
// in the user's cache folder.

// ChecklistRequest is one call of checklist.
type ChecklistRequest struct {
	Dir string
	// Base, Full and Findings name the change, without a Run: from the newest
	// certificate by default, or from the certificate Base names.
	Base     string
	Full     bool
	Findings []analysis.Finding
	// Review prints what a review must still answer.
	Review bool
	// Start begins the work, or goes on with the work under way, and prints the
	// step for the session that drives it instead of the agent's task: which
	// agent to start, and to run gate when it has ended.
	Start bool
}

func (r ChecklistRequest) namesChange() bool {
	return r.Base != "" || r.Full || len(r.Findings) > 0
}

// folder is the project folder as one command finds it.
type folder struct {
	root  string
	files project.Repository
	links map[string]string
	// hashes holds every accounted file outside the results folder, fingerprint
	// all of them in one.
	hashes      map[string]string
	fingerprint string
	// certs holds the hashes of the certificates in the results folder, which
	// the work may not change either.
	certs map[string]string
	cur   *project.Project
}

// readFolder reads the project folder that holds dir.
func (e *Engine) readFolder(dir string) (*folder, error) {
	if e.Folder == nil {
		return nil, errors.New("this build reads no project folder")
	}
	root, err := e.Folder.Root(dir)
	if err != nil {
		return nil, err
	}
	files, links, err := e.Folder.Open(root)
	if err != nil {
		return nil, err
	}
	hashes, err := hashFiles(files)
	if err != nil {
		return nil, err
	}
	cur, err := project.Load(files)
	if err != nil {
		return nil, err
	}
	certs := map[string]string{}
	list, err := files.Files()
	if err != nil {
		return nil, err
	}
	for _, file := range list {
		if isResult(file) && strings.HasSuffix(file, ".json") {
			data, err := files.Read(file)
			if err != nil {
				return nil, err
			}
			certs[file] = fileHash(data)
		}
	}
	return &folder{root: root, files: files, links: links, hashes: hashes, fingerprint: fingerprint(hashes), certs: certs, cur: cur}, nil
}

// changedOwn lists what changed since the work began that the work may not
// change: the owner's files and the certificates.
func (f *folder) changedOwn(w *Work) []string {
	return append(changedFrom(w.Spec, f.specHashes()), changedFrom(w.Certificates, f.certs)...)
}

// specHashes are the hashes the work may not change: Markitect's folder except the
// registry pages and the results, the harness, and the .gitignore files, which say
// what the folder holds. The owner changes them before the work.
func (f *folder) specHashes() map[string]string {
	out := map[string]string{}
	for file, h := range f.hashes {
		if ownersFile(file) {
			out[file] = h
		}
	}
	return out
}

// ownersFile says whether a file is one the work may not change.
func ownersFile(file string) bool {
	return (strings.HasPrefix(file, project.Dir+"/") && !registryPage(file)) || project.IsHarness(file) || path.Base(file) == ".gitignore"
}

// changedFrom lists the files whose hashes differ between was and now.
func changedFrom(was, now map[string]string) []string {
	var out []string
	for f, h := range now {
		if was[f] != h {
			out = append(out, f)
		}
	}
	for f := range was {
		if _, ok := now[f]; !ok {
			out = append(out, f)
		}
	}
	return sortedStrings(out)
}

func sortedStrings(s []string) []string {
	m := map[string]bool{}
	for _, v := range s {
		m[v] = true
	}
	return sortedKeys(m)
}

// certificatePath resolves the certificate a user names: a path in the project
// folder, a name in its results folder, or a file as given; its Result page names
// it too. It returns the path relative to the project folder when it lies inside
// it.
func certificatePath(root, name string) (string, error) {
	candidates := []string{name}
	if !filepath.IsAbs(name) {
		candidates = []string{filepath.Join(root, filepath.FromSlash(name)), filepath.Join(root, filepath.FromSlash(project.ResultsDir), name), name}
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := ReadCertificate(abs); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("%w: %v", ErrUsage, err)
		}
		abs = strings.TrimSuffix(abs, ".md")
		if !strings.HasSuffix(abs, ".json") {
			abs += ".json"
		}
		if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel), nil
		}
		return abs, nil
	}
	return "", fmt.Errorf("%w: %s is no certificate: name the .json or .md file gate wrote under %s", ErrUsage, name, project.ResultsDir)
}

// inProject is a path the work holds, on disk.
func inProject(root, file string) string {
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(root, filepath.FromSlash(file))
}

// workFor is the work checklist names: with --base, --full or --findings the
// change they name, begun afresh unless it is the one under way; otherwise the
// work under way, or the change from the newest certificate when there is none.
// A plain checklist does not begin the work again when the spec, the harness or
// the certificates changed under it: refusal says so, and naming the change
// begins it again. With no certificate the list holds every Definition.
func (e *Engine) workFor(f *folder, req ChecklistRequest, stored *Work) (w *Work, refusal string, err error) {
	newest, _, err := newestCertificate(f.root)
	if err != nil {
		return nil, "", err
	}
	base, full := newest, req.Full
	if req.Base != "" {
		if base, err = certificatePath(f.root, req.Base); err != nil {
			return nil, "", err
		}
	}
	if base == "" {
		full = true
	}
	findings, named := req.Findings, req.namesChange()
	// A base certificate gone from the folder, as after a change of branch, is a
	// changed certificate like any other: a plain checklist refuses.
	if stored != nil && !req.namesChange() {
		if changed := f.changedOwn(stored); len(changed) > 0 {
			// The skill's call is the owner's step, so it begins the work
			// again, from the newest certificate and with the Findings and the
			// whole spec the owner named; an agent's plain checklist is refused.
			if !req.Start {
				return nil, restartText(changed), nil
			}
			full, findings, named = full || stored.Full, stored.Findings, stored.Named
		} else if stored.Named || stored.Base == base {
			return stored, "", nil
		}
	}
	if stored != nil && req.namesChange() && len(f.changedOwn(stored)) == 0 && stored.Named && stored.Base == base && stored.Full == full && sameFindings(stored.Findings, req.Findings) {
		return stored, "", nil
	}
	return &Work{Project: f.root, Base: base, Full: full, Findings: findings, Named: named, Spec: f.specHashes(), Certificates: f.certs, Answers: map[string]Given{}, Began: e.now()}, "", nil
}

// restartText says that what the work may not change changed under it. An agent
// reads it too, so it names no command that would begin the work again: that is
// the owner's step.
func restartText(changed []string) string {
	var b strings.Builder
	b.WriteString("These changed since the work began, and the work may not change them:\n\n")
	for _, c := range changed {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	b.WriteString("\nPut them back: a change to the spec is its own step, before the work. If the owner changed them on purpose, stop and say so; only the owner begins the work again.\n")
	return b.String()
}

func sameFindings(a, b []analysis.Finding) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// list rebuilds the list of the work for the folder's state, and returns the
// impact and the project at the base.
func (e *Engine) list(f *folder, w *Work) (analysis.Impact, *project.Project, error) {
	before, changed := f.cur, []analysis.Change(nil)
	if w.Base != "" {
		cert, err := ReadCertificate(inProject(f.root, w.Base))
		if err != nil {
			return analysis.Impact{}, nil, err
		}
		if before, err = cert.Project(); err != nil {
			return analysis.Impact{}, nil, err
		}
		changed = cert.Changed(f.hashes)
	}
	imp := analysis.ComputeImpact(analysis.ImpactInput{Base: before, Current: f.cur, Changed: changed, Findings: w.Findings, Full: w.Full})
	w.Items = mergeItems(w.Items, checklist(imp, before, f.cur, w.Findings))
	w.Notes = imp.Notes
	return imp, before, nil
}

// Checklist prints the task and the list. In a folder with an open Run it is the
// Run's; otherwise the list of the change to the folder, which answer and gate then
// work on. ok is false when there is no list, because the spec has errors.
func (e *Engine) Checklist(ctx context.Context, req ChecklistRequest) (text string, ok bool, err error) {
	open, err := e.openRun(ctx, req.Dir)
	if err != nil {
		return "", false, err
	}
	if open != nil {
		if req.Start {
			return fmt.Sprintf("Run %s is open in this folder, so no work begins beside it. markitect %s --skill says how to go on with it, and markitect stop \"why\" ends it.\n", open.Run, open.Command), false, nil
		}
		if req.namesChange() {
			return "", false, fmt.Errorf("%w: run %s is open in this folder and has its own list; --base, --full and --findings are for work without a run", ErrUsage, open.Run)
		}
		text, err := e.runChecklist(ctx, req.Dir, req.Review)
		return text, err == nil, err
	}
	f, err := e.readFolder(req.Dir)
	if err != nil {
		return "", false, err
	}
	if req.Start && !harnessOnDisk(f.root) {
		return noGuard, false, nil
	}
	if errs := f.cur.SpecErrors(); len(errs) > 0 {
		return "The spec has errors, so there is no list. Fix them in the spec first:\n\n" + listErrors(errs), false, nil
	}
	var w Work
	var imp analysis.Impact
	var before *project.Project
	var restarted []string
	refusal := ""
	if err := e.updateWork(f.root, func(stored *Work) (*Work, error) {
		if stored == nil && req.Review && !req.namesChange() && !req.Start {
			// A reviewer is started for work under way; without one there is
			// nothing to review.
			refusal = "No work is under way in this folder, so there is nothing to review. End now, and say so.\n"
			return nil, nil
		}
		if req.Start && stored != nil && !req.namesChange() {
			restarted = f.changedOwn(stored)
		}
		next, why, err := e.workFor(f, req, stored)
		if err != nil {
			return nil, err
		}
		if why != "" {
			refusal = why
			return stored, nil
		}
		if imp, before, err = e.list(f, next); err != nil {
			return nil, err
		}
		w = *next
		if req.Start && req.Review && len(next.Items) == 0 {
			// A review that finds nothing to review leaves no work behind, which
			// would otherwise hold the next change to its base.
			return nil, nil
		}
		return next, nil
	}); err != nil {
		return "", false, err
	}
	if refusal != "" {
		return refusal, false, nil
	}
	if req.Start {
		return startStep(w, f, req.Review, restarted), true, nil
	}
	changes := changedText(imp, before)
	if !req.Review {
		var b strings.Builder
		b.WriteString(stateTask)
		if wrong := sentBack(w); len(wrong) > 0 {
			b.WriteString("\nThe last review found these wrong. Repair them, unless you already have; a reviewer looks at them again:\n\n")
			for _, s := range wrong {
				fmt.Fprintf(&b, "- %s\n", s)
			}
		}
		fmt.Fprintf(&b, "\n%s This list says what the change reaches and where the registry says to look. It is your to-do list, and you answer nothing on it:\n\n", baseLine(w))
		b.WriteString(renderList(w.Items))
		b.WriteString(changes)
		return b.String(), true, nil
	}
	if len(w.Items) == 0 {
		return "Nothing to review: the change reaches nothing in the spec. End now.\n", true, nil
	}
	valid, _ := w.standing(f)
	var todo []Item
	var earlier []string
	for _, it := range w.Items {
		if _, ok := valid[it.Key]; ok {
			continue
		}
		todo = append(todo, it)
		if a, ok := w.Answers[it.Key]; ok && a.Word == AnswerWrong {
			earlier = append(earlier, fmt.Sprintf("%s: %s", it.Key, a.Explanation))
		}
	}
	if len(todo) == 0 {
		return "Every item has an answer for the files as they are now, so nothing is left to review. End now.\n", true, nil
	}
	return stateReviewPrompt(baseLine(w), earlier, todo, changes), true, nil
}

// noGuard is what --start says in a folder without the harness, as a Run refuses
// to start without it.
const noGuard = "The harness is not complete in this project: the work needs the builder and reviewer agents, and the hook in " + SettingsFile + " that keeps each agent to its part. Run markitect update, which brings them in line, and try again; if it says it left the settings as they are, add the hook it names by hand.\n"

// harnessOnDisk says whether the folder holds both agents and the hook that
// guards answer and checklist.
func harnessOnDisk(root string) bool {
	for _, file := range []string{reviewerFile, builderFile, SettingsFile} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil || (file == SettingsFile && !HasGuard(data)) {
			return false
		}
	}
	return true
}

// sentBack lists the items whose latest answer is wrong, with what the reviewer
// found, for the builder that repairs them.
func sentBack(w Work) []string {
	var out []string
	for _, it := range w.Items {
		if a, ok := w.Answers[it.Key]; ok && a.Word == AnswerWrong {
			out = append(out, fmt.Sprintf("%s: %s", it.Key, a.Explanation))
		}
	}
	return out
}

// startStep tells the session that drives the work which agent to start. The
// builder goes through gate itself, which has it start the reviewer; a review of
// finished work needs only the reviewer. Either way the driver runs gate when the
// agent has ended and does what it prints, so the process stays in the tool. Work
// whose review is complete, or that gate has stopped, goes straight to gate.
// restarted names what changed under the work that this call began again.
func startStep(w Work, f *folder, review bool, restarted []string) string {
	model := f.cur.Manifest.Model
	var b strings.Builder
	if len(restarted) > 0 {
		fmt.Fprintf(&b, "These changed since the work under way began, so it begins again: %s.\n\n", strings.Join(restarted, ", "))
	}
	fmt.Fprintf(&b, "%s The list has %d %s.\n\n", baseLine(w), len(w.Items), plural(len(w.Items), "item", "items"))
	_, missing := w.standing(f)
	rounds, _ := w.rounds()
	stopped := len(w.specProblems()) > 0 || rounds > repairRounds
	switch {
	case review && len(w.Items) == 0:
		b.WriteString("Nothing to review: the change reaches nothing in the spec, so there is no agent to start and nothing to commit.\n")
	case review && len(missing) == 0:
		b.WriteString("Every item has an answer for the files as they are now. Run markitect gate --findings and do what it prints.\n")
	case review:
		fmt.Fprintf(&b, "Start the %s agent%s with exactly this message:\n\n    Review the work.\n\nIt takes its task from markitect checklist --review. When it has ended, run markitect gate --findings and do what it prints.\n", ReviewerAgent, onModel(model))
	case len(w.Items) == 0:
		b.WriteString("Nothing on the list to build. Run markitect gate and do what it prints.\n")
	case len(missing) == 0 || stopped:
		b.WriteString("The work under way has its review. Run markitect gate and do what it prints.\n")
	default:
		fmt.Fprintf(&b, "Start the %s agent%s with exactly this message:\n\n    Build the work.\n\nIt takes its task from markitect checklist and goes through markitect gate, which has it start the reviewer. When it has ended, run markitect gate and do what it prints.\n", BuilderAgent, onModel(model))
	}
	return b.String()
}

// baseLine says where the change starts.
func baseLine(w Work) string {
	switch {
	case w.Base == "" && w.Full:
		return "There is no certificate yet, so every Definition is on the list."
	case w.Base == "":
		return "The list holds the Findings."
	case w.Full:
		return fmt.Sprintf("The base is the certificate %s, and every Definition is on the list.", w.Base)
	}
	return fmt.Sprintf("The base is the certificate %s: `markitect impact --base %s` shows what changed since.", w.Base, w.Base)
}

// changedText shows what changed in the spec since the base: each changed or
// removed Definition's text as the certificate has it.
func changedText(imp analysis.Impact, before *project.Project) string {
	var b strings.Builder
	for _, c := range imp.Definitions {
		if c.Change == analysis.Added || before == nil {
			continue
		}
		d := before.Model.Definitions[c.Path]
		if d == nil || d.Library != "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("\n## Definitions before the change\n\nWhat the base certificate holds of each Definition that changed or is gone; the spec in the folder holds what it says now.\n")
		}
		fmt.Fprintf(&b, "\n### %s (%s)\n\n````markdown\n%s````\n", c.Path, c.Change, strings.TrimRight(string(d.Data), "\n")+"\n")
	}
	return b.String()
}

// accounted is the folder's files to check pointers against: a file the walk did
// not find does not exist for an answer.
type accounted struct {
	repo project.Repository
	set  map[string]bool
}

func (a accounted) Files() ([]string, error) { return a.repo.Files() }

func (a accounted) Read(file string) ([]byte, error) {
	if !a.set[file] {
		return nil, fs.ErrNotExist
	}
	return a.repo.Read(file)
}

func (a accounted) GitIgnored(file string) bool { return a.repo.GitIgnored(file) }

func (f *folder) accounted() accounted {
	set := map[string]bool{}
	files, _ := f.files.Files()
	for _, file := range files {
		set[file] = true
	}
	return accounted{repo: f.files, set: set}
}

// noWork is what answer and gate say when no checklist began the work.
const noWork = "No checklist began this work. Run markitect checklist, which says what to do, and then markitect gate.\n"

// answerState stores a reviewer's answers for the folder as it is. Each location
// must exist in the folder's current files, and each answer is bound to the files
// its item names and the files it points to.
func (e *Engine) answerState(dir string, raw []byte) (string, bool, error) {
	f, err := e.readFolder(dir)
	if err != nil {
		return "", false, err
	}
	if errs := f.cur.SpecErrors(); len(errs) > 0 {
		return "The spec has errors, so no answer is stored. End now, and say so:\n\n" + listErrors(errs), false, nil
	}
	var problems, missing []string
	none := false
	if err := e.updateWork(f.root, func(w *Work) (*Work, error) {
		if w == nil {
			none = true
			return nil, nil
		}
		if _, _, err := e.list(f, w); err != nil {
			return nil, err
		}
		answers, refused := parseAnswers(raw, w.Items)
		checked, links := checkAnswers(f.accounted(), w.Items, answers)
		if problems = append(refused, checked...); len(problems) > 0 {
			return w, nil
		}
		byKey := map[string]Item{}
		for _, it := range w.Items {
			byKey[it.Key] = it
		}
		for k, a := range answers {
			w.Answers[k] = given(byKey[k], a, f)
		}
		w.MissingFor = dedupe(append(w.MissingFor, links...))
		var valid map[string]Given
		valid, missing = w.standing(f)
		if len(missing) == 0 {
			w.review(f.fingerprint, valid)
		}
		return w, nil
	}); err != nil {
		return "", false, err
	}
	switch {
	case none:
		return "No work is under way in this folder, so nothing is stored. End now, and say so.\n", false, nil
	case len(problems) > 0:
		var b strings.Builder
		b.WriteString("These answers are not accepted, and nothing from this call is stored. Correct them and call markitect answer again:\n\n")
		for _, p := range problems {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		return b.String(), false, nil
	case len(missing) > 0:
		return fmt.Sprintf("Stored. %d items still have no answer:\n\n- %s\n", len(missing), strings.Join(missing, "\n- ")), false, nil
	}
	return "Stored: every item has an answer for the files as they are now. You are done; report in one line and end.\n", true, nil
}

// stateStatus says how the work in the folder stands without a Run, and changes
// nothing.
func (e *Engine) stateStatus(dir string) string {
	var b strings.Builder
	b.WriteString("\nWithout a run, in this folder:\n")
	f, err := e.readFolder(dir)
	if err != nil {
		fmt.Fprintf(&b, "- The folder could not be read: %v\n", err)
		return b.String()
	}
	w, err := e.loadWork(f.root)
	switch {
	case err != nil:
		fmt.Fprintf(&b, "- The work could not be read: %v\n", err)
	case w == nil:
		b.WriteString("- No work is under way; markitect checklist begins it.\n")
	case len(f.cur.SpecErrors()) > 0:
		b.WriteString("- The spec has errors, so there is no list; markitect check lists them.\n")
	default:
		if changed := f.changedOwn(w); len(changed) > 0 {
			fmt.Fprintf(&b, "- What the work may not change changed since it began: %s.\n", strings.Join(changed, ", "))
		}
		if _, _, err := e.list(f, w); err != nil {
			fmt.Fprintf(&b, "- The list could not be made: %v\n", err)
			break
		}
		valid, missing := w.standing(f)
		wrong, _ := verdictsFor(w.Items, func(key string) Answer { return valid[key].Answer }, settleLean)
		rounds, _ := w.rounds()
		fmt.Fprintf(&b, "- %s The list has %d %s: %d with an answer for the files as they are now, %d without, %d wrong. Repair rounds: %d of %d.\n",
			baseLine(*w), len(w.Items), plural(len(w.Items), "item", "items"), len(valid), len(missing), len(wrong), rounds, repairRounds)
		for _, d := range w.specProblems() {
			fmt.Fprintf(&b, "- A spec problem waits for the owner: %s\n", oneLine(d.Matter))
		}
	}
	newest, cert, err := newestCertificate(f.root)
	switch {
	case err != nil:
		fmt.Fprintf(&b, "- The certificates could not be read: %v\n", err)
	case cert == nil:
		b.WriteString("- No certificate yet, so the first list covers every Definition.\n")
	default:
		diff := cert.Differences(f.hashes)
		if len(diff) == 0 {
			fmt.Fprintf(&b, "- The folder matches the newest certificate, %s.\n", newest)
		} else {
			fmt.Fprintf(&b, "- %d %s differ from the newest certificate, %s: %s.\n", len(diff), plural(len(diff), "file", "files"), newest, strings.Join(clipList(diff, 10), ", "))
		}
	}
	return b.String()
}

// clipList keeps the first n entries, and says how many more there are.
func clipList(list []string, n int) []string {
	if len(list) <= n {
		return list
	}
	return append(append([]string(nil), list[:n]...), fmt.Sprintf("and %d more", len(list)-n))
}

// stateTask is the builder's task without a Run, which checklist prints each time,
// so that compaction cannot lose it.
const stateTask = fixedPart + `
You build in this folder, and nobody can answer you.

Make this project match the spec. Start with the list below.

You are done when markitect gate passes. Run it when you think the work is done, and do what it says each time.

Update the registry when files change: every file is listed on the page of its owning Package with the Definitions it realizes and a short reason, or it is ignored. When a Definition is gone, its files go or get a new justified basis.

Do not change the spec, the manifest, the lock or anything else under .markitect/ except the registry pages, and do not change AGENTS.md, CLAUDE.md or the installed Markitect skills and agents. Change only what the list asks for: gate refuses a changed file that lies in no item.

If the spec contradicts itself or leaves open something that changes behaviour at the project's boundary, stop and say so in your report: the owner decides. A gap with no such effect is yours to decide. If something the project needs is missing from this machine, such as a program or a service, stop and say so.
`

// stateReviewPrompt is what checklist --review prints without a Run.
func stateReviewPrompt(base string, earlier []string, items []Item, changes string) string {
	var b strings.Builder
	b.WriteString(fixedPart)
	fmt.Fprintf(&b, "\nThis change is under review, and you owe it no benefit of the doubt. %s Each item names the files the registry lists for it.\n\n", base)
	if len(earlier) > 0 {
		b.WriteString("An earlier review of this work found these, and they were sent back:\n\n")
		for _, e := range earlier {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		b.WriteString("\nThe other answers stand. Answer only the items below.\n\n")
	}
	b.WriteString("## Checklist\n\n")
	b.WriteString(renderList(items))
	b.WriteString(answerFormat)
	b.WriteString(changes)
	return b.String()
}
