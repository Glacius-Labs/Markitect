package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"markitect/internal/app"
	"markitect/internal/authoring"
	"markitect/internal/core"
	"markitect/internal/format"
	"markitect/internal/migrate"
	"markitect/internal/release"
	"markitect/internal/source"
)

var version = "0.1.0-rc.3-dev"

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
		fmt.Fprintln(errout, "usage: markitect <check|verify|inventory|context|impact|find|explain|authoring|review|render|format|migrate|schema|package|version> [--repo PATH] [--revision COMMIT]")
		return 2
	}
	command := args[0]
	if command == "version" {
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
	name := fs.String("name", "", "context entry name")
	namespace := fs.String("namespace", "", "context entry namespace")
	query := fs.String("query", "", "literal search text (find)")
	write := fs.Bool("write", false, "write managed outputs in an isolated worktree")
	check := fs.Bool("check", false, "check rendered outputs (default)")
	output := fs.String("output", "", "new release output directory (package)")
	reviewConfig := fs.String("config", "", "repository-relative review configuration in the fixed snapshot")
	reviewReport := fs.String("report", "", "completed reviewer report to record (local UTF-8 file)")
	reviewEvidence := fs.String("evidence", "", "previous advisory review record to evaluate (local YAML file)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	allowed := map[string]bool{}
	if command != "authoring" {
		allowed["repo"] = true
	}
	if command != "schema" && command != "package" && command != "authoring" {
		allowed["revision"] = true
	}
	switch command {
	case "context", "review", "explain":
		allowed["kind"] = true
		allowed["name"] = true
		allowed["namespace"] = true
		if command == "review" {
			allowed["config"] = true
			allowed["report"] = true
			allowed["evidence"] = true
		}
	case "find":
		allowed["query"] = true
		allowed["kind"] = true
		allowed["namespace"] = true
	case "impact":
		allowed["base"] = true
	case "render", "schema", "format":
		allowed["write"] = true
		allowed["check"] = true
	case "migrate":
		allowed["write"] = true
	case "package":
		allowed["output"] = true
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
	if *write && ((command != "render" && command != "migrate" && command != "schema" && command != "format") || *revision != "" || *check) {
		fmt.Fprintln(errout, "--write only supports render, migrate or schema on the working tree")
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
	if command == "migrate" {
		snap, err := source.Load(*root, *revision)
		if err != nil {
			return fail(err)
		}
		files, err := migrate.Konfyra(snap)
		if err != nil {
			return fail(err)
		}
		paths := make([]string, 0, len(files))
		for name := range files {
			paths = append(paths, name)
		}
		sort.Strings(paths)
		if *write {
			paths, err = app.WriteMigration(*root, snap, files)
			if err != nil {
				return fail(err)
			}
		}
		candidates, err := migrate.CandidateDependencies(snap)
		if err != nil {
			return fail(err)
		}
		return emit(map[string]any{"status": "migration-plan", "written": *write, "files": paths, "provisional": snap.Provisional, "revision": snap.Revision, "requiresDependencyReview": true, "dependencyCandidates": candidates})
	}
	if command == "inventory" {
		snap, err := source.Load(*root, *revision)
		if err != nil {
			return fail(err)
		}
		items := app.LegacyInventory(snap)
		coverage := "legacy candidates only; no semantic dependency inference or validity claim"
		if _, ok := snap.Files["markitect.yaml"]; ok {
			p, err := app.Parse(snap)
			if err != nil {
				return fail(err)
			}
			items = append(p.Inventory, items...)
			coverage = "typed canonical resources and remaining legacy candidates; generated views are not counted twice"
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
		matches, err := app.Find(p, app.FindQuery{Query: *query, Kind: *kind, Namespace: *namespace})
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
		explanation, err := app.Explain(p, key)
		if err != nil {
			return fail(err)
		}
		return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: explanation})
	case "review":
		return runReview(*root, p, *namespace, *kind, *name, *reviewConfig, *reviewReport, *reviewEvidence, toolDigest, emit, fail)
	case "format":
		files, err := app.Format(*root, p, *write)
		if err != nil {
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
		c, err := app.CompileContext(p, *namespace+"/"+*kind+"/"+*name, version, toolDigest)
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
			return fail(err)
		}
		result.Coverage = "typed graph, owned outputs and fixed profile repository gates, all from one immutable Git snapshot; semantic review remains separate"
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
