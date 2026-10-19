package run

import (
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func toolNames(turn Turn) []string {
	var names []string
	for _, tool := range turn.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func TestCustomMandateAndVerifyGuidanceReplaceTheApplyTexts(t *testing.T) {
	f := newFixture(t)
	f.cfg.Mandate = "Extract the model candidate from the fixed artifacts."
	f.cfg.VerifyGuidance = "Blocking means the model candidate misstates an artifact."
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	var executes, verifies int
	for _, turn := range f.agents.turns {
		if turn.Kind == ExecuteTurn && turn.Resume != "" {
			continue // a resumed turn carries only new items, not the mandate
		}
		contains(t, "prompt of "+turn.ID, turn.Prompt, f.cfg.Mandate)
		if strings.Contains(turn.Prompt, mandate) {
			t.Errorf("prompt of %s keeps the Apply mandate", turn.ID)
		}
		switch turn.Kind {
		case ExecuteTurn:
			executes++
			contains(t, "Execute prompt", turn.Prompt, "Carry out your responsibility now")
		case VerifyTurn:
			verifies++
			contains(t, "Verify prompt", turn.Prompt, "write build or test output only under /tmp. "+f.cfg.VerifyGuidance+"\n")
			if strings.Contains(turn.Prompt, extraDepthNote) {
				t.Errorf("Verify prompt of %s carries Apply's extra-depth sentence", turn.ID)
			}
			if strings.Contains(turn.Prompt, blockingGuidance) || strings.Contains(turn.Prompt, "Blocking means a concrete model violation") {
				t.Errorf("Verify prompt of %s keeps the Apply guidance", turn.ID)
			}
		}
	}
	if executes == 0 || verifies == 0 {
		t.Fatalf("turns: %d Execute, %d Verify", executes, verifies)
	}
}

func TestEmptyMandateAndGuidanceKeepTheApplyTexts(t *testing.T) {
	f := newFixture(t)
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	for _, turn := range f.agents.turns {
		if turn.Kind == ExecuteTurn && turn.Resume != "" {
			continue
		}
		contains(t, "prompt of "+turn.ID, turn.Prompt, mandate)
		switch turn.Kind {
		case ExecuteTurn:
			contains(t, "Execute prompt", turn.Prompt, "everything left under /work is captured and must be an accounted artifact.\n")
			if strings.Contains(turn.Prompt, "The tool check") {
				t.Errorf("Execute prompt of %s mentions the check tool", turn.ID)
			}
		case VerifyTurn:
			contains(t, "Verify prompt", turn.Prompt, "write build or test output only under /tmp. "+blockingGuidance+" "+extraDepthNote+"\n")
		}
	}
}

func TestCheckToolIsOfferedOnlyToExecuteTurnsAndOnlyWithCheckTool(t *testing.T) {
	for _, offered := range []bool{false, true} {
		f := newFixture(t)
		f.cfg.CheckTool = offered
		f.agents.script = func(c *call) error {
			a := c.tool(toolCheck, nil)
			switch {
			case c.turn.Kind == VerifyTurn && !a.Refused:
				t.Errorf("Verify turn %s was served check", c.turn.ID)
			case c.turn.Kind == ExecuteTurn && a.Refused == offered:
				t.Errorf("offered %t, Execute turn %s got refused %t: %s", offered, c.turn.ID, a.Refused, a.Text)
			}
			return c.defaultScript()
		}
		out := f.run()
		if out.Status != Verified {
			t.Fatalf("%s %s", out.Status, out.Reason)
		}
		for _, turn := range f.agents.turns {
			has := slices.Contains(toolNames(turn), toolCheck)
			if want := offered && turn.Kind == ExecuteTurn; has != want {
				t.Errorf("turn %s tools %v, check offered %t, want %t", turn.ID, toolNames(turn), has, want)
			}
			if turn.Kind == ExecuteTurn && turn.Resume == "" && strings.Contains(turn.Prompt, "The tool check") != offered {
				t.Errorf("Execute prompt of %s mentions check: %t, offered %t", turn.ID, !offered, offered)
			}
			if turn.Kind == VerifyTurn && strings.Contains(turn.Prompt, "The tool check") {
				t.Errorf("Verify prompt of %s mentions check", turn.ID)
			}
		}
		if n := f.ev.count("check.requested"); (n > 0) != offered {
			t.Errorf("offered %t, %d check.requested events", offered, n)
		}
	}
}

func TestCheckAnswersWithTheCallersFindingsFirst(t *testing.T) {
	f := newFixture(t)
	f.cfg.CheckTool = true
	f.cfg.Account = func(tree snapshot.Tree) []Finding {
		if _, ok := tree.Lookup(mustPath(t, "lending/reminders/ok.go")); ok {
			return nil
		}
		return []Finding{
			{Owner: "library", Obligation: "library obligation", Observed: "library observed"},
			{Owner: "lending/reminders", Obligation: "first own obligation", Observed: "first own observed"},
			{Owner: "nobody", Obligation: "stray obligation", Observed: "stray observed"},
			{Owner: "lending/reminders", Obligation: "second own obligation", Observed: "second own observed"},
		}
	}
	var own, all string
	f.agents.script = func(c *call) error {
		if c.is(1, "lending/reminders", ExecuteTurn) {
			own = c.tool(toolCheck, nil).Text
			all = c.tool(toolCheck, map[string]any{"scope": "all"}).Text
			if a := c.tool(toolCheck, map[string]any{"scope": "everything"}); !a.Refused {
				t.Errorf("an unknown scope was served: %s", a.Text)
			}
			if a := c.tool(toolCheck, map[string]any{"path": "x"}); !a.Refused {
				t.Errorf("an unknown argument was served: %s", a.Text)
			}
			c.write("lending/reminders/ok.go", "ok")
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	contains(t, "own answer", own, "Yours:", "first own obligation: first own observed", "second own obligation: second own observed", "2 finding(s) of other Managers")
	if strings.Contains(own, "library observed") || strings.Contains(own, "stray observed") {
		t.Errorf("default scope lists findings of others:\n%s", own)
	}
	if strings.Index(own, "first own observed") > strings.Index(own, "second own observed") || strings.Index(own, "second own observed") > strings.Index(own, "2 finding(s)") {
		t.Errorf("own findings do not come first:\n%s", own)
	}
	contains(t, "all answer", all, "Yours:", "Of other Managers:", "[library] library obligation: library observed", "[.] stray obligation: stray observed (reported for unknown owner \"nobody\")")
	if strings.Index(all, "second own observed") > strings.Index(all, "Of other Managers:") {
		t.Errorf("own findings do not come first:\n%s", all)
	}

	events := f.ev.kinds("check.requested")
	if len(events) != 2 {
		t.Fatalf("check.requested events %+v", events)
	}
	for _, ev := range events {
		data, _ := ev.Data.(map[string]any)
		if ev.Manager != "lending/reminders" || data["own"] != 2 || data["others"] != 2 {
			t.Errorf("event %+v", ev)
		}
	}
	if out.Rounds[0].Green != true || len(out.Findings) != 0 {
		t.Errorf("a check use must not raise findings: %+v %+v", out.Rounds, out.Findings)
	}
}

func TestCheckPeeksWhileWritersRunAndRefusesAnUncertainPeek(t *testing.T) {
	f := newFixture(t)
	f.cfg.CheckTool = true
	f.cfg.Account = func(tree snapshot.Tree) []Finding {
		if e, ok := tree.Lookup(mustPath(t, "library/new.go")); ok && string(e.Bytes()) == "package library" {
			return []Finding{{Owner: "library", Obligation: "new file is registered", Observed: "library/new.go is unexplained"}}
		}
		return nil
	}
	var seen, refused, retried ToolAnswer
	f.agents.script = func(c *call) error {
		if c.is(1, "library", ExecuteTurn) {
			c.write("library/new.go", "package library")
			// The turn mounts the shared volume for writing: Peek still observes it.
			seen = c.tool(toolCheck, nil)
			f.ws.mu.Lock()
			f.ws.peekUncertain = true
			f.ws.mu.Unlock()
			refused = c.tool(toolCheck, nil)
			f.ws.mu.Lock()
			f.ws.peekUncertain = false
			f.ws.mu.Unlock()
			c.write("library/new.go", "package library // registered")
			retried = c.tool(toolCheck, nil)
		}
		return c.defaultScript()
	}
	out := f.run()
	if seen.Refused || !strings.Contains(seen.Text, "library/new.go is unexplained") {
		t.Errorf("peek during a write: %+v", seen)
	}
	if !refused.Refused || !strings.Contains(refused.Text, "the volume changed while it was observed") {
		t.Errorf("uncertain peek: %+v", refused)
	}
	if retried.Refused {
		t.Errorf("a later peek was refused: %+v", retried)
	}
	// The refusal is an answer, not a failure of the run.
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	var reasons int
	for _, ev := range f.ev.kinds("check.requested") {
		if data, _ := ev.Data.(map[string]any); data["refused"] != nil {
			reasons++
		}
	}
	if reasons != 1 {
		t.Errorf("%d refused check.requested events", reasons)
	}
}
