package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
)

var fullGitCommitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

type commandOptions struct {
	root           string
	runManifest    string
	revision       string
	apiVersion     string
	base           string
	kind           string
	name           string
	namespace      string
	areaPath       string
	packageName    string
	query          string
	write          bool
	check          bool
	output         string
	bundlePath     string
	bundleSHA      string
	action         string
	adapter        string
	plan           string
	reviewConfig   string
	reviewReport   string
	reviewEvidence string
	flagCount      int
}

func parseOptions(command string, args []string, allowed map[string]bool, out, errout io.Writer) (commandOptions, int, bool) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	root := fs.String("repo", ".", "repository root")
	runManifest := fs.String("run", "", "committed fixed-run context manifest (context)")
	revision := fs.String("revision", "", "fixed Git revision (omitted: provisional working tree)")
	apiVersion := fs.String("api-version", "", "resource API version for extension resources")
	base := fs.String("base", "", "base revision for impact")
	kind := fs.String("kind", "", "context entry kind")
	name := fs.String("name", "", "resource or project name")
	namespace := fs.String("namespace", "", "resource namespace (initial area name for init)")
	areaPath := fs.String("path", "", "new ownership area path (init; defaults to .markitect/areas/<namespace>)")
	packageName := fs.String("package", "", "exact content package identity (omitted: local entry)")
	query := fs.String("query", "", "literal search text (find)")
	write := fs.Bool("write", false, "write planned files in an isolated worktree")
	check := fs.Bool("check", false, "check rendered outputs (default)")
	output := fs.String("output", "", "absent output directory (package) or ZIP file (bundle)")
	bundlePath := fs.String("bundle", "", "local release ZIP to validate and install")
	bundleSHA := fs.String("sha256", "", "expected SHA-256 of the release ZIP")
	action := fs.String("action", "", "reconciliation action: observe, plan, apply or verify")
	adapter := fs.String("adapter", "", "configured reconciliation adapter name")
	plan := fs.String("plan", "", "saved concrete YAML reconciliation plan")
	reviewConfig := fs.String("config", "", "repository-relative review configuration in the fixed snapshot")
	reviewReport := fs.String("report", "", "completed reviewer report to record (local UTF-8 file)")
	reviewEvidence := fs.String("evidence", "", "previous advisory review record to evaluate (local YAML file)")
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
	fs.Visit(func(f *flag.Flag) {
		if !allowed[f.Name] {
			invalid = f.Name
		}
	})
	if invalid != "" {
		fmt.Fprintf(errout, "--%s does not apply to %s\n", invalid, command)
		return commandOptions{}, 2, true
	}
	if *write && ((command != "render" && command != "schema" && command != "format" && command != "install" && command != "init" && command != "reconcile") || *revision != "" || *check || (command == "reconcile" && *action != "apply")) {
		fmt.Fprintln(errout, "--write only supports render, format, schema, install, init or reconcile --action apply on the working tree")
		return commandOptions{}, 2, true
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
	if *runManifest != "" && (command != "context" || !fullGitCommitID.MatchString(*revision) || *apiVersion != "" || *kind != "" || *name != "" || *namespace != "" || *packageName != "") {
		fmt.Fprintln(errout, "--run is only valid for context with --revision and supplies its own entry")
		return commandOptions{}, 2, true
	}
	return commandOptions{
		root:           *root,
		runManifest:    *runManifest,
		revision:       *revision,
		apiVersion:     *apiVersion,
		base:           *base,
		kind:           *kind,
		name:           *name,
		namespace:      *namespace,
		areaPath:       *areaPath,
		packageName:    *packageName,
		query:          *query,
		write:          *write,
		check:          *check,
		output:         *output,
		bundlePath:     *bundlePath,
		bundleSHA:      *bundleSHA,
		action:         *action,
		adapter:        *adapter,
		plan:           *plan,
		reviewConfig:   *reviewConfig,
		reviewReport:   *reviewReport,
		reviewEvidence: *reviewEvidence,
		flagCount:      fs.NFlag(),
	}, 0, false
}
