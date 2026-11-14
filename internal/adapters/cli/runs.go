package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// runUsage is the run command's usage.
const runUsage = `usage:
  markitect run start KIND [--strategy STRATEGY] [--trigger owner|routine|claude] [--repo DIR]
  markitect run slice [PACKAGES] [--repo DIR]
  markitect run end OUTCOME [--rounds N] [--repo DIR]`

// Triggers are what may start a run: the owner, a routine, or Claude on its own.
var triggers = []string{"owner", "routine", "claude"}

// runs appends what a driver says of a run to the project's history (DEC-114):
// that it starts, that a slice or a round starts, and that it ends. The driver
// gives only what the tool cannot know; the tool adds the time and what the
// list, the record and the harness configuration say then.
func (c CLI) runs(ctx context.Context, args []string) (int, error) {
	var o options
	var strategy, trigger, rounds string
	fs := c.flags("run", &o)
	fs.StringVar(&strategy, "strategy", "", "with start: how the run works, sequential, parallel or the name of a custom workflow")
	fs.StringVar(&trigger, "trigger", "owner", "with start: what started the run, owner, routine or claude")
	fs.StringVar(&rounds, "rounds", "", "with end: how many rounds the run went")
	all, err := parse(fs, args)
	if err != nil {
		return Misused, err
	}
	if len(all) == 0 {
		return Misused, fmt.Errorf("names what the run does: start, slice or end\n%s", runUsage)
	}
	what, positional := all[0], all[1:]
	allowed := map[string][]string{"start": {"repo", "strategy", "trigger"}, "slice": {"repo"}, "end": {"repo", "rounds"}}
	if _, ok := allowed[what]; !ok {
		return Misused, fmt.Errorf("run starts, slices or ends, not %q\n%s", what, runUsage)
	}
	var stray []string
	fs.Visit(func(f *flag.Flag) {
		if !slices.Contains(allowed[what], f.Name) {
			stray = append(stray, "--"+f.Name)
		}
	})
	if len(stray) > 0 {
		return Misused, fmt.Errorf("run %s takes no %s\n%s", what, strings.Join(stray, " or "), runUsage)
	}
	var ev run.Event
	switch what {
	case "start":
		ev, err = c.runStart(ctx, o.repo, positional, strategy, trigger)
	case "slice":
		ev, err = c.runSlice(ctx, o.repo, positional)
	case "end":
		ev, err = c.runEnd(ctx, o.repo, positional, rounds)
	}
	if err != nil {
		return Misused, err
	}
	if err := c.Record.Append(ctx, o.repo, ev); err != nil {
		return Misused, err
	}
	fmt.Fprintln(c.Stdout, said(ev))
	return OK, nil
}

func (c CLI) runStart(ctx context.Context, dir string, positional []string, strategy, trigger string) (run.Event, error) {
	if len(positional) != 1 || !slices.Contains(harness.Skills, positional[0]) {
		return run.Event{}, fmt.Errorf("names the kind of run, one of %s\n%s", strings.Join(harness.Skills, ", "), runUsage)
	}
	kind := positional[0]
	if strategy != "" {
		if err := harness.Check("strategy."+kind, strategy); err != nil || strategy == harness.Auto {
			return run.Event{}, fmt.Errorf("--strategy is how the run works, sequential, parallel or a custom workflow, not %q", strategy)
		}
	}
	if !slices.Contains(triggers, trigger) {
		return run.Event{}, fmt.Errorf("--trigger is one of %s, not %q", strings.Join(triggers, ", "), trigger)
	}
	st, err := c.Record.Status(ctx, run.StatusRequest{Dir: dir})
	if err != nil {
		return run.Event{}, err
	}
	ws, err := c.Folders.Project(dir)
	if err != nil {
		return run.Event{}, err
	}
	cfg, err := harnessConfig(ws)
	if err != nil {
		return run.Event{}, err
	}
	roles := map[string]run.Role{}
	for _, r := range harness.Roles {
		model, effort := cfg.Agent(r)
		roles[r] = run.Role{Model: model, Effort: effort}
	}
	items, slicesOf, counts := len(st.Items), len(st.Slices), run.CountsOf(st)
	return run.Event{Type: run.EventRunStart, Kind: kind, Strategy: strategy, Trigger: trigger, Items: &items, Slices: &slicesOf, Roles: roles, Counts: &counts}, nil
}

func (c CLI) runSlice(ctx context.Context, dir string, positional []string) (run.Event, error) {
	if len(positional) > 1 {
		return run.Event{}, fmt.Errorf("names the slice's Packages, separated by commas, as its markitect status --slice lists them, or nothing for a round over the whole list\n%s", runUsage)
	}
	var names []string
	if len(positional) == 1 {
		if names = run.SliceNames(positional[0]); len(names) == 0 {
			return run.Event{}, fmt.Errorf("names Packages, separated by commas, as markitect status --slices prints them")
		}
	}
	st, err := c.Record.Status(ctx, run.StatusRequest{Dir: dir, Slice: names})
	if err != nil {
		return run.Event{}, err
	}
	items := len(st.Items)
	return run.Event{Type: run.EventRunSlice, Packages: names, Items: &items}, nil
}

func (c CLI) runEnd(ctx context.Context, dir string, positional []string, rounds string) (run.Event, error) {
	if len(positional) != 1 || !slices.Contains(run.Outcomes, positional[0]) {
		return run.Event{}, fmt.Errorf("names how the run ended, one of %s\n%s", strings.Join(run.Outcomes, ", "), runUsage)
	}
	ev := run.Event{Type: run.EventRunEnd, Outcome: positional[0]}
	if rounds != "" {
		n, err := strconv.Atoi(rounds)
		if err != nil || n < 0 {
			return run.Event{}, fmt.Errorf("--rounds counts rounds, not %q", rounds)
		}
		ev.Rounds = &n
	}
	st, err := c.Record.Status(ctx, run.StatusRequest{Dir: dir})
	if err != nil {
		return run.Event{}, err
	}
	counts := run.CountsOf(st)
	ev.Counts = &counts
	// An audit's end counts what it opened in the inbox, by kind.
	if ws, err := c.Folders.Project(dir); err == nil {
		if data, err := ws.Read(project.HistoryFile); err == nil {
			if start, ok := run.ParseHistory(data).OpenRun(); ok && start.Kind == "audit" {
				if b, err := c.Record.Inbox(ctx, dir); err == nil {
					if found := b.Found("audit", start.Time); len(found) > 0 {
						ev.Found = found
					}
				}
			}
		}
	}
	return ev, nil
}

// said is the line the run command prints for an event it appended.
func said(ev run.Event) string {
	switch ev.Type {
	case run.EventRunStart:
		if n := ev.Counts.SpecErrors; n > 0 {
			return fmt.Sprintf("Run started: %s, with %d %s in the spec and so no list.", ev.Kind, n, plural(n, "error", "errors"))
		}
		return fmt.Sprintf("Run started: %s, %d %s on the list in %d %s.", ev.Kind, *ev.Items, plural(*ev.Items, "item", "items"), *ev.Slices, plural(*ev.Slices, "slice", "slices"))
	case run.EventRunSlice:
		what := "a round over the whole list"
		if len(ev.Packages) > 0 {
			what = "the slice of " + strings.Join(ev.Packages, ", ")
		}
		return fmt.Sprintf("Started %s: %d %s.", what, *ev.Items, plural(*ev.Items, "item", "items"))
	}
	c := ev.Counts
	text := fmt.Sprintf("Run ended: %s; %d verified, %d left (%d fail, %d spec-problem), %d skip", ev.Outcome, c.Verified, c.Left, c.Fail, c.SpecProblem, c.Skip)
	if c.SpecErrors > 0 || c.RealizationErrors > 0 {
		text += fmt.Sprintf("; %d %s in the spec, %d in the realization", c.SpecErrors, plural(c.SpecErrors, "error", "errors"), c.RealizationErrors)
	}
	return text + "."
}

// hookWait is how long the hook waits for the lock of the record to append a
// refusal; past it the line is dropped.
const hookWait = time.Second

// refused appends to the history of the project the call came from what the
// hook refused. It never fails: a hook must not stop an agent.
func (c CLI) refused(ctx context.Context, input, out []byte) {
	var in struct {
		Cwd       string         `json:"cwd"`
		AgentType string         `json:"agent_type"`
		ToolName  string         `json:"tool_name"`
		ToolInput map[string]any `json:"tool_input"`
	}
	var said struct {
		Output struct {
			Why string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if c.Record == nil || json.Unmarshal(input, &in) != nil || json.Unmarshal(out, &said) != nil || in.Cwd == "" {
		return
	}
	target := ""
	for _, key := range []string{"file_path", "notebook_path", "subagent_type", "command"} {
		if v, ok := in.ToolInput[key].(string); ok && v != "" {
			target = v
			break
		}
	}
	if r := []rune(target); len(r) > 200 {
		target = string(r[:200]) + "…"
	}
	_ = c.Record.AppendWithin(ctx, in.Cwd, run.Event{Type: run.EventRefusal, Agent: in.AgentType, Tool: in.ToolName, Target: target, Why: said.Output.Why}, hookWait)
}
