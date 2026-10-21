package apply_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The expectations below are read off the library M1 model and the documented
// selection rule (own Definitions, their direct outgoing links, direct local
// incoming links and the type contracts of everything selected; no further
// business links), not off the selector or the renderer.
const (
	reminders           = "lending/reminders"
	assessOverdueEnd    = "a loan already returned in current state remains excluded even when assessing an earlier date."
	loanPolicyEnd       = "A lower limit neither deletes existing loans nor forces their return; while at or above that limit, the member cannot start another loan."
	assessOverdueLinks  = "links to: library/inputs, library/library, lending/loan, lending/loan-policy, members/member"
	loanPolicyLinks     = "links to: lending/borrow, lending/maxOpenLoans, lending/loan, lending/loanPeriodDays, lending/return"
	remindersLinkReason = "included because: lending/reminders/assess-overdue links to it"
)

// remindersFull is what the Brief of lending/reminders states in full.
var remindersFull = []string{
	"lending/reminders/assess-overdue",
	// direct outgoing links
	"library/inputs", "library/library", "lending/loan", "lending/loan-policy", "members/member",
	// direct local incoming links
	"library/callable-interface", "library/realization",
	// type contracts of the selection, closed over the contracts' own types:
	// the Kind and Property contracts of Core with their properties
	"software/UseCase", "software/Rule", "software/Concept", "lending/LendingPolicy", "lending/maxOpenLoans", "lending/loanPeriodDays",
	"markitect/core/Kind", "markitect/core/Property", "markitect/core/name", "markitect/core/properties", "markitect/core/target",
	"markitect/core/type", "markitect/core/values", "markitect/core/minCount", "markitect/core/maxCount",
}

// remindersExcluded are linked only from selected context, a second hop.
var remindersExcluded = []string{"lending/borrow", "lending/return", "catalog/availability", "catalog/copy", "members/eligibility", "members/register-member"}

// blockEnd is where a stated Definition ends: a blank line before the next
// Definition heading or Brief section. A body may hold blank lines itself.
var blockEnd = regexp.MustCompile(`\n\n(?:[a-z0-9][a-zA-Z0-9._/-]*/[a-zA-Z0-9._-]+ \(|Included Definitions|Required checks|Your own Definitions are yours|Never realize|Configured )`)

// block returns the block a Brief or lookup states for one Definition: from
// its heading to the next Definition or section.
func block(text, id string) (string, bool) {
	start := strings.Index(text, "\n"+id+" (")
	if start < 0 {
		return "", false
	}
	rest := text[start+1:]
	if end := blockEnd.FindStringIndex(rest); end != nil {
		rest = rest[:end[0]]
	}
	return rest, true
}

// missing lists what a stated block lacks of its expected content.
func missing(text, id string, want ...string) []string {
	b, ok := block(text, id)
	if !ok {
		return []string{id + " is not stated in full"}
	}
	var out []string
	for _, w := range want {
		if !strings.Contains(b, w) {
			out = append(out, fmt.Sprintf("%s lacks %q", id, w))
		}
	}
	return out
}

// loanPolicyContent is what an included loan-policy must keep: its decisive
// final sentence, its numeric values, its resolved links, its owner and why it
// is included.
func loanPolicyContent(text string) []string {
	return missing(text, "lending/loan-policy", loanPolicyEnd, "maxOpenLoans: 5", "loanPeriodDays: 14", loanPolicyLinks, "(owned by lending)", remindersLinkReason)
}

func teamByPath(cfg run.Config) map[string]run.Manager {
	out := map[string]run.Manager{}
	for _, m := range cfg.Team {
		out[m.Path] = m
	}
	return out
}

// A Manager's Brief states its own obligations and the selected contracts in
// full, with every condition, value, resolved link and owner, and nothing of
// the second hop; a cut or changed packet fails the same check.
func TestBriefStatesTheSelectedContextInFull(t *testing.T) {
	req, _ := layoutRequest(t)
	cfg, err := apply.BuildApplyConfig(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	m := teamByPath(cfg)[reminders]
	brief := m.Brief
	if !slices.Equal(m.Obligations, []string{"lending/reminders/assess-overdue"}) {
		t.Fatalf("own obligations %v", m.Obligations)
	}
	for _, id := range remindersFull {
		if _, ok := block(brief, id); !ok {
			t.Errorf("%s is not stated in full", id)
		}
	}
	for _, id := range remindersExcluded {
		if _, ok := block(brief, id); ok {
			t.Errorf("%s, a second hop, is stated", id)
		}
	}
	if gaps := append(missing(brief, "lending/reminders/assess-overdue", assessOverdueEnd, assessOverdueLinks), loanPolicyContent(brief)...); len(gaps) > 0 {
		t.Fatalf("the Brief lost content: %v", gaps)
	}
	if own, _ := block(brief, "lending/reminders/assess-overdue"); strings.Contains(own, "owned by") || strings.Contains(own, "included because") {
		t.Errorf("the own obligation is labeled as foreign context: %s", own)
	}
	for _, id := range []string{"library/callable-interface", "library/realization"} {
		if gaps := missing(brief, id, "(owned by library)", "included because: it links to lending/reminders/assess-overdue"); len(gaps) > 0 {
			t.Errorf("incoming link: %v", gaps)
		}
	}
	for name, faulty := range map[string]string{
		"cut final sentence": strings.Replace(brief, " while at or above that limit, the member cannot start another loan.", "", 1),
		"changed value":      strings.Replace(brief, "maxOpenLoans: 5", "maxOpenLoans: 6", 1),
		"lost links":         strings.Replace(brief, loanPolicyLinks, "", 1),
		"lost owner":         strings.Replace(brief, "lending/loan-policy (lending/LendingPolicy): loan-policy (owned by lending)", "lending/loan-policy (lending/LendingPolicy): loan-policy", 1),
	} {
		if faulty == brief || len(loanPolicyContent(faulty)) == 0 {
			t.Errorf("the check misses a %s", name)
		}
	}
}

// localDefinitions reads every local Definition identity of the library M1
// model from its source files.
func localDefinitions(t *testing.T) []string {
	t.Helper()
	root := "../../docs/design/canonical-model/library-example/m1/model"
	var ids []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		pkg, _ := filepath.Rel(root, filepath.Dir(path))
		for _, line := range strings.Split(string(raw), "\n") {
			if id, ok := strings.CutPrefix(strings.TrimSpace(line), "id: "); ok {
				ids = append(ids, filepath.ToSlash(pkg)+"/"+id)
			}
		}
		return nil
	})
	if err != nil || len(ids) < 20 {
		t.Fatalf("model files: %v %v", ids, err)
	}
	slices.Sort(ids)
	return ids
}

// Root's project inventory lists every local Definition with its owner for
// reference only; the coverage record keeps those apart from what is stated in
// full, and every Manager's record matches its Brief.
func TestTheCoverageRecordSeparatesFullTextFromTheInventory(t *testing.T) {
	req, _ := layoutRequest(t)
	cfg, binding, err := apply.BuildApplyBinding(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	team := teamByPath(cfg)
	local := localDefinitions(t)
	root := team["."].Brief
	inventory, _, _ := strings.Cut(root[strings.Index(root, "Project obligations"):], "\n\n")
	for _, id := range local {
		owner := id[:strings.LastIndex(id, "/")]
		if !strings.Contains(inventory, "- "+id+" (") || !strings.Contains(inventory, "owned by "+owner+"\n") {
			t.Errorf("Root's inventory lacks %s owned by %s", id, owner)
		}
	}
	for _, mb := range binding.Managers {
		brief := team[mb.Manager].Brief
		if mb.Context.Brief != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(brief))) {
			t.Errorf("%s: the record does not bind its Brief", mb.Manager)
		}
		for _, id := range mb.Context.Full {
			if _, ok := block(brief, id); !ok {
				t.Errorf("%s: %s is recorded as full but not stated", mb.Manager, id)
			}
		}
		for _, id := range mb.Context.Reference {
			if _, ok := block(brief, id); ok || slices.Contains(mb.Context.Full, id) {
				t.Errorf("%s: %s is recorded as reference but stated in full", mb.Manager, id)
			}
		}
		switch mb.Manager {
		case ".":
			full := slices.Clone(mb.Context.Full)
			if !slices.Equal(full, []string{"library/callable-interface"}) || len(mb.Context.Reference) != len(local)-1 {
				t.Errorf("Root: full %v, %d references of %d", full, len(mb.Context.Reference), len(local))
			}
		case reminders:
			full := slices.Sorted(slices.Values(mb.Context.Full))
			if want := slices.Sorted(slices.Values(remindersFull)); !slices.Equal(full, want) || len(mb.Context.Reference) != 0 {
				t.Errorf("reminders: full %v, want %v; references %v", full, want, mb.Context.Reference)
			}
		}
	}
}

// The Agents port receives the packet: a new work conversation and the first
// turn of every fresh Verify context carry the recipient's own Brief, a
// continued conversation relies on its history, a compacted one or one the
// provider did not continue gets the Brief again, and a reviewer never continues an author's conversation. A
// lookup answers from the bound model in full, and run evidence relates each
// turn's delivery and each lookup to its payload.
func TestTheBriefReachesEveryFreshAndRehydratedContext(t *testing.T) {
	req := strategyRequest(t, "direct")
	cfg, err := apply.BuildApplyConfig(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	team := teamByPath(cfg)
	agents := &preservingAgents{t: t, obligations: map[string][]string{},
		lookups: map[string][]string{reminders: {"lending/loan-policy"}}, compacted: map[string]bool{"lending": true}, forgets: map[string]bool{".": true}}
	for _, m := range cfg.Team {
		agents.obligations[m.Path] = m.Obligations
	}
	evidence := &memoryEvidence{}
	req.Agents, req.Workspaces, req.Runner, req.Evidence = agents, &treeWorkspaces{}, &passingRunner{}, evidence
	req.Publisher = &fakePublisher{receipt: apply.PublicationReceipt{ID: "run-1", Outcome: apply.PublicationNoChange}}
	result, err := apply.Apply(context.Background(), req)
	if err != nil || result.Outcome.Status != run.Verified {
		t.Fatalf("outcome %s %s: %v", result.Outcome.Status, result.Outcome.Reason, err)
	}
	delivery := map[string]map[string]any{}
	for _, e := range evidence.events {
		if data, _ := e.Data.(map[string]any); e.Kind == "turn.start" {
			delivery[e.Turn] = data
		}
	}
	header := func(path string) string {
		if path == "." {
			return "You are the Manager of Root."
		}
		return "You are the Manager of Package " + path + "."
	}
	seen := map[string]int{}
	rehydrated := map[string]bool{}
	works := map[string]bool{}
	for _, turn := range agents.turns {
		if turn.Kind == run.ExecuteTurn {
			works["conv-"+turn.Home] = true
		}
	}
	for i, turn := range agents.turns {
		brief := strings.TrimSpace(team[turn.Manager].Brief)
		carries := strings.Contains(turn.Prompt, brief) && strings.Contains(turn.Prompt, header(turn.Manager))
		data := delivery[turn.ID]
		key := string(turn.Kind) + " " + turn.Home
		seen[key]++
		for _, other := range cfg.Team {
			if other.Path != turn.Manager && strings.Contains(turn.Prompt, header(other.Path)) {
				t.Errorf("turn of %s carries the Brief of %s", turn.Manager, other.Path)
			}
		}
		if data["prompt"] != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(turn.Prompt))) {
			t.Errorf("turn %d of %s: the recorded prompt digest is not the delivered prompt's", i, turn.Manager)
		}
		switch {
		case seen[key] == 1:
			if turn.Resume != "" || !carries || data["context"] != "brief" {
				t.Errorf("first %s turn of %s: resume %q, carries %v, context %v", turn.Kind, turn.Manager, turn.Resume, carries, data["context"])
			}
		case data["context"] == "rehydrated":
			rehydrated[turn.Manager] = true
			if turn.Manager != "lending" && turn.Manager != "." || turn.Resume == "" || !carries {
				t.Errorf("rehydrated turn of %s: resume %q, carries %v", turn.Manager, turn.Resume, carries)
			}
		default:
			if turn.Resume == "" || carries || data["context"] != "history" {
				t.Errorf("continued %s turn of %s: resume %q, carries %v, context %v", turn.Kind, turn.Manager, turn.Resume, carries, data["context"])
			}
		}
		if turn.Kind == run.VerifyTurn && (works[turn.Resume] || strings.HasSuffix(turn.Home, "-m")) {
			t.Errorf("reviewer of %s continues an author's conversation or home: %q %q", turn.Manager, turn.Resume, turn.Home)
		}
	}
	if !rehydrated["lending"] || !rehydrated["."] {
		t.Errorf("a compacted or lost conversation never received its Brief again: %v", rehydrated)
	}
	answer := agents.answers[reminders+" lending/loan-policy"]
	if gaps := loanPolicyContent(strings.Replace(answer, "(context owned by lending, bound model):\n", "(owned by lending)\n"+remindersLinkReason+"\n", 1)); len(gaps) > 3 || !strings.Contains(answer, "bound model") || !strings.Contains(answer, loanPolicyEnd) || !strings.Contains(answer, "maxOpenLoans: 5") || !strings.Contains(answer, loanPolicyLinks) {
		t.Errorf("the lookup lost content: %q", answer)
	}
	if brief := agents.answers[reminders+" brief"]; !strings.Contains(brief, strings.TrimSpace(team[reminders].Brief)) {
		t.Errorf("look_up brief does not return the Brief: %q", brief)
	}
	var recorded bool
	for _, e := range evidence.events {
		if data, _ := e.Data.(map[string]any); e.Kind == "lookup.definition" && data["id"] == "lending/loan-policy" {
			recorded = data["answer"] == fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(answer))) && e.Manager == reminders && e.Turn != ""
		}
	}
	if !recorded {
		t.Error("the lookup is not recorded with its answer's digest")
	}
}
