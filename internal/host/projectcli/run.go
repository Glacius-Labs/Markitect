package projectcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

// Run executes one closed project action. It composes the project frontend
// services; model compilation and impact semantics stay in their owning packages.
func Run(args []string, out, errout io.Writer) int {
	opts, help, err := parse(args, errout)
	if err != nil {
		fmt.Fprintf(errout, "markitect project: %v\n", err)
		return 2
	}
	if help {
		printUsage(out, opts.action)
		return 0
	}
	if out == nil || errout == nil {
		return 2
	}
	if err := runAction(opts, out); err != nil {
		fmt.Fprintf(errout, "markitect project %s: %v\n", opts.action, err)
		var outcome *projectOutcomeError
		if errors.As(err, &outcome) {
			return outcome.code
		}
		return 2
	}
	return 0
}

func runAction(opts options, out io.Writer) error {
	ctx := context.Background()
	if opts.action == "run" || opts.action == "resume" || opts.action == "verify" || (opts.action == "distill" && opts.generate) {
		bounded, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		ctx = bounded
	}
	switch opts.action {
	case "schema":
		return writeJSON(out, projectmodel.Schema())
	case "init":
		plan, err := projectwork.Init(opts.repo, opts.name, opts.write)
		if err != nil {
			return err
		}
		return writeJSON(out, plan)
	case "setup", "doctor":
		project, err := projectwork.Load(opts.repo, "")
		if err != nil {
			return err
		}
		if opts.action == "doctor" {
			setupOptions := projectsetup.Options{Provider: opts.provider, ToolRoot: opts.toolRoot, ProviderExecutable: opts.providerExecutable}
			report, err := projectsetup.Doctor(project, setupOptions)
			if err != nil {
				return err
			}
			if err := writeJSON(out, report); err != nil {
				return err
			}
			for _, check := range report.Checks {
				if check.Status == "blocked" || check.Status == "missing" {
					return &projectOutcomeError{code: 1, message: "one or more local prerequisites need attention"}
				}
			}
			return nil
		}
		setupOptions, err := setupOptions(opts)
		if err != nil {
			return err
		}
		preview, err := projectsetup.PreviewEdit(project, setupOptions)
		if err != nil {
			return err
		}
		if opts.write {
			if opts.expect != preview.EditPlan.Digest {
				return fmt.Errorf("--expect does not match the exact runtime edit plan digest %s", preview.EditPlan.Digest)
			}
			applied, err := projectwork.ApplyEdit(opts.repo, preview.EditPlan, preview.EditPlan.BaseDigest)
			if err != nil {
				return err
			}
			preview.EditPlan = applied
		}
		return writeJSON(out, preview)
	case "check", "index", "context", "document", "edit":
		project, err := projectwork.Load(opts.repo, opts.revision)
		if err != nil {
			return err
		}
		switch opts.action {
		case "check":
			if err := writeJSON(out, struct {
				ProjectDigest string                 `json:"projectDigest"`
				Revision      string                 `json:"revision"`
				Provisional   bool                   `json:"provisional"`
				Status        string                 `json:"status"`
				Findings      []projectmodel.Finding `json:"findings"`
				Unknown       []string               `json:"unknown"`
			}{project.Digest, project.Revision, project.Provisional, project.Report.Status, project.Report.Findings, project.Report.Unknown}); err != nil {
				return err
			}
			if project.Report.Status != "succeeded" {
				return &projectOutcomeError{code: 1, message: "project report is " + project.Report.Status}
			}
			return nil
		case "index":
			return writeJSON(out, project.Report)
		case "context":
			result, err := projectmodel.Context(project.Report, opts.manager)
			if err != nil {
				return err
			}
			return writeJSON(out, result)
		case "document":
			content, err := projectwork.Document(project, opts.write)
			if err != nil {
				return err
			}
			_, err = io.WriteString(out, content)
			return err
		case "edit":
			data, err := readRecord(opts.repo, opts.input)
			if err != nil {
				return err
			}
			mutation, err := projectwork.DecodeMutation(data)
			if err != nil {
				return err
			}
			plan, err := projectwork.PlanEdit(project, mutation)
			if err != nil {
				return err
			}
			if opts.write {
				if opts.expect != plan.Digest {
					return fmt.Errorf("--expect does not match the exact edit plan digest %s", plan.Digest)
				}
				plan, err = projectwork.ApplyEdit(opts.repo, plan, plan.BaseDigest)
				if err != nil {
					return err
				}
			}
			return writeJSON(out, plan)
		}
	case "impact":
		base, err := projectwork.Load(opts.repo, opts.base)
		if err != nil {
			return fmt.Errorf("load base project: %w", err)
		}
		candidate, err := projectwork.Load(opts.repo, opts.revision)
		if err != nil {
			return fmt.Errorf("load candidate project: %w", err)
		}
		return writeJSON(out, projectmodel.Impact(base.Report, candidate.Report))
	case "discover":
		data, err := readRecord(opts.repo, opts.request)
		if err != nil {
			return err
		}
		request, err := projectadoption.DecodeDiscoveryRequest(data)
		if err != nil {
			return err
		}
		discovery, err := projectadoption.Discover(opts.repo, request)
		if err != nil {
			return err
		}
		encoded, err := projectadoption.EncodeDiscovery(discovery)
		if err != nil {
			return err
		}
		return emitRecord(opts.repo, opts.output, encoded, out)
	case "distill":
		discoveryBytes, err := readRecord(opts.repo, opts.discovery)
		if err != nil {
			return err
		}
		discovery, err := projectadoption.DecodeDiscovery(discoveryBytes)
		if err != nil {
			return err
		}
		if opts.generate {
			head, err := source.GitOutput(opts.repo, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
			if err != nil {
				return fmt.Errorf("generated distillation requires a committed HEAD: %w", err)
			}
			fixedProject, err := projectwork.Load(opts.repo, strings.TrimSpace(string(head)))
			if err != nil {
				return fmt.Errorf("load committed project HEAD before generated distillation: %w", err)
			}
			if fixedProject.Provisional || fixedProject.Revision == "" || fixedProject.Report.Status != "succeeded" {
				return errors.New("generated distillation requires a committed HEAD with a successful project check")
			}
			workingProject, err := projectwork.Load(opts.repo, "")
			if err != nil {
				return err
			}
			if workingProject.Snapshot == nil || fixedProject.Snapshot == nil || workingProject.Snapshot.Digest() != fixedProject.Snapshot.Digest() {
				return errors.New("selected project inputs differ from committed HEAD; commit accepted model, runtime, inventory, or selection changes before generated distillation")
			}
			runtimeConfig, err := projectrun.LoadRuntime(opts.repo)
			if err != nil {
				return err
			}
			rootManager := ""
			for _, manager := range fixedProject.Report.Managers {
				if manager.Parent == "" && manager.Namespace == "" {
					if rootManager != "" {
						return errors.New("project has multiple root Managers")
					}
					rootManager = manager.ID
				}
			}
			agent, ok := runtimeConfig.Agents[rootManager]
			if rootManager == "" || !ok {
				return errors.New("configured runtime has no agent mapped to the active root Manager")
			}
			agentConfig, err := agent.AgentConfig()
			if err != nil {
				return err
			}
			targetContext, err := projectadoption.TargetContextForProject(fixedProject)
			if err != nil {
				return err
			}
			prices, err := parsePositiveRates(opts.inputMicros, opts.outputMicros, opts.maxCost)
			if err != nil {
				return err
			}
			report, receipt, err := projectadoption.GenerateDistillation(ctx, opts.repo, discovery, agentConfig, projectadoption.DistillationRunOptions{
				MaxTimeout: agentConfig.Timeout, MaxStdoutBytes: agentConfig.MaxStdoutBytes, MaxStderrBytes: agentConfig.MaxStderrBytes,
				MaxCostMicros: prices.maxCost, InputPriceMicrosPerMillion: prices.input, OutputPriceMicrosPerMillion: prices.output,
				TargetContext: targetContext,
			})
			if err != nil {
				return err
			}
			encoded, err := projectadoption.EncodeDistillation(report)
			if err != nil {
				return err
			}
			receiptBytes, err := json.Marshal(receipt)
			if err != nil {
				return err
			}
			receiptPath, err := derivedReceiptPath(opts.output)
			if err != nil {
				return err
			}
			digests, err := writeRecords(opts.repo, map[string][]byte{opts.output: encoded, receiptPath: receiptBytes})
			if err != nil {
				return err
			}
			return writeJSON(out, map[string]string{"reportPath": opts.output, "reportDigest": digests[opts.output], "receiptPath": receiptPath, "receiptDigest": digests[receiptPath]})
		}
		reportBytes, err := readRecord(opts.repo, opts.report)
		if err != nil {
			return err
		}
		report, err := projectadoption.DecodeDistillation(reportBytes, discovery)
		if err != nil {
			return err
		}
		encoded, err := projectadoption.EncodeDistillation(report)
		if err != nil {
			return err
		}
		return emitRecord(opts.repo, opts.output, encoded, out)
	case "resolve":
		discoveryBytes, err := readRecord(opts.sourceRepo, opts.discovery)
		if err != nil {
			return err
		}
		discovery, err := projectadoption.DecodeDiscovery(discoveryBytes)
		if err != nil {
			return err
		}
		reportBytes, err := readRecord(opts.sourceRepo, opts.report)
		if err != nil {
			return err
		}
		report, err := projectadoption.DecodeDistillation(reportBytes, discovery)
		if err != nil {
			return err
		}
		choiceBytes, err := readRecord(opts.sourceRepo, opts.input)
		if err != nil {
			return err
		}
		choices, err := decodeResolutionChoices(choiceBytes)
		if err != nil {
			return err
		}
		target, err := projectwork.Load(opts.repo, opts.revision)
		if err != nil {
			return err
		}
		resolution, err := buildResolution(discovery, report, target, choices)
		if err != nil {
			return err
		}
		encoded, err := projectadoption.EncodeResolution(resolution)
		if err != nil {
			return err
		}
		return emitRecord(opts.sourceRepo, opts.output, encoded, out)
	case "adopt":
		discoveryBytes, err := readRecord(opts.sourceRepo, opts.discovery)
		if err != nil {
			return err
		}
		discovery, err := projectadoption.DecodeDiscovery(discoveryBytes)
		if err != nil {
			return err
		}
		reportBytes, err := readRecord(opts.sourceRepo, opts.report)
		if err != nil {
			return err
		}
		report, err := projectadoption.DecodeDistillation(reportBytes, discovery)
		if err != nil {
			return err
		}
		resolutionBytes, err := readRecord(opts.sourceRepo, opts.resolution)
		if err != nil {
			return err
		}
		resolution, err := projectadoption.DecodeResolution(resolutionBytes, discovery, report)
		if err != nil {
			return err
		}
		target, err := projectwork.Load(opts.repo, opts.revision)
		if err != nil {
			return err
		}
		schemaDigest, buildDigest, err := projectadoption.CurrentBindings(projectmodel.Schema())
		if err != nil {
			return err
		}
		if opts.write {
			planBytes, err := readRecord(opts.sourceRepo, opts.plan)
			if err != nil {
				return err
			}
			plan, err := projectadoption.DecodeAdoptionPlan(planBytes)
			if err != nil {
				return err
			}
			if err := projectadoption.ValidateAdoptionPlan(plan); err != nil {
				return err
			}
			receipt, err := projectadoption.ApplyAdoption(opts.sourceRepo, opts.repo, target, discovery, report, resolution, plan, opts.expect, schemaDigest, buildDigest)
			if err != nil {
				return err
			}
			return writeJSON(out, receipt)
		}
		plan, err := projectadoption.PlanAdoption(opts.sourceRepo, target, discovery, report, resolution, schemaDigest, buildDigest)
		if err != nil {
			return err
		}
		encoded, err := projectadoption.EncodeAdoptionPlan(plan)
		if err != nil {
			return err
		}
		return emitRecord(opts.sourceRepo, opts.output, encoded, out)
	case "plan":
		request := projectrun.PlanRequest{Goal: opts.goal, Managers: append([]string(nil), opts.managers...), BaseRevision: opts.revision, SinceRevision: opts.since, ExecuteAuthorized: opts.write}
		plan, err := projectrun.Plan(projectRunHost(), opts.repo, opts.revision, request)
		if err != nil {
			return err
		}
		return writeJSON(out, plan)
	case "run":
		report, err := projectrun.Run(ctx, projectRunHost(), projectrun.ProcessInvoker{}, opts.repo, opts.plan)
		if err != nil {
			return err
		}
		return writeJSON(out, report)
	case "resume":
		report, err := projectrun.Resume(ctx, projectRunHost(), projectrun.ProcessInvoker{}, opts.repo, opts.run)
		if err != nil {
			return err
		}
		return writeJSON(out, report)
	case "status":
		report, err := projectrun.Status(opts.repo, opts.run)
		if err != nil {
			return err
		}
		return writeJSON(out, report)
	case "verify":
		report, err := projectrun.Verify(ctx, projectRunHost(), projectrun.ProcessInvoker{}, opts.repo, opts.run)
		if err != nil {
			return err
		}
		return writeJSON(out, report)
	case "apply":
		host := projectRunHost()
		if !opts.write {
			preflight, err := projectrun.PreflightApply(host, opts.repo, opts.run, opts.candidate)
			if err != nil {
				return err
			}
			if preflight.PlanID != opts.plan {
				return fmt.Errorf("requested plan does not match the verified run")
			}
			return writeJSON(out, preflight)
		}
		report, err := projectrun.Apply(host, projectrun.ProcessInvoker{}, opts.repo, projectrun.ApplyRequest{
			RunID: opts.run, PlanID: opts.plan, CandidateID: opts.candidate,
			TargetBranch: opts.branch, ExpectedHead: opts.head, ExpectedWorktree: opts.tree,
			ExpectedVerificationDigest: opts.expect,
		})
		if err != nil {
			return err
		}
		if report.Status != projectrun.StatusApplied {
			return &projectOutcomeError{code: 1, message: "project apply status is " + report.Status}
		}
		return writeJSON(out, report)
	default:
		return errors.New("action requires a project service that is not wired in this package build")
	}
	return fmt.Errorf("unsupported project action %q", opts.action)
}

func projectRunHost() projectrun.Host {
	return projectrun.Host{
		Load:         projectwork.Load,
		FromSnapshot: projectwork.FromSnapshot,
		PlanEdit:     projectwork.PlanEdit,
		ApplyEdit:    projectwork.ApplyEdit,
	}
}

func setupOptions(opts options) (projectsetup.Options, error) {
	rates, err := parsePositiveRates(opts.inputMicros, opts.outputMicros, opts.maxCost)
	if err != nil {
		return projectsetup.Options{}, err
	}
	return projectsetup.Options{
		Provider: opts.provider, Model: opts.model, Effort: opts.effort, ToolRoot: opts.toolRoot,
		ProviderExecutable: opts.providerExecutable, InputMicrosPerMillion: rates.input,
		OutputMicrosPerMillion: rates.output, MaxCostMicros: rates.maxCost,
	}, nil
}

type rateOptions struct{ input, output, maxCost int64 }

func parsePositiveRates(inputText, outputText, maxText string) (rateOptions, error) {
	var result rateOptions
	values := []struct {
		name string
		text string
		dest *int64
	}{
		{"input-micros-per-million", inputText, &result.input},
		{"output-micros-per-million", outputText, &result.output},
		{"max-cost-micros", maxText, &result.maxCost},
	}
	for _, value := range values {
		parsed, err := strconv.ParseInt(value.text, 10, 64)
		if err != nil || parsed < 0 {
			return result, fmt.Errorf("--%s must be a nonnegative integer", value.name)
		}
		*value.dest = parsed
	}
	if result.input == 0 && result.output == 0 {
		return result, errors.New("at least one input/output price rate must be positive")
	}
	if result.maxCost <= 0 || result.maxCost > 1_000_000_000_000 {
		return result, errors.New("--max-cost-micros must be between 1 and 1000000000000")
	}
	if result.input > 1_000_000_000_000 || result.output > 1_000_000_000_000 {
		return result, errors.New("price rates must not exceed 1000000000000 micros per million tokens")
	}
	return result, nil
}

func derivedReceiptPath(output string) (string, error) {
	if !strings.HasPrefix(output, ".markitect/drafts/") || !strings.HasSuffix(output, ".json") {
		return "", errors.New("generated distillation --output must be an explicit .json path under .markitect/drafts/")
	}
	return strings.TrimSuffix(output, ".json") + ".receipt.json", nil
}

type projectOutcomeError struct {
	code    int
	message string
}

func (e *projectOutcomeError) Error() string { return e.message }

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printUsage(out io.Writer, action string) {
	if action == "" {
		_, _ = io.WriteString(out, "Usage: markitect project <action> [flags]\nActions: schema init check index context impact document edit discover distill resolve adopt setup doctor plan run resume status verify apply\n")
		return
	}
	if spec, ok := actionSpecs[action]; ok {
		_, _ = fmt.Fprintf(out, "Usage: markitect project %s\n", spec.usage)
	}
}
