package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// commandGate and commandGateFindings are the Command of gate's Results: the check
// of the work, and the report of a review of finished work.
const (
	commandGate         = "gate"
	commandGateFindings = "gate --findings"
)

// GateRequest is one call of gate.
type GateRequest struct {
	Dir string
	// Verify only checks that the folder matches its newest certificate.
	Verify bool
	// Findings closes a review of finished work: the Findings go into the folder,
	// and no certificate.
	Findings bool
}

// Gate checks the work in the project folder without a Run, as submit does in
// one, minus what a Run needs, and without Git (DEC-097). It does the first thing
// that applies, and each refusal says the one next step. Done means the work
// passed, the folder matches its certificate, or the report of a review is
// written.
func (e *Engine) Gate(ctx context.Context, req GateRequest) (Verdict, error) {
	if req.Verify && req.Findings {
		return Verdict{}, fmt.Errorf("%w: gate takes --verify or --findings, not both", ErrUsage)
	}
	if open, err := e.openRun(ctx, req.Dir); err != nil {
		return Verdict{}, err
	} else if open != nil {
		return Verdict{Text: fmt.Sprintf("Run %s is open in this folder, and gate does not check a run's work. Go on with it with markitect submit, or end it with markitect stop \"why\".\n", open.Run)}, nil
	}
	f, err := e.readFolder(req.Dir)
	if err != nil {
		return Verdict{}, err
	}
	if req.Verify {
		return verifyFolder(f)
	}
	if req.Findings {
		return e.gateFindings(f)
	}
	w, err := e.loadWork(f.root)
	if err != nil {
		return Verdict{}, err
	}
	if w == nil {
		// Without work under way, a folder that matches its newest certificate
		// has nothing to check.
		if name, cert, err := newestCertificate(f.root); err != nil {
			return Verdict{}, err
		} else if cert != nil && len(cert.Differences(f.hashes)) == 0 {
			return Verdict{Done: true, Text: fmt.Sprintf("Nothing to check: the folder matches its newest certificate, %s.\n", name)}, nil
		}
		return Verdict{Text: noWork}, nil
	}
	// Row 1: what the work may not change.
	if changed := f.changedOwn(w); len(changed) > 0 {
		return Verdict{Text: restartText(changed)}, nil
	}
	var links []string
	for _, file := range sortedKeys(f.links) {
		if leavesTree(file, f.links[file]) {
			links = append(links, fmt.Sprintf("%s is a link that leaves the folder, to %s", file, f.links[file]))
		}
	}
	if len(links) > 0 {
		return listed("A link may not leave the folder. Remove these, then run markitect gate again:", links), nil
	}
	// Row 3 comes first when the spec itself has errors: there is no list then.
	if len(f.cur.SpecErrors()) > 0 {
		return listed("markitect check fails on the work. Fix it, then run markitect gate again:", errorLines(f.cur)), nil
	}
	w, valid, missing, err := e.reviewState(f)
	if err != nil {
		return Verdict{}, err
	}
	// Row 2: a changed file that lies in no item.
	if left, err := outsideItems(f, w); err != nil {
		return Verdict{}, err
	} else if len(left) > 0 {
		return listed(fmt.Sprintf("These files changed since the certificate %s and lie in no item of the list, so nobody reviews them. List them on the registry, so that their Definitions are reviewed, or put them back; then run markitect gate again:", w.Base), left), nil
	}
	// Row 3: check, of the spec and the realization.
	if len(f.cur.Errors) > 0 {
		return listed("markitect check fails on the work. Fix it, then run markitect gate again:", errorLines(f.cur)), nil
	}
	// Row 5: a spec problem of any finished review stands until the spec changes.
	if decisions := w.specProblems(); len(decisions) > 0 {
		file, err := e.writeResult(f, w, valid, StateSpecProblem, "a reviewer found a spec problem: the spec needs the owner", decisions)
		if err != nil {
			return Verdict{}, err
		}
		var b strings.Builder
		b.WriteString("A reviewer found a spec problem, which only the owner can settle. Stop, and give the owner this:\n\n")
		for _, d := range decisions {
			fmt.Fprintf(&b, "- %s\n", oneLine(d.Matter))
		}
		fmt.Fprintf(&b, "\nThe Result is %s. Once the owner has settled the spec, the owner begins the work again.\n", file)
		return Verdict{Text: b.String()}, nil
	}
	// Row 6, the end: the repair rounds ran out.
	rounds, sent := w.rounds()
	if rounds > repairRounds {
		file, err := e.writeResult(f, w, valid, StateRefused, "the repair rounds ran out, and these stay open: "+sent[len(sent)-1], nil)
		if err != nil {
			return Verdict{}, err
		}
		return Verdict{Text: fmt.Sprintf("The reviewer still found something wrong after %d repair rounds, the most the work gets. Stop, and give the owner the Result %s. A change to the spec begins the work again.\n", repairRounds, file)}, nil
	}
	// Row 4: items without an answer for the files as they are now.
	if len(missing) > 0 {
		return Verdict{Text: reviewNext(len(missing), f.cur.Manifest.Model, "markitect gate")}, nil
	}
	// Row 6: wrong answers go back, one round per reviewed state.
	if wrong, _ := verdictsFor(w.Items, func(key string) Answer { return valid[key].Answer }, settleLean); len(wrong) > 0 {
		return listed(fmt.Sprintf("The reviewer found these wrong (repair round %d of %d). Repair them, then run markitect gate again:", rounds, repairRounds), wrong), nil
	}
	// Row 7: nothing left. The certificate goes into the folder, and the work is
	// done; when nothing changed and the list is empty, there is nothing to
	// certify again.
	if len(w.Items) == 0 && w.Base != "" {
		if base, err := ReadCertificate(inProject(f.root, w.Base)); err == nil && len(base.Differences(f.hashes)) == 0 {
			if err := e.updateWork(f.root, func(*Work) (*Work, error) { return nil, nil }); err != nil {
				return Verdict{}, err
			}
			return Verdict{Done: true, Text: fmt.Sprintf("Nothing to check: the folder matches the certificate %s, and the list is empty.\n", w.Base)}, nil
		}
	}
	cert, err := newCertificate(f.files, f.hashes, f.cur, e.now())
	if err != nil {
		return Verdict{}, err
	}
	cert.Base, cert.Full = w.Base, w.Full
	stamp := gateStamp(e.now(), f.fingerprint)
	certFile := path.Join(project.ResultsDir, stamp+".json")
	r := e.gateResult(w, valid, stamp, StatePassed, "every item has an answer for the files as they are now, and none is wrong", nil)
	r.Certificate = certFile
	cert.Items = r.Items
	data, err := json.MarshalIndent(cert, "", "  ")
	if err != nil {
		return Verdict{}, err
	}
	if err := writeFolder(f.root, certFile, append(data, '\n')); err != nil {
		return Verdict{}, err
	}
	page := strings.TrimSuffix(certFile, ".json") + ".md"
	if err := writeFolder(f.root, page, Page(r)); err != nil {
		return Verdict{}, err
	}
	if err := e.updateWork(f.root, func(*Work) (*Work, error) { return nil, nil }); err != nil {
		return Verdict{}, err
	}
	return Verdict{Done: true, Text: fmt.Sprintf("The work passes: every item has an answer for the files as they are now, and none is wrong. Its certificate is %s, with the Result %s. Commit the work together with both, in one commit.\n", certFile, page)}, nil
}

// reviewState rebuilds the list, reads the answers that stand on the folder's
// state, and records the state as reviewed when every item has one.
func (e *Engine) reviewState(f *folder) (*Work, map[string]Given, []string, error) {
	var w Work
	var valid map[string]Given
	var missing []string
	err := e.updateWork(f.root, func(stored *Work) (*Work, error) {
		if stored == nil {
			return nil, fmt.Errorf("%w: the work of %s went while gate ran", ErrUsage, f.root)
		}
		if _, _, err := e.list(f, stored); err != nil {
			return nil, err
		}
		valid, missing = stored.standing(f)
		if len(missing) == 0 {
			stored.review(f.fingerprint, valid)
		}
		w = *stored
		return stored, nil
	})
	return &w, valid, missing, err
}

// outsideItems lists the files that changed since the base certificate and lie in
// no item of the list. Without a certificate every Definition is on the list, and
// nothing is compared.
func outsideItems(f *folder, w *Work) ([]string, error) {
	if w.Base == "" {
		return nil, nil
	}
	cert, err := ReadCertificate(inProject(f.root, w.Base))
	if err != nil {
		return nil, err
	}
	in := map[string]bool{}
	for _, it := range w.Items {
		for _, file := range it.Look {
			in[file] = true
		}
	}
	var out []string
	for _, c := range cert.Changed(f.hashes) {
		// The owner's files were checked against the start of the work (row 1).
		if !in[c.Path] && !ownersFile(c.Path) {
			out = append(out, fmt.Sprintf("%s (%s)", c.Path, c.Change))
		}
	}
	return out, nil
}

// verifyFolder checks that the folder matches its newest certificate.
func verifyFolder(f *folder) (Verdict, error) {
	name, cert, err := newestCertificate(f.root)
	if err != nil {
		return Verdict{}, err
	}
	if cert == nil {
		return Verdict{Text: fmt.Sprintf("No certificate in %s, so nothing is certified here.\n", project.ResultsDir)}, nil
	}
	if diff := cert.Differences(f.hashes); len(diff) > 0 {
		return listed(fmt.Sprintf("These files do not match the newest certificate, %s: they changed without a gate that passed.", name), clipList(diff, maxRefusals)), nil
	}
	return Verdict{Done: true, Text: fmt.Sprintf("The folder matches its newest certificate, %s.\n", name)}, nil
}

// gateFindings closes a review of finished work: the Findings page and the Result
// go into the folder, and no certificate.
func (e *Engine) gateFindings(f *folder) (Verdict, error) {
	if errs := f.cur.SpecErrors(); len(errs) > 0 {
		return Verdict{Text: "The spec has errors, so there is nothing to review. Fix them in the spec first:\n\n" + listErrors(errs)}, nil
	}
	if w, err := e.loadWork(f.root); err != nil {
		return Verdict{}, err
	} else if w == nil {
		return Verdict{Text: "No checklist began this review. Run markitect checklist --base CERTIFICATE --review, have the reviewer answer, then run markitect gate --findings.\n"}, nil
	}
	w, valid, missing, err := e.reviewState(f)
	if err != nil {
		return Verdict{}, err
	}
	if len(missing) > 0 {
		return Verdict{Text: reviewNext(len(missing), f.cur.Manifest.Model, "markitect gate --findings")}, nil
	}
	rep := verifyReport(w.Items, func(key string) Answer { return valid[key].Answer }, f.cur)
	stamp := gateStamp(e.now(), f.fingerprint)
	r := e.gateResult(w, valid, stamp, rep.state, rep.why, rep.decisions)
	r.Command, r.Findings, r.Unsettled = commandGateFindings, rep.findings, rep.unsettled
	file := path.Join(project.ResultsDir, stamp+"-verify.md")
	if err := writeFolder(f.root, file, Page(r)); err != nil {
		return Verdict{}, err
	}
	// The review is over.
	if err := e.updateWork(f.root, func(*Work) (*Work, error) { return nil, nil }); err != nil {
		return Verdict{}, err
	}
	if len(rep.findings) == 0 {
		return Verdict{Done: true, Text: fmt.Sprintf("The review ended as %s: %s. The Result is %s, and there is no certificate.\n", rep.state, rep.why, file)}, nil
	}
	findings := path.Join(project.ResultsDir, stamp+"-findings.md")
	if err := writeFolder(f.root, findings, FindingsPage(r)); err != nil {
		return Verdict{}, err
	}
	return Verdict{Done: true, Text: fmt.Sprintf("The review ended as %s: %s. The Findings are in %s and the Result is %s; there is no certificate. To repair them, begin the work with: markitect checklist --findings %s\n", rep.state, rep.why, findings, file, findings)}, nil
}

// writeResult writes gate's Result page for a refusal into the folder, and says
// where.
func (e *Engine) writeResult(f *folder, w *Work, valid map[string]Given, state, why string, decisions []OwnerDecision) (string, error) {
	// One page per state and outcome: calling gate again rewrites it.
	stamp := gatePrefix + f.fingerprint[:8] + "-" + state
	file := path.Join(project.ResultsDir, stamp+".md")
	return file, writeFolder(f.root, file, Page(e.gateResult(w, valid, stamp, state, why, decisions)))
}

// gateResult is gate's Result as it stands. The tool writes it from what happened,
// never an agent; stamp is the name of its pages.
func (e *Engine) gateResult(w *Work, valid map[string]Given, stamp, state, why string, decisions []OwnerDecision) *Result {
	rounds, sent := w.rounds()
	r := &Result{Format: ResultFormat, Run: strings.TrimPrefix(stamp, gatePrefix), Command: commandGate, State: state, Why: why,
		Base: w.Base, Full: w.Full, OwnerDecisions: decisions, Repairs: rounds, SentBack: sent, MissingLinks: w.MissingFor, ImpactNotes: w.Notes,
		Began: w.Began, Ended: e.now()}
	if r.Base == "" {
		r.Base = "none"
	}
	for _, it := range w.Items {
		a := valid[it.Key]
		r.Items = append(r.Items, ResultItem{Item: it.Key, Kind: it.Kind, Definition: it.Definition, Look: it.Look, Answer: a.Word, Pointers: a.Pointers, Explanation: a.Explanation})
	}
	sort.SliceStable(r.Items, func(i, j int) bool { return r.Items[i].Item < r.Items[j].Item })
	return r
}

// writeFolder writes a file of the project folder.
func writeFolder(root, file string, data []byte) error {
	p := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// reviewNext says to start the reviewer, and what to run when it has ended.
func reviewNext(missing int, model, again string) string {
	return fmt.Sprintf(`%d %s no answer for the files as they are now. Start the %s agent%s with exactly this message:

    Review the work.

It takes its task from markitect checklist --review. When it has ended, run %s again.
`, missing, plural(missing, "item has", "items have"), ReviewerAgent, onModel(model), again)
}

// listed is a refusal with its lines.
func listed(head string, lines []string) Verdict {
	var b strings.Builder
	b.WriteString(head)
	b.WriteString("\n\n")
	for _, l := range lines {
		fmt.Fprintf(&b, "- %s\n", l)
	}
	return Verdict{Text: b.String()}
}
