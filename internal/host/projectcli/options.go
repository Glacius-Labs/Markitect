package projectcli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type options struct {
	action, repo, revision, base                                        string
	name, manager, goal                                                 string
	provider, model, effort                                             string
	toolRoot, providerExecutable                                        string
	inputMicros, outputMicros                                           string
	maxCost, since                                                      string
	operation, provenance, event, documentPath                          string
	explorationID, scope, actor, authority, decisionRef, acknowledgedAt string
	brownfieldAction, sessionID                                         string
	sourceRepo                                                          string
	input, request                                                      string
	output                                                              string
	discovery, report, resolution                                       string
	plan, run, candidate                                                string
	expect, branch, head, tree                                          string
	managers                                                            []string
	write, help, generate                                               bool
	acknowledgeStructure                                                bool
}

type actionSpec struct {
	usage    string
	flags    []string
	required []string
	write    bool
}

var actionSpecs = map[string]actionSpec{
	"explore":    {usage: "explore --repo PATH [--exploration ID | --input RECORD.json] [--expect PLAN_DIGEST --write]", flags: []string{"repo", "revision", "exploration", "input", "expect", "write"}, required: []string{"repo"}, write: true},
	"readiness":  {usage: "readiness --repo PATH --exploration ID --scope ID [--acknowledge-structure --actor ACTOR --authority TEXT --decision-ref REF --acknowledged-at RFC3339 --expect DIGEST --write]", flags: []string{"repo", "revision", "exploration", "scope", "acknowledge-structure", "actor", "authority", "decision-ref", "acknowledged-at", "expect", "write"}, required: []string{"repo", "exploration", "scope"}, write: true},
	"brownfield": {usage: "brownfield --repo TARGET [--source-repo SOURCE] --brownfield-action start|begin|context|propose|integrate|iterate|run|resolve|plan|apply-adoption|resume [--revision COMMIT] [--session ID] [--input STAGE.json] [--expect DIGEST --write]", flags: []string{"repo", "source-repo", "revision", "brownfield-action", "session", "input", "expect", "write"}, required: []string{"repo", "brownfield-action"}, write: true},
	"deliver":    {usage: "deliver --repo PATH --exploration ID --scope ID [--run ID] --write", flags: []string{"repo", "exploration", "scope", "run", "write"}, required: []string{"repo", "exploration", "scope", "write"}, write: true},
	"schema":     {usage: "schema", flags: []string{}},
	"init":       {usage: "init --repo PATH --name NAME [--write]", flags: []string{"repo", "name", "write"}, required: []string{"repo", "name"}, write: true},
	"check":      {usage: "check --repo PATH [--revision COMMIT]", flags: []string{"repo", "revision"}, required: []string{"repo"}},
	"index":      {usage: "index --repo PATH [--revision COMMIT]", flags: []string{"repo", "revision"}, required: []string{"repo"}},
	"context":    {usage: "context --repo PATH --manager ID [--revision COMMIT]", flags: []string{"repo", "revision", "manager"}, required: []string{"repo", "manager"}},
	"impact":     {usage: "impact --repo PATH --base COMMIT --revision COMMIT", flags: []string{"repo", "base", "revision"}, required: []string{"repo", "base", "revision"}},
	"document":   {usage: "document --repo PATH [--revision COMMIT] [--write]", flags: []string{"repo", "revision", "write"}, required: []string{"repo"}, write: true},
	"edit":       {usage: "edit --repo PATH --input MUTATION.json [--revision COMMIT] [--expect PLAN_DIGEST --write]", flags: []string{"repo", "revision", "input", "expect", "write"}, required: []string{"repo", "input"}, write: true},
	"discover":   {usage: "discover --repo PATH --request DISCOVERY-REQUEST.json [--output DISCOVERY.json]", flags: []string{"repo", "request", "output"}, required: []string{"repo", "request"}},
	"distill":    {usage: "distill --repo PATH --discovery DISCOVERY.json --report DISTILLATION.json [--output VALIDATED-DISTILLATION.json] | distill --repo PATH --discovery DISCOVERY.json --generate --write --output DISTILLATION.json --input-micros-per-million N --output-micros-per-million N --max-cost-micros N", flags: []string{"repo", "discovery", "report", "output", "generate", "write", "input-micros-per-million", "output-micros-per-million", "max-cost-micros"}, required: []string{"repo", "discovery"}, write: true},
	"adopt":      {usage: "adopt --repo TARGET --source-repo SOURCE --revision TARGET_COMMIT --discovery DISCOVERY.json --report DISTILLATION.json --resolution RESOLUTION.json [--output PLAN.json] [--plan PLAN.json --expect PLAN_DIGEST --write]", flags: []string{"repo", "source-repo", "revision", "discovery", "report", "resolution", "plan", "output", "expect", "write"}, required: []string{"repo", "source-repo", "revision", "discovery", "report", "resolution"}, write: true},
	"plan":       {usage: "plan --repo PATH --goal TEXT [--operation apply|cleanup|reconcile] [--manager ID ...] [--revision COMMIT] [--since OLD_COMMIT] [--write]", flags: []string{"repo", "goal", "operation", "manager", "revision", "since", "exploration", "scope", "write"}, required: []string{"repo", "goal"}, write: true},
	"run":        {usage: "run --repo PATH --plan PLAN_ID --write", flags: []string{"repo", "plan", "write"}, required: []string{"repo", "plan", "write"}, write: true},
	"resume":     {usage: "resume --repo PATH --run RUN_ID --write", flags: []string{"repo", "run", "write"}, required: []string{"repo", "run", "write"}, write: true},
	"repair":     {usage: "repair --repo PATH --run RUN_ID [--write]", flags: []string{"repo", "run", "write"}, required: []string{"repo", "run"}, write: true},
	"status":     {usage: "status --repo PATH --run RUN_ID", flags: []string{"repo", "run"}, required: []string{"repo", "run"}},
	"verify":     {usage: "verify --repo PATH (--run RUN_ID | --revision COMMIT) --write", flags: []string{"repo", "run", "revision", "write"}, required: []string{"repo", "write"}, write: true},
	"coverage":   {usage: "coverage --repo PATH [--revision COMMIT]", flags: []string{"repo", "revision"}, required: []string{"repo"}},
	"cleanup":    {usage: "cleanup --repo PATH --goal TEXT [--revision COMMIT] [--write] (creates a plan; run/verify/apply separately)", flags: []string{"repo", "goal", "revision", "exploration", "scope", "write"}, required: []string{"repo", "goal"}, write: true},
	"reconcile":  {usage: "reconcile --repo PATH --goal TEXT [--revision COMMIT] [--write] (creates a plan; run/verify/apply separately)", flags: []string{"repo", "goal", "revision", "exploration", "scope", "write"}, required: []string{"repo", "goal"}, write: true},
	"brief":      {usage: "brief --repo PATH --since COMMIT --revision COMMIT --provenance TEXT [--expect STATE_DIGEST --write]", flags: []string{"repo", "since", "revision", "provenance", "expect", "write"}, required: []string{"repo", "since", "revision", "provenance"}, write: true},
	"briefings":  {usage: "briefings --repo PATH [--manager ID]", flags: []string{"repo", "manager"}, required: []string{"repo"}},
	"dismiss":    {usage: "dismiss --repo PATH --event ID --manager ID --expect STATE_DIGEST --write", flags: []string{"repo", "event", "manager", "expect", "write"}, required: []string{"repo", "event", "manager", "expect", "write"}, write: true},
	"onboard":    {usage: "onboard --repo PATH --provider codex|claude|both [--document-path PATH] [--expect PLAN_DIGEST --write]", flags: []string{"repo", "provider", "document-path", "expect", "write"}, required: []string{"repo", "provider"}, write: true},
	"apply":      {usage: "apply --repo PATH --plan PLAN_ID --run RUN_ID --candidate ID [--branch BRANCH --head COMMIT --worktree DIGEST --expect VERIFY_DIGEST --write]", flags: []string{"repo", "plan", "run", "candidate", "branch", "head", "worktree", "expect", "write"}, required: []string{"repo", "plan", "run", "candidate"}, write: true},
	"setup":      {usage: "setup --repo PATH --tool-root MARKITECT_SOURCE --provider codex|claude --model MODEL [--effort high] --input-micros-per-million N --output-micros-per-million N --max-cost-micros N [--provider-executable PATH] [--expect EDIT_DIGEST --write]", flags: []string{"repo", "tool-root", "provider", "model", "effort", "provider-executable", "input-micros-per-million", "output-micros-per-million", "max-cost-micros", "expect", "write"}, required: []string{"repo", "tool-root", "provider", "model", "input-micros-per-million", "output-micros-per-million", "max-cost-micros"}, write: true},
	"doctor":     {usage: "doctor --repo PATH --tool-root MARKITECT_SOURCE --provider codex|claude [--provider-executable PATH]", flags: []string{"repo", "tool-root", "provider", "provider-executable"}, required: []string{"repo", "tool-root", "provider"}},
	"resolve":    {usage: "resolve --repo TARGET --source-repo SOURCE --revision TARGET_COMMIT --discovery DISCOVERY.json --report DISTILLATION.json --input CHOICES.json [--output RESOLUTION.json]", flags: []string{"repo", "source-repo", "revision", "discovery", "report", "input", "output"}, required: []string{"repo", "source-repo", "revision", "discovery", "report", "input"}},
}

func parse(args []string, errout io.Writer) (options, bool, error) {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return options{}, true, nil
	}
	if len(args) > 0 && args[0] == "project" {
		args = args[1:]
		if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
			return options{}, true, nil
		}
	}
	if len(args) == 0 {
		return options{}, false, errors.New("project requires an action; use --help")
	}
	action := args[0]
	spec, ok := actionSpecs[action]
	if !ok {
		return options{}, false, fmt.Errorf("unknown project action %q", action)
	}
	fs := flag.NewFlagSet("project "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values := map[string]*string{}
	for _, name := range spec.flags {
		switch name {
		case "write", "generate", "acknowledge-structure":
			continue
		case "manager":
			continue
		default:
			values[name] = fs.String(name, "", "")
		}
	}
	var managers stringList
	if contains(spec.flags, "manager") {
		fs.Var(&managers, "manager", "manager ID (repeatable)")
	}
	write := false
	if contains(spec.flags, "write") {
		fs.BoolVar(&write, "write", false, "apply the already-authorized, reviewed operation")
	}
	generate := false
	acknowledgeStructure := false
	if contains(spec.flags, "acknowledge-structure") {
		fs.BoolVar(&acknowledgeStructure, "acknowledge-structure", false, "record caller-authorized acknowledgement of the exact proposed structure")
	}
	if contains(spec.flags, "generate") {
		fs.BoolVar(&generate, "generate", false, "generate one agent-assisted report using the configured runtime")
	}
	help := false
	fs.BoolVar(&help, "help", false, "show action usage")
	fs.BoolVar(&help, "h", false, "show action usage")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options{action: action}, true, nil
		}
		return options{}, false, err
	}
	if help {
		return options{action: action}, true, nil
	}
	if fs.NArg() != 0 {
		return options{}, false, errors.New("unexpected positional arguments")
	}
	if write && !spec.write {
		return options{}, false, fmt.Errorf("--write does not apply to project %s", action)
	}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, required := range spec.required {
		if action == "distill" && required == "report" && generate {
			continue
		}
		if required == "write" {
			if !write {
				return options{}, false, fmt.Errorf("project %s requires --write", action)
			}
			continue
		}
		if required == "manager" {
			if len(managers) == 0 {
				return options{}, false, fmt.Errorf("project %s requires at least one --manager", action)
			}
			continue
		}
		if !seen[required] || *values[required] == "" {
			return options{}, false, fmt.Errorf("project %s requires --%s", action, required)
		}
	}
	if action == "distill" {
		if generate {
			if !write {
				return options{}, false, errors.New("project distill --generate requires --write to invoke the configured agent")
			}
			if *values["report"] != "" {
				return options{}, false, errors.New("project distill --generate cannot be combined with --report")
			}
			for _, name := range []string{"output", "input-micros-per-million", "output-micros-per-million", "max-cost-micros"} {
				if !seen[name] || *values[name] == "" {
					return options{}, false, fmt.Errorf("project distill --generate requires --%s", name)
				}
			}
		} else {
			if write {
				return options{}, false, errors.New("project distill --write is valid only with --generate")
			}
			if *values["report"] == "" {
				return options{}, false, errors.New("project distill requires --report unless --generate is used")
			}
		}
	}
	if action == "verify" && ((*values["run"] == "") == (*values["revision"] == "")) {
		return options{}, false, errors.New("project verify requires exactly one of --run or --revision")
	}
	if action == "plan" && seen["operation"] {
		switch *values["operation"] {
		case "apply", "cleanup", "reconcile":
		default:
			return options{}, false, errors.New("project plan --operation must be apply, cleanup or reconcile")
		}
	}
	if (action == "context" || action == "briefings" || action == "dismiss") && len(managers) > 1 {
		return options{}, false, fmt.Errorf("project %s accepts only one --manager", action)
	}
	if (action == "brief" || action == "onboard") && write && *values["expect"] == "" {
		return options{}, false, fmt.Errorf("project %s --write requires --expect with the preview digest", action)
	}
	if (action == "edit" || action == "adopt") && write && *values["expect"] == "" {
		return options{}, false, fmt.Errorf("project %s --write requires --expect with the reviewed plan digest", action)
	}
	if action == "setup" && write && *values["expect"] == "" {
		return options{}, false, errors.New("project setup --write requires --expect with the exact runtime edit plan digest")
	}
	if action == "setup" && !write && seen["expect"] {
		return options{}, false, errors.New("project setup --expect is valid only together with --write")
	}
	if action == "adopt" && write && *values["plan"] == "" {
		return options{}, false, errors.New("project adopt --write requires --plan with the exact reviewed adoption plan")
	}
	if action == "apply" && *values["expect"] == "" {
		if write {
			return options{}, false, errors.New("project apply --write requires --expect with the exact verification digest")
		}
	}
	if action == "apply" && write {
		for _, required := range []string{"branch", "head", "worktree"} {
			if !seen[required] || *values[required] == "" {
				return options{}, false, fmt.Errorf("project apply --write requires --%s from the read-only preflight", required)
			}
		}
	}
	o := options{action: action, write: write, generate: generate, acknowledgeStructure: acknowledgeStructure, managers: append([]string(nil), managers...)}
	if len(managers) > 0 {
		o.manager = managers[0]
	}
	assign := func(flagName string, target *string) {
		if value := values[flagName]; value != nil {
			*target = *value
		}
	}
	assign("repo", &o.repo)
	assign("source-repo", &o.sourceRepo)
	assign("revision", &o.revision)
	assign("base", &o.base)
	assign("name", &o.name)
	assign("goal", &o.goal)
	assign("provider", &o.provider)
	assign("model", &o.model)
	assign("effort", &o.effort)
	assign("tool-root", &o.toolRoot)
	assign("provider-executable", &o.providerExecutable)
	assign("input-micros-per-million", &o.inputMicros)
	assign("output-micros-per-million", &o.outputMicros)
	assign("max-cost-micros", &o.maxCost)
	assign("since", &o.since)
	assign("input", &o.input)
	assign("request", &o.request)
	assign("output", &o.output)
	assign("discovery", &o.discovery)
	assign("report", &o.report)
	assign("resolution", &o.resolution)
	assign("plan", &o.plan)
	assign("run", &o.run)
	assign("candidate", &o.candidate)
	assign("expect", &o.expect)
	assign("branch", &o.branch)
	assign("head", &o.head)
	assign("worktree", &o.tree)
	assign("operation", &o.operation)
	assign("provenance", &o.provenance)
	assign("event", &o.event)
	assign("document-path", &o.documentPath)
	assign("exploration", &o.explorationID)
	assign("scope", &o.scope)
	assign("actor", &o.actor)
	assign("authority", &o.authority)
	assign("decision-ref", &o.decisionRef)
	assign("acknowledged-at", &o.acknowledgedAt)
	assign("brownfield-action", &o.brownfieldAction)
	assign("session", &o.sessionID)
	_ = errout
	return o, false, nil
}

type stringList []string

func (s *stringList) String() string { return fmt.Sprint([]string(*s)) }
func (s *stringList) Set(value string) error {
	if value == "" {
		return errors.New("value must not be empty")
	}
	*s = append(*s, value)
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
