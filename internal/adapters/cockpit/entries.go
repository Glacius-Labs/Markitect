package cockpit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Entry is one record as the timeline and the event list show it: a factual
// one-line text, the detail it refers to and, for an exchange, its link.
type Entry struct {
	Seq     int       `json:"seq"`
	Mark    string    `json:"mark"`
	At      time.Time `json:"at,omitzero"`
	Kind    string    `json:"kind"`
	Pass    string    `json:"pass,omitempty"`
	Manager string    `json:"manager,omitempty"`
	Turn    string    `json:"turn,omitempty"`
	Class   string    `json:"class"`
	Text    string    `json:"text"`
	Ref     string    `json:"ref,omitempty"`
	Link    *Link     `json:"link,omitempty"`
}

// Link is the direction and channel of one recorded exchange. Channel "agent"
// is Agent-authored communication, "submission" an Agent's report to its
// decider, "orchestrator" a control or acceptance event the Orchestrator
// records, and "owner" the owner channel. A link records a send, not a
// delivery or a reading.
type Link struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Channel string `json:"channel"`
	Purpose string `json:"purpose"`
}

// Entries describes records as timeline entries.
func Entries(records []Record) []Entry {
	out := make([]Entry, 0, len(records))
	for _, r := range records {
		out = append(out, entry(r))
	}
	return out
}

func entry(r Record) Entry {
	e := Entry{Seq: r.Seq, Mark: r.Mark, At: r.At, Kind: r.Kind, Pass: r.Pass, Manager: r.Manager, Turn: r.Turn}
	if r.Invalid != "" {
		e.Class, e.Text, e.Kind = "unreadable", "Unreadable record: "+r.Invalid, "unreadable"
		return e
	}
	var d map[string]json.RawMessage
	_ = json.Unmarshal(r.Data, &d)
	str := func(key string) string {
		var s string
		_ = json.Unmarshal(d[key], &s)
		return s
	}
	flag := func(key string) bool {
		var b bool
		_ = json.Unmarshal(d[key], &b)
		return b
	}
	who := labelOf(r.Manager)
	decider := func() string {
		by := str("by")
		switch by {
		case "", "the Orchestrator":
			return nodeOrchestrator
		case "markitect":
			return nodeOrchestrator
		}
		return by
	}
	switch r.Kind {
	case "run.start":
		var managers []string
		_ = json.Unmarshal(d["managers"], &managers)
		e.Class, e.Text = "control", fmt.Sprintf("Run started with %d Manager(s)", len(managers))
	case "pass.start":
		access := "read-only"
		if flag("writes") {
			access = "owned writes"
		}
		e.Class, e.Ref = "control", "pass:"+r.Pass
		e.Text = fmt.Sprintf("Pass %s started: %s, %s", r.Pass, str("purpose"), access)
		switch {
		case flag("review") && str("purpose") == "assess":
			e.Text = fmt.Sprintf("Pass %s started: Model assessment with fresh reviewers, read-only", r.Pass)
		case flag("review"):
			e.Text = fmt.Sprintf("Pass %s started: %s with fresh reviewers, read-only", r.Pass, str("purpose"))
		}
	case "pass.sealed":
		e.Class, e.Ref, e.Text = "control", "pass:"+r.Pass, fmt.Sprintf("%s Pass %s sealed %s", passTitle(str("purpose")), r.Pass, strings.ReplaceAll(str("verdict"), "_", " "))
	case "manager.state_changed":
		e.Class, e.Ref = "state", "manager:"+r.Manager
		e.Text = fmt.Sprintf("%s %s → %s: %s", who, str("from"), str("to"), str("why"))
		if w := str("waitingFor"); w != "" {
			e.Text += "; waits for " + w
		}
	case "manager.release":
		e.Class, e.Ref, e.Text = "control", "manager:"+r.Manager, fmt.Sprintf("%s released: %s", who, str("why"))
		e.Link = &Link{From: nodeOrchestrator, To: r.Manager, Channel: "orchestrator", Purpose: "release"}
	case "manager.unblocked":
		e.Class, e.Ref, e.Text = "control", "manager:"+r.Manager, fmt.Sprintf("%s unblocked: %s", who, str("why"))
	case "manager.disabled":
		e.Class, e.Ref, e.Text = "problem", "manager:"+r.Manager, fmt.Sprintf("%s disabled after repeated failed turns", who)
	case "turn.start":
		access := "writes"
		if flag("readOnly") {
			access = "read-only"
		}
		if mode := str("mode"); mode != "" {
			access += ", " + mode
		}
		if ctx := str("context"); ctx != "" {
			access += ", context " + ctx
		}
		e.Class, e.Ref, e.Text = "work", "turn:"+r.Turn, fmt.Sprintf("%s: %s turn %s started (%s)", who, str("role"), shortTurn(r.Turn), access)
	case "turn.end":
		var reports []string
		_ = json.Unmarshal(d["reports"], &reports)
		text := fmt.Sprintf("%s: turn %s ended", who, shortTurn(r.Turn))
		if len(reports) > 0 {
			text += ", reported " + strings.Join(reports, ", ")
		} else {
			text += " without a report"
		}
		if n := intOf(d["probes"]); n > 0 {
			text += fmt.Sprintf("; %d probe%s", n, plural(n))
		}
		if !flag("stopped") {
			text += "; container stop not observed"
		}
		e.Class, e.Ref, e.Text = "work", "turn:"+r.Turn, text
	case "report.progress":
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), who+" reported progress"
	case "report.wait":
		waitFor := str("for")
		if stage := str("stage"); stage != "" {
			waitFor = fmt.Sprintf("the accepted %s of %s", stage, labelOf(waitFor))
		}
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s reported waiting for %s", who, waitFor)
	case "proposal.submitted":
		id := str("proposal")
		e.Class, e.Ref, e.Text = "exchange", "proposal:"+id, fmt.Sprintf("%s submitted %s %s", who, str("kind"), id)
		to := nodeOrchestrator
		if parent := parentOf(r.Manager); parent != "" {
			to = parent
		}
		e.Link = &Link{From: r.Manager, To: to, Channel: "submission", Purpose: str("kind") + " " + id}
	case "proposal.checked":
		id := str("proposal")
		result := "passed"
		if !flag("passed") {
			result = "failed"
		}
		e.Class, e.Ref, e.Text = "control", "proposal:"+id, fmt.Sprintf("%s %s its technical check", id, result)
	case "basis.accepted", "result.accepted", "basis.returned", "result.returned", "result.reopened":
		id := str("proposal")
		kind, verb, _ := strings.Cut(r.Kind, ".")
		from := decider()
		e.Class, e.Ref = "exchange", "proposal:"+id
		e.Text = fmt.Sprintf("%s %s %s %s of %s", nodeLabel(from), verb, kind, id, who)
		channel := "agent"
		if from == nodeOrchestrator {
			channel = "orchestrator"
		}
		e.Link = &Link{From: from, To: r.Manager, Channel: channel, Purpose: verb + " " + id}
	case "basis.invalidated":
		e.Class, e.Ref, e.Text = "control", "proposal:"+str("proposal"), fmt.Sprintf("%s revised its basis %s", who, str("proposal"))
	case "proposal.invalidated", "proposal.withdrawn":
		e.Class, e.Ref, e.Text = "control", "proposal:"+str("proposal"), fmt.Sprintf("%s %s", str("proposal"), strings.TrimPrefix(r.Kind, "proposal."))
	case "finding.recorded", "finding.closed", "finding.disputed":
		id := str("id")
		kind := observationLabel(str("kind"), flag("blocking"))
		verb := strings.TrimPrefix(r.Kind, "finding.")
		if verb == "recorded" {
			verb = "raised"
		}
		e.Class, e.Ref = "evidence", "finding:"+id
		e.Text = fmt.Sprintf("%s %s %s: owner %s, raised by %s", kind, id, verb, labelOf(str("owner")), labelOf(str("raisedBy")))
		if verb == "closed" {
			e.Text = fmt.Sprintf("%s %s closed as %s", kind, id, str("closedAs"))
		}
		if class := str("correction"); class != "" && str("kind") == "model" {
			e.Text += ", class " + class
		}
	case "question.asked":
		id := str("id")
		from, to := str("from"), str("to")
		e.Class, e.Ref, e.Text = "exchange", "question:"+id, fmt.Sprintf("%s asked %s %s", labelOf(from), nodeLabel(participant(to)), id)
		channel := "agent"
		if to == recordedUser {
			channel = "owner"
		}
		e.Link = &Link{From: from, To: participant(to), Channel: channel, Purpose: "question " + id}
		if str("about") != "" {
			e.Text += " (serving " + str("about") + ")"
		}
	case "question.answered":
		id := str("id")
		from, to := participant(str("to")), str("from")
		e.Class, e.Ref, e.Text = "exchange", "question:"+id, fmt.Sprintf("%s answered %s", nodeLabel(from), id)
		channel := "agent"
		if from == nodeUser {
			channel = "owner"
		}
		e.Link = &Link{From: from, To: to, Channel: channel, Purpose: "answer " + id}
	case "question.lapsed":
		id := str("id")
		e.Class, e.Ref, e.Text = "exchange", "question:"+id, fmt.Sprintf("%s lapsed unanswered when its Pass closed", id)
	case "message":
		e.Class, e.Ref, e.Text = "exchange", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s informed %s", labelOf(str("from")), labelOf(str("to")))
		e.Link = &Link{From: str("from"), To: str("to"), Channel: "agent", Purpose: "information"}
	case "problem.raised":
		text := fmt.Sprintf("%s raised %s problem %s", who, str("category"), str("id"))
		if flag("abort") {
			text += " and ends the run"
		} else if flag("escalated") {
			text += ", escalated"
		}
		e.Class, e.Ref, e.Text = "problem", "problem:"+str("id"), text
	case "helper.delegated", "helper.start", "helper.end":
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s: helper %s (%s) %s", who, str("helper"), str("role"), strings.TrimPrefix(r.Kind, "helper."))
	case "check.requested":
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), who+" ran an advisory File check"
	case "candidate":
		e.Class, e.Ref, e.Text = "evidence", "pass:"+r.Pass, "Immutable candidate captured: "+short(str("tree"))
	case "check":
		result := "passed"
		if !flag("passed") {
			result = "failed"
		}
		e.Class, e.Ref, e.Text = "evidence", "check:"+r.Pass+"/"+str("name"), fmt.Sprintf("Required check %s %s on the candidate", str("name"), result)
	case "gate":
		e.Class, e.Ref, e.Text = "evidence", "pass:"+r.Pass, fmt.Sprintf("Whole-stand Model check and File check %s", str("at"))
	case "registry.view":
		e.Class, e.Ref, e.Text = "control", "pass:"+r.Pass, "Derived registry view recorded"
	case "outcome":
		e.Class, e.Text = "control", "Outcome recorded: "+str("status")
		if reason := str("reason"); reason != "" {
			e.Text += " (" + clip(reason, 120) + ")"
		}
	case "lookup.definition":
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s looked up %s", who, str("id"))
		if flag("refused") {
			e.Text += " (refused: too large)"
		}
	case "lookup.brief":
		e.Class, e.Ref, e.Text = "work", "seq:"+fmt.Sprint(r.Seq), who+" looked up its Brief again"
	case "dependency.bound":
		e.Class, e.Ref, e.Text = "control", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s's work now depends on the accepted %s of %s (%s)", who, str("stage"), labelOf(str("on")), str("proposal"))
		if flag("rebound") {
			e.Text = fmt.Sprintf("%s's work was rebound to the revised %s of %s (%s)", who, str("stage"), labelOf(str("on")), str("proposal"))
		}
	case "dependency.unbound":
		e.Class, e.Ref, e.Text = "control", "seq:"+fmt.Sprint(r.Seq), fmt.Sprintf("%s's dependency on the accepted %s of %s was refused: it would close a cycle", who, str("stage"), labelOf(str("on")))
	case "probe.requested":
		e.Class, e.Ref, e.Text = "evidence", "probe:"+str("id"), fmt.Sprintf("Probe %s requested by %s on %s for %s", str("id"), who, standText(str("standKind")), str("obligation"))
	case "probe.receipt":
		e.Class, e.Ref, e.Text = "evidence", "probe:"+str("id"), fmt.Sprintf("Probe %s on %s: %s", str("id"), standText(str("standKind")), str("outcome"))
		if flag("uncertain") {
			e.Text += " (uncertain)"
		}
	case "probe.disposed":
		var disposition struct {
			As      string `json:"as"`
			By      string `json:"by"`
			Pending bool   `json:"pending"`
		}
		_ = json.Unmarshal(d["disposition"], &disposition)
		e.Class, e.Ref = "evidence", "probe:"+str("receipt")
		switch {
		case d["voided"] != nil:
			e.Text = fmt.Sprintf("The disposition of probe %s was withdrawn", str("receipt"))
		case disposition.Pending:
			e.Text = fmt.Sprintf("Probe %s disposed as %s by %s, pending an independent review", str("receipt"), strings.ReplaceAll(disposition.As, "_", " "), labelOf(disposition.By))
		default:
			e.Text = fmt.Sprintf("Probe %s disposed as %s by %s", str("receipt"), strings.ReplaceAll(disposition.As, "_", " "), labelOf(disposition.By))
		}
	case "extract.phase":
		e.Class, e.Text = "control", fmt.Sprintf("Extract phase %s ended %s", str("phase"), str("status"))
	case "environment.release":
		// Written by earlier builds that kept a working environment between turns.
		e.Class, e.Text = "runtime", "environment release"
	case "owner.error":
		e.Class, e.Text = "problem", fmt.Sprintf("Owner channel error for %s", str("question"))
	case "workspace.materialize", "workspace.close", "home.remove":
		e.Class, e.Text = "runtime", strings.ReplaceAll(r.Kind, ".", " ")
	default:
		e.Class, e.Text = "other", "Record "+r.Kind
	}
	if e.Ref == "" {
		e.Ref = "seq:" + fmt.Sprint(r.Seq)
	}
	return e
}

// observationLabel names a finding's kind (owner refinement of 21 October
// 2026). A record of an earlier build has no kind; its blocking flag decides.
func observationLabel(kind string, blocking bool) string {
	switch kind {
	case "deviation":
		return "Deviation"
	case "suspected":
		return "Suspicion"
	case "model":
		return "Model problem"
	case "suggestion":
		return "Suggestion"
	}
	if blocking {
		return "Genuine finding"
	}
	return "Note"
}

// standText says which stand a probe ran on.
func standText(kind string) string {
	switch kind {
	case "candidate":
		return "the candidate"
	case "accepted":
		return "the accepted stand"
	case "proposal":
		return "a checked proposal"
	case "live":
		return "the live stand"
	}
	return "a stand"
}

func intOf(raw json.RawMessage) int {
	var n int
	_ = json.Unmarshal(raw, &n)
	return n
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func parentOf(path string) string {
	if path == "." || path == "" {
		return ""
	}
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return "."
}

func short(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}
