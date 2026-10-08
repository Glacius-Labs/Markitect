package projectcli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type options struct {
	action, repo, revision, base  string
	name, manager, goal           string
	sourceRepo                    string
	input, request                string
	output                        string
	discovery, report, resolution string
	plan, run, candidate          string
	expect, branch, head, tree    string
	managers                      []string
	write, help                   bool
}

type actionSpec struct {
	usage    string
	flags    []string
	required []string
	write    bool
}

var actionSpecs = map[string]actionSpec{
	"init":     {usage: "init --repo PATH --name NAME [--write]", flags: []string{"repo", "name", "write"}, required: []string{"repo", "name"}, write: true},
	"check":    {usage: "check --repo PATH [--revision COMMIT]", flags: []string{"repo", "revision"}, required: []string{"repo"}},
	"index":    {usage: "index --repo PATH [--revision COMMIT]", flags: []string{"repo", "revision"}, required: []string{"repo"}},
	"context":  {usage: "context --repo PATH --manager ID [--revision COMMIT]", flags: []string{"repo", "revision", "manager"}, required: []string{"repo", "manager"}},
	"impact":   {usage: "impact --repo PATH --base COMMIT --revision COMMIT", flags: []string{"repo", "base", "revision"}, required: []string{"repo", "base", "revision"}},
	"document": {usage: "document --repo PATH [--revision COMMIT] [--write]", flags: []string{"repo", "revision", "write"}, required: []string{"repo"}, write: true},
	"edit":     {usage: "edit --repo PATH --input MUTATION.json [--revision COMMIT] [--expect PLAN_DIGEST --write]", flags: []string{"repo", "revision", "input", "expect", "write"}, required: []string{"repo", "input"}, write: true},
	"discover": {usage: "discover --repo PATH --request DISCOVERY-REQUEST.json [--output DISCOVERY.json]", flags: []string{"repo", "request", "output"}, required: []string{"repo", "request"}},
	"distill":  {usage: "distill --repo PATH --discovery DISCOVERY.json --report DISTILLATION.json [--output VALIDATED-DISTILLATION.json]", flags: []string{"repo", "discovery", "report", "output"}, required: []string{"repo", "discovery", "report"}},
	"adopt":    {usage: "adopt --repo TARGET --source-repo SOURCE --revision TARGET_COMMIT --discovery DISCOVERY.json --report DISTILLATION.json --resolution RESOLUTION.json [--output PLAN.json] [--plan PLAN.json --expect PLAN_DIGEST --write]", flags: []string{"repo", "source-repo", "revision", "discovery", "report", "resolution", "plan", "output", "expect", "write"}, required: []string{"repo", "source-repo", "revision", "discovery", "report", "resolution"}, write: true},
	"plan":     {usage: "plan --repo PATH --goal TEXT [--manager ID ...] [--revision COMMIT] [--write]", flags: []string{"repo", "goal", "manager", "revision", "write"}, required: []string{"repo", "goal"}, write: true},
	"run":      {usage: "run --repo PATH --plan PLAN_ID --write", flags: []string{"repo", "plan", "write"}, required: []string{"repo", "plan", "write"}, write: true},
	"resume":   {usage: "resume --repo PATH --run RUN_ID --write", flags: []string{"repo", "run", "write"}, required: []string{"repo", "run", "write"}, write: true},
	"status":   {usage: "status --repo PATH --run RUN_ID", flags: []string{"repo", "run"}, required: []string{"repo", "run"}},
	"verify":   {usage: "verify --repo PATH --run RUN_ID --write", flags: []string{"repo", "run", "write"}, required: []string{"repo", "run", "write"}, write: true},
	"apply":    {usage: "apply --repo PATH --plan PLAN_ID --run RUN_ID --candidate ID [--branch BRANCH --head COMMIT --worktree DIGEST --expect VERIFY_DIGEST --write]", flags: []string{"repo", "plan", "run", "candidate", "branch", "head", "worktree", "expect", "write"}, required: []string{"repo", "plan", "run", "candidate"}, write: true},
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
		case "write":
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
	if (action == "edit" || action == "adopt") && write && *values["expect"] == "" {
		return options{}, false, fmt.Errorf("project %s --write requires --expect with the reviewed plan digest", action)
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
	o := options{action: action, write: write, managers: append([]string(nil), managers...)}
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
