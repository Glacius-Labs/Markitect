package projectcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
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
	switch opts.action {
	case "init":
		plan, err := projectwork.Init(opts.repo, opts.name, opts.write)
		if err != nil {
			return err
		}
		return writeJSON(out, plan)
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
	default:
		return errors.New("action requires its coordinated execution or adoption service, which is not wired in this package build")
	}
	return fmt.Errorf("unsupported project action %q", opts.action)
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
		_, _ = io.WriteString(out, "Usage: markitect project <action> [flags]\nActions: init check index context impact document edit discover distill adopt plan run resume status verify apply\n")
		return
	}
	if spec, ok := actionSpecs[action]; ok {
		_, _ = fmt.Fprintf(out, "Usage: markitect project %s\n", spec.usage)
	}
}
