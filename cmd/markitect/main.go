package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/licenses"
	"github.com/Glacius-Labs/Markitect/internal/release"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

var version = "0.4.1"

type report struct {
	Tool        string            `yaml:"tool"`
	Version     string            `yaml:"version"`
	ToolDigest  string            `yaml:"toolDigest,omitempty"`
	Revision    string            `yaml:"revision,omitempty"`
	Provisional bool              `yaml:"provisional"`
	Digest      string            `yaml:"digest,omitempty"`
	Status      string            `yaml:"status"`
	Coverage    string            `yaml:"coverage"`
	Inventory   []app.Entry       `yaml:"inventory,omitempty"`
	Diagnostics []core.Diagnostic `yaml:"diagnostics,omitempty"`
	Files       []string          `yaml:"files,omitempty"`
	Gates       []app.GateResult  `yaml:"gates,omitempty"`
}

type queryEnvelope struct {
	Version        string `yaml:"version"`
	ToolDigest     string `yaml:"toolDigest"`
	Revision       string `yaml:"revision,omitempty"`
	Provisional    bool   `yaml:"provisional"`
	SnapshotDigest string `yaml:"snapshotDigest"`
	Result         any    `yaml:"result"`
}

func run(args []string, out, errout io.Writer) int {
	if len(args) == 0 {
		printUsage(errout)
		return 2
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		if len(args) == 1 {
			printUsage(out)
			return 0
		}
		if args[0] == "help" && len(args) == 2 {
			return run([]string{args[1], "--help"}, out, errout)
		}
		fmt.Fprintln(errout, "help accepts at most one command")
		return 2
	}
	command := args[0]
	allowed, known := commandFlags(command)
	if !known {
		fmt.Fprintf(errout, "unknown command %q\n", command)
		return 2
	}
	if command == "version" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprintln(out, "usage: markitect version")
			return 0
		}
		if len(args) != 1 {
			fmt.Fprintln(errout, "version accepts no arguments")
			return 2
		}
		fmt.Fprintf(out, "Markitect %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return 0
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	root := fs.String("repo", ".", "repository root")
	revision := fs.String("revision", "", "fixed Git revision (omitted: provisional working tree)")
	base := fs.String("base", "", "base revision for impact")
	kind := fs.String("kind", "", "context entry kind")
	name := fs.String("name", "", "resource or project name")
	namespace := fs.String("namespace", "", "resource namespace (initial area name for init)")
	areaPath := fs.String("path", "", "new ownership area path (init)")
	packageName := fs.String("package", "", "exact content package identity (omitted: local entry)")
	query := fs.String("query", "", "literal search text (find)")
	write := fs.Bool("write", false, "write planned files in an isolated worktree")
	check := fs.Bool("check", false, "check rendered outputs (default)")
	output := fs.String("output", "", "absent output directory (package) or ZIP file (bundle)")
	bundlePath := fs.String("bundle", "", "local release ZIP to validate and install")
	bundleSHA := fs.String("sha256", "", "expected SHA-256 of the release ZIP")
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
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	invalid := ""
	fs.Visit(func(f *flag.Flag) {
		if !allowed[f.Name] {
			invalid = f.Name
		}
	})
	if invalid != "" {
		fmt.Fprintf(errout, "--%s does not apply to %s\n", invalid, command)
		return 2
	}
	if *write && ((command != "render" && command != "schema" && command != "format" && command != "install" && command != "init") || *revision != "" || *check) {
		fmt.Fprintln(errout, "--write only supports render, format, schema, install or init on the working tree")
		return 2
	}
	emit := func(value any) int {
		data, err := app.YAML(value)
		if err != nil {
			fmt.Fprintln(errout, err)
			return 2
		}
		if _, err = out.Write(data); err != nil {
			fmt.Fprintln(errout, err)
			return 2
		}
		return 0
	}
	fail := func(err error) int { fmt.Fprintln(errout, err); return 2 }
	if command == "licenses" {
		if _, err := io.WriteString(out, licenses.Text); err != nil {
			return fail(err)
		}
		return 0
	}
	if command == "init" {
		return runInit(*root, *name, *namespace, *areaPath, *write, emit, fail)
	}
	if command == "bundle" {
		return runBundle(*root, *revision, *output, emit, fail)
	}
	if command == "pack" {
		return runPack(*root, *revision, *output, emit, fail)
	}
	if command == "install" {
		return runInstall(*root, *bundlePath, *bundleSHA, *write, emit, fail)
	}
	if command == "authoring" {
		if fs.NFlag() != 0 {
			fmt.Fprintln(errout, "authoring accepts no flags; its compiled context has no repository or revision")
			return 2
		}
		digest, err := currentToolDigest()
		if err != nil {
			return fail(err)
		}
		ctx, err := authoring.Context(version, digest)
		if err != nil {
			return fail(err)
		}
		return emit(ctx)
	}
	if command == "schema" {
		schemas, err := format.Schemas()
		if err != nil {
			return fail(err)
		}
		for name, data := range schemas {
			target := filepath.Join(*root, filepath.FromSlash(name))
			data = append([]byte("# Generated by Markitect; edit Go validation declarations.\n"), data...)
			schemas[name] = data
			if !*write {
				existing, err := os.ReadFile(target)
				if err != nil || string(existing) != string(data) {
					fmt.Fprintln(errout, "schema drift:", name)
					return 1
				}
			}
		}
		if *write {
			if err = app.WriteSchemas(*root, schemas); err != nil {
				return fail(err)
			}
		}
		return emit(map[string]any{"status": "passed", "schemas": len(schemas), "version": version})
	}
	if command == "package" {
		if *output == "" {
			return fail(fmt.Errorf("package requires --output pointing to an absent directory"))
		}
		if _, err := os.Lstat(*output); !os.IsNotExist(err) {
			return fail(fmt.Errorf("release output must not already exist"))
		}
		archive, lock, err := release.Package(*root, version)
		if err != nil {
			return fail(err)
		}
		if err = os.MkdirAll(filepath.Join(*output, "tools", "markitect"), 0755); err != nil {
			return fail(err)
		}
		if err = os.WriteFile(filepath.Join(*output, "tools", "markitect", "source.zip"), archive, 0644); err != nil {
			return fail(err)
		}
		if err = os.WriteFile(filepath.Join(*output, "markitect.lock.yaml"), lock, 0644); err != nil {
			return fail(err)
		}
		return emit(map[string]any{"status": "packaged", "version": version, "output": *output})
	}
	if command == "inventory" {
		snap, err := source.Load(*root, *revision)
		if err != nil {
			return fail(err)
		}
		items := app.MarkdownInventory(snap)
		coverage := "ordinary Markdown candidates only; no resource classification, semantic dependency inference or validity claim"
		if _, ok := snap.Files["markitect.yaml"]; ok {
			p, err := app.Parse(snap)
			if err != nil {
				return fail(err)
			}
			items = append(p.Inventory, items...)
			coverage = "typed canonical resources and ordinary Markdown candidates; generated views are not counted twice"
		}
		return emit(report{Tool: "Markitect", Version: version, Revision: snap.Revision, Provisional: snap.Provisional, Digest: snap.Digest(), Status: "inventory", Coverage: coverage, Inventory: items})
	}
	if command != "check" && command != "verify" && command != "context" && command != "impact" && command != "find" && command != "explain" && command != "review" && command != "render" && command != "format" {
		return fail(fmt.Errorf("unknown command %q", command))
	}
	p, err := app.Load(*root, *revision)
	if err != nil {
		return fail(err)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	result := report{Tool: "Markitect", Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, Digest: p.Snapshot.Digest(), Status: "passed", Coverage: "typed YAML graph and Markitect-owned outputs; external repository gates and semantic review remain separate", Inventory: p.Inventory, Diagnostics: p.Diagnostics}
	if len(p.Diagnostics) > 0 {
		result.Status = "failed"
		if code := emit(result); code != 0 {
			return code
		}
		return 1
	}
	switch command {
	case "find":
		matches, err := app.Find(p, app.FindQuery{Query: *query, Kind: *kind, Namespace: *namespace, Package: *packageName})
		if err != nil {
			return fail(err)
		}
		return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: matches})
	case "explain":
		if *kind == "" || *name == "" {
			return fail(fmt.Errorf("explain requires --kind and --name"))
		}
		if *kind == "Project" {
			if *namespace != "" {
				return fail(fmt.Errorf("Project explain identity does not accept --namespace"))
			}
		} else if *namespace == "" {
			return fail(fmt.Errorf("explain requires --namespace for namespaced resources"))
		}
		key := "/" + *kind + "/" + *name
		if *namespace != "" {
			key = *namespace + "/" + *kind + "/" + *name
		}
		if *packageName != "" {
			key = *packageName + "::" + key
		}
		explanation, err := app.Explain(p, key)
		if err != nil {
			return fail(err)
		}
		return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: explanation})
	case "review":
		return runReview(*root, p, *packageName, *namespace, *kind, *name, *reviewConfig, *reviewReport, *reviewEvidence, toolDigest, emit, fail)
	case "format":
		files, err := app.Format(*root, p, *write)
		if err != nil {
			if len(files) > 0 {
				if code := emit(map[string]any{"status": "failed", "written": files, "recovery": "Inspect the listed paths and Git diff before retrying; the complete operation is not a filesystem transaction."}); code != 0 {
					return code
				}
			}
			return fail(err)
		}
		code := emit(map[string]any{"changed": files, "written": *write})
		if code != 0 {
			return code
		}
		if !*write && len(files) > 0 {
			return 1
		}
		return 0
	case "context":
		if *kind == "" || *name == "" || *namespace == "" {
			return fail(fmt.Errorf("context requires --kind, --name and --namespace"))
		}
		key := (core.Ref{Package: *packageName, Namespace: *namespace, Kind: *kind, Name: *name}).GraphKey("", "", "")
		c, err := app.CompileContext(p, key, version, toolDigest)
		if err != nil {
			return fail(err)
		}
		return emit(c)
	case "impact":
		if *base == "" || *revision == "" {
			return fail(fmt.Errorf("impact requires fixed --base and --revision"))
		}
		previous, err := app.Load(*root, *base)
		if err != nil {
			return fail(err)
		}
		if len(previous.Diagnostics) > 0 {
			return fail(fmt.Errorf("base has unresolved diagnostics; inspect the base separately"))
		}
		return emit(app.Changes(previous, p))
	case "render":
		if *write {
			files, err := app.WriteOutputs(*root, p)
			if err != nil {
				return fail(err)
			}
			result.Files = files
			result.Status = "rendered"
			return emit(result)
		}
	}
	result.Diagnostics = app.CheckOutputs(p)
	if len(result.Diagnostics) > 0 {
		result.Status = "failed"
	}
	if command == "verify" && result.Status == "passed" {
		result.Gates, err = app.VerifyRepository(p)
		if err != nil {
			var verifyErr *app.VerifyError
			if !errors.As(err, &verifyErr) {
				verifyErr = &app.VerifyError{Kind: "incomplete-evidence", Err: err}
			}
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
				Code:    "verify." + verifyErr.Kind,
				Path:    verifyErr.Gate,
				Message: verifyErr.Error(),
			})
			fmt.Fprintf(errout, "verify: %s\n", verifyErr)
			result.Status = "incomplete"
			result.Coverage = "typed graph and Markitect-owned outputs passed; repository verification incomplete"
			exitCode := 2
			if verifyErr.Kind == "gate-failure" {
				result.Status = "failed"
				result.Coverage = "typed graph and Markitect-owned outputs passed; repository verification stopped at a failing check; later checks were not run"
				exitCode = 1
			}
			if code := emit(result); code != 0 {
				return code
			}
			return exitCode
		}
		result.Coverage = "typed graph, owned outputs, and all declared repository checks passed from one immutable Git snapshot; semantic review remains separate"
		for _, gate := range result.Gates {
			if gate.ExitCode != 0 {
				result.Status = "failed"
			}
		}
	}
	code := emit(result)
	if code != 0 {
		return code
	}
	if result.Status == "failed" {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func currentToolDigest() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executableBytes, err := os.ReadFile(executable)
	if err != nil {
		return "", err
	}
	return app.Hash(executableBytes), nil
}
