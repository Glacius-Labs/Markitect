package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
)

var fullGitCommitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

type commandOptions struct {
	scope                 string
	expect                string
	workspace             string
	queue                 string
	decision              string
	root                  string
	runManifest           string
	revision              string
	apiVersion            string
	base                  string
	kind                  string
	name                  string
	namespace             string
	packageName           string
	query                 string
	write                 bool
	check                 bool
	analyzePolicyFailures bool
	output                string
	bundlePath            string
	bundleSHA             string
	action                string
	adapter               string
	plan                  string
	applyResult           string
	coverage              string
	reviewConfig          string
	reviewReport          string
	reviewEvidence        string
	runtime               string
	goalInput             string
	goalRecommendations   string
	goalSelection         string
	flagCount             int
}

func parseOptions(command string, args []string, allowed map[string]bool, out, errout io.Writer) (commandOptions, int, bool) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	scope := fs.String("scope", "", "owner-supplied exact adoption scope YAML (prepare)")
	expect := fs.String("expect", "", "exact reviewed digest required by the selected write or canonical adoption action")
	workspace := fs.String("workspace", "", "captured immutable external evidence workspace (copy-me)")
	queue := fs.String("queue", "", "explicit evidence/candidate queue YAML (copy-me)")
	decision := fs.String("decision", "", "optional supplied immutable review decision YAML (copy-me)")
	root := fs.String("repo", ".", "repository root")
	runManifest := fs.String("run", "", "committed fixed-run context manifest (context)")
	revision := fs.String("revision", "", "fixed Git revision (omitted: provisional working tree)")
	apiVersion := fs.String("api-version", "", "resource API version for extension resources")
	base := fs.String("base", "", "base revision for impact, canonical verification or adoption")
	kind := fs.String("kind", "", "context entry kind")
	name := fs.String("name", "", "resource or project name")
	namespace := fs.String("namespace", "", "resource namespace")
	packageName := fs.String("package", "", "exact content package identity (omitted: local entry)")
	query := fs.String("query", "", "literal search text (find)")
	write := fs.Bool("write", false, "explicitly write planned files (prepare: reviewed external capture)")
	check := fs.Bool("check", false, "check rendered outputs (default)")
	analyzePolicyFailures := fs.Bool("analyze-policy-failures", false, "allow read-only analysis of structurally valid policy failures (context, impact)")
	output := fs.String("output", "", "absent directory (package/prepare) or ZIP file (bundle)")
	bundlePath := fs.String("bundle", "", "local release ZIP to validate and install")
	bundleSHA := fs.String("sha256", "", "expected SHA-256 of the release ZIP")
	action := fs.String("action", "", "action for reconciliation, projection or canonical operations")
	adapter := fs.String("adapter", "", "configured reconciliation adapter name")
	plan := fs.String("plan", "", "saved concrete action plan; controller apply reads a closed JSON reviewed run")
	applyResult := fs.String("apply-result", "", "saved canonical controller Apply result JSON (controller-verify)")
	coverage := fs.String("coverage", "", "explicit repository-relative artifact accounting configuration (projection)")
	reviewConfig := fs.String("config", "", "repository-relative review configuration in the fixed snapshot")
	reviewReport := fs.String("report", "", "local report or owner-supplied selection input for the active action")
	reviewEvidence := fs.String("evidence", "", "previous advisory review record or canonical Projection Record array (local file)")
	runtime := fs.String("runtime", "", "explicit external canonical controller runtime configuration (closed JSON)")
	goalInput := fs.String("goal-input", "", "explicit goal and supplied Module package bytes (closed JSON)")
	goalRecommendations := fs.String("goal-recommendations", "", "exact saved goal recommendation result JSON")
	goalSelection := fs.String("goal-selection", "", "explicit selected recommendation IDs (closed JSON)")
	fs.Usage = func() {
		fmt.Fprintf(out, "usage: markitect %s [options]\n", command)
		fs.VisitAll(func(f *flag.Flag) {
			if allowed[f.Name] {
				fmt.Fprintf(out, "  --%-12s %s\n", f.Name, f.Usage)
			}
		})
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return commandOptions{}, 0, true
		}
		return commandOptions{}, 2, true
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return commandOptions{}, 2, true
	}
	invalid := ""
	analyzePolicyFailuresProvided := false
	decisionProvided := false
	canonicalSelectors := map[string]bool{}
	providedFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
		if f.Name == "decision" {
			decisionProvided = true
		}
		if f.Name == "analyze-policy-failures" {
			analyzePolicyFailuresProvided = true
		}
		if !allowed[f.Name] {
			invalid = f.Name
		}
		if command == "canonical" && (f.Name == "api-version" || f.Name == "kind" || f.Name == "namespace" || f.Name == "name") {
			canonicalSelectors[f.Name] = true
		}
	})
	if invalid != "" {
		fmt.Fprintf(errout, "--%s does not apply to %s\n", invalid, command)
		return commandOptions{}, 2, true
	}
	if providedFlags["apply-result"] && (command != "canonical" || *action != "controller-verify") {
		fmt.Fprintln(errout, "--apply-result applies only to canonical --action controller-verify")
		return commandOptions{}, 2, true
	}
	goalAction := command == "canonical" && isCanonicalGoalAction(*action)
	if goalAction {
		if *goalInput == "" || *runtime == "" {
			fmt.Fprintln(errout, "canonical goal actions require --goal-input and --runtime")
			return commandOptions{}, 2, true
		}
		allowedGoal := map[string]bool{"action": true, "goal-input": true, "goal-recommendations": true, "goal-selection": true, "runtime": true}
		for name := range providedFlags {
			if !allowedGoal[name] {
				fmt.Fprintf(errout, "--%s does not apply to canonical goal actions\n", name)
				return commandOptions{}, 2, true
			}
		}
		if *action == "goal-recommend" && (providedFlags["goal-recommendations"] || providedFlags["goal-selection"]) {
			fmt.Fprintln(errout, "goal-recommend does not accept recommendations or selection")
			return commandOptions{}, 2, true
		}
		if *action == "goal-propose" && (*goalRecommendations == "" || *goalSelection == "") {
			fmt.Fprintln(errout, "goal-propose requires --goal-recommendations and --goal-selection")
			return commandOptions{}, 2, true
		}
	} else if providedFlags["goal-input"] || providedFlags["goal-recommendations"] || providedFlags["goal-selection"] {
		fmt.Fprintln(errout, "goal inputs apply only to canonical goal-recommend or goal-propose")
		return commandOptions{}, 2, true
	}
	if command == "copy-me" {
		if *action != "" && *action != "infer" {
			fmt.Fprintln(errout, "copy-me supports the default validation action or --action infer")
			return commandOptions{}, 2, true
		}
		if *action == "infer" {
			if *workspace == "" || *queue == "" || *runtime == "" {
				fmt.Fprintln(errout, "copy-me --action infer requires --workspace, --queue and --runtime")
				return commandOptions{}, 2, true
			}
			if decisionProvided {
				fmt.Fprintln(errout, "copy-me --action infer does not accept --decision")
				return commandOptions{}, 2, true
			}
		} else if *runtime != "" {
			fmt.Fprintln(errout, "copy-me --runtime applies only to --action infer")
			return commandOptions{}, 2, true
		}
	}
	canonicalApply := command == "canonical" && *action == "apply"
	canonicalDurableAdoptionApply := command == "canonical" && *action == "adopt" && *runtime != ""
	canonicalControllerApply := command == "canonical" && (*action == "controller-apply" || *action == "controller-refresh-apply")
	canonicalControllerVerify := command == "canonical" && *action == "controller-verify"
	if *write && ((command != "render" && command != "schema" && command != "format" && command != "install" && command != "reconcile" && command != "prepare" && command != "projection" && !canonicalApply && !canonicalDurableAdoptionApply && !canonicalControllerApply && !canonicalControllerVerify) || (*revision != "" && !canonicalApply && !canonicalDurableAdoptionApply && !canonicalControllerApply && !canonicalControllerVerify) || *check || ((command == "reconcile" || command == "projection" || command == "canonical") && *action != "apply" && !canonicalDurableAdoptionApply && !canonicalControllerApply && !canonicalControllerVerify)) {
		fmt.Fprintln(errout, "--write supports render, format, schema, install, reconcile/projection apply, canonical apply/controller-apply/controller-verify, durable canonical adoption, or prepare external capture")
		return commandOptions{}, 2, true
	}
	if command == "projection" {
		if *action != "observe" && *action != "plan" && *action != "apply" && *action != "verify" {
			fmt.Fprintln(errout, "projection requires --action observe, plan, apply or verify")
			return commandOptions{}, 2, true
		}
		if *reviewConfig == "" || *coverage == "" {
			fmt.Fprintln(errout, "projection requires explicit --config and --coverage")
			return commandOptions{}, 2, true
		}
		if *action == "apply" && (!*write || *plan == "") {
			fmt.Fprintln(errout, "projection apply requires --write and --plan")
			return commandOptions{}, 2, true
		}
		if *action != "apply" && (*write || *plan != "" || *reviewReport != "" || *expect != "") {
			fmt.Fprintln(errout, "projection --write, --plan, --report and --expect apply only to apply")
			return commandOptions{}, 2, true
		}
		if (*reviewReport == "") != (*expect == "") {
			fmt.Fprintln(errout, "projection AI candidate --report and reviewed --expect digest are required together")
			return commandOptions{}, 2, true
		}
		if *action == "verify" && !fullGitCommitID.MatchString(*revision) {
			fmt.Fprintln(errout, "projection verify requires a full immutable --revision")
			return commandOptions{}, 2, true
		}
	}
	if command == "reconcile" {
		if *action != "observe" && *action != "plan" && *action != "apply" && *action != "verify" {
			fmt.Fprintln(errout, "reconcile requires --action observe, plan, apply or verify")
			return commandOptions{}, 2, true
		}
		if *adapter == "" {
			fmt.Fprintln(errout, "reconcile requires --adapter")
			return commandOptions{}, 2, true
		}
		if (*action == "apply" || *action == "verify") && *plan == "" {
			fmt.Fprintln(errout, "reconcile apply and verify require --plan")
			return commandOptions{}, 2, true
		}
		if (*action == "observe" || *action == "plan") && *plan != "" {
			fmt.Fprintln(errout, "--plan applies only to reconcile apply or verify")
			return commandOptions{}, 2, true
		}
		if *action == "apply" && !*write {
			fmt.Fprintln(errout, "reconcile apply requires explicit --write")
			return commandOptions{}, 2, true
		}
		if *action != "apply" && *write {
			fmt.Fprintln(errout, "--write applies only to reconcile --action apply")
			return commandOptions{}, 2, true
		}
	}
	if command == "canonical" && !goalAction {
		if *action != "model" && *action != "modules" && *action != "context" && *action != "request" && *action != "plan" && *action != "apply" && *action != "impact" && *action != "reconcile-plan" && *action != "verify" && *action != "adopt-plan" && *action != "adopt" && !isCanonicalControllerAction(*action) {
			fmt.Fprintln(errout, "canonical requires --action model, modules, context, request, impact, reconcile-plan, plan, apply, verify, adopt-plan, adopt, controller-audit, controller-propose, controller-execute, controller-apply or controller-verify")
			return commandOptions{}, 2, true
		}
		if *reviewConfig == "" {
			fmt.Fprintln(errout, "canonical requires --config in the selected source snapshot")
			return commandOptions{}, 2, true
		}
		if isCanonicalControllerAction(*action) {
			if *runtime == "" {
				fmt.Fprintf(errout, "canonical %s requires explicit --runtime closed JSON configuration\n", *action)
				return commandOptions{}, 2, true
			}
			if *action == "controller-audit" {
				auditFlags := map[string]bool{"action": true, "repo": true, "config": true, "runtime": true, "base": true, "revision": true}
				for name := range providedFlags {
					if !auditFlags[name] {
						fmt.Fprintf(errout, "--%s does not apply to canonical controller-audit\n", name)
						return commandOptions{}, 2, true
					}
				}
			}
			if *action == "controller-verify" && providedFlags["apply-result"] {
				if providedFlags["base"] || providedFlags["revision"] {
					fmt.Fprintln(errout, "canonical controller-verify --apply-result cannot be combined with --base or --revision")
					return commandOptions{}, 2, true
				}
				if *applyResult == "" {
					fmt.Fprintln(errout, "canonical controller-verify --apply-result requires a file path")
					return commandOptions{}, 2, true
				}
			} else if !fullGitCommitID.MatchString(*base) || !fullGitCommitID.MatchString(*revision) {
				fmt.Fprintf(errout, "canonical %s requires full immutable --base and --revision values\n", *action)
				return commandOptions{}, 2, true
			}
			if *apiVersion != "" || *kind != "" || *namespace != "" || *name != "" || (*reviewReport != "" && *action != "controller-refresh-propose") || *reviewEvidence != "" {
				fmt.Fprintf(errout, "canonical %s does not accept selectors, --report or --evidence\n", *action)
				return commandOptions{}, 2, true
			}
			switch *action {
			case "controller-refresh-propose":
				if *write || *plan != "" || *expect != "" || *reviewReport == "" {
					fmt.Fprintln(errout, "canonical controller-refresh-propose requires --report with exact Projection IDs and forbids --write, --plan and --expect")
					return commandOptions{}, 2, true
				}
			case "controller-audit", "controller-propose", "controller-execute":
				if *write || *plan != "" || *expect != "" {
					fmt.Fprintf(errout, "canonical %s is read-only and does not accept --write, --plan or --expect\n", *action)
					return commandOptions{}, 2, true
				}
			case "controller-apply", "controller-refresh-apply":
				if !*write || *plan == "" || *expect == "" {
					fmt.Fprintln(errout, "canonical controller-apply requires --write, --plan and --expect")
					return commandOptions{}, 2, true
				}
			case "controller-verify":
				if *plan != "" || *expect != "" {
					fmt.Fprintln(errout, "canonical controller-verify does not accept --plan or --expect")
					return commandOptions{}, 2, true
				}
			}
		} else {
			durableAdoption := (*action == "adopt-plan" || *action == "adopt") && *runtime != ""
			if *runtime != "" && !durableAdoption {
				fmt.Fprintln(errout, "canonical --runtime applies only to controller actions or durable adoption")
				return commandOptions{}, 2, true
			}
			selectorFields := []string{*apiVersion, *kind, *namespace, *name}
			selectedCount := 0
			for _, field := range selectorFields {
				if field != "" {
					selectedCount++
				}
			}
			if *action == "context" || *action == "request" || *action == "plan" || *action == "apply" || *action == "adopt-plan" || *action == "adopt" {
				if !canonicalSelectors["api-version"] || !canonicalSelectors["kind"] || !canonicalSelectors["namespace"] || !canonicalSelectors["name"] || *apiVersion == "" || *kind == "" || *name == "" {
					fmt.Fprintf(errout, "canonical %s requires exact --api-version, --kind, --namespace and --name\n", *action)
					return commandOptions{}, 2, true
				}
			} else if selectedCount != 0 || len(canonicalSelectors) != 0 {
				fmt.Fprintln(errout, "canonical --api-version, --kind, --namespace and --name apply only to context, request, plan or apply")
				return commandOptions{}, 2, true
			}
			if *write && !canonicalApply && !canonicalDurableAdoptionApply {
				fmt.Fprintln(errout, "canonical actions are read-only")
				return commandOptions{}, 2, true
			}
			if (*action == "request" || *action == "plan" || *action == "apply" || *action == "impact" || *action == "reconcile-plan" || *action == "verify" || *action == "adopt-plan" || *action == "adopt") && !fullGitCommitID.MatchString(*revision) {
				fmt.Fprintf(errout, "canonical %s requires a full immutable --revision\n", *action)
				return commandOptions{}, 2, true
			}
			if *action == "impact" || *action == "reconcile-plan" || *action == "verify" || *action == "adopt-plan" || *action == "adopt" {
				if !fullGitCommitID.MatchString(*base) {
					fmt.Fprintf(errout, "canonical %s requires a full immutable --base revision\n", *action)
					return commandOptions{}, 2, true
				}
				if (*action == "impact" || *action == "reconcile-plan" || *action == "verify") && (*write || *plan != "" || *reviewReport != "" || *expect != "") {
					fmt.Fprintf(errout, "canonical %s is read-only and does not accept --plan, --report or --expect\n", *action)
					return commandOptions{}, 2, true
				}
				if *action == "impact" && *reviewEvidence != "" {
					fmt.Fprintln(errout, "canonical impact does not accept --evidence")
					return commandOptions{}, 2, true
				}
				if *action == "verify" && *reviewEvidence == "" {
					fmt.Fprintln(errout, "canonical verify requires --evidence with one Projection Record")
					return commandOptions{}, 2, true
				}
				if *action == "adopt-plan" || *action == "adopt" {
					if *plan != "" || *reviewReport == "" || (*action == "adopt-plan" && *write) || (*action == "adopt" && *runtime == "" && *write) || (*action == "adopt" && *runtime != "" && !*write) {
						fmt.Fprintf(errout, "canonical %s requires --report, does not use --plan, and durable adopt requires --runtime plus --write\n", *action)
						return commandOptions{}, 2, true
					}
					if *action == "adopt-plan" && *expect != "" {
						fmt.Fprintln(errout, "canonical adopt-plan does not accept --expect")
						return commandOptions{}, 2, true
					}
					if *action == "adopt" && *expect == "" {
						fmt.Fprintln(errout, "canonical adopt requires --expect with the exact reviewed PlanDigest")
						return commandOptions{}, 2, true
					}
				}
			} else if *base != "" {
				fmt.Fprintln(errout, "canonical --base applies only to --action impact, reconcile-plan, verify, adopt-plan or adopt")
				return commandOptions{}, 2, true
			}
			if *action == "plan" {
				if *write || *plan != "" || *expect != "" {
					fmt.Fprintln(errout, "canonical plan is read-only; --plan and --expect apply only to apply")
					return commandOptions{}, 2, true
				}
			} else if *action == "apply" {
				if !*write || *plan == "" || *expect == "" {
					fmt.Fprintln(errout, "canonical apply requires --write, --plan and --expect")
					return commandOptions{}, 2, true
				}
			} else if *action != "impact" && *action != "reconcile-plan" && *action != "verify" && *action != "adopt-plan" && *action != "adopt" && (*write || *plan != "" || *reviewReport != "" || *expect != "") {
				fmt.Fprintln(errout, "canonical --write, --plan, --report and --expect apply only to apply; --report is also allowed for plan candidates")
				return commandOptions{}, 2, true
			}
			if *action == "reconcile-plan" && (*plan != "" || *reviewReport != "" || *expect != "") {
				fmt.Fprintln(errout, "canonical reconcile-plan does not accept --plan, --report or --expect")
				return commandOptions{}, 2, true
			}
			if *reviewEvidence != "" && *action != "reconcile-plan" && *action != "verify" {
				fmt.Fprintln(errout, "canonical --evidence applies only to --action reconcile-plan or verify")
				return commandOptions{}, 2, true
			}
			if (*action == "adopt-plan" || *action == "adopt") && *reviewEvidence != "" {
				fmt.Fprintln(errout, "canonical adoption uses --report, not --evidence")
				return commandOptions{}, 2, true
			}
			if *action == "plan" && *reviewReport != "" && *expect != "" {
				fmt.Fprintln(errout, "canonical plan does not accept --expect")
				return commandOptions{}, 2, true
			}
		}
	}
	if *runManifest != "" && (command != "context" || !fullGitCommitID.MatchString(*revision) || *apiVersion != "" || *kind != "" || *name != "" || *namespace != "" || *packageName != "") {
		fmt.Fprintln(errout, "--run is only valid for context with --revision and supplies its own entry")
		return commandOptions{}, 2, true
	}
	if analyzePolicyFailuresProvided && command == "context" && *runManifest != "" {
		fmt.Fprintln(errout, "--analyze-policy-failures does not apply to context --run")
		return commandOptions{}, 2, true
	}
	return commandOptions{
		scope: *scope, expect: *expect, workspace: *workspace, queue: *queue, decision: *decision,
		root:                  *root,
		runManifest:           *runManifest,
		revision:              *revision,
		apiVersion:            *apiVersion,
		base:                  *base,
		kind:                  *kind,
		name:                  *name,
		namespace:             *namespace,
		packageName:           *packageName,
		query:                 *query,
		write:                 *write,
		check:                 *check,
		analyzePolicyFailures: *analyzePolicyFailures,
		output:                *output,
		bundlePath:            *bundlePath,
		bundleSHA:             *bundleSHA,
		action:                *action,
		adapter:               *adapter,
		plan:                  *plan,
		applyResult:           *applyResult,
		runtime:               *runtime,
		goalInput:             *goalInput,
		goalRecommendations:   *goalRecommendations,
		goalSelection:         *goalSelection,
		coverage:              *coverage,
		reviewConfig:          *reviewConfig,
		reviewReport:          *reviewReport,
		reviewEvidence:        *reviewEvidence,
		flagCount:             fs.NFlag(),
	}, 0, false
}
