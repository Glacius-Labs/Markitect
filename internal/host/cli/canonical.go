package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const canonicalPlanReportAPIVersion = "markitect.canonical-plan/v1alpha1"

type canonicalPlanReport struct {
	APIVersion      string                    `yaml:"apiVersion"`
	Status          string                    `yaml:"status"`
	Revision        string                    `yaml:"revision"`
	Provisional     bool                      `yaml:"provisional"`
	SourceDigest    string                    `yaml:"sourceDigest"`
	Request         map[string]any            `yaml:"request"`
	CandidateDigest string                    `yaml:"candidateDigest"`
	Outputs         map[string][]byte         `yaml:"outputs"`
	Plan            *projectionengine.Plan    `yaml:"plan"`
	Escalations     []canonicalPlanEscalation `yaml:"escalations,omitempty"`
}

type canonicalPlanEscalation struct {
	Code     string `yaml:"code"`
	Identity string `yaml:"identity,omitempty"`
	Message  string `yaml:"message"`
}

func runCanonical(o commandOptions, emit func(any) int, fail func(error) int) int {
	if isCanonicalControllerAction(o.action) {
		return runCanonicalController(o, emit, fail)
	}
	requirePins := o.action != "modules"
	loadRevision := o.revision
	if o.action == "verify" || o.action == "adopt-plan" || o.action == "adopt" {
		loadRevision = o.base
	}
	loaded, err := host.LoadCanonicalSource(o.root, loadRevision, o.reviewConfig, requirePins)
	if err != nil {
		return fail(err)
	}
	base := map[string]any{
		"revision":     loaded.Snapshot.ID,
		"provisional":  loaded.Snapshot.Provisional,
		"sourceDigest": loaded.Snapshot.Digest(),
	}
	switch o.action {
	case "modules":
		previews := append([]host.CanonicalModulePreview(nil), loaded.Previews...)
		sort.Slice(previews, func(i, j int) bool { return previews[i].Pin.Name < previews[j].Pin.Name })
		base["status"] = "preview"
		base["modules"] = previews
		return emit(base)
	case "model":
		if len(loaded.Diagnostics) > 0 {
			base["status"] = "failed"
			base["diagnostics"] = loaded.Diagnostics
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		base["status"] = "passed"
		base["model"] = loaded.Model
		base["modules"] = loaded.Previews
		base["projectors"] = loaded.Activation.Projectors
		return emit(base)
	case "context":
		if len(loaded.Diagnostics) > 0 {
			base["status"] = "failed"
			base["diagnostics"] = loaded.Diagnostics
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		identity := core.DefinitionIdentity{APIVersion: o.apiVersion, Kind: o.kind, Namespace: o.namespace, Name: o.name}
		definition, found := loaded.Model.Definition(identity)
		if !found {
			base["status"] = "failed"
			base["diagnostics"] = []core.Diagnostic{{Code: "definition.notfound", Identity: identity.Key(), Message: "the exact selected Definition is not in the explicitly listed source paths"}}
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		var selectedKind core.Kind
		var selectedSchema core.Schema
		for _, schema := range loaded.Model.Schemas {
			if schema.APIVersion == identity.APIVersion {
				selectedSchema = schema
				selectedKind = schema.Kinds[identity.Kind]
				break
			}
		}
		references := make([]core.Edge, 0)
		for _, edge := range loaded.Model.Edges {
			if edge.From == identity.Key() {
				references = append(references, edge)
			}
		}
		base["status"] = "selected"
		base["selection"] = map[string]any{
			"definition":         definition,
			"schema":             map[string]any{"apiVersion": selectedSchema.APIVersion, "purpose": selectedSchema.Purpose, "kind": selectedKind},
			"outgoingReferences": references,
		}
		return emit(base)
	case "request":
		if len(loaded.Diagnostics) > 0 {
			base["status"] = "failed"
			base["diagnostics"] = loaded.Diagnostics
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		identity := core.DefinitionIdentity{APIVersion: o.apiVersion, Kind: o.kind, Namespace: o.namespace, Name: o.name}
		selected, found := loaded.Model.Definition(identity)
		if !found {
			return fail(fmt.Errorf("the exact selected Projection Definition is not in the explicitly listed source paths"))
		}
		var targets map[string][]byte
		var err error
		targets, err = selectedProjectionTargetFiles(selected, loaded.Snapshot.Files)
		if err != nil {
			return fail(err)
		}
		request, err := canonical.BindProjection(loaded.Model, loaded.Activation, loaded.Config.ProjectionBindings, identity, targets)
		if err != nil {
			return fail(err)
		}
		base["status"] = "bound"
		base["request"] = canonicalRequestOutput(request)
		return emit(base)
	case "impact":
		if len(loaded.Diagnostics) > 0 {
			base["status"] = "failed"
			base["diagnostics"] = loaded.Diagnostics
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		baseSource, err := host.LoadCanonicalSource(o.root, o.base, o.reviewConfig, true)
		if err != nil {
			return fail(err)
		}
		if len(baseSource.Diagnostics) > 0 {
			base["status"] = "failed"
			base["diagnostics"] = baseSource.Diagnostics
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		impact, err := host.AnalyzeCanonicalImpact(baseSource, loaded, nil)
		if err != nil {
			return fail(err)
		}
		base["status"] = "analyzed"
		base["baseRevision"] = impact.BaseRevision
		base["candidateRevision"] = impact.CandidateRevision
		base["impact"] = impact
		return emit(base)
	case "plan", "apply":
		return runCanonicalPlanApply(o, base, emit, fail)
	case "reconcile-plan":
		return runCanonicalReconcilePlan(o, loaded, base, emit, fail)
	case "verify":
		return runCanonicalVerify(o, loaded, base, emit, fail)
	case "adopt-plan", "adopt":
		return runCanonicalAdoption(o, loaded, base, emit, fail)
	default:
		return fail(fmt.Errorf("canonical requires --action model, modules, context, request, impact, reconcile-plan, plan, apply, verify, adopt-plan or adopt"))
	}
}

func canonicalRequestOutput(request canonical.ProjectionRequest) map[string]any {
	return map[string]any{
		"revision": request.Revision, "modelDigest": request.ModelDigest, "requestDigest": request.RequestDigest,
		"projection": request.Projection, "binding": request.Binding, "modulePin": request.ModulePin, "projector": request.Projector,
		"definitions": request.Definitions, "schemas": request.Schemas, "policies": request.Policies,
		"edges": request.Edges, "externalEdges": request.ExternalEdges,
		"targetRepository": request.TargetRepository, "targetPath": request.TargetPath, "targetPrefix": request.TargetPrefix,
		"targetFiles": request.TargetFiles, "targetDigests": request.TargetDigests,
	}
}

const (
	canonicalControllerConfigLimit = 1 << 20
	canonicalReviewedRunLimit      = 32 << 20
)

func isCanonicalControllerAction(action string) bool {
	switch action {
	case "controller-propose", "controller-execute", "controller-apply", "controller-verify", "controller-refresh-propose", "controller-refresh-apply":
		return true
	default:
		return false
	}
}

func runCanonicalController(o commandOptions, emit func(any) int, fail func(error) int) int {
	runtimeBytes, err := readBoundedControllerFile(o.runtime, canonicalControllerConfigLimit)
	if err != nil {
		return fail(fmt.Errorf("read controller runtime configuration: %w", err))
	}
	cfg, err := host.DecodeCanonicalControllerConfig(runtimeBytes)
	if err != nil {
		return fail(fmt.Errorf("decode controller runtime configuration: %w", err))
	}

	if o.action == "controller-refresh-propose" || o.action == "controller-refresh-apply" {
		return runCanonicalEvidenceRefresh(o, cfg, emit, fail)
	}

	switch o.action {
	case "controller-propose":
		proposal, err := host.ProposeCanonicalController(o.root, o.base, o.revision, o.reviewConfig, cfg)
		if err != nil {
			if code := emit(proposal); code != 0 {
				return code
			}
			return fail(err)
		}
		if code := emit(proposal); code != 0 {
			return code
		}
		return canonicalControllerStatusExit(proposal.Status)

	case "controller-execute":
		toolDigest, err := currentToolDigest()
		if err != nil {
			return fail(err)
		}
		run, err := host.ExecuteCanonicalController(context.Background(), o.root, o.base, o.revision, o.reviewConfig, cfg, version, toolDigest)
		if err != nil {
			if code := emit(run); code != 0 {
				return code
			}
			return fail(err)
		}
		if code := emit(run); code != 0 {
			return code
		}
		return canonicalControllerStatusExit(run.Status)

	case "controller-apply":
		planBytes, err := readBoundedControllerFile(o.plan, canonicalReviewedRunLimit)
		if err != nil {
			return fail(fmt.Errorf("read reviewed controller run: %w", err))
		}
		run, err := host.DecodeCanonicalReviewedRun(planBytes)
		if err != nil {
			return fail(fmt.Errorf("decode reviewed controller run JSON: %w", err))
		}
		if run.Proposal.Plan.BaseRevision != o.base || run.Proposal.Plan.Revision != o.revision {
			return fail(errors.New("--base and --revision must exactly match the saved reviewed run"))
		}
		if run.Digest != o.expect {
			return fail(errors.New("--expect must exactly match the saved reviewed run digest"))
		}
		applied, err := host.ApplyCanonicalController(o.root, o.reviewConfig, cfg, run, o.expect, o.write)
		if err != nil {
			if code := emit(applied); code != 0 {
				return code
			}
			return fail(err)
		}
		if code := emit(applied); code != 0 {
			return code
		}
		return canonicalControllerStatusExit(applied.Status)

	case "controller-verify":
		verification, err := host.VerifyCanonicalController(context.Background(), o.root, o.base, o.revision, o.reviewConfig, cfg, o.write)
		if err != nil {
			if code := emit(verification); code != 0 {
				return code
			}
			return fail(err)
		}
		if code := emit(verification); code != 0 {
			return code
		}
		return canonicalControllerStatusExit(verification.Status)
	default:
		return fail(fmt.Errorf("unsupported canonical controller action %q", o.action))
	}
}

func canonicalControllerStatusExit(status string) int {
	switch status {
	case "failed", "blocked", "escalated", "refused", "partial-failure":
		return 1
	case "incomplete":
		return 2
	case "planned", "passed", "materialized-unverified", "no-materialization-work":
		return 0
	default:
		return 2
	}
}

func readBoundedControllerFile(path string, max int64) ([]byte, error) {
	if path == "" {
		return nil, errors.New("path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("file exceeds %d-byte limit", max)
	}
	return data, nil
}

func runCanonicalReconcilePlan(o commandOptions, current *host.CanonicalSource, base map[string]any, emit func(any) int, fail func(error) int) int {
	previous, err := host.LoadCanonicalSource(o.root, o.base, o.reviewConfig, true)
	if err != nil {
		return fail(err)
	}
	if len(previous.Diagnostics) > 0 || len(current.Diagnostics) > 0 {
		base["status"] = "failed"
		base["baseDiagnostics"] = previous.Diagnostics
		base["diagnostics"] = current.Diagnostics
		if code := emit(base); code != 0 {
			return code
		}
		return 1
	}
	observed, err := source.Load(o.root, "")
	if err != nil {
		return fail(err)
	}
	active, err := readCanonicalProjectionRecords(o.reviewEvidence)
	if err != nil {
		return fail(fmt.Errorf("read active Projection Records: %w", err))
	}
	plan, err := host.PlanCanonicalReconciliation(previous, current, observed, active)
	if err != nil {
		return fail(err)
	}
	base["status"] = plan.Status
	base["baseRevision"] = previous.Snapshot.ID
	base["candidateRevision"] = current.Snapshot.ID
	base["plan"] = plan
	if code := emit(base); code != 0 {
		return code
	}
	if plan.Status == "escalated" {
		return 1
	}
	return 0
}

func readCanonicalProjectionRecords(file string) ([]records.ProjectionRecord, error) {
	if file == "" {
		return nil, nil
	}
	data, err := readCanonicalLocalInput(file, 8<<20)
	if err != nil {
		return nil, err
	}
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return nil, fmt.Errorf("decode a closed JSON array of active Projection Records: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var active []records.ProjectionRecord
	if err := decoder.Decode(&active); err != nil {
		return nil, fmt.Errorf("decode a closed JSON array of active Projection Records: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("active Projection Records input must contain exactly one JSON value")
	}
	if active == nil {
		return nil, fmt.Errorf("active Projection Records input must be a JSON array")
	}
	if len(active) > 4096 {
		return nil, fmt.Errorf("active Projection Records input exceeds 4096 records")
	}
	seenIDs := map[string]bool{}
	for _, record := range active {
		if err := records.ValidateProjectionRecord(record); err != nil {
			return nil, fmt.Errorf("validate active Projection Record: %w", err)
		}
		if seenIDs[record.ID] {
			return nil, fmt.Errorf("duplicate active Projection Record %q", record.ID)
		}
		seenIDs[record.ID] = true
	}
	return active, nil
}

func readCanonicalProjectionRecord(file string) (records.ProjectionRecord, error) {
	data, err := readCanonicalLocalInput(file, 8<<20)
	if err != nil {
		return records.ProjectionRecord{}, err
	}
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return records.ProjectionRecord{}, fmt.Errorf("decode a closed JSON Projection Record: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record records.ProjectionRecord
	if err := decoder.Decode(&record); err != nil {
		return records.ProjectionRecord{}, fmt.Errorf("decode a closed JSON Projection Record: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return records.ProjectionRecord{}, fmt.Errorf("Projection Record input must contain exactly one JSON object")
	}
	if err := records.ValidateProjectionRecord(record); err != nil {
		return records.ProjectionRecord{}, fmt.Errorf("validate Projection Record: %w", err)
	}
	return record, nil
}

func runCanonicalVerify(o commandOptions, fixed *host.CanonicalSource, base map[string]any, emit func(any) int, fail func(error) int) int {
	record, err := readCanonicalProjectionRecord(o.reviewEvidence)
	if err != nil {
		return fail(fmt.Errorf("read Projection Record: %w", err))
	}
	target, err := source.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	verification, verifyErr := host.VerifyCanonicalProjection(fixed, target, record, records.VerifierIdentity{ID: "markitect.canonical-fixed-checks", Version: version, Digest: toolDigest})
	base["status"] = "verified"
	base["evidenceRevision"] = verification.EvidenceRevision
	base["evidenceSnapshotDigest"] = verification.EvidenceSnapshotDigest
	base["result"] = verification.Result
	base["gates"] = verification.Gates
	if verifyErr != nil {
		var classified *host.VerifyError
		if errors.As(verifyErr, &classified) && (classified.Kind == "gate-failure" || classified.Kind == "incomplete-evidence") {
			status := verification.Result.Outcome
			if status == "" {
				status = "incomplete"
			}
			base["status"] = status
			base["verificationError"] = verifyErr.Error()
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		return fail(verifyErr)
	}
	base["status"] = verification.Result.Outcome
	return emit(base)
}

func runCanonicalAdoption(o commandOptions, fixed *host.CanonicalSource, base map[string]any, emit func(any) int, fail func(error) int) int {
	target, err := source.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	selection, err := readCanonicalAdoptionSelection(o.reviewReport)
	if err != nil {
		return fail(fmt.Errorf("read adoption selection: %w", err))
	}
	identity := core.DefinitionIdentity{APIVersion: o.apiVersion, Kind: o.kind, Namespace: o.namespace, Name: o.name}
	base["sourceRevision"] = fixed.Snapshot.ID
	base["evidenceRevision"] = target.ID
	if o.action == "adopt-plan" {
		plan, err := host.PrepareCanonicalAdoption(fixed, target, identity, selection)
		if err != nil {
			var classified *host.VerifyError
			if errors.As(err, &classified) && classified.Kind == "incomplete-evidence" {
				base["status"] = "incomplete"
				base["adoptionError"] = err.Error()
				if code := emit(base); code != 0 {
					return code
				}
				return 1
			}
			return fail(err)
		}
		base["status"] = "planned"
		base["plan"] = map[string]any{
			"apiVersion": plan.APIVersion, "planDigest": plan.PlanDigest, "evidenceRevision": plan.EvidenceRevision,
			"record": plan.Record, "unmatchedArtifacts": plan.UnmatchedArtifacts,
		}
		return emit(base)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	adoption, adoptErr := host.AdoptCanonicalProjection(fixed, target, identity, selection, o.expect, records.VerifierIdentity{ID: "markitect.canonical-adoption-verifier", Version: version, Digest: toolDigest})
	base["verification"] = adoption.Verification
	base["unmatchedArtifacts"] = adoption.UnmatchedArtifacts
	if adoptErr != nil {
		var classified *host.VerifyError
		if errors.As(adoptErr, &classified) && (classified.Kind == "gate-failure" || classified.Kind == "incomplete-evidence") {
			status := adoption.Verification.Result.Outcome
			if status == "" {
				status = "incomplete"
			}
			base["status"] = status
			base["adoptionError"] = adoptErr.Error()
			if code := emit(base); code != 0 {
				return code
			}
			return 1
		}
		return fail(adoptErr)
	}
	if adoption.Record == nil {
		return fail(fmt.Errorf("adoption verification passed without producing a Projection Record"))
	}
	base["status"] = "adopted"
	base["record"] = adoption.Record
	return emit(base)
}

func readCanonicalAdoptionSelection(file string) (host.CanonicalAdoptionSelection, error) {
	data, err := readCanonicalLocalInput(file, 1<<20)
	if err != nil {
		return host.CanonicalAdoptionSelection{}, err
	}
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("decode a closed JSON adoption selection: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var selection host.CanonicalAdoptionSelection
	if err := decoder.Decode(&selection); err != nil {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("decode a closed JSON adoption selection: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption selection must contain exactly one JSON object")
	}
	if selection.Artifacts == nil || len(selection.Artifacts) == 0 || len(selection.Artifacts) > 4096 {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption selection must list 1..4096 exact artifacts")
	}
	if selection.ActiveRecords == nil || len(selection.ActiveRecords) > 4096 {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption selection must include an activeRecords array of at most 4096 records")
	}
	recordIDs, projectionIDs := map[string]bool{}, map[string]bool{}
	for _, record := range selection.ActiveRecords {
		if err := records.ValidateProjectionRecord(record); err != nil {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("invalid active Projection Record: %w", err)
		}
		if recordIDs[record.ID] {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("active Projection Record %q is duplicated", record.ID)
		}
		if projectionIDs[record.ProjectionID] {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("active Projection %q has multiple supplied Records", record.ProjectionID)
		}
		recordIDs[record.ID], projectionIDs[record.ProjectionID] = true, true
	}
	if strings.TrimSpace(selection.ReviewReference) != selection.ReviewReference || selection.ReviewReference == "" || len(selection.ReviewReference) > 4096 {
		return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption selection requires a nonempty exact reviewReference of at most 4096 bytes")
	}
	seen := map[string]bool{}
	for _, artifact := range selection.Artifacts {
		if strings.TrimSpace(artifact) != artifact || artifact == "" {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption artifact paths must be nonempty exact repository paths")
		}
		if err := projectionengine.ValidateRelativePath(artifact); err != nil {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption artifact path %q is invalid: %w", artifact, err)
		}
		if seen[artifact] {
			return host.CanonicalAdoptionSelection{}, fmt.Errorf("adoption artifact path %q is duplicated", artifact)
		}
		seen[artifact] = true
	}
	return selection, nil
}

func rejectDuplicateCanonicalJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanCanonicalJSONValue(decoder, 0); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("input must contain exactly one JSON value")
	}
	return nil
}

func scanCanonicalJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("JSON nesting exceeds 64 levels")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = true
			if err := scanCanonicalJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return fmt.Errorf("invalid JSON object")
		}
	case '[':
		for decoder.More() {
			if err := scanCanonicalJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return fmt.Errorf("invalid JSON array")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func runCanonicalPlanApply(o commandOptions, base map[string]any, emit func(any) int, fail func(error) int) int {
	fixed, err := host.LoadCanonicalSource(o.root, o.revision, o.reviewConfig, true)
	if err != nil {
		return fail(err)
	}
	if len(fixed.Diagnostics) != 0 {
		base["status"] = "failed"
		base["diagnostics"] = fixed.Diagnostics
		if code := emit(base); code != 0 {
			return code
		}
		return 1
	}
	observed, err := source.Load(o.root, "")
	if err != nil {
		return fail(err)
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	identity := core.DefinitionIdentity{APIVersion: o.apiVersion, Kind: o.kind, Namespace: o.namespace, Name: o.name}
	var candidate []byte
	if o.reviewReport != "" {
		candidate, err = readCanonicalLocalInput(o.reviewReport, 8<<20)
		if err != nil {
			return fail(fmt.Errorf("read candidate report: %w", err))
		}
	}
	prepared, err := host.PrepareCanonicalProjection(fixed, observed, identity, version, toolDigest, candidate, fixed.Config.Checks...)
	if err != nil {
		return fail(err)
	}
	report := canonicalPlanReport{
		APIVersion: canonicalPlanReportAPIVersion, Revision: fixed.Snapshot.ID,
		Provisional: fixed.Snapshot.Provisional, SourceDigest: fixed.Snapshot.Digest(),
		Request: canonicalRequestOutput(prepared.Request), CandidateDigest: prepared.CandidateDigest,
		Outputs: prepared.Outputs,
	}
	if len(prepared.Escalations) > 0 || prepared.Plan == nil {
		report.Status = "escalated"
		for _, escalation := range prepared.Escalations {
			report.Escalations = append(report.Escalations, canonicalPlanEscalation{Code: escalation.Code, Identity: escalation.Identity, Message: escalation.Message})
		}
		if code := emit(report); code != 0 {
			return code
		}
		return 1
	}
	if o.action == "plan" {
		report.Status = "planned"
		report.Plan = prepared.Plan
		return emit(report)
	}

	planBytes, err := readCanonicalLocalInput(o.plan, 8<<20)
	if err != nil {
		return fail(fmt.Errorf("read saved plan: %w", err))
	}
	reviewedReport, err := decodeCanonicalPlanReport(planBytes)
	if err != nil {
		return fail(err)
	}
	if reviewedReport.APIVersion != canonicalPlanReportAPIVersion || reviewedReport.Status != "planned" || reviewedReport.Provisional || reviewedReport.Revision != fixed.Snapshot.ID || reviewedReport.SourceDigest != fixed.Snapshot.Digest() || reviewedReport.CandidateDigest != prepared.CandidateDigest || reviewedReport.Plan == nil || !reflect.DeepEqual(*reviewedReport.Plan, *prepared.Plan) {
		return fail(fmt.Errorf("saved canonical plan report does not match the freshly prepared source, candidate and Plan"))
	}
	if !sameCanonicalYAMLValue(reviewedReport.Request, canonicalRequestOutput(prepared.Request)) {
		return fail(fmt.Errorf("saved canonical plan report request was changed"))
	}
	if !sameCanonicalJSON(reviewedReport.Outputs, prepared.Outputs) || len(reviewedReport.Escalations) != 0 {
		return fail(fmt.Errorf("saved canonical plan report outputs or escalation state was changed"))
	}
	if o.expect != prepared.CandidateDigest {
		return fail(fmt.Errorf("--expect digest does not match freshly prepared candidate/output bytes"))
	}
	applied, applyErr := host.ApplyCanonicalProjection(o.root, fixed, observed, prepared, o.expect, o.write)
	base["status"] = applied.Status
	base["written"] = applied.Written
	base["record"] = applied.Record
	if applyErr != nil {
		base["error"] = applyErr.Error()
		if code := emit(base); code != 0 {
			return code
		}
		return 1
	}
	return emit(base)
}

func readCanonicalLocalInput(file string, maxBytes int64) ([]byte, error) {
	info, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxBytes {
		return nil, fmt.Errorf("file must be a regular file containing 1..%d bytes", maxBytes)
	}
	input, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(input, maxBytes+1))
	closeErr := input.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file grew beyond the %d byte limit while reading", maxBytes)
	}
	return data, nil
}

func decodeCanonicalPlanReport(data []byte) (canonicalPlanReport, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return canonicalPlanReport{}, fmt.Errorf("decode saved canonical plan report: %w", err)
	}
	if containsYAMLAlias(&document) {
		return canonicalPlanReport{}, fmt.Errorf("saved canonical plan report does not allow YAML aliases")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var report canonicalPlanReport
	if err := decoder.Decode(&report); err != nil {
		return canonicalPlanReport{}, fmt.Errorf("decode saved canonical plan report: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return canonicalPlanReport{}, fmt.Errorf("saved canonical plan report must contain exactly one YAML document")
	}
	return report, nil
}

func sameCanonicalJSON(left, right any) bool {
	a, errA := json.Marshal(left)
	b, errB := json.Marshal(right)
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

func sameCanonicalYAMLValue(left, right any) bool {
	leftBytes, leftErr := yaml.Marshal(left)
	rightBytes, rightErr := yaml.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	var leftValue, rightValue any
	if yaml.Unmarshal(leftBytes, &leftValue) != nil || yaml.Unmarshal(rightBytes, &rightValue) != nil {
		return false
	}
	return sameCanonicalJSON(leftValue, rightValue)
}

func containsYAMLAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if containsYAMLAlias(child) {
			return true
		}
	}
	return false
}

func selectedProjectionTargetFiles(definition core.Definition, files map[string][]byte) (map[string][]byte, error) {
	target, ok := definition.Spec["target"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("selected Projection target must be an object")
	}
	path, ok := target["path"].(string)
	if !ok || strings.TrimSpace(path) != path || path == "" {
		return nil, fmt.Errorf("selected Projection target.path must be a nonempty exact path prefix")
	}
	prefix := strings.TrimSuffix(path, "/")
	if prefix == "." {
		prefix = ""
	}
	selected := map[string][]byte{}
	for file, data := range files {
		if prefix == "" || file == prefix || strings.HasPrefix(file, prefix+"/") {
			selected[file] = data
		}
	}
	return selected, nil
}
