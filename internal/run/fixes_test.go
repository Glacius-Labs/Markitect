package run

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// pollUntil waits until cond holds; the turn fails after a timeout. The script
// must not hold a lock the other turns need while it waits.
func (c *call) pollUntil(cond func() bool) error {
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		if cond() {
			return nil
		}
		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
	}
	c.t.Errorf("turn %s timed out waiting", c.turn.ID)
	return errors.New("timeout")
}

// note is a non-blocking finding; it never holds a verdict back.
func note(owner, observed string) map[string]any {
	return map[string]any{"owner": owner, "obligation": owner + " obligation", "observed": observed, "blocking": false}
}

// findingsNow copies the findings of the run as they stand.
func findingsNow(r *Run) []Finding {
	r.e.mu.Lock()
	defer r.e.mu.Unlock()
	var out []Finding
	for _, f := range r.e.findings {
		out = append(out, *f)
	}
	return out
}

var resultRE = regexp.MustCompile(`- decide (P\d+), the result of`)

// proposalOf finds the ID a Manager's prompt names for a decision, such as
// "the basis of library"; empty when the prompt shows none.
func proposalOf(c *call, kind, manager string) string {
	re := regexp.MustCompile(`decide (P\d+), the ` + kind + ` of ` + regexp.QuoteMeta(manager) + `\b`)
	if m := re.FindStringSubmatch(c.turn.Prompt); m != nil {
		return m[1]
	}
	return ""
}

// checkedProposal is the first proposal of the given kind that a Manager
// submitted and that passed its technical check; empty while none did.
func checkedProposal(f *fixture, kind, manager string) string {
	for _, sub := range f.ev.kinds("proposal.submitted") {
		data := sub.Data.(map[string]any)
		if sub.Manager != manager || data["kind"] != kind {
			continue
		}
		id := data["proposal"].(string)
		for _, ev := range f.ev.kinds("proposal.checked") {
			if d := ev.Data.(map[string]any); d["proposal"] == id && d["passed"] == true {
				return id
			}
		}
	}
	return ""
}

// turnEnds counts the ended turns of a Manager in a role.
func turnEnds(f *fixture, manager, role string) int {
	n := 0
	for _, ev := range f.ev.kinds("turn.end") {
		if ev.Manager == manager && ev.Data.(map[string]any)["role"] == role {
			n++
		}
	}
	return n
}

// A basis a parent returns changes no finding: the finding it raised is not
// recorded, so the accepted resubmission records it exactly once, with one ID.
func TestReturnedSubmissionRaisesNoFinding(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var r *Run
	submits, returned := 0, false
	var afterReturn []Finding
	showedProposed := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			submits++
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{note("library", "the interface names no error case")}}))
			return nil
		case c.turn.Manager == "." && !returned && proposalOf(c, kindBasis, "library") != "":
			// The parent sees the proposed finding, but it is not recorded yet.
			showedProposed = strings.Contains(c.turn.Prompt, "Finding (proposed) [note]")
			c.must(toolReport, map[string]any{"kind": kindReturn, "proposal": proposalOf(c, kindBasis, "library"), "reason": "name the error case first"})
			returned = true
			afterReturn = findingsNow(r)
		}
		c.protocol()
		return nil
	}
	r = f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || !returned || submits != 2 {
		t.Fatalf("stopped %v, verdict %s, returned %v, submits %d: %s", res.Stopped, res.Verdict, returned, submits, out.Reason)
	}
	if len(afterReturn) != 0 || !showedProposed {
		t.Errorf("findings after the return %+v; the parent saw the proposed finding: %v", afterReturn, showedProposed)
	}
	if len(out.Findings) != 1 || out.Findings[0].ID != "F1" || out.Findings[0].Owner != "library" || len(f.ev.kinds("finding.recorded")) != 1 {
		t.Errorf("findings %+v; want the one of the accepted resubmission", out.Findings)
	}
}

// A closure a parent returns closes nothing: the finding stays open and holds
// the Pass; the accepted resubmission closes it exactly once.
func TestReturnedClosureLeavesTheFindingOpen(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var r *Run
	returned := false
	var afterReturn []Finding
	closure := []any{map[string]any{"id": "F1", "reason": "the notice is fixed", "evidence": "lending/reminders/r.go sends it on the due date"}}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.is("p2", "lending") && strings.HasPrefix(c.now(), "integrate"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"closed": closure}))
			return nil
		case c.is("p2", ".") && !returned && proposalOf(c, kindResult, "lending") != "":
			c.must(toolReport, map[string]any{"kind": kindReturn, "proposal": proposalOf(c, kindResult, "lending"), "reason": "show the notice test first"})
			returned = true
			afterReturn = findingsNow(r)
		}
		c.protocol()
		return nil
	}
	r = f.open()
	first := r.Pass(implementSpec())
	second := r.Pass(implementSpec())
	out := f.finish(r, "")
	if first.Verdict != NeedsCorrection || second.Stopped || !returned {
		t.Fatalf("first %s, second stopped %v, returned %v: %s", first.Verdict, second.Stopped, returned, out.Reason)
	}
	if len(afterReturn) != 1 || afterReturn[0].Closed {
		t.Errorf("after the return %+v; the finding must stay open", afterReturn)
	}
	f1 := out.Findings[0]
	if second.Verdict != Satisfied || !f1.Closed || f1.ClosedPass != "p2" || f1.ClosedEvidence == "" || len(f.ev.kinds("finding.closed")) != 1 {
		t.Errorf("second %s, F1 %+v, %d closures", second.Verdict, f1, len(f.ev.kinds("finding.closed")))
	}
}

// A submission whose File check fails is returned before it takes effect: no
// finding is recorded for it, the accepted resubmission records it once.
func TestSubmissionFailingTheFileCheckRaisesNoFinding(t *testing.T) {
	f := newFixture(t)
	f.cfg.FileCheck = func(tree snapshot.Tree) []Finding {
		if _, ok := tree.Lookup(mustPath(t, "library/new.go")); ok {
			if _, ok := tree.Lookup(mustPath(t, "library/registry")); !ok {
				return []Finding{{Owner: "library", Obligation: "file.unexplained", Observed: "library/new.go is not registered"}}
			}
		}
		return nil
	}
	var mu sync.Mutex
	returns, recordedAtRetry := 0, -1
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			c.write("library/new.go", "package library")
			if strings.Contains(c.turn.Prompt, "did not pass the File check") {
				returns++
				recordedAtRetry = len(f.ev.kinds("finding.recorded"))
				c.write("library/registry", "new.go")
			}
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{note("library", "new.go needs a doc comment")}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(prepareSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied || returns != 1 {
		t.Fatalf("stopped %v, verdict %s, technical returns %d: %s", res.Stopped, res.Verdict, returns, out.Reason)
	}
	if recordedAtRetry != 0 || len(out.Findings) != 1 || out.Findings[0].ID != "F1" {
		t.Errorf("%d findings recorded before the retry; final %+v", recordedAtRetry, out.Findings)
	}
}

// acceptLocked checks a stale composite again: when a sibling's submission was
// accepted after the check and the composition now fails the File check, the
// submission goes back to its owner without taking effect.
func TestAcceptanceRechecksTheCompositeAgainstTheCurrentStand(t *testing.T) {
	f := newFixture(t)
	f.cfg.FileCheck = func(tree snapshot.Tree) []Finding {
		_, lending := tree.Lookup(mustPath(t, "lending/new.go"))
		_, mirror := tree.Lookup(mustPath(t, "library/mirror.go"))
		if lending && !mirror {
			return []Finding{{Owner: "library", Obligation: "file.unexplained", Observed: "library/mirror.go does not mirror lending/new.go"}}
		}
		return nil
	}
	var mu sync.Mutex
	staled := false
	var staleAnswer string
	recordedAtRetry := -1
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			c.write("lending/new.go", "package lending")
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			mu.Lock()
			if strings.Contains(c.turn.Prompt, "did not pass the File check") {
				recordedAtRetry = len(f.ev.kinds("finding.recorded"))
				c.write("library/mirror.go", "package library")
			}
			mu.Unlock()
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{note("library", "mirror.go needs a doc comment")}}))
			return nil
		case c.turn.Manager == "." && len(c.pendingDecisions()) > 0:
			mu.Lock()
			first := !staled
			staled = true
			mu.Unlock()
			if first {
				// Both bases passed their check against the same accepted stand;
				// accepting lending's first changes the stand library was checked on.
				if err := c.pollUntil(func() bool {
					return checkedProposal(f, kindBasis, "lending") != "" && checkedProposal(f, kindBasis, "library") != ""
				}); err != nil {
					return err
				}
				c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": checkedProposal(f, kindBasis, "lending")})
				a := c.tool(toolReport, map[string]any{"kind": kindAccept, "proposal": checkedProposal(f, kindBasis, "library")})
				mu.Lock()
				staleAnswer = a.Text
				mu.Unlock()
				return nil
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(prepareSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	if !strings.Contains(staleAnswer, "could not be accepted") || !strings.Contains(staleAnswer, "went back to library") {
		t.Errorf("decision text %q", staleAnswer)
	}
	if recordedAtRetry != 0 || len(out.Findings) != 1 {
		t.Errorf("%d findings recorded before library resubmitted; final %+v", recordedAtRetry, out.Findings)
	}
	if content, ok := f.ws.file("run1-work", "library/mirror.go"); !ok || content == "" {
		t.Errorf("library never resubmitted with its mirror")
	}
}

// The Pass verdict follows Root's integrated result and the open findings. A
// parent's finding on its child is fixed by the child in a later work Pass,
// where the child reports satisfied although the finding is still open: only
// its raiser closes it, in its own later result. Without the closure the Pass
// still needs correction. Before the fix the child's satisfied was refused.
func TestPassVerdictFollowsRootResultAndOpenFindings(t *testing.T) {
	for _, closes := range []bool{true, false} {
		t.Run(fmt.Sprintf("raiser closes %v", closes), func(t *testing.T) {
			f := newFixture(t)
			f.agents.script = func(c *call) error {
				switch {
				case c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
					c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
					return nil
				case c.is("p2", "lending/reminders") && strings.HasPrefix(c.now(), "work on"):
					c.write("lending/reminders/r.go", "r fixed")
				case closes && c.is("p2", "lending") && strings.HasPrefix(c.now(), "integrate"):
					c.must(toolReport, c.envelope(kindResult, map[string]any{"closed": []any{map[string]any{"id": "F1", "reason": "the child fixed the notice", "evidence": "lending/reminders/r.go on this stand"}}}))
					return nil
				}
				c.protocol()
				return nil
			}
			r := f.open()
			first := r.Pass(implementSpec())
			second := r.Pass(implementSpec())
			out := f.finish(r, "")
			if first.Stopped || second.Stopped {
				t.Fatalf("stopped: %s", out.Reason)
			}
			if first.Verdict != NeedsCorrection || !slices.Equal(first.Corrections, []Purpose{Implement}) {
				t.Errorf("first Pass %s, corrections %v", first.Verdict, first.Corrections)
			}
			var child ManagerRecord
			for _, m := range out.Passes[1].Managers {
				if m.Manager == "lending/reminders" {
					child = m
				}
			}
			if child.Verdict != Satisfied {
				t.Errorf("the child reported %s for its corrected work while its parent's finding awaited the parent", child.Verdict)
			}
			want := NeedsCorrection
			if closes {
				want = Satisfied
			}
			f1 := out.Findings[0]
			if second.Verdict != want || f1.Closed != closes || closes && (f1.ClosedPass != "p2" || f1.RaisedBy != "lending") {
				t.Errorf("second Pass %s (want %s), F1 %+v", second.Verdict, want, f1)
			}
		})
	}
}

// A parent shown two pending results that decides only one in a turn gets the
// rest put before it again, so the Pass completes instead of stalling.
func TestPartialDecisionsAreRepromptedWithoutStall(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	partial := false
	var earlier []string
	var rest, reprompt string
	bothResults := func() bool {
		return checkedProposal(f, kindResult, "lending") != "" && checkedProposal(f, kindResult, "library") != ""
	}
	f.agents.script = func(c *call) error {
		if c.turn.Manager != "." {
			c.protocol()
			return nil
		}
		pend := c.pendingDecisions()
		var results []string
		for _, m := range resultRE.FindAllStringSubmatch(c.turn.Prompt, -1) {
			results = append(results, m[1])
		}
		mu.Lock()
		done, prior := partial, slices.Clone(earlier)
		mu.Unlock()
		switch {
		case !done && len(results) >= 2:
			// It accepts the result it already saw before, leaving the newer one.
			keep := 0
			for i, id := range results {
				if slices.Contains(prior, id) {
					keep = i
				}
			}
			mu.Lock()
			partial, rest = true, results[1-keep]
			mu.Unlock()
			c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": results[keep]})
			return nil
		case !done && len(results) == 1:
			mu.Lock()
			earlier = append(earlier, results...)
			mu.Unlock()
			// Hold this turn until both results wait, then decide no result. The
			// children's bases still get their decision, or no result would come.
			accepted := map[string]bool{}
			return c.pollUntil(func() bool {
				for _, child := range []string{"lending", "library"} {
					if id := checkedProposal(f, kindBasis, child); id != "" && !accepted[id] {
						accepted[id] = !c.tool(toolReport, map[string]any{"kind": kindAccept, "proposal": id}).Refused
					}
				}
				return bothResults()
			})
		case done && reprompt == "" && slices.Equal(pend, []string{rest}):
			mu.Lock()
			reprompt = c.turn.Prompt
			mu.Unlock()
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	if !partial || rest == "" || !strings.Contains(reprompt, "Still open for you: decide "+rest+",") {
		t.Errorf("partial %v, rest %q, re-prompt:\n%s", partial, rest, reprompt)
	}
}

// A Manager whose helper still works does not count a turn without a report as
// empty: it receives messages in several turns while the helper runs, is never
// told it ended without a report, and the run does not halt.
func TestManagerWithRunningHelperIsNotCountedEmpty(t *testing.T) {
	f := newFixture(t)
	withWorker(f, "library", Helper{Role: "architect", Profile: "arch"})
	release := make(chan struct{})
	var mu sync.Mutex
	delegated, sent := false, false
	messages := maxEmptyTurns + 3
	f.agents.script = func(c *call) error {
		switch {
		case c.helper():
			if err := c.wait(release); err != nil {
				return err
			}
			c.must(toolReport, map[string]any{"summary": "reviewed the interface"})
			return nil
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && !strings.Contains(c.turn.Prompt, "Your architect h1"):
			mu.Lock()
			first := !delegated
			delegated = true
			mu.Unlock()
			if first {
				c.must(toolDelegate, map[string]any{"role": "architect", "task": "review the interface"})
			}
			// Later turns only receive messages while the helper works.
			return nil
		case c.turn.Manager == "." && len(c.pendingDecisions()) > 0:
			mu.Lock()
			first := !sent
			sent = true
			mu.Unlock()
			if first {
				if err := c.pollUntil(func() bool { return f.ev.index("helper.start", "library", "", nil) >= 0 }); err != nil {
					return err
				}
				for i := 0; i < messages; i++ {
					before := turnEnds(f, "library", "manager")
					c.must(toolSend, map[string]any{"to": "library", "kind": sendInfo, "text": fmt.Sprintf("note %d", i)})
					if err := c.pollUntil(func() bool { return turnEnds(f, "library", "manager") > before }); err != nil {
						return err
					}
				}
				close(release)
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	if n := turnEnds(f, "library", "manager"); n < messages+2 {
		t.Errorf("library had %d turns; want the delegating turn, %d message turns and the one after the helper", n, messages)
	}
	if prompt := promptOf(f, "library", "ended without a report"); prompt != "" {
		t.Errorf("library was told it ended without a report while its helper worked")
	}
}

// A blocked Manager's findings to correct survive a send in a communication-only
// turn: the send answer does not hand them over, and the turn that does the work
// after it unblocked still receives them.
func TestBlockedManagerKeepsItsFindingsAcrossASend(t *testing.T) {
	f := newFixture(t)
	const observed = "the loan interface lacks a due date"
	var mu sync.Mutex
	var blockedSend, blockedPrompt, productivePrompt string
	asked := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if dependsOn(c, "lending", "library", "basis") {
			return nil
		}
		if c.planning() {
			c.protocol()
			return nil
		}
		switch {
		case c.is("p2", ".") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{map[string]any{"owner": "lending", "obligation": "lending/borrow", "expected": "a due date on every loan", "observed": observed, "blocking": true}}}))
			return nil
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && !asked:
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendQuestion, "text": "which loan fields does lending expose?"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			asked = true
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Question Q1 from your child library"):
			c.must(toolSend, map[string]any{"to": "lending", "kind": sendQuestion, "text": "library asks which loan fields you expose", "about": "Q1"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q2"})
			for _, id := range c.pendingDecisions() {
				c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
			}
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Answer to your question Q2"):
			c.must(toolSend, map[string]any{"to": "library", "kind": sendAnswer, "text": "copy, member and due date", "about": "Q1"})
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "your own work waits for") && slices.Contains(c.pendingQuestions(), "Q2"):
			blockedPrompt = c.turn.Prompt
			blockedSend = c.must(toolSend, map[string]any{"to": "parent", "kind": sendAnswer, "text": "copy, member and due date", "about": "Q2"}).Text
			return nil
		case c.is("p2", "lending") && strings.HasPrefix(c.now(), "work on"):
			productivePrompt = c.turn.Prompt
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	r.Pass(implementSpec())
	f.finish(r, "")
	if blockedPrompt == "" || productivePrompt == "" {
		t.Fatalf("lending had a communication-only turn: %v, a productive turn: %v", blockedPrompt != "", productivePrompt != "")
	}
	if strings.Contains(blockedSend, observed) || !strings.Contains(blockedSend, "No new items for you") {
		t.Errorf("the send answer handed over the finding: %q", blockedSend)
	}
	// The parent's accepted basis shows the finding too; what counts is the items.
	items := func(prompt string) string { _, rest, _ := strings.Cut(prompt, "Items:"); return rest }
	if strings.Contains(items(blockedPrompt), observed) {
		t.Errorf("the communication-only turn was handed the finding to correct")
	}
	if !strings.Contains(items(productivePrompt), observed) {
		t.Errorf("the finding never reached lending's productive turn:\n%s", productivePrompt)
	}
}

// Only a review finding can be disputed, and only by a reviewer other than its
// raiser whose responsibility holds the raiser or the owner, once per candidate.
func TestDisputesNeedAReviewFindingAndAnEntitledDisputer(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var r *Run
	refusals := map[string]string{}
	say := func(key, text string) {
		mu.Lock()
		refusals[key] = text
		mu.Unlock()
	}
	dispute := func(ids ...string) []any {
		var out []any
		for _, id := range ids {
			out = append(out, map[string]any{"id": id, "reason": "the notice follows the model", "evidence": "lending/reminders/r.go and its test"})
		}
		return out
	}
	f.agents.script = func(c *call) error {
		if c.turn.Kind != VerifyTurn {
			c.protocol()
			return nil
		}
		switch {
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			// A sibling's responsibility holds neither raiser nor owner.
			if err := c.pollUntil(func() bool { return len(f.ev.kinds("finding.recorded")) > 0 }); err != nil {
				return err
			}
			say("sibling", c.basis(map[string]any{"disputed": dispute("F1")}).Text)
		case c.turn.Manager == "lending/reminders" && strings.HasPrefix(c.now(), "work on"):
			if err := c.pollUntil(func() bool { return len(f.ev.kinds("finding.recorded")) > 0 }); err != nil {
				return err
			}
			// Twice in one report is a second dispute on the same candidate.
			say("twice", c.basis(map[string]any{"disputed": dispute("F1", "F1")}).Text)
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"disputed": dispute("F1")}))
			return nil
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "integrate"):
			say("raiser", c.result(map[string]any{"verdict": string(NeedsCorrection), "disputed": dispute("F1")}).Text)
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		case c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate"):
			// The owner's dispute was accepted on this candidate already.
			say("again", c.result(map[string]any{"verdict": string(NeedsCorrection), "disputed": dispute("F1")}).Text)
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r = f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != NeedsCorrection {
		t.Fatalf("stopped %v, verdict %s: %s", res.Stopped, res.Verdict, out.Reason)
	}
	for key, want := range map[string]string{
		"sibling": "neither the raiser nor the owner",
		"twice":   "already disputed on this candidate",
		"raiser":  "you raised",
		"again":   "already disputed on this candidate",
	} {
		if !strings.Contains(refusals[key], want) {
			t.Errorf("%s: refusal %q does not contain %q", key, refusals[key], want)
		}
	}
	if d := out.Findings[0].Disputes; len(d) != 1 || d[0].By != "lending/reminders" {
		t.Errorf("disputes %+v; want only the owner's", d)
	}
	if !r.Reassess() {
		t.Errorf("a review kept open only by a fresh dispute must be reassessed")
	}
}

// A finding of a work Pass is no review finding: it cannot be disputed in a
// review, whoever asks.
func TestWorkPassFindingCannotBeDisputed(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.is("p2", ".") && strings.HasPrefix(c.now(), "integrate"):
			refusal = c.result(map[string]any{"verdict": string(NeedsCorrection), "disputed": []any{map[string]any{"id": "F1", "reason": "fine", "evidence": "x"}}}).Text
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != NeedsCorrection || out.Findings[0].Review || len(out.Findings[0].Disputes) != 0 {
		t.Fatalf("stopped %v, verdict %s, F1 %+v: %s", res.Stopped, res.Verdict, out.Findings[0], out.Reason)
	}
	if !strings.Contains(refusal, "no open genuine review finding") {
		t.Errorf("refusal %q", refusal)
	}
}

// Every closure states its evidence, resolved or disproved; a blank one does
// not count. The same closure with evidence closes the finding.
func TestClosureWithoutEvidenceIsRefused(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	refusals := map[string]string{}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.is("p2", "lending") && strings.HasPrefix(c.now(), "work on"):
			try := func(key string, closed map[string]any) {
				closed["id"], closed["reason"] = "F1", "the child fixed the notice"
				refusals[key] = c.basis(map[string]any{"closed": []any{closed}}).Text
			}
			try("missing", map[string]any{})
			try("blank", map[string]any{"evidence": "  ", "as": ClosedResolved})
			try("disproved", map[string]any{"as": ClosedDisproved})
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"closed": []any{map[string]any{"id": "F1", "reason": "the child fixed the notice", "evidence": "lending/reminders/r.go on this stand"}}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	for _, key := range []string{"missing", "blank", "disproved"} {
		if !strings.Contains(refusals[key], "a closure states its evidence") {
			t.Errorf("%s: refusal %q", key, refusals[key])
		}
	}
	if f1 := out.Findings[0]; res.Stopped || res.Verdict != Satisfied || !f1.Closed || f1.ClosedEvidence == "" {
		t.Errorf("stopped %v, verdict %s, F1 %+v", res.Stopped, res.Verdict, f1)
	}
}

// An accepted result that leaves an own obligation open keeps the Pass at
// needs_correction even when the finding its verdict rested on is closed.
func TestOpenCoverageKeepsThePassAtNeedsCorrection(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.is("p2", "lending/reminders") && strings.HasPrefix(c.now(), "submit your complete result"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection),
				"obligations": []any{map[string]any{"definition": "lending/reminders/notice", "status": coverOpen, "conditions": "notice timing remains uncovered", "note": "the notice timing is not realized yet"}}}))
			return nil
		case c.is("p2", "lending") && strings.HasPrefix(c.now(), "integrate"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"closed": []any{map[string]any{"id": "F1", "reason": "looks done", "evidence": "lending/reminders/r.go on this stand"}}}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	second := r.Pass(implementSpec())
	out := f.finish(r, "")
	if second.Stopped || !out.Findings[0].Closed || second.Verdict != NeedsCorrection {
		t.Fatalf("stopped %v, F1 closed %v, verdict %s: an open obligation was sealed clean", second.Stopped, out.Findings[0].Closed, second.Verdict)
	}
}

// After a work Pass the open findings of work Passes route the work, whichever
// Pass raised them; a review finding waiting for the next review does not.
func TestWorkPassCorrectionsAreTheOpenWorkFindings(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		switch {
		case c.turn.Kind != VerifyTurn && c.is("p1", "lending") && strings.HasPrefix(c.now(), "work on"):
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{genuine("lending/reminders", "lending/reminders/notice")}}))
			return nil
		case c.turn.Kind == VerifyTurn && c.is("p2", ".") && strings.HasPrefix(c.now(), "work on"):
			finding := genuine("library", "library/interface")
			finding["correction"] = string(Plan)
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"findings": []any{finding}}))
			return nil
		case c.turn.Kind == VerifyTurn && c.is("p2", ".") && strings.HasPrefix(c.now(), "integrate"):
			c.must(toolReport, c.envelope(kindResult, map[string]any{"verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	first := r.Pass(implementSpec())
	review := r.Pass(verifySpec())
	third := r.Pass(implementSpec())
	f.finish(r, "")
	if first.Stopped || review.Stopped || third.Stopped {
		t.Fatal("a Pass stopped")
	}
	if !slices.Equal(review.Corrections, []Purpose{Implement, Plan}) && !slices.Equal(review.Corrections, []Purpose{Plan, Implement}) {
		t.Errorf("review corrections %v, want the open work and review findings", review.Corrections)
	}
	// F1 of p1 is still open and raised nothing new in p3: it repeats the work;
	// the review's plan finding waits for the next review.
	if third.Verdict != NeedsCorrection || !slices.Equal(third.Corrections, []Purpose{Implement}) {
		t.Errorf("third Pass %s, corrections %v", third.Verdict, third.Corrections)
	}
}
