package run

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var probeIDRE = regexp.MustCompile(`-r\d+$`)

// probeOutcomes makes the fake runner answer probes by their file content:
// "FAIL" exits 1, "HANG" times out uncertainly, anything else exits 0.
// Required checks keep passing.
func probeOutcomes(r CheckRequest) (CheckReceipt, error, bool) {
	if !probeIDRE.MatchString(string(r.ID)) {
		return CheckReceipt{}, nil, false
	}
	rec := CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true}
	for _, e := range r.Tree.Entries() {
		switch string(e.Bytes()) {
		case "FAIL":
			rec.ExitCode, rec.Output = 1, []byte("AssertionError: expected exit 0, got 2")
		case "HANG":
			rec.Exited, rec.TimedOut, rec.Uncertain, rec.Reason = false, true, true, "the probe timed out"
		}
	}
	return rec, nil, true
}

func probe(obligation, claim, path, content string) map[string]any {
	return map[string]any{"probe": map[string]any{"obligation": obligation, "claim": claim, "expected": "the model's outcome for " + claim,
		"check": "tests", "path": path, "content": content, "args": []string{path}}}
}

var (
	receiptRE       = regexp.MustCompile(`Receipt (R\d+):`)
	listedReceiptRE = regexp.MustCompile(`Receipt (R\d+) by`)
)

func receiptOf(t testing.TB, a ToolAnswer) string {
	t.Helper()
	m := receiptRE.FindStringSubmatch(a.Text)
	if a.Refused || m == nil {
		t.Fatalf("probe not run: %+v", a)
	}
	return m[1]
}

// withEvidence returns the turn's full coverage citing the receipt for its
// first obligation.
func withEvidence(c *call, receipt string) []map[string]any {
	cov := c.coverage()
	cov[0]["evidence"] = []map[string]string{{"claim": "the probed claim", "receipt": receipt}}
	return cov
}

func probeFixture(t *testing.T) *fixture {
	f := newFixture(t)
	f.addCheck("tests", ".")
	f.cfg.Checks[0].Request.Args = []string{"-B", "-m", "unittest"}
	f.cfg.Checks[0].ProfileArgs = 1
	f.checks.fail = probeOutcomes
	return f
}

// The Orchestrator, not the Agent, makes a receipt: the claim and expectation
// are recorded before the probe runs, the probe file is added to the immutable
// candidate in the check's directory without replacing a project file, the
// check's profile runs it with the profile's own arguments first, and every
// outcome is kept. Before this, a report's evidence was free text that nobody
// could retrieve or bind (old behavior: a done claim with a suite total only
// was accepted in a review).
func TestProbeReceiptsAreBoundAndRetained(t *testing.T) {
	f := probeFixture(t)
	f.cfg.Start = treeOf(t, map[string]string{"go.mod": "module x\n", "library/a.go": "a", "lending/l.go": "l", "lending/reminders/r.go": "r", "realization/app.py": "app"})
	var mu sync.Mutex
	var suiteOnly, replaced, fabricated, foreign ToolAnswer
	var receipt string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			cov := c.coverage()
			delete(cov[0], "evidence")
			suiteOnly = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"obligations": cov, "evidence": []string{"51 tests passed"}}))
			replaced = c.tool(toolCheck, probe("library/interface", "a hyphen name", "app.py", "PASS"))
			foreign = c.tool(toolCheck, probe("lending/borrow", "a loan", "probe_loan.py", "PASS"))
			fabricated = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, "R99")}))
			receipt = receiptOf(t, c.must(toolCheck, probe("library/interface", "a hyphen name", "probe_names.py", "PASS")))
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, receipt)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Stopped || out.Status != Verified {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	if !suiteOnly.Refused || !strings.Contains(suiteOnly.Text, "needs evidence") {
		t.Errorf("a suite total alone covered a Definition: %+v", suiteOnly)
	}
	if !replaced.Refused || !strings.Contains(replaced.Text, "never replaces a project file") {
		t.Errorf("a probe replaced a candidate file: %+v", replaced)
	}
	if !foreign.Refused || !strings.Contains(foreign.Text, "outside your responsibility") {
		t.Errorf("a probe of another responsibility ran: %+v", foreign)
	}
	if !fabricated.Refused || !strings.Contains(fabricated.Text, "no probe receipt") {
		t.Errorf("a receipt written into the report was admitted: %+v", fabricated)
	}
	if len(out.Probes) != 1 || out.Probes[0].ID != receipt {
		t.Fatalf("probes %+v", out.Probes)
	}
	got := out.Probes[0]
	candidate := out.Passes[1].Tree
	if got.Stand != candidate || got.StandKind != standCandidate || got.Outcome != ProbePassed || got.Path != "realization/probe_names.py" || got.Content != "PASS" || got.ContentDigest == "" {
		t.Errorf("receipt %+v on candidate %s", got, candidate)
	}
	var probeReq *CheckRequest
	for i, req := range f.checks.reqs {
		if probeIDRE.MatchString(string(req.ID)) {
			probeReq = &f.checks.reqs[i]
		}
	}
	if probeReq == nil || !slices.Equal(probeReq.Args, []string{"-B", "probe_names.py"}) || probeReq.Tree.Digest().String() != got.Tree {
		t.Fatalf("probe request %+v", probeReq)
	}
	if entry, ok := probeReq.Tree.Lookup(mustPath(t, "realization/app.py")); !ok || string(entry.Bytes()) != "app" {
		t.Errorf("the probe tree changed a candidate file")
	}
	requested := f.ev.index("probe.requested", "library", "", nil)
	recorded := f.ev.index("probe.receipt", "library", "", nil)
	if requested < 0 || recorded < requested {
		t.Errorf("the expectation was not on record before the probe ran: %d, %d", requested, recorded)
	}
}

// A failed probe stays adverse evidence: it cannot support a claim, it blocks
// a done claim and a satisfied result until it is disposed of, and a genuine
// finding that cites it disposes of it as a violation. Old behavior: a reviewer
// could leave a failed observation out of its report and claim satisfied.
func TestFailedProbeBlocksUntilAFindingCitesIt(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var cited, undisposed, satisfied ToolAnswer
	var receipt string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			receipt = receiptOf(t, c.must(toolCheck, probe("library/interface", "a hyphen name", "probe_names.py", "FAIL")))
			cited = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, receipt)}))
			undisposed = c.tool(toolReport, c.envelope(kindBasis, nil))
			finding := genuine("library", "library/interface")
			finding["receipts"] = []string{receipt}
			cov := c.coverage()
			cov[0]["status"], cov[0]["note"] = coverOpen, "the hyphen name is refused"
			delete(cov[0], "evidence")
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": cov, "findings": []any{finding}}))
			return nil
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit") {
			satisfied = c.result(nil)
			cov := c.coverage()
			cov[0]["status"], cov[0]["note"] = coverOpen, "the hyphen name is refused"
			delete(cov[0], "evidence")
			c.must(toolReport, c.envelope(kindResult, map[string]any{"obligations": cov, "verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if !cited.Refused || !strings.Contains(cited.Text, "cannot support") {
		t.Errorf("a failed receipt supported a claim: %+v", cited)
	}
	if !undisposed.Refused || !strings.Contains(undisposed.Text, "has no disposition") {
		t.Errorf("a done claim hid its failed receipt: %+v", undisposed)
	}
	if !satisfied.Refused {
		t.Errorf("a satisfied result was admitted with an open finding: %+v", satisfied)
	}
	if res.Verdict != NeedsCorrection || out.Status == Verified || r.Reassess() {
		t.Errorf("verdict %s, status %s", res.Verdict, out.Status)
	}
	d := out.Probes[0].Disposition
	if d == nil || d.As != DisposedViolation || d.Finding == "" || d.Pending {
		t.Errorf("disposition %+v", d)
	}
	if !slices.Contains(out.Findings[0].Receipts, receipt) {
		t.Errorf("finding %+v does not cite %s", out.Findings[0], receipt)
	}
}

// A non-Root reviewer may dispose of its own failed probe as an erroneous
// expectation; its parent sees the receipt and the disproof and decides
// independently. Root has no parent: its own disproof awaits a fresh review on
// the same candidate, which confirms it or raises a finding.
func TestExpectationErrorsNeedAnIndependentAssessment(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var parentSaw, freshRootSaw bool
	var rootSelf, rerunFailed ToolAnswer
	verifies := 0
	disprove := func(receipt string) map[string]any {
		return map[string]any{"receipt": receipt, "as": DisposedExpectationError, "reason": "the model does not fix the exit code", "evidence": "library/interface states no exit code"}
	}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		review := c.turn.Kind == VerifyTurn
		if review && c.turn.Manager == "." && strings.HasPrefix(c.now(), "work on") {
			verifies++
		}
		switch {
		case review && verifies == 1 && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			own := receiptOf(t, c.must(toolCheck, probe("library/interface", "an exit code", "probe_exit.py", "FAIL")))
			rerunFailed = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": []any{map[string]any{"receipt": own, "as": DisposedRerun, "rerun": own}}}))
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": []any{disprove(own)}}))
			return nil
		case review && verifies == 1 && c.turn.Manager == "." && proposalOf(c, kindBasis, "library") != "":
			parentSaw = strings.Contains(c.turn.Prompt, "as an erroneous expectation") && strings.Contains(c.turn.Prompt, "Probe:\n") && strings.Contains(c.turn.Prompt, "FAIL")
		case review && verifies == 1 && c.turn.Manager == "." && strings.HasPrefix(c.now(), "work on"):
			own := receiptOf(t, c.must(toolCheck, probe("library/interface", "a cross-area call", "probe_cross.py", "FAIL")))
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": []any{disprove(own)}}))
			return nil
		case review && verifies == 1 && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate"):
			rootSelf = c.must(toolReport, c.envelope(kindResult, nil))
			return nil
		case review && verifies == 2 && c.turn.Manager == "." && strings.HasPrefix(c.now(), "work on"):
			freshRootSaw = strings.Contains(c.turn.Prompt, "still need an independent disposition") && strings.Contains(c.turn.Prompt, "awaiting an independent assessment")
			var pending []any
			for _, m := range listedReceiptRE.FindAllStringSubmatch(c.turn.Prompt, -1) {
				pending = append(pending, disprove(m[1]))
			}
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": pending}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	first := r.Pass(verifySpec())
	if first.Verdict != NeedsCorrection || !r.Reassess() {
		t.Fatalf("Root's own disproof made the review clean: verdict %s, reassess %v", first.Verdict, r.Reassess())
	}
	second := r.Pass(verifySpec())
	out := f.finish(r, "")
	if second.Verdict != Satisfied || out.Status != Verified {
		t.Fatalf("second verdict %s, status %s: %s", second.Verdict, out.Status, out.Reason)
	}
	if !parentSaw || !freshRootSaw || rootSelf.Refused {
		t.Errorf("parent saw %v, fresh Root saw %v, Root's satisfied result %+v", parentSaw, freshRootSaw, rootSelf)
	}
	if !rerunFailed.Refused || !strings.Contains(rerunFailed.Text, "never by running it again") {
		t.Errorf("a failed assertion was disposed of by running it again: %+v", rerunFailed)
	}
	for _, p := range out.Probes {
		if p.Disposition == nil || p.Disposition.Pending || p.Disposition.As != DisposedExpectationError {
			t.Errorf("receipt %s disposition %+v", p.ID, p.Disposition)
		}
		if p.Requester == "." && p.Disposition.Pass == p.Pass {
			t.Errorf("Root's own disproof counted as independent: %+v", p.Disposition)
		}
	}
}

// A Runtime error is no assertion: it neither passes nor shows the expected
// refusal, it cannot be explained away as an expectation error, and only an
// exact rerun of the same probe (probe.rerun) that passes on the same candidate
// replaces it; a different probe under the same claim cannot. An exit that says
// the probe could not be executed is an error too, not a failed assertion.
// Before the review fix, any later passing receipt with the same claim text
// disposed of an error.
func TestProbeErrorsStayUnresolvedUntilAnExactRerunPasses(t *testing.T) {
	f := probeFixture(t)
	var runs sync.Map
	f.checks.fail = func(r CheckRequest) (CheckReceipt, error, bool) {
		if !probeIDRE.MatchString(string(r.ID)) {
			return CheckReceipt{}, nil, false
		}
		rec := CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true}
		for _, e := range r.Tree.Entries() {
			switch string(e.Bytes()) {
			case "FLAKY":
				// The first two runs of the probe time out; the third passes.
				n, _ := runs.LoadOrStore("flaky", 0)
				runs.Store("flaky", n.(int)+1)
				if n.(int) < 2 {
					rec.Exited, rec.TimedOut, rec.Uncertain = false, true, true
				}
			case "NOEXEC":
				rec.ExitCode = 127
			}
		}
		return rec, nil, true
	}
	var mu sync.Mutex
	var asExpectation, otherProbe, rerunNew, noexec ToolAnswer
	var hung, erred, passed, noexecID string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			hung = receiptOf(t, c.must(toolCheck, probe("library/interface", "a slow command", "probe_slow.py", "FLAKY")))
			asExpectation = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": []any{map[string]any{"receipt": hung, "as": DisposedExpectationError, "reason": "slow", "evidence": "none"}}}))
			weaker := receiptOf(t, c.must(toolCheck, probe("library/interface", "a slow command", "probe_weak.py", "PASS")))
			otherProbe = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"dispositions": []any{map[string]any{"receipt": hung, "as": DisposedRerun, "rerun": weaker}}}))
			rerunNew = c.tool(toolCheck, map[string]any{"probe": map[string]any{"rerun": weaker}})
			noexec = c.must(toolCheck, probe("library/interface", "a missing interpreter", "probe_noexec.py", "NOEXEC"))
			erred = receiptOf(t, c.must(toolCheck, map[string]any{"probe": map[string]any{"rerun": hung}}))
			// A repetition of a repetition repeats the original probe.
			passed = receiptOf(t, c.must(toolCheck, map[string]any{"probe": map[string]any{"rerun": erred}}))
			noexecID = receiptRE.FindStringSubmatch(noexec.Text)[1]
			noexecAgain := receiptOf(t, c.must(toolCheck, map[string]any{"probe": map[string]any{"rerun": noexecID}}))
			finding := genuine("library", "library/interface")
			finding["observed"], finding["receipts"] = "the probe cannot be executed, so the claim has no evidence", []string{noexecID, noexecAgain}
			cov := c.coverage()
			cov[0]["status"], cov[0]["note"] = coverOpen, "the claim lacks evidence"
			delete(cov[0], "evidence")
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": cov, "findings": []any{finding},
				"dispositions": []any{map[string]any{"receipt": hung, "as": DisposedRerun, "rerun": passed}}}))
			return nil
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit") {
			cov := c.coverage()
			cov[0]["status"], cov[0]["note"] = coverOpen, "the claim lacks evidence"
			delete(cov[0], "evidence")
			c.must(toolReport, c.envelope(kindResult, map[string]any{"obligations": cov, "verdict": string(NeedsCorrection)}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if res.Verdict != NeedsCorrection || out.Status == Verified {
		t.Fatalf("verdict %s, status %s: %s", res.Verdict, out.Status, out.Reason)
	}
	byID := map[string]ProbeReceipt{}
	for _, p := range out.Probes {
		byID[p.ID] = p
	}
	for _, id := range []string{hung, erred} {
		if p := byID[id]; p.Outcome != ProbeError || p.Disposition == nil || p.Disposition.As != DisposedRerun || p.Disposition.Rerun != passed {
			t.Errorf("error receipt %+v is not disposed of by %s", p, passed)
		}
	}
	if p := byID[passed]; p.Outcome != ProbePassed || p.Rerun != hung || p.ContentDigest != byID[hung].ContentDigest || !slices.Equal(p.Args, byID[hung].Args) {
		t.Errorf("the rerun %+v is not the exact probe of %s", p, hung)
	}
	if p := byID[noexecID]; p.Outcome != ProbeError || !strings.Contains(p.Reason, "exit 127") {
		t.Errorf("an unexecutable probe counted as an assertion: %+v", p)
	}
	// The provider's schema admits a rerun on its own.
	var schemaOf struct {
		Properties struct {
			Probe struct {
				Required   []string       `json:"required"`
				Properties map[string]any `json:"properties"`
			} `json:"probe"`
		} `json:"properties"`
	}
	for _, turn := range f.agents.all() {
		for _, tool := range turn.Tools {
			if tool.Name == toolCheck && turn.Kind == VerifyTurn {
				if err := json.Unmarshal(tool.InputSchema, &schemaOf); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if _, ok := schemaOf.Properties.Probe.Properties["rerun"]; !ok || len(schemaOf.Properties.Probe.Required) > 0 {
		t.Errorf("the check schema does not admit a rerun alone: %+v", schemaOf.Properties.Probe)
	}
	if !asExpectation.Refused || !strings.Contains(asExpectation.Text, "error of the Runtime") {
		t.Errorf("a Runtime error was explained away: %+v", asExpectation)
	}
	if !otherProbe.Refused {
		t.Errorf("a different probe under the same claim replaced the error: %+v", otherProbe)
	}
	if !rerunNew.Refused || !strings.Contains(rerunNew.Text, "only a Runtime error is run again") {
		t.Errorf("a passing receipt was run again: %+v", rerunNew)
	}
}

// Evidence binds claims to receipts: a receipt item carries the receipt's own
// claim, and one receipt cannot be cited for two claims. Before the review fix
// one trivial passing receipt could be cited under every claim.
func TestOneReceiptSupportsOneClaim(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var twice ToolAnswer
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") {
			id := receiptOf(t, c.must(toolCheck, probe("library/interface", "an ordinary name is stored", "probe_names.py", "PASS")))
			cov := c.coverage()
			cov[0]["evidence"] = []map[string]string{{"claim": "every input class", "receipt": id}, {"claim": "the hyphen name", "receipt": id}}
			twice = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"obligations": cov}))
			cov[0]["evidence"] = []map[string]string{{"claim": "every input class", "receipt": id}}
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": cov}))
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	r.Pass(verifySpec())
	f.finish(r, "")
	if !twice.Refused || !strings.Contains(twice.Text, "one receipt supports one claim") {
		t.Errorf("one receipt supported two claims: %+v", twice)
	}
	var claims []string
	for _, ev := range f.ev.kinds("proposal.submitted") {
		if ev.Manager != "library" || ev.Pass != "p2" {
			continue
		}
		env := ev.Data.(map[string]any)["envelope"].(envelope)
		for _, c := range env.Obligations {
			for _, item := range c.Evidence {
				claims = append(claims, item.Claim)
			}
		}
	}
	if !slices.Contains(claims, "an ordinary name is stored") || slices.Contains(claims, "every input class") {
		t.Errorf("evidence claims %q are not the receipt's", claims)
	}
}

// A disposition stands on the submission that made it: when the parent reopens
// that result, an expectation-error disproof falls with it and the receipt
// blocks again until a new accepted submission disposes of it. Before the
// review fix the disposition survived the reopening.
func TestDispositionsFallWithTheirReopenedSubmission(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var receipt string
	reopened, rejected := false, ToolAnswer{}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		review := c.turn.Kind == VerifyTurn
		switch {
		case review && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			receipt = receiptOf(t, c.must(toolCheck, probe("library/interface", "an exit code", "probe_exit.py", "FAIL")))
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": []map[string]any{{"definition": "library/interface", "status": coverOpen, "note": "assessed in the result"}}}))
			return nil
		case review && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit") && !reopened:
			c.must(toolReport, c.envelope(kindResult, map[string]any{"dispositions": []any{map[string]any{"receipt": receipt, "as": DisposedExpectationError, "reason": "the model fixes no exit code", "evidence": "library/interface"}}}))
			return nil
		case review && c.turn.Manager == "library" && strings.Contains(c.turn.Prompt, "reopened your accepted result"):
			rejected = c.result(nil)
			c.must(toolReport, c.envelope(kindResult, map[string]any{"dispositions": []any{map[string]any{"receipt": receipt, "as": DisposedExpectationError, "reason": "the model fixes no exit code; checked again", "evidence": "library/interface"}}}))
			return nil
		case review && c.turn.Manager == "." && strings.HasPrefix(c.now(), "integrate") && !reopened:
			reopened = true
			c.must(toolReport, map[string]any{"kind": kindReopen, "child": "library", "reason": "the disproof needs a second look"})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	res := r.Pass(verifySpec())
	out := f.finish(r, "")
	if !reopened || res.Verdict != Satisfied || out.Status != Verified {
		t.Fatalf("reopened %v, verdict %s, status %s: %s", reopened, res.Verdict, out.Status, out.Reason)
	}
	if !rejected.Refused || !strings.Contains(rejected.Text, "no disposition") {
		t.Errorf("the reopened result's disposition still counted: %+v", rejected)
	}
	voided := false
	for _, ev := range f.ev.kinds("probe.disposed") {
		if _, ok := ev.Data.(map[string]any)["voided"]; ok {
			voided = true
		}
	}
	if !voided || out.Probes[0].Disposition == nil || !strings.Contains(out.Probes[0].Disposition.Reason, "checked again") {
		t.Errorf("voided %v, disposition %+v", voided, out.Probes[0].Disposition)
	}
}

// A receipt supports only the stand it ran on: a review cannot cite a receipt
// of an earlier candidate, and a work submission cannot cite a live receipt
// when its area changed afterwards; the unchanged one is accepted.
func TestStaleReceiptsSupportNothing(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var oldReceipt string
	var staleReview ToolAnswer
	staleWork := false
	verifies, works := 0, 0
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		review := c.turn.Kind == VerifyTurn
		switch {
		case !review && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			works++
			if verifies > 0 {
				// A later implementation changes the candidate.
				c.write("library/a.go", "a corrected")
				break
			}
			if strings.Contains(c.turn.Prompt, "did not pass the File check") && strings.Contains(c.turn.Prompt, "Probe again after your last change") {
				staleWork = true
			}
			live := receiptOf(t, c.must(toolCheck, probe("library/interface", "the program", "probe_live.py", "PASS")))
			if works == 1 {
				c.write("library/a.go", "a changed after the probe")
			}
			c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, live)}))
			return nil
		case review && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on"):
			verifies++
			if verifies == 1 {
				oldReceipt = receiptOf(t, c.must(toolCheck, probe("library/interface", "the program", "probe_v.py", "PASS")))
				c.must(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, oldReceipt)}))
				return nil
			}
			staleReview = c.tool(toolReport, c.envelope(kindBasis, map[string]any{"obligations": withEvidence(c, oldReceipt)}))
		}
		c.protocol()
		return nil
	}
	r := f.open()
	for _, spec := range []PassSpec{implementSpec(), verifySpec(), implementSpec(), verifySpec()} {
		if res := r.Pass(spec); res.Stopped {
			t.Fatalf("stopped in %s", res.Record.ID)
		}
	}
	out := f.finish(r, "")
	if out.Status != Verified || works != 3 {
		t.Fatalf("status %s after %d work turns of library: %s", out.Status, works, out.Reason)
	}
	if !staleWork {
		t.Errorf("a live receipt supported a submission whose area changed after it")
	}
	if !staleReview.Refused || !strings.Contains(staleReview.Text, "stale receipt") {
		t.Errorf("a receipt of an earlier candidate supported this one: %+v", staleReview)
	}
	accepted := 0
	for _, ev := range f.ev.kinds("basis.accepted") {
		if ev.Manager == "library" && ev.Pass == "p1" {
			accepted++
		}
	}
	if accepted != 1 {
		t.Errorf("library's basis with an unchanged live receipt was accepted %d times", accepted)
	}
}

// A probe keeps time for the turn's report: late in a turn it is refused
// instead of running into the turn's timeout, which would fail the whole turn
// (as in the first V1 evaluation, where slow probes timed out reviewer turns).
func TestProbeKeepsTimeForTheTurnsReport(t *testing.T) {
	f := probeFixture(t)
	var mu sync.Mutex
	var offset time.Duration
	base := time.Now()
	f.cfg.Clock = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return base.Add(offset)
	}
	var late, early ToolAnswer
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && late.Text == "" {
			mu.Lock()
			offset += c.turn.Timeout - c.turn.Timeout/8
			mu.Unlock()
			late = c.tool(toolCheck, probe("library/interface", "a late claim", "probe_late.py", "PASS"))
			mu.Lock()
			offset = 0
			mu.Unlock()
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" && strings.HasPrefix(c.now(), "work on") && early.Text == "" && late.Text != "" {
			early = c.tool(toolCheck, probe("library/interface", "an early claim", "probe_early.py", "PASS"))
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(implementSpec())
	r.Pass(verifySpec())
	f.finish(r, "")
	if !late.Refused || !strings.Contains(late.Text, "Too little of this turn is left") {
		t.Errorf("a probe ran into the end of its turn: %+v", late)
	}
	if early.Refused || !strings.Contains(early.Text, "after 0 seconds") {
		t.Errorf("a probe with time left was refused: %+v", early)
	}
}
