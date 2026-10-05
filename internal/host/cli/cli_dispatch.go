package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/host"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"github.com/Glacius-Labs/Markitect/internal/tooling/licenses"
)

func dispatchCommand(command string, o commandOptions, out, errout io.Writer, emit func(any) int, fail func(error) int) int {
	if command == "prepare" {
		return runPrepare(o, emit, fail)
	}
	if command == "copy-me" {
		return runCopyMe(o, emit, fail)
	}
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
	if command == "canonical" {
		return runCanonical(o, emit, fail)
	}
	if command == "package" {
		return runPackage(o, emit, fail)
	}
	if command == "inventory" {
		return runInventory(o, emit, fail)
	}
	if command != "check" && command != "verify" && command != "context" && command != "impact" && command != "find" && command != "explain" && command != "review" && command != "render" && command != "format" && command != "reconcile" && command != "model" && command != "projection" {
		return fail(fmt.Errorf("unknown command %q", command))
	}
	if command == "context" && o.runManifest != "" {
		toolDigest, err := currentToolDigest()
		if err != nil {
			return fail(err)
		}
		return runContextManifest(o, toolDigest, emit, fail)
	}
	p, err := host.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	if command == "model" {
		model, err := host.CompileModel(p)
		if err != nil {
			return fail(err)
		}
		if code := emit(model); code != 0 {
			return code
		}
		if model.ValidationStatus != "passed" {
			return 1
		}
		return 0
	}
	structuralCoverage := "typed YAML graph and Markitect-owned outputs"
	if (command == "check" || command == "verify") && p.Graph.Project.Spec.Documentation != nil {
		structuralCoverage += " and configured documentation routers"
	}
	result := report{Tool: "Markitect", Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.ID, Provisional: p.Snapshot.Provisional, Digest: p.Snapshot.Digest(), Status: "passed", Coverage: structuralCoverage + "; external repository gates and semantic review remain separate", Inventory: p.Inventory, Diagnostics: p.Diagnostics, PolicyResults: p.Graph.Core.PolicyResults}
	allowPolicyFailures := o.analyzePolicyFailures && (command == "context" || command == "impact")
	if len(p.Diagnostics) > 0 && (!allowPolicyFailures || len(p.StructuralDiagnostics()) > 0) {
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
		return runReview(o.root, p, o.packageName, o.apiVersion, o.namespace, o.kind, o.name, o.reviewConfig, o.reviewReport, o.reviewEvidence, toolDigest, emit, fail)
	case "format":
		return runFormat(o, p, emit, fail)
	case "projection":
		return runProjection(o, p, emit, fail)
	case "reconcile":
		return runReconcile(o, p, emit, fail)
	case "context":
		return runContext(o, p, toolDigest, emit, fail)
	case "impact":
		return runImpact(o, p, emit, fail)
	case "render":
		if o.write {
			files, err := host.WriteOutputs(o.root, p)
			if err != nil {
				return fail(err)
			}
			result.Files = files
			result.Status = "rendered"
			return emit(result)
		}
	}
	result.Diagnostics = host.CheckOutputs(p)
	if command == "check" || command == "verify" {
		result.Diagnostics = append(result.Diagnostics, host.CheckDocumentationRouters(p)...)
	}
	if len(result.Diagnostics) > 0 {
		result.Status = "failed"
	}

	configPath, coveragePath, projectionActive, registrationErr := host.RegisteredProjection(p)
	if registrationErr != nil {
		return fail(registrationErr)
	}
	if projectionActive && (command == "check" || command == "verify") {
		var projectionReport host.ProjectionReport
		if command == "verify" && result.Status == "passed" {
			projectionReport, err = host.VerifyRepresentations(p, configPath, coveragePath, version, toolDigest)
		} else {
			projectionReport, err = host.ObserveRepresentations(p, configPath, coveragePath, version, toolDigest)
		}
		if err != nil {
			return fail(err)
		}
		result.Projections = &projectionReport
		result.Gates = projectionReport.Checks
		result.Coverage += "; registered projection state is separately reported; check does not establish AI semantics"
		if command == "verify" && result.Status == "passed" {
			switch projectionReport.Status {
			case "converged":
				result.Coverage = structuralCoverage + " and registered projection contracts converged relative to declared fixed-snapshot checks and artifact roots; semantic review remains separate"
			case "drift", "failed", "blocked":
				result.Status = "failed"
			default:
				result.Status = "incomplete"
			}
		}
	}
	if command == "verify" && result.Status == "passed" && !projectionActive {
		result.Gates, err = host.VerifyRepository(p)
		if err != nil {
			var verifyErr *host.VerifyError
			if !errors.As(err, &verifyErr) {
				verifyErr = &host.VerifyError{Kind: "incomplete-evidence", Err: err}
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
	if result.Status == "incomplete" {
		return 2
	}
	return 0
}
