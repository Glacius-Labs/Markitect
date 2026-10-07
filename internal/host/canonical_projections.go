package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/dotnet"
)

var canonicalRevisionPattern = regexp.MustCompile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$")

type CanonicalCandidateFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"`
}

// CanonicalCandidate is an exact UTF-8 file set bound to the request that
// produced it. Apply requires approval of the digest of these raw JSON bytes.
type CanonicalCandidate struct {
	RequestDigest string                   `json:"requestDigest"`
	Files         []CanonicalCandidateFile `json:"files"`
}

type CanonicalProjectionEscalation struct {
	Code     string
	Identity string
	Message  string
}

type PreparedCanonicalProjection struct {
	Request         canonical.ProjectionRequest
	Plan            *projectionengine.Plan
	CandidateDigest string
	Outputs         map[string][]byte
	OutputModes     map[string]string
	Escalations     []CanonicalProjectionEscalation
	candidateRaw    []byte
	checks          []authoring.Check
}

type CanonicalProjectionApply struct {
	Status  string
	Written []string
	Record  *records.ProjectionRecord
}

// Prepare binds one canonical Projection to a fixed source and an observed
// provisional target snapshot. The Host selects its static Projector
// implementation from the activated Module and Projector identities.
func PrepareCanonicalProjection(fixed *CanonicalSource, observed *snapshot.Snapshot, selected core.DefinitionIdentity, toolVersion, toolDigest string, candidate []byte, checks ...authoring.Check) (PreparedCanonicalProjection, error) {
	prepared := PreparedCanonicalProjection{Outputs: map[string][]byte{}, OutputModes: map[string]string{}}
	if err := validateCanonicalProjectionSnapshots(fixed, observed); err != nil {
		return prepared, err
	}
	if strings.TrimSpace(toolVersion) == "" || !validSHA256(toolDigest) {
		return prepared, errors.New("a tool version and sha256 tool digest are required")
	}
	if len(fixed.Diagnostics) != 0 {
		return prepared, errors.New("canonical model has structural diagnostics")
	}
	if strings.TrimSpace(fixed.Model.Digest) == "" || fixed.Model.Revision != fixed.Snapshot.ID {
		return prepared, errors.New("canonical Model does not bind the fixed source snapshot")
	}
	if err := validateCanonicalSourceUnchanged(fixed, observed); err != nil {
		return prepared, err
	}

	var parsed *CanonicalCandidate
	if len(candidate) > 0 {
		value, err := decodeCanonicalCandidate(candidate)
		if err != nil {
			return prepared, err
		}
		parsed = &value
	}
	projection, _ := fixed.Model.Definition(selected)
	targetFiles, err := canonicalProjectionTargetFiles(projection, observed.Files)
	if err != nil {
		return prepared, err
	}
	request, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, selected, targetFiles)
	if err != nil {
		return prepared, err
	}
	prepared.Request = request
	isCandidate, err := selectedHostProjector(request)
	if err != nil {
		return prepared, err
	}
	prepared.candidateRaw = append([]byte(nil), candidate...)
	prepared.checks = cloneAuthoringChecks(checks)
	prepared.Escalations = nil
	if parsed != nil && parsed.RequestDigest != request.RequestDigest {
		return prepared, fmt.Errorf("candidate requestDigest does not match freshly bound Projection request")
	}
	checkNames, checkEscalations, err := selectedCanonicalChecks(request.Projector, isCandidate, checks)
	if err != nil {
		return prepared, err
	}
	prepared.Escalations = append(prepared.Escalations, checkEscalations...)

	targets := []string{}
	var desired map[string][]byte
	if !isCandidate {
		if parsed != nil {
			return prepared, errors.New("deterministic Projector does not accept supplied candidate bytes")
		}
		if len(prepared.Escalations) == 0 {
			rendered, modes, escalations, err := renderCanonicalDeterministicProjection(fixed, observed, request, checks)
			if err != nil {
				return prepared, err
			}
			prepared.Escalations = append(prepared.Escalations, escalations...)
			if len(prepared.Escalations) == 0 {
				desired = rendered
				prepared.Outputs = cloneByteMap(rendered)
				prepared.OutputModes = cloneStringMap(modes)
				for target := range rendered {
					if err := projectionengine.ValidateRelativePath(target); err != nil {
						return prepared, err
					}
					targets = append(targets, target)
				}
				sort.Strings(targets)
			}
		}
	} else {
		if parsed == nil {
			prepared.Escalations = append(prepared.Escalations, CanonicalProjectionEscalation{Code: "projection.candidate-required", Message: "the .NET candidate Projector requires a request-bound UTF-8 candidate envelope"})
			return prepared, nil
		}
		candidateFiles, candidateModes, err := candidateMap(*parsed, request.TargetPrefix)
		if err != nil {
			return prepared, err
		}
		for p := range candidateFiles {
			targets = append(targets, p)
		}
		sort.Strings(targets)
		if len(prepared.Escalations) == 0 {
			guidance := projectionGuidance(request.Policies)
			evaluated := dotnet.Evaluate(dotnet.Input{Definitions: request.Definitions, Schemas: request.Schemas, Guidance: guidance, AllowedPaths: targets, CandidateFiles: candidateFiles})
			for _, e := range evaluated.Escalations {
				prepared.Escalations = append(prepared.Escalations, CanonicalProjectionEscalation{Code: e.Code, Identity: e.Identity, Message: e.Message})
			}
			for _, d := range evaluated.Diagnostics {
				prepared.Escalations = append(prepared.Escalations, CanonicalProjectionEscalation{Code: d.Code, Message: d.Message})
			}
			if len(prepared.Escalations) == 0 {
				prepared.Outputs = cloneByteMap(evaluated.Files)
				prepared.OutputModes = candidateModes
			}
		}
	}
	prepared.CandidateDigest = candidateDigest(candidate, prepared.Outputs, prepared.OutputModes)
	if len(targets) == 0 {
		return prepared, nil
	}
	if isCandidate && len(checkNames) == 0 {
		return prepared, nil
	}

	mode := projectionengine.ModeDeterministic
	if isCandidate {
		mode = projectionengine.ModeAI
	}
	contract := projectionengine.Contract{ID: "selected-projection", Sources: make([]string, 0, len(request.Definitions)), Representation: request.Projector.Target,
		Materializer:       projectionengine.Materializer{Name: request.ModulePin.Name + "/" + request.Projector.ID, Version: request.Projector.Version, Mode: mode},
		VerificationChecks: checkNames, Targets: make([]projectionengine.TargetPath, 0, len(targets))}
	for _, definition := range request.Definitions {
		contract.Sources = append(contract.Sources, definition.Identity().Key())
	}
	sort.Strings(contract.Sources)
	for _, target := range targets {
		contract.Targets = append(contract.Targets, projectionengine.TargetPath{Path: target})
	}
	protected := canonicalSourcePaths(fixed)
	for _, definition := range request.Definitions {
		protected = append(protected, definition.Source.Path)
	}
	for _, policy := range request.Policies {
		protected = append(protected, policy.Source.Path)
	}
	protected = sortedUniquePaths(protected)
	facts := make([]projectionengine.SourceProvenance, 0, len(request.Definitions))
	for _, definition := range request.Definitions {
		facts = append(facts, projectionengine.SourceProvenance{Key: definition.Identity().Key(), Kind: definition.Kind, Source: projectionengine.SourceInfo{Path: definition.Source.Path, Line: definition.Source.Line, Digest: definition.Source.Digest}})
	}
	plan, err := projectionengine.Build(projectionengine.Input{
		ModelDigest: request.ModelDigest, SnapshotDigest: sha256Prefix(observed.Digest()),
		Config:       projectionengine.Config{APIVersion: projectionengine.ConfigAPIVersion, Version: projectionengine.ConfigVersion, Contracts: []projectionengine.Contract{contract}},
		ConfigDigest: projectionConfigDigest(request.RequestDigest, selectedAuthoringChecks(checkNames, checks)), IntentDigest: request.RequestDigest,
		ToolName: request.ModulePin.Name + "/" + request.Projector.ID, ToolVersion: toolVersion, ToolDigest: toolDigest,
		Sources: facts, Files: request.TargetFiles, FileModes: artifactModesForFiles(request.TargetFiles, observed.Modes), Desired: desired, DesiredModes: prepared.OutputModes, ProtectedPaths: protected, OwnershipPaths: protected,
		Governance: projectionengine.Governance{Status: projectionengine.GovernanceNotConfigured},
	})
	if err != nil {
		return prepared, err
	}
	prepared.Plan = &plan
	return prepared, nil
}

// Apply requires explicit write authorization and exact digest approval. It
// freshly prepares and compares every request, plan, candidate, and output
// binding before invoking the Host's guarded writer.
func ApplyCanonicalProjection(root string, fixed *CanonicalSource, observed *snapshot.Snapshot, reviewed PreparedCanonicalProjection, acceptedCandidateDigest string, write bool) (CanonicalProjectionApply, error) {
	report := CanonicalProjectionApply{}
	if !write {
		return report, errors.New("projection apply requires explicit write=true")
	}
	if acceptedCandidateDigest == "" || acceptedCandidateDigest != reviewed.CandidateDigest {
		return report, errors.New("accepted digest does not match reviewed candidate bytes")
	}
	if reviewed.Plan == nil {
		return report, errors.New("reviewed Projection has no exact-target plan")
	}
	fresh, err := PrepareCanonicalProjection(fixed, observed, reviewed.Request.Projection.Identity(), reviewed.Plan.ToolVersion, reviewed.Plan.ToolDigest, reviewed.candidateRaw, reviewed.checks...)
	if err != nil {
		return report, err
	}
	if fresh.Plan == nil || len(fresh.Escalations) != 0 || len(fresh.Outputs) == 0 {
		return report, errors.New("fresh Projection request is incomplete or escalated")
	}
	if !sameCanonicalProjectionRequest(fresh.Request, reviewed.Request) || fresh.Plan.PlanDigest != reviewed.Plan.PlanDigest || fresh.CandidateDigest != acceptedCandidateDigest || outputDigest(fresh.Outputs, fresh.OutputModes) != outputDigest(reviewed.Outputs, reviewed.OutputModes) {
		return report, errors.New("reviewed Projection bindings or output bytes are stale or changed")
	}
	written, writeErr := WriteProjectionArtifacts(root, observed, fresh.Outputs, fresh.OutputModes)
	report.Written = append([]string(nil), written...)
	alreadyMaterialized := len(written) == 0 && writeErr == nil
	if len(written) == 0 {
		if writeErr != nil {
			return report, writeErr
		}
		for name, content := range fresh.Outputs {
			old, exists := observed.Files[name]
			if !exists || !bytes.Equal(old, content) || observed.Modes[name] != artifactMode(fresh.OutputModes, name) {
				return report, errors.New("writer reported no writes but reviewed outputs are not already materialized")
			}
			written = append(written, name)
		}
		sort.Strings(written)
	}
	report.Written = append([]string(nil), written...)
	state := records.StateMaterializedUnverified
	if writeErr != nil {
		state = records.StatePartialFailure
	}
	actual, observeErr := observeCanonicalProjectionOutputs(root, written)
	if observeErr != nil {
		return report, fmt.Errorf("outputs remain; post-write artifact observation failed: %w", observeErr)
	}
	record, recordErr := buildCanonicalProjectionRecord(fresh, observed, actual, written, state)
	if recordErr != nil {
		return report, recordErr
	}
	report.Record = &record
	report.Status = state
	if alreadyMaterialized {
		report.Status = "already-materialized"
	}
	return report, writeErr
}

// observeCanonicalProjectionOutputs reads only the exact artifacts that Apply
// will bind into its record. Repository-wide freshness was already checked by
// the guarded writer; this readback confirms the materialized artifact bytes
// and modes without widening the record's evidence scope.
func observeCanonicalProjectionOutputs(root string, paths []string) (*snapshot.Snapshot, error) {
	selected, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return nil, err
	}
	if len(selected.MissingPaths) != 0 {
		return nil, fmt.Errorf("materialized projection artifacts are missing: %s", strings.Join(selected.MissingPaths, ", "))
	}
	return selected.Snapshot, nil
}

func selectedHostProjector(request canonical.ProjectionRequest) (bool, error) {
	entrypoint, err := canonicalWorkflowEntrypoint(request)
	return entrypoint == "dotnet", err
}

func selectedCanonicalChecks(projector canonical.ProjectorRegistration, candidate bool, supplied []authoring.Check) ([]string, []CanonicalProjectionEscalation, error) {
	byName := map[string]authoring.Check{}
	for _, check := range supplied {
		if err := authoring.ValidateCheck(check); err != nil {
			return nil, nil, fmt.Errorf("invalid supplied Project check %q: %w", check.Name, err)
		}
		if _, ok := byName[check.Name]; ok {
			return nil, nil, fmt.Errorf("duplicate supplied Project check %q", check.Name)
		}
		byName[check.Name] = check
	}
	names := append([]string(nil), projector.RequiredChecks...)
	if len(names) == 0 {
		for name := range byName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var escalations []CanonicalProjectionEscalation
	for _, name := range names {
		if _, ok := byName[name]; !ok {
			escalations = append(escalations, CanonicalProjectionEscalation{Code: "projection.check-missing", Message: fmt.Sprintf("Projector requires explicitly supplied Project check %q", name)})
		}
	}
	if candidate && len(names) == 0 {
		escalations = append(escalations, CanonicalProjectionEscalation{Code: "projection.checks-required", Message: "candidate preparation requires at least one explicitly supplied named Project check"})
	}
	return names, escalations, nil
}
func projectionGuidance(policies []core.Definition) map[string]string {
	result := map[string]string{}
	for _, policy := range policies {
		source, ok := policy.Spec["sourceKind"].(map[string]any)
		if !ok {
			continue
		}
		api, _ := source["apiVersion"].(string)
		kind, _ := source["kind"].(string)
		guidance, _ := policy.Spec["guidance"].(string)
		if api != "" && kind != "" && strings.TrimSpace(guidance) != "" {
			result[(core.KindIdentity{APIVersion: api, Kind: kind}).Key()] = guidance
		}
	}
	return result
}
func selectedAuthoringChecks(names []string, supplied []authoring.Check) []authoring.Check {
	byName := make(map[string]authoring.Check, len(supplied))
	for _, check := range supplied {
		byName[check.Name] = check
	}
	out := make([]authoring.Check, 0, len(names))
	for _, name := range names {
		check := byName[name]
		out = append(out, cloneAuthoringCheck(check))
	}
	return out
}
func projectionConfigDigest(requestDigest string, checks []authoring.Check) string {
	payload := struct {
		RequestDigest string
		Checks        []authoring.Check
	}{requestDigest, checks}
	encoded, _ := json.Marshal(payload)
	return sha256Prefix(sha256Hex(encoded))
}
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanCandidateJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("candidate envelope must contain exactly one JSON value")
	}
	return nil
}
func scanCandidateJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case 123:
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("candidate object member name is not a string")
			}
			if seen[key] {
				return fmt.Errorf("candidate object repeats member %q", key)
			}
			seen[key] = true
			if err := scanCandidateJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case 91:
		for decoder.More() {
			if err := scanCandidateJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter in candidate envelope")
	}
}

func decodeCanonicalCandidate(data []byte) (CanonicalCandidate, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return CanonicalCandidate{}, fmt.Errorf("candidate envelope: %w", err)
	}
	var candidate CanonicalCandidate
	if len(data) == 0 || !utf8.Valid(data) {
		return candidate, errors.New("candidate envelope must be nonempty UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&candidate); err != nil {
		return candidate, fmt.Errorf("decode candidate envelope: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return candidate, errors.New("candidate envelope must contain exactly one JSON value")
	}
	if candidate.RequestDigest == "" || len(candidate.Files) == 0 {
		return candidate, errors.New("candidate envelope requires requestDigest and at least one file")
	}
	seen := map[string]bool{}
	for _, file := range candidate.Files {
		if file.Path == "" || seen[file.Path] {
			return candidate, fmt.Errorf("candidate has empty or duplicate path %q", file.Path)
		}
		seen[file.Path] = true
		if !utf8.ValidString(file.Content) || strings.IndexByte(file.Content, 0) >= 0 {
			return candidate, fmt.Errorf("candidate %q must be UTF-8 text without NUL", file.Path)
		}
	}
	return candidate, nil
}
func canonicalProjectionTargetFiles(projection core.Definition, observed map[string][]byte) (map[string][]byte, error) {
	target, ok := projection.Spec["target"].(map[string]any)
	if !ok {
		return nil, errors.New("Projection target must be an object")
	}
	rawPrefix, ok := target["path"].(string)
	if !ok || strings.TrimSpace(rawPrefix) == "" {
		return nil, errors.New("Projection target.path must be a nonempty prefix")
	}
	prefix := path.Clean(strings.TrimSuffix(rawPrefix, "/"))
	if prefix == "." || strings.HasPrefix(prefix, "../") || path.IsAbs(prefix) {
		return nil, fmt.Errorf("Projection target path %q is not a safe relative prefix", rawPrefix)
	}
	files := map[string][]byte{}
	for name, content := range observed {
		if prefix == "" || name == prefix || strings.HasPrefix(name, prefix+"/") {
			files[name] = content
		}
	}
	return files, nil
}

func candidateMap(candidate CanonicalCandidate, prefix string) (map[string][]byte, map[string]string, error) {
	out := make(map[string][]byte, len(candidate.Files))
	modes := make(map[string]string, len(candidate.Files))
	for _, file := range candidate.Files {
		if err := projectionengine.ValidateRelativePath(file.Path); err != nil {
			return nil, nil, fmt.Errorf("candidate path: %w", err)
		}
		if prefix != "" && !strings.HasPrefix(file.Path, strings.TrimSuffix(prefix, "/")+"/") {
			return nil, nil, fmt.Errorf("candidate path %q is outside exact target prefix %q", file.Path, prefix)
		}
		mode := file.Mode
		if mode == "" {
			mode = snapshot.RegularMode
		}
		if mode != snapshot.RegularMode && mode != snapshot.ExecutableMode {
			return nil, nil, fmt.Errorf("candidate mode for %q must be 100644 or 100755", file.Path)
		}
		out[file.Path] = []byte(file.Content)
		modes[file.Path] = mode
	}
	return out, modes, nil
}
func validateCanonicalProjectionSnapshots(fixed *CanonicalSource, observed *snapshot.Snapshot) error {
	if fixed == nil || fixed.Snapshot == nil {
		return errors.New("fixed canonical source is required")
	}
	if fixed.Snapshot.Provisional || !canonicalRevisionPattern.MatchString(fixed.Snapshot.ID) {
		return errors.New("canonical Projection requires a fixed full Git revision")
	}
	if observed == nil || !observed.Provisional {
		return errors.New("Prepare requires a provisional observed target snapshot")
	}
	if fixed.Model.Revision != fixed.Snapshot.ID {
		return errors.New("compiled Model revision does not match its fixed source snapshot")
	}
	return nil
}
func validateCanonicalSourceUnchanged(fixed *CanonicalSource, observed *snapshot.Snapshot) error {
	for _, name := range canonicalSourcePaths(fixed) {
		old, ok := fixed.Snapshot.Files[name]
		if !ok {
			return fmt.Errorf("fixed canonical source %q is absent from its snapshot", name)
		}
		current, ok := observed.Files[name]
		if !ok || !bytes.Equal(old, current) || fixed.Snapshot.Modes[name] != observed.Modes[name] {
			return fmt.Errorf("canonical source %q changed since its fixed revision", name)
		}
	}
	return nil
}
func canonicalSourcePaths(source *CanonicalSource) []string {
	paths := []string{source.ConfigPath}
	paths = append(paths, source.Config.Definitions...)
	for _, m := range source.Config.Modules {
		paths = append(paths, m.Manifest)
		paths = append(paths, m.Files...)
	}
	return sortedUniquePaths(paths)
}
func sortedUniquePaths(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	for _, v := range values {
		if v != "" && (len(out) == 0 || out[len(out)-1] != v) {
			out = append(out, v)
		}
	}
	return append([]string(nil), out...)
}
func buildCanonicalProjectionRecord(p PreparedCanonicalProjection, observed, actual *snapshot.Snapshot, written []string, state string) (records.ProjectionRecord, error) {
	request, plan := p.Request, *p.Plan
	artifacts := make([]records.Artifact, 0, len(written))
	facts := make([]records.ArtifactFact, 0, len(written))
	for _, target := range written {
		content, ok := p.Outputs[target]
		if !ok {
			return records.ProjectionRecord{}, fmt.Errorf("writer reported unreviewed path %q", target)
		}
		digest := sha256Prefix(sha256Hex(content))
		materialized, exists := actual.Files[target]
		if !exists || !bytes.Equal(materialized, content) {
			return records.ProjectionRecord{}, fmt.Errorf("post-write observation does not confirm reviewed bytes for %q", target)
		}
		mode := actual.Modes[target]
		if mode != snapshot.RegularMode && mode != snapshot.ExecutableMode {
			return records.ProjectionRecord{}, fmt.Errorf("post-write observation has unsupported mode %q for %q", mode, target)
		}
		if state != records.StatePartialFailure && mode != artifactMode(p.OutputModes, target) {
			return records.ProjectionRecord{}, fmt.Errorf("post-write observation mode %q does not match reviewed mode %q for %q", mode, artifactMode(p.OutputModes, target), target)
		}
		change := records.ChangeCreated
		if old, exists := observed.Files[target]; exists {
			if bytes.Equal(old, content) && observed.Modes[target] == mode {
				change = records.ChangeRetained
			} else {
				change = records.ChangeModified
			}
		}
		artifacts = append(artifacts, records.Artifact{Path: target, Digest: digest, Mode: mode, Change: change})
		facts = append(facts, records.ArtifactFact{Path: target, Role: records.RoleProjectionTarget, Digest: digest, Mode: mode})
	}
	targetDigest, err := records.TargetSnapshotDigest(facts)
	if err != nil {
		return records.ProjectionRecord{}, err
	}
	scope := make([]string, 0, len(request.Definitions))
	for _, d := range request.Definitions {
		scope = append(scope, d.Identity().Key())
	}
	sort.Strings(scope)
	policies := make([]string, 0, len(request.Policies))
	for _, d := range request.Policies {
		policies = append(policies, d.Identity().Key())
	}
	sort.Strings(policies)
	return records.NewProjectionRecord(records.ProjectionRecord{
		Revision: request.Revision, ModelDigest: request.ModelDigest, PlanDigest: sha256Prefix(plan.PlanDigest), InputSnapshotDigest: sha256Prefix(observed.Digest()), RequestDigest: request.RequestDigest,
		Module: records.ModuleIdentity{Name: request.ModulePin.Name, Version: request.ModulePin.Version, Digest: request.ModulePin.Digest}, ProjectionID: request.Projection.Identity().Key(),
		Projector: records.ProjectorIdentity{ID: request.Projector.ID, Version: request.Projector.Version}, ScopeIDs: scope, PolicyIDs: policies, Artifacts: artifacts, TargetSnapshotDigest: targetDigest, State: state,
	})
}
func cloneAuthoringChecks(values []authoring.Check) []authoring.Check {
	out := make([]authoring.Check, len(values))
	for i, v := range values {
		out[i] = cloneAuthoringCheck(v)
	}
	return out
}
func cloneAuthoringCheck(value authoring.Check) authoring.Check {
	value.Run = append([]string(nil), value.Run...)
	if value.TimeoutSeconds != nil {
		seconds := *value.TimeoutSeconds
		value.TimeoutSeconds = &seconds
	}
	return value
}
func candidateDigest(raw []byte, outputs map[string][]byte, modes map[string]string) string {
	if len(raw) > 0 {
		return sha256Prefix(sha256Hex(raw))
	}
	return outputDigest(outputs, modes)
}
func outputDigest(outputs map[string][]byte, modes map[string]string) string {
	paths := make([]string, 0, len(outputs))
	for p := range outputs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	type item struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
		Mode   string `json:"mode"`
	}
	items := make([]item, 0, len(paths))
	for _, p := range paths {
		items = append(items, item{Path: p, Digest: sha256Hex(outputs[p]), Mode: artifactMode(modes, p)})
	}
	encoded, _ := json.Marshal(items)
	return sha256Prefix(sha256Hex(encoded))
}

func regularModes(outputs map[string][]byte) map[string]string {
	modes := make(map[string]string, len(outputs))
	for name := range outputs {
		modes[name] = snapshot.RegularMode
	}
	return modes
}

func artifactModesForFiles(files map[string][]byte, modes map[string]string) map[string]string {
	selected := make(map[string]string, len(files))
	for name := range files {
		if mode := modes[name]; mode != "" {
			selected[name] = mode
		}
	}
	return selected
}

func artifactMode(modes map[string]string, name string) string {
	if mode := modes[name]; mode != "" {
		return mode
	}
	return snapshot.RegularMode
}
func sameCanonicalProjectionRequest(a, b canonical.ProjectionRequest) bool {
	left, leftErr := json.Marshal(a)
	right, rightErr := json.Marshal(b)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func sha256Prefix(value string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}
func validSHA256(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value[7:]) != value[7:] {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
