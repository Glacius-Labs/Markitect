package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/licenses"
)

func dispatchCommand(command string, o commandOptions, out, errout io.Writer, emit func(any) int, fail func(error) int) int {
	if command == "licenses" {
		if _, err := io.WriteString(out, licenses.Text); err != nil {
			return fail(err)
		}
		return 0
	}
	if command == "init" {
		return runInit(o.root, o.name, o.namespace, o.areaPath, o.write, emit, fail)
	}
	if command == "bundle" {
		return runBundle(o.root, o.revision, o.output, emit, fail)
	}
	if command == "pack" {
		return runPack(o.root, o.revision, o.output, emit, fail)
	}
	if command == "install" {
		return runInstall(o.root, o.bundlePath, o.bundleSHA, o.write, emit, fail)
	}
	if command == "authoring" {
		return runAuthoring(o, errout, emit, fail)
	}
	if command == "schema" {
		return runSchema(o, errout, emit, fail)
	}
	if command == "package" {
		return runPackage(o, emit, fail)
	}
	if command == "inventory" {
		return runInventory(o, emit, fail)
	}
	if command != "check" && command != "verify" && command != "context" && command != "impact" && command != "find" && command != "explain" && command != "review" && command != "render" && command != "format" {
		return fail(fmt.Errorf("unknown command %q", command))
	}
	p, err := app.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	structuralCoverage := "typed YAML graph and Markitect-owned outputs"
	if (command == "check" || command == "verify") && p.Graph.Project.Spec.Documentation != nil {
		structuralCoverage += " and configured documentation routers"
	}
	result := report{Tool: "Markitect", Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, Digest: p.Snapshot.Digest(), Status: "passed", Coverage: structuralCoverage + "; external repository gates and semantic review remain separate", Inventory: p.Inventory, Diagnostics: p.Diagnostics}
	if len(p.Diagnostics) > 0 {
		result.Status = "failed"
		if code := emit(result); code != 0 {
			return code
		}
		return 1
	}
	switch command {
	case "find":
		return runFind(o, p, toolDigest, emit, fail)
	case "explain":
		return runExplain(o, p, toolDigest, emit, fail)
	case "review":
		return runReview(o.root, p, o.packageName, o.namespace, o.kind, o.name, o.reviewConfig, o.reviewReport, o.reviewEvidence, toolDigest, emit, fail)
	case "format":
		return runFormat(o, p, emit, fail)
	case "context":
		return runContext(o, p, toolDigest, emit, fail)
	case "impact":
		return runImpact(o, p, emit, fail)
	case "render":
		if o.write {
			files, err := app.WriteOutputs(o.root, p)
			if err != nil {
				return fail(err)
			}
			result.Files = files
			result.Status = "rendered"
			return emit(result)
		}
	}
	result.Diagnostics = app.CheckOutputs(p)
	if command == "check" || command == "verify" {
		result.Diagnostics = append(result.Diagnostics, app.CheckDocumentationRouters(p)...)
	}
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
			result.Coverage = structuralCoverage + " passed; repository verification incomplete"
			exitCode := 2
			if verifyErr.Kind == "gate-failure" {
				result.Status = "failed"
				result.Coverage = structuralCoverage + " passed; repository verification stopped at a failing check; later checks were not run"
				exitCode = 1
			}
			if code := emit(result); code != 0 {
				return code
			}
			return exitCode
		}
		result.Coverage = structuralCoverage + " and all declared repository checks passed from one immutable Git snapshot; semantic review remains separate"
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
