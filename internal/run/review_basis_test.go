package run

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

// These scripted Agents test an enforced exposure boundary, not their ability
// to discover natural-language conditions or judge arbitrary assertions.
func TestModelFirstBasisPrecedesCandidateAndAuthorEvidence(t *testing.T) {
	f := probeFixture(t)
	f.cfg.OwnerRequest = "AUTHOR_CANDIDATE_HINT"
	var mu sync.Mutex
	var blind, opened bool
	var home string
	hasCandidate := func(workspace string) bool {
		f.ws.mu.Lock()
		defer f.ws.mu.Unlock()
		volume := f.ws.byName[workspace]
		if volume == nil {
			t.Errorf("unknown mounted workspace %s", workspace)
			return false
		}
		_, exists := volume.files["library/a.go"]
		return exists
	}
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
			c.must(toolSend, map[string]any{"to": "library", "kind": sendInfo, "text": "MODEL_QUESTION: derive allowed classes from the Definition"})
		}
		if c.turn.Kind == VerifyTurn && c.turn.Manager == "library" {
			if strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
				blind, home = true, c.turn.Home
				if hasCandidate(c.turn.Workspace) || !c.turn.ReadOnly || len(c.turn.Mounts) > 0 {
					t.Fatal("the model-first turn received the artifact candidate")
				}
				if strings.Contains(c.turn.Prompt, "CANDIDATE_HINT") || slices.ContainsFunc(c.turn.Tools, func(s ToolSpec) bool { return s.Name == toolCheck || s.Name == toolDelegate }) {
					t.Fatal("candidate hint or candidate tools exposed before accepted basis")
				}
				for _, topic := range []string{"checks", "probe", "registry", "findings", "plan", "state", "questions"} {
					if a := c.tool(toolLookUp, map[string]any{"topic": topic, "id": "R1"}); !a.Refused {
						t.Errorf("pre-basis lookup %s exposed candidate/author information", topic)
					}
				}
				for _, tool := range []string{toolCheck, toolDelegate} {
					if a := c.tool(tool, map[string]any{}); !a.Refused {
						t.Errorf("direct pre-basis %s invocation was admitted", tool)
					}
				}
				brief := c.must(toolLookUp, map[string]any{"topic": "brief"}).Text
				if strings.Contains(brief, "AUTHOR_CANDIDATE_HINT") || strings.Contains(brief, "Required checks on this candidate:") {
					t.Fatal("brief lookup bypassed candidate withholding")
				}
				c.must(toolLookUp, map[string]any{"topic": "definition", "id": "library/interface"})
				if !strings.Contains(c.turn.Prompt, "MODEL_QUESTION") {
					t.Error("ordinary model communication was withheld")
				}
			} else if strings.HasPrefix(c.now(), "submit your complete result") {
				opened = true
				if !hasCandidate(c.turn.Workspace) || c.turn.Home != home || c.turn.Resume == "" {
					t.Fatal("candidate was not opened in the same independent conversation/home")
				}
				for _, text := range []string{"model-derived cases; candidate assessment pending", "AUTHOR_CANDIDATE_HINT", "Required checks on this candidate:"} {
					if !strings.Contains(c.turn.Prompt, text) {
						t.Errorf("post-basis assignment lacks %q", text)
					}
				}
				if a := c.tool(toolReport, map[string]any{"kind": kindRevise, "reason": "replace expectations after reading candidate"}); !a.Refused {
					t.Fatal("accepted precommitment replaced after candidate access")
				}
				c.must(toolLookUp, map[string]any{"topic": "registry"})
			}
		}
		if c.turn.Kind == VerifyTurn && strings.HasPrefix(c.now(), "coordinate model questions") {
			// Accepting the final child basis may open the global barrier, but
			// this already-running model-only turn must remain blind.
			for _, id := range c.pendingDecisions() {
				c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
			}
			if hasCandidate(c.turn.Workspace) {
				t.Error("coordination turn mounted the candidate")
			}
			if a := c.tool(toolLookUp, map[string]any{"topic": "checks"}); !a.Refused {
				t.Error("last acceptance exposed candidate information mid-turn")
			}
			if a := c.tool(toolReport, c.envelope(kindResult, nil)); !a.Refused {
				t.Error("model-only turn submitted an artifact result")
			}
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	spec := verifySpec()
	spec.ModelBasis = true
	res := r.Pass(spec)
	out := f.finish(r, "")
	if res.Stopped || out.Status != Verified || !blind || !opened {
		t.Fatalf("status %s: %s; blind %v, opened %v", out.Status, out.Reason, blind, opened)
	}
	accepted := f.ev.index("basis.accepted", "library", "", nil)
	access := f.ev.index("review.candidate_opened", "library", "", nil)
	if accepted < 0 || access <= accepted {
		t.Fatalf("candidate access %d preceded basis acceptance %d", access, accepted)
	}
	for _, manager := range teamPaths {
		if basis := f.ev.index("basis.accepted", manager, "", nil); basis < 0 || access <= basis {
			t.Errorf("candidate opened before required %s basis: accepted %d, access %d", manager, basis, access)
		}
	}
}

func TestSubmittedModelBasisDoesNotOpenCandidateInTheSameTurn(t *testing.T) {
	f := probeFixture(t)
	var checked bool
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
			c.must(toolReport, c.envelope(kindBasis, nil))
			if a := c.tool(toolCheck, probe("library/interface", "too early", "probe.py", "PASS")); !a.Refused {
				t.Error("merely submitted basis opened candidate probes")
			}
			checked = true
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	spec := verifySpec()
	spec.ModelBasis = true
	r.Pass(spec)
	out := f.finish(r, "")
	if !checked || out.Status != Verified {
		t.Fatalf("status %s: %s; submission checked %v", out.Status, out.Reason, checked)
	}
}

func TestFailedModelBasisTurnNeverOpensTheCandidate(t *testing.T) {
	f := probeFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Manager == "library" && strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
			c.must(toolReport, c.envelope(kindBasis, nil))
			return errTest
		}
		c.protocol()
		return nil
	}
	r := f.open()
	spec := verifySpec()
	spec.ModelBasis = true
	r.Pass(spec)
	out := f.finish(r, "")
	if out.Status != Incomplete || f.ev.index("review.candidate_opened", "library", "", nil) >= 0 || f.ev.index("basis.accepted", "library", "", nil) >= 0 {
		t.Fatalf("failed provisional basis became accepted: %s %s", out.Status, out.Reason)
	}
}

func TestModelBasisRevisionBeforeGlobalExposureInvalidatesConsumers(t *testing.T) {
	f := probeFixture(t)
	var revised bool
	libraryAccepted := make(chan struct{})
	var release sync.Once
	f.agents.script = func(c *call) error {
		// Establish one parked accepted child and one stopped checked child,
		// without racing a still-running child whose reset refusal is normal.
		if c.turn.Manager == "lending" && strings.Contains(c.turn.Prompt, "MODEL-FIRST BASIS:") {
			if err := c.wait(libraryAccepted); err != nil {
				return err
			}
		}
		if c.turn.Manager == "." && !revised && strings.HasPrefix(c.now(), "coordinate model questions") && len(c.pendingDecisions()) > 0 && f.ev.index("basis.accepted", "library", "", nil) >= 0 {
			c.must(toolReport, map[string]any{"kind": kindRevise, "reason": "clarified model-derived integration conditions before candidate access"})
			revised = true
			return nil
		}
		c.protocol()
		if c.turn.Manager == "." && f.ev.index("basis.accepted", "library", "", nil) >= 0 {
			release.Do(func() { close(libraryAccepted) })
		}
		return nil
	}
	r := f.open()
	spec := verifySpec()
	spec.ModelBasis = true
	r.Pass(spec)
	out := f.finish(r, "")
	if !revised || out.Status != Verified {
		t.Fatalf("revision %v, status %s: %s", revised, out.Status, out.Reason)
	}
	if f.ev.index("basis.invalidated", ".", "", nil) < 0 {
		t.Fatal("old accepted basis was not invalidated")
	}
	if len(f.ev.kinds("proposal.invalidated")) == 0 {
		t.Fatal("old dependent proposal was not invalidated")
	}
	f.ev.mu.Lock()
	defer f.ev.mu.Unlock()
	lastBasis := map[string]int{}
	for index, ev := range f.ev.events {
		if ev.Kind == "basis.invalidated" && ev.Manager == "." {
			lastBasis = map[string]int{}
		}
		if ev.Kind == "basis.accepted" {
			lastBasis[ev.Manager] = index
		}
		if ev.Kind == "review.candidate_opened" {
			for _, manager := range teamPaths {
				if accepted, ok := lastBasis[manager]; !ok || accepted >= index {
					t.Errorf("candidate opened without current accepted %s basis", manager)
				}
			}
		}
	}
}
