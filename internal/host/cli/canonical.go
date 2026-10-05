package cli

import (
	"bytes"
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
	requirePins := o.action != "modules"
	loadRevision := o.revision
	if o.action == "verify" {
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
	default:
		return fail(fmt.Errorf("canonical requires --action model, modules, context, request, impact, reconcile-plan, plan, apply or verify"))
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
