package cockpit

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The Orchestrator words a wait in a few fixed forms (run.computeState and the
// release conditions). Matching them only reads what the wait names; a text in
// no known form keeps its wording and names nothing.
var (
	waitDecision   = regexp.MustCompile(`^the decision of (Root \(\.\)|Manager (.+)|the Orchestrator) on (P\d+)$`)
	waitDependency = regexp.MustCompile(`^the accepted (basis|result) of (Root \(\.\)|Manager (.+?)) \((.*)\)$`)
	waitParent     = regexp.MustCompile(`^the accepted basis of (Root \(\.\)|Manager (.+))$`)
	waitQuestion   = regexp.MustCompile(`^Q\d+$`)
)

const maxCauseDepth = 6

// classifyWaits reads every Manager's wait and follows its cause through the
// same view.
func classifyWaits(v *View) {
	byPath := map[string]*ManagerView{}
	for i := range v.Managers {
		byPath[v.Managers[i].Path] = &v.Managers[i]
	}
	questions := map[string]QuestionView{}
	for _, q := range v.Questions {
		questions[q.ID] = q
	}
	proposals := map[string]ProposalView{}
	for _, pr := range v.Proposals {
		proposals[pr.ID] = pr
	}
	for i := range v.Managers {
		if w := v.Managers[i].Wait; w != nil {
			classify(w, &v.Managers[i], byPath, questions)
			outdated(w, &v.Managers[i], byPath, questions, proposals)
		}
	}
	for i := range v.Managers {
		if w := v.Managers[i].Wait; w != nil {
			w.Cause = cause(v.Managers[i].Path, byPath, map[string]bool{}, 0)
		}
	}
}

func classify(w *Wait, m *ManagerView, byPath map[string]*ManagerView, questions map[string]QuestionView) {
	node := func(label, path string) string {
		if label == "Root (.)" {
			return "."
		}
		if label == "the Orchestrator" {
			return nodeOrchestrator
		}
		return path
	}
	switch t := w.Text; {
	case t == "":
		w.Kind = "unnamed"
	case t == "the results of its children":
		w.Kind = "children"
		for _, c := range m.Children {
			if child := byPath[c]; child == nil || child.State != string(run.Done) {
				w.On = append(w.On, c)
			}
		}
	case t == "the results of its helpers":
		w.Kind = "helpers"
	case waitDecision.MatchString(t):
		s := waitDecision.FindStringSubmatch(t)
		w.Kind, w.On, w.Refs = "decision", []string{node(s[1], s[2])}, []string{"proposal:" + s[3]}
	case waitDependency.MatchString(t):
		s := waitDependency.FindStringSubmatch(t)
		w.Kind, w.On, w.Need = "dependency", []string{node(s[2], s[3])}, s[4]
	case waitParent.MatchString(t):
		s := waitParent.FindStringSubmatch(t)
		w.Kind, w.On = "parent-basis", []string{node(s[1], s[2])}
	case waitQuestion.MatchString(t):
		w.Kind, w.Refs = "answer", []string{"question:" + t}
		if q, ok := questions[t]; ok {
			w.On = []string{participant(q.To)}
		}
	case t == run.User:
		w.Kind, w.On = "contact", []string{nodeUser}
	case byPath[t] != nil:
		w.Kind, w.On = "contact", []string{t}
	default:
		w.Kind = "unrecognized"
	}
}

// outdated marks a recorded wait whose named condition the record shows
// fulfilled after the wait began. The Orchestrator records a state change but
// not a changed wait text while a Manager stays Waiting, so such a wait is the
// last wording, not the current reason. What still blocks it is then read from
// the record only: the unaccepted dependencies of its accepted plan.
func outdated(w *Wait, m *ManagerView, byPath map[string]*ManagerView, questions map[string]QuestionView, proposals map[string]ProposalView) {
	accepted := func(owner, stage string) int {
		o := byPath[owner]
		if o == nil {
			return 0
		}
		id := o.Basis
		if stage == "result" {
			id = o.Result
		}
		return proposals[id].DecSeq
	}
	switch w.Kind {
	case "parent-basis":
		if m.Released && m.ReleasedSeq > w.Since {
			w.Outdated = fmt.Sprintf("%s was released at record %d once its parent's basis was accepted.", labelOf(m.Path), m.ReleasedSeq)
		}
	case "dependency":
		stage := "basis"
		if strings.HasPrefix(w.Text, "the accepted result") {
			stage = "result"
		}
		if seq := accepted(w.On[0], stage); seq > w.Since {
			w.Outdated = fmt.Sprintf("The %s of %s it names was accepted at record %d.", stage, nodeLabel(w.On[0]), seq)
		}
	case "decision":
		if pr, ok := proposals[strings.TrimPrefix(w.Refs[0], "proposal:")]; ok && pr.DecSeq > w.Since {
			w.Outdated = fmt.Sprintf("%s was %s at record %d.", pr.ID, pr.Status, pr.DecSeq)
		}
	case "answer":
		if q, ok := questions[w.Text]; ok && q.Status != "open" {
			seq := q.AnsweredSeq
			if q.Status == "lapsed" {
				seq = q.LapsedSeq
			}
			if seq > w.Since {
				w.Outdated = fmt.Sprintf("%s was %s at record %d.", q.ID, q.Status, seq)
			}
		}
	}
	if w.Outdated == "" {
		return
	}
	w.On = nil
	for _, d := range m.Declared {
		if !d.Accepted && !slices.Contains(w.On, d.Owner) {
			w.On = append(w.On, d.Owner)
		}
	}
}

func participant(path string) string {
	if path == run.User {
		return nodeUser
	}
	return path
}

// cause follows what a waiting Manager waits on, as far as those participants
// wait themselves. A participant already on the path is marked as a repeat
// and not followed again.
func cause(path string, byPath map[string]*ManagerView, seen map[string]bool, depth int) []Step {
	m := byPath[path]
	if m == nil || m.Wait == nil || depth >= maxCauseDepth {
		return nil
	}
	seen[path] = true
	defer delete(seen, path)
	var out []Step
	for _, on := range m.Wait.On {
		step := Step{Node: on, Depth: depth}
		switch on {
		case nodeOrchestrator:
			step.Doing = "decides on a Root submission once its technical check passed"
		case nodeUser:
			step.Doing = "receives Root's question through the owner channel"
		default:
			n := byPath[on]
			if n == nil {
				step.Doing = "is not part of the recorded organization"
				break
			}
			step.State, step.Doing = n.State, doing(n)
			if n.Wait != nil {
				step.Wait = n.Wait.Text
			}
			if seen[on] {
				step.Repeat = true
				out = append(out, step)
				continue
			}
			out = append(out, step)
			out = append(out, cause(on, byPath, seen, depth+1)...)
			continue
		}
		out = append(out, step)
	}
	return out
}

// doing says in a few words what a Manager does according to the record.
func doing(m *ManagerView) string {
	switch {
	case m.Turn != nil:
		access := "writes"
		if m.Turn.ReadOnly {
			access = "read-only"
		}
		return fmt.Sprintf("%s turn %s is running (%s)", m.Turn.Role, m.Turn.Short, access)
	case m.Checking:
		return "its submission is being checked"
	case m.State == string(run.Ready):
		return "is ready for its next turn"
	case m.State == string(run.Done):
		return "is done in this Pass"
	case m.State == string(run.Waiting) && m.Wait != nil && m.Wait.Outdated != "":
		if len(m.Wait.On) == 0 {
			return "waits; the record shows no current reason"
		}
		names := make([]string, len(m.Wait.On))
		for i, on := range m.Wait.On {
			names[i] = labelOf(on)
		}
		return "waits; its accepted plan still needs " + strings.Join(names, ", ")
	case m.State == string(run.Waiting) && m.Wait != nil:
		return "waits for " + m.Wait.Text
	case !m.Released && m.State == "":
		return "has not started in this Pass"
	}
	return "has no recorded state in this Pass"
}

func nodeLabel(node string) string {
	switch node {
	case nodeOrchestrator:
		return "the Orchestrator"
	case nodeUser:
		return "the owner"
	}
	return labelOf(node)
}

func nodeRef(node string) string {
	if strings.HasPrefix(node, "@") {
		return ""
	}
	return "manager:" + node
}

// account states the view's situation in a few factual sentences.
func account(v *View) []Sentence {
	var out []Sentence
	if v.Run.Started.IsZero() && len(v.Passes) == 0 {
		return append(out, Sentence{{Text: "No run record has been read up to this point."}})
	}
	out = append(out, statusSentence(v))
	var working Sentence
	for _, m := range v.Managers {
		if m.State != string(run.Working) {
			continue
		}
		if len(working) == 0 {
			working = append(working, Segment{Text: "Working: "})
		} else {
			working = append(working, Segment{Text: ", "})
		}
		working = append(working, Segment{Text: m.Label, Ref: "manager:" + m.Path}, Segment{Text: " (" + doing(&m) + ")"})
	}
	if len(working) > 0 {
		out = append(out, append(working, Segment{Text: "."}))
	}
	waits := 0
	for _, m := range v.Managers {
		if m.Wait == nil || waits == 3 || v.Status.Outcome != "" {
			continue
		}
		waits++
		s := Sentence{{Text: m.Label, Ref: "manager:" + m.Path}, {Text: " waits for "}}
		if m.Wait.Outdated != "" {
			s = Sentence{{Text: m.Label, Ref: "manager:" + m.Path}, {Text: " is Waiting; its recorded reason is outdated"}}
			if len(m.Wait.On) > 0 {
				s = append(s, Segment{Text: ", and its accepted plan still needs "})
				for i, on := range m.Wait.On {
					if i > 0 {
						s = append(s, Segment{Text: ", "})
					}
					s = append(s, Segment{Text: labelOf(on), Ref: "manager:" + on})
				}
			}
			out = append(out, append(s, Segment{Text: "."}))
			continue
		}
		switch m.Wait.Kind {
		case "decision":
			s = append(s, Segment{Text: nodeLabel(m.Wait.On[0]), Ref: nodeRef(m.Wait.On[0])}, Segment{Text: " to decide on "}, Segment{Text: strings.TrimPrefix(m.Wait.Refs[0], "proposal:"), Ref: m.Wait.Refs[0]})
		case "children":
			s = append(s, Segment{Text: "the results of its children"})
			for i, c := range m.Wait.On {
				sep := ": "
				if i > 0 {
					sep = ", "
				}
				s = append(s, Segment{Text: sep}, Segment{Text: labelOf(c), Ref: "manager:" + c})
			}
		case "answer":
			s = append(s, Segment{Text: "the answer to "}, Segment{Text: strings.TrimPrefix(m.Wait.Refs[0], "question:"), Ref: m.Wait.Refs[0]})
			if len(m.Wait.On) > 0 {
				s = append(s, Segment{Text: " from "}, Segment{Text: nodeLabel(m.Wait.On[0]), Ref: nodeRef(m.Wait.On[0])})
			}
		default:
			s = append(s, Segment{Text: m.Wait.Text})
		}
		if len(m.Wait.Cause) > 0 && m.Wait.Kind != "children" {
			first := m.Wait.Cause[0]
			if first.Doing != "" && !strings.HasPrefix(first.Node, "@") {
				s = append(s, Segment{Text: "; " + nodeLabel(first.Node) + " " + first.Doing})
			}
		}
		out = append(out, append(s, Segment{Text: "."}))
	}
	for _, q := range v.Questions {
		if q.To == run.User && q.Status == "open" {
			out = append(out, Sentence{{Text: "Root asked the owner "}, {Text: q.ID, Ref: "question:" + q.ID}, {Text: "; no answer is recorded."}})
		}
	}
	var open Sentence
	for _, f := range v.Findings {
		if f.Blocking && !f.Closed {
			if len(open) == 0 {
				open = append(open, Segment{Text: "Open genuine findings: "})
			} else {
				open = append(open, Segment{Text: ", "})
			}
			open = append(open, Segment{Text: f.ID, Ref: "finding:" + f.ID}, Segment{Text: " (owner " + labelOf(f.Owner) + ")"})
		}
	}
	if len(open) > 0 {
		out = append(out, append(open, Segment{Text: "."}))
	}
	return out
}

func statusSentence(v *View) Sentence {
	if v.Status.Outcome != "" {
		s := Sentence{{Text: "The Orchestrator recorded the outcome " + v.Status.Outcome}}
		if v.Status.Reason != "" {
			s = append(s, Segment{Text: ": " + clip(v.Status.Reason, 200)})
		}
		s = append(s, Segment{Text: "."})
		if r := v.Status.Report; r != nil {
			s = append(s, Segment{Text: " Run outcome: " + r.Outcome + "."})
			if pub := r.Publication; pub != nil && pub.Outcome == "no_change" {
				s = append(s, Segment{Text: " Publication no_change: the verified candidate equals the captured source; nothing was written and the guard held."})
			} else if pub := r.Publication; pub != nil {
				after := 0
				for _, path := range pub.Paths {
					if path.State == "after" {
						after++
					}
				}
				s = append(s, Segment{Text: fmt.Sprintf(" Publication %s: %d of %d paths read back with the published content.", pub.Outcome, after, len(pub.Paths))})
			} else {
				s = append(s, Segment{Text: " Nothing was published."})
			}
		}
		return s
	}
	if len(v.Passes) == 0 {
		return Sentence{{Text: "The run started; no Pass has started yet."}}
	}
	pv := v.Passes[len(v.Passes)-1]
	s := Sentence{{Text: passName(pv) + " Pass "}, {Text: pv.ID, Ref: "pass:" + pv.ID}}
	if pv.Sealed {
		s = append(s, Segment{Text: " was sealed " + pv.Verdict + "."})
		return s
	}
	s = append(s, Segment{Text: " is running"})
	if pv.Correction {
		s = append(s, Segment{Text: " to correct "})
		for i, f := range pv.Addresses {
			if i > 0 {
				s = append(s, Segment{Text: ", "})
			}
			s = append(s, Segment{Text: f, Ref: "finding:" + f})
		}
	}
	return append(s, Segment{Text: "."})
}

func passName(pv PassView) string {
	if pv.Purpose == "" {
		return "A"
	}
	return strings.ToUpper(pv.Purpose[:1]) + pv.Purpose[1:]
}
