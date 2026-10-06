package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/consumers/artifactcoverage"
	"github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/consumers/projections"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const MaterializationVersion = "markitect.example.org/materialization/v1alpha1"

type Materialization struct {
	APIVersion   string            `yaml:"apiVersion"`
	PlanDigest   string            `yaml:"planDigest"`
	Files        map[string]string `yaml:"files"`
	recordDigest string
	parsedDigest string
}

// ProjectionReport distinguishes target evidence from complete scope accounting.
// Convergence is relative to configured checks and an immutable snapshot.
type ProjectionReport struct {
	VerificationBoundary string                   `yaml:"verificationBoundary,omitempty"`
	Status               string                   `yaml:"status"`
	Plan                 projections.Plan         `yaml:"plan"`
	Renderer             *ReconcileObservation    `yaml:"renderer,omitempty"`
	Accounting           *artifactcoverage.Report `yaml:"accounting,omitempty"`
	Checks               []GateResult             `yaml:"checks,omitempty"`
	EvidenceError        string                   `yaml:"evidenceError,omitempty"`
	Written              []string                 `yaml:"written,omitempty"`
}

func projectionInput(p *Project, configPath, coveragePath, toolVersion, toolDigest string) (projections.Input, error) {
	var input projections.Input
	if p == nil || p.Snapshot == nil {
		return input, errors.New("a captured Project is required")
	}
	if err := source.ValidateIncludedPaths([]string{configPath}); err != nil {
		return input, err
	}
	data, ok := p.Snapshot.Files[configPath]
	if !ok {
		return input, fmt.Errorf("projection configuration %q is absent from snapshot", configPath)
	}
	config, err := projections.ParseConfig(data)
	if err != nil {
		return input, err
	}
	model, err := compileAdapterModel(p) // Strict existing policy/structure boundary.
	if err != nil {
		return input, err
	}
	desired, rendererOwners, err := GenerateOutputsWithOwners(p)
	if err != nil {
		return input, err
	}
	if coveragePath == "" {
		return input, errors.New("explicit accounting configuration is required")
	}
	if err := source.ValidateIncludedPaths([]string{coveragePath}); err != nil {
		return input, err
	}
	coverageBytes, ok := p.Snapshot.Files[coveragePath]
	if !ok {
		return input, fmt.Errorf("accounting config %s absent from snapshot", coveragePath)
	}
	coverage, err := artifactcoverage.ParseConfig(coverageBytes)
	if err != nil {
		return input, err
	}
	if err := validateArtifactCoverageSourcePaths(coverage, coveragePath, p.Snapshot); err != nil {
		return input, err
	}
	protected := map[string]string{configPath: "projection configuration", coveragePath: "artifact accounting configuration", ".artifacts/markitect": "Host writer workspace"}
	for _, record := range coverage.Spec.Tooling {
		protected[record.Path] = "tool-owned:" + record.Owner
	}
	for _, record := range coverage.Spec.Vendor {
		protected[record.Path] = "vendor:" + record.Owner
	}
	for _, record := range coverage.Spec.Exclusions {
		protected[record.Path] = "explicit exclusion"
	}
	for _, owner := range canonicalArtifactOwners(p, model) {
		protected[owner.Path] = owner.Owner
	}
	for _, owner := range resolvedArtifactInputs(p) {
		protected[owner.Path] = owner.Owner
	}
	for _, pin := range p.Graph.Project.Spec.Packages {
		protected[pin.Archive] = "pinned package"
	}
	targets := map[string]bool{}
	selectedDesired := map[string][]byte{}
	protectedPaths := make([]string, 0, len(protected))
	for path := range protected {
		protectedPaths = append(protectedPaths, path)
	}
	sort.Strings(protectedPaths)
	checks := map[string]bool{}
	for _, check := range p.Graph.Project.Spec.Checks {
		checks[check.Name] = true
	}
	for _, c := range config.Contracts {
		for _, name := range c.VerificationChecks {
			if !checks[name] {
				return input, fmt.Errorf("projection %s names undeclared Project check %q", c.ID, name)
			}
		}
		if c.Materializer.Mode == "deterministic" && (c.Materializer.Name != "markitect-render" || c.Materializer.Version != "v1alpha1") {
			return input, fmt.Errorf("unregistered deterministic materializer %s@%s", c.Materializer.Name, c.Materializer.Version)
		}
		for _, target := range c.Targets {
			if owner, exists := protected[target.Path]; exists {
				return input, fmt.Errorf("projection target %s overlaps canonical/input owner %s", target.Path, owner)
			}
			if c.Materializer.Mode == "ai" && len(rendererOwners[target.Path]) > 0 {
				return input, fmt.Errorf("AI target %s is already renderer-owned", target.Path)
			}
			if c.Materializer.Mode == "deterministic" {
				if _, exists := desired[target.Path]; !exists {
					return input, fmt.Errorf("deterministic target %s has no registered renderer", target.Path)
				}
				selectedDesired[target.Path] = desired[target.Path]
				for _, owner := range rendererOwners[target.Path] {
					found := false
					for _, selected := range c.Sources {
						if owner == selected {
							found = true
						}
					}
					if !found {
						return input, fmt.Errorf("projection %s omits renderer source owner %s for %s", c.ID, owner, target.Path)
					}
				}
			}
			targets[target.Path] = true
		}
	}
	// Unknown non-target bytes conservatively bind the plan. This is not an
	// inference that all those bytes are canonical semantic facts.
	hashes := map[string]string{}
	for name, content := range p.Snapshot.Files {
		if !targets[name] {
			hashes[name] = hashBytes(content)
		}
	}
	input = projections.Input{Model: model, Config: config, ConfigDigest: hashBytes(data), IntentDigest: digestStringMap(hashes), ToolName: "Markitect", ToolVersion: toolVersion, ToolDigest: toolDigest, Files: p.Snapshot.Files, Desired: selectedDesired, ProtectedPaths: protectedPaths}
	return input, nil
}

func PlanRepresentations(p *Project, configPath, coveragePath, toolVersion, toolDigest string) (projections.Plan, error) {
	input, err := projectionInput(p, configPath, coveragePath, toolVersion, toolDigest)
	if err != nil {
		return projections.Plan{}, err
	}
	plan, err := projections.Build(input)
	if err != nil {
		return projections.Plan{}, err
	}
	accounting, err := representationAccounting(p, configPath, coveragePath)
	if err != nil {
		return projections.Plan{}, err
	}
	renderer, err := ObserveProjection(p, toolVersion, toolDigest)
	if err != nil {
		return projections.Plan{}, err
	}
	composeRepresentationObservations(&plan, &renderer, &accounting)
	if plan.Status == "converged" {
		plan.Status = "incomplete"
		plan.Diagnostics = append(plan.Diagnostics, projections.Diagnostic{Code: "projection.verification-not-run", Severity: "warning", Message: "Read-only plan has not run Project checks; contract targets may match, but immutable Verify is required for configured convergence"})
	}
	return bindRepresentationPlan(plan)
}

func composeRepresentationObservations(plan *projections.Plan, renderer *ReconcileObservation, accounting *artifactcoverage.Report) {
	for _, finding := range accounting.Findings {
		plan.Diagnostics = append(plan.Diagnostics, projections.Diagnostic{Code: "projection.accounting." + finding.Code, Severity: "error", Path: finding.Path, Message: finding.Message})
	}
	if accounting.Status != "passed" {
		plan.Status = "drift"
	}
	declared := map[string]bool{}
	for _, contract := range plan.Contracts {
		for _, target := range contract.Targets {
			declared[target.Path] = true
		}
	}
	for code, paths := range map[string][]string{"drift": renderer.Drift, "stale": renderer.Stale, "conflict": renderer.Conflicts} {
		for _, name := range paths {
			plan.Status = "drift"
			if !declared[name] {
				plan.Diagnostics = append(plan.Diagnostics, projections.Diagnostic{Code: "projection.renderer-" + code, Severity: "error", Path: name, Message: "Required Project renderer output has " + code + " outside selected contract targets"})
			}
		}
	}
}

func bindRepresentationPlan(plan projections.Plan) (projections.Plan, error) {
	sort.Slice(plan.Diagnostics, func(i, j int) bool {
		a, b := plan.Diagnostics[i], plan.Diagnostics[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.ContractID != b.ContractID {
			return a.ContractID < b.ContractID
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Check != b.Check {
			return a.Check < b.Check
		}
		return a.Message < b.Message
	})
	// Bind the composed Host status/causes as well as the pure contract plan.
	plan.PlanDigest = ""
	encoded, err := json.Marshal(plan)
	if err != nil {
		return projections.Plan{}, err
	}
	plan.PlanDigest = strings.TrimPrefix(hashBytes(encoded), "sha256:")
	return plan, nil
}

// ObserveRepresentations runs no project code and writes nothing. AI content
// remains incomplete until Verify supplies fixed-snapshot check evidence.
func ObserveRepresentations(p *Project, configPath, coveragePath, toolVersion, toolDigest string) (ProjectionReport, error) {
	plan, err := PlanRepresentations(p, configPath, coveragePath, toolVersion, toolDigest)
	if err != nil {
		return ProjectionReport{}, err
	}
	report := ProjectionReport{Plan: plan, Status: plan.Status}
	renderer, renderErr := ObserveProjection(p, toolVersion, toolDigest)
	if renderErr != nil {
		return report, renderErr
	}
	report.Renderer = &renderer
	if len(renderer.Drift) > 0 || len(renderer.Stale) > 0 || len(renderer.Conflicts) > 0 {
		report.Status = "drift"
	}
	accounting, err := representationAccounting(p, configPath, coveragePath)
	if err != nil {
		report.EvidenceError = err.Error()
		if report.Status != "drift" {
			report.Status = "incomplete"
		}
		return report, nil
	}
	report.Accounting = &accounting
	if accounting.Status != "passed" {
		report.Status = "drift"
	}
	return report, nil
}

func representationAccounting(p *Project, configPath, coveragePath string) (artifactcoverage.Report, error) {
	if coveragePath == "" {
		return artifactcoverage.Report{}, errors.New("explicit --coverage config is required to claim governed-scope convergence")
	}
	if err := source.ValidateIncludedPaths([]string{coveragePath}); err != nil {
		return artifactcoverage.Report{}, err
	}
	data, ok := p.Snapshot.Files[coveragePath]
	if !ok {
		return artifactcoverage.Report{}, fmt.Errorf("accounting config %s absent from snapshot", coveragePath)
	}
	config, err := artifactcoverage.ParseConfig(data)
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	config.SourcePath = coveragePath
	if err := validateArtifactCoverageSourcePaths(config, coveragePath, p.Snapshot); err != nil {
		return artifactcoverage.Report{}, err
	}
	input, err := projectionInput(p, configPath, coveragePath, "accounting", "accounting")
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	_, owners, err := GenerateOutputsWithOwners(p)
	if err != nil {
		return artifactcoverage.Report{}, err
	}
	inventory := artifactcoverage.Inventory{SnapshotDigest: p.Snapshot.Digest(), Files: p.Snapshot.Files, CanonicalOwners: canonicalArtifactOwners(p, input.Model), InputOwners: resolvedArtifactInputs(p), GeneratedOwners: rendererArtifactOwners(owners)}
	// The contract file owns mapping mechanics, not semantic source meaning.
	inventory.CanonicalOwners = append(inventory.CanonicalOwners, artifactcoverage.OwnerFact{Path: configPath, Owner: "projection-contract"})
	for _, c := range input.Config.Contracts {
		for _, target := range c.Targets {
			for _, tool := range config.Spec.Tooling {
				if strings.EqualFold(tool.Path, target.Path) {
					return artifactcoverage.Report{}, fmt.Errorf("projection target %s overlaps tool ownership", target.Path)
				}
			}
			for _, vendor := range config.Spec.Vendor {
				if strings.EqualFold(vendor.Path, target.Path) {
					return artifactcoverage.Report{}, fmt.Errorf("projection target %s overlaps vendor ownership", target.Path)
				}
			}
			for _, excluded := range config.Spec.Exclusions {
				if strings.EqualFold(excluded.Path, target.Path) {
					return artifactcoverage.Report{}, fmt.Errorf("projection target %s is explicitly excluded", target.Path)
				}
			}
			if c.Materializer.Mode == "ai" {
				inventory.ProjectedOwners = append(inventory.ProjectedOwners, artifactcoverage.OwnerFact{Path: target.Path, Owner: c.ID})
			}
		}
	}
	return artifactcoverage.Check(config, inventory)
}

func VerifyRepresentations(p *Project, configPath, coveragePath, toolVersion, toolDigest string) (ProjectionReport, error) {
	input, err := projectionInput(p, configPath, coveragePath, toolVersion, toolDigest)
	if err != nil {
		return ProjectionReport{}, err
	}
	if p.Snapshot.Provisional {
		return ProjectionReport{}, errors.New("projection verify requires an immutable --revision")
	}
	baseline, err := projections.Build(input)
	if err != nil {
		return ProjectionReport{}, err
	}
	contractDigests := map[string]string{}
	for _, c := range baseline.Contracts {
		contractDigests[c.ID] = c.ContractDigest
	}
	results, verifyErr := VerifyRepresentationChecks(p)
	statuses := map[string]string{}
	failedGate := false
	for _, result := range results {
		if result.ExitCode == 0 {
			statuses[result.Name] = "passed"
		} else if result.ExitCode == -1 {
			statuses[result.Name] = "incomplete"
		} else {
			statuses[result.Name] = "failed"
			failedGate = true
		}
	}
	for _, c := range input.Config.Contracts {
		hashes := map[string]string{}
		for _, target := range c.Targets {
			if data, ok := input.Files[target.Path]; ok {
				hashes[target.Path] = strings.TrimPrefix(hashBytes(data), "sha256:")
			}
		}
		for _, name := range c.VerificationChecks {
			status := statuses[name]
			if status == "" {
				status = "incomplete"
			}
			input.Evidence = append(input.Evidence, projections.CheckEvidence{ContractID: c.ID, ContractDigest: contractDigests[c.ID], Check: name, Status: status, SnapshotDigest: p.Snapshot.Digest(), IntentDigest: input.IntentDigest, TargetDigests: hashes})
		}
	}
	plan, err := projections.Build(input)
	if err != nil {
		return ProjectionReport{}, err
	}
	report := ProjectionReport{Plan: plan, Status: plan.Status, Checks: results, VerificationBoundary: "Configured fixed Project check results and snapshot integrity only. Checker independence and semantic sufficiency are owner-supplied requirements, not proved by this verifier."}
	renderer, renderErr := ObserveProjection(p, toolVersion, toolDigest)
	if renderErr != nil {
		return report, renderErr
	}
	report.Renderer = &renderer
	if len(renderer.Drift) > 0 || len(renderer.Stale) > 0 || len(renderer.Conflicts) > 0 {
		report.Status = "drift"
	}
	if verifyErr != nil {
		report.EvidenceError = verifyErr.Error()
		if failedGate {
			report.Status = "drift"
		} else if report.Status != "drift" {
			report.Status = "incomplete"
		}
	}
	accounting, err := representationAccounting(p, configPath, coveragePath)
	if err != nil {
		report.EvidenceError = err.Error()
		if report.Status != "drift" {
			report.Status = "incomplete"
		}
	} else {
		report.Accounting = &accounting
		if accounting.Status != "passed" {
			report.Status = "drift"
		}
	}
	if report.Accounting != nil {
		composeRepresentationObservations(&plan, &renderer, report.Accounting)
	}
	for _, result := range results {
		if result.ExitCode != 0 {
			code, message := "projection.check-failed", "Required Project check failed"
			if result.ExitCode == -1 {
				code, message = "projection.check-incomplete", "Required Project check evidence unavailable or invalidated"
			}
			plan.Diagnostics = append(plan.Diagnostics, projections.Diagnostic{Code: code, Severity: "error", Check: result.Name, Message: message})
		}
	}
	if report.EvidenceError != "" {
		plan.Diagnostics = append(plan.Diagnostics, projections.Diagnostic{Code: "projection.verification-evidence", Severity: "error", Message: "Required verification evidence failed, was unavailable or invalidated; see report checks and evidenceError"})
	}
	plan.Status = report.Status
	report.Plan, err = bindRepresentationPlan(plan)
	if err != nil {
		return report, err
	}
	return report, nil
}

func ReadMaterialization(filename string) (Materialization, error) {
	var candidate Materialization
	data, err := readProjectionRecord(filename)
	if err != nil {
		return candidate, err
	}
	if err := decodeProjectionRecord(data, &candidate); err != nil {
		return candidate, err
	}
	if candidate.APIVersion != MaterializationVersion || candidate.PlanDigest == "" {
		return candidate, errors.New("invalid materialization identity")
	}
	for name, content := range candidate.Files {
		if !utf8.ValidString(content) {
			return candidate, fmt.Errorf("candidate %s is not UTF-8", name)
		}
	}
	candidate.recordDigest = hashBytes(data)
	candidate.parsedDigest = materializationDigest(candidate)
	return candidate, nil
}

// MaterializationRecordDigest identifies the exact externally reviewed file.
// It is supplied evidence identity, not reviewer authentication.
func MaterializationRecordDigest(candidate Materialization) string { return candidate.recordDigest }
func materializationDigest(candidate Materialization) string {
	data, _ := YAML(candidate)
	return hashBytes(data)
}
func ReadRepresentationPlan(filename string) (projections.Plan, error) {
	var plan projections.Plan
	data, err := readProjectionRecord(filename)
	if err != nil {
		return plan, err
	}
	err = decodeProjectionRecord(data, &plan)
	return plan, err
}
func readProjectionRecord(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("projection record must be regular")
	}
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 2<<20 || !utf8.Valid(data) {
		return nil, errors.New("projection record must be UTF-8 and at most 2 MiB")
	}
	return data, nil
}
func decodeProjectionRecord(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("projection record must contain exactly one document")
	}
	return nil
}

func ApplyRepresentations(root string, p *Project, configPath, coveragePath, toolVersion, toolDigest string, saved projections.Plan, candidate *Materialization, expectedCandidateDigest string) (ProjectionReport, error) {
	fresh, err := PlanRepresentations(p, configPath, coveragePath, toolVersion, toolDigest)
	if err != nil {
		return ProjectionReport{}, err
	}
	if !yamlEqual(fresh, saved) {
		return ProjectionReport{}, errors.New("projection plan is stale or modified; replan exact desired and observed inputs")
	}
	input, err := projectionInput(p, configPath, coveragePath, toolVersion, toolDigest)
	if err != nil {
		return ProjectionReport{}, err
	}
	// Validate explicit accounting/config collisions before mutation; missing outputs
	// are expected during materialization, unmanaged/conflicting files are not.
	accounting, err := representationAccounting(p, configPath, coveragePath)
	if err != nil {
		return ProjectionReport{}, err
	}
	for _, finding := range accounting.Findings {
		if finding.Code != "missing-projected" && finding.Code != "missing-generated" && finding.Code != "stale-root" {
			return ProjectionReport{}, fmt.Errorf("artifact accounting blocks apply: %s: %s", finding.Code, finding.Message)
		}
	}
	contents := map[string][]byte{}
	allowed := map[string]bool{}
	for _, c := range input.Config.Contracts {
		for _, target := range c.Targets {
			if c.Materializer.Mode == "deterministic" {
				contents[target.Path] = input.Desired[target.Path]
			} else {
				allowed[target.Path] = true
			}
		}
	}
	if len(allowed) > 0 && candidate == nil {
		return ProjectionReport{}, errors.New("AI materialization requires a complete candidate and explicitly reviewed --expect digest")
	}
	if candidate != nil {
		if expectedCandidateDigest == "" || candidate.recordDigest != expectedCandidateDigest || candidate.parsedDigest != materializationDigest(*candidate) {
			return ProjectionReport{}, errors.New("candidate bytes differ from the explicitly reviewed digest; review the exact record again")
		}
		if len(candidate.Files) != len(allowed) {
			return ProjectionReport{}, errors.New("candidate must provide every exact AI target; partial materialization is unsupported")
		}
		if candidate.PlanDigest != saved.PlanDigest || candidate.APIVersion != MaterializationVersion {
			return ProjectionReport{}, errors.New("materialization is not bound to this plan")
		}
		for name, content := range candidate.Files {
			if !allowed[name] || !utf8.ValidString(content) {
				return ProjectionReport{}, fmt.Errorf("candidate writes an unowned/non-UTF-8 target %s", name)
			}
			if isCanonicalProjectionEnvelope([]byte(content), p.Graph.Registry) {
				return ProjectionReport{}, fmt.Errorf("candidate target %s contains a typed canonical resource; change desired state explicitly", name)
			}
			contents[name] = []byte(content)
		}
	}
	for name, content := range contents {
		if observed, ok := p.Snapshot.Files[name]; ok && bytes.Equal(content, observed) {
			delete(contents, name)
		}
	}
	written, err := writeRepresentationContents(root, p, contents)
	return ProjectionReport{Status: "materialized-unverified", Plan: saved, Written: written}, err
}

// Writes reuse Host path, branch, lock and atomic-file safety. Local multi-file
// apply is deliberately not a transaction; exact partial progress is returned.
func writeRepresentationContents(root string, p *Project, contents map[string][]byte) ([]string, error) {
	if p == nil || p.Snapshot == nil {
		return nil, errors.New("a source snapshot is required")
	}
	if len(contents) == 0 {
		return verifyNoopWrite(root, p)
	}
	return WriteProjectionContents(root, p.Snapshot, contents)
}

// WriteProjectionContents is the Host's guarded writer for a previously validated
// plan. Callers must bind exact candidate bytes, canonical intent and owned paths
// before calling; this low-level function does not grant projection authority.
// No rollback is implied: every successfully written path is returned on failure.
func WriteProjectionContents(root string, captured *snapshot.Snapshot, contents map[string][]byte) ([]string, error) {
	return WriteProjectionArtifacts(root, captured, contents, nil)
}

// WriteProjectionArtifacts applies exact bytes and Git regular-file modes under
// the same guarded write protocol. A requested executable bit must be confirmed
// by the post-write source observation; unsupported filesystems fail explicitly.
func WriteProjectionArtifacts(root string, captured *snapshot.Snapshot, contents map[string][]byte, modes map[string]string) ([]string, error) {
	if captured == nil || !captured.Provisional {
		return nil, errors.New("projection writes require the provisional isolated working tree")
	}
	if err := validateProjectionWriteModes(contents, modes); err != nil {
		return nil, err
	}
	branch := ""
	var err error
	if hasGitMetadata(root) {
		branch, err = writeBranchName(root)
		if err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)
	fresh, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if fresh.Digest() != captured.Digest() {
		return nil, errors.New("source changed since capture")
	}
	if len(contents) == 0 {
		return nil, nil
	}
	for _, name := range names {
		if _, err := safeDestination(root, name); err != nil {
			return nil, err
		}
	}
	lockPath, err := safeDestination(root, ".artifacts/markitect/write.lock")
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("another writer owns lock: %w", err)
	}
	lock.Close()
	defer os.Remove(lockPath)
	var written []string
	for _, name := range names {
		if branch != "" {
			if err := ensureWriteBranch(root, branch); err != nil {
				return written, err
			}
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return written, err
		}
		observed, exists := captured.Files[name]
		desiredMode := projectionOutputMode(modes, name)
		current, readErr := os.ReadFile(dest)
		if exists && (readErr != nil || !bytes.Equal(observed, current)) {
			return written, fmt.Errorf("target changed during apply: %s", name)
		}
		if !exists && !os.IsNotExist(readErr) {
			return written, fmt.Errorf("target appeared during apply: %s", name)
		}
		if exists && bytes.Equal(observed, contents[name]) && captured.Modes[name] == desiredMode {
			continue
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return written, err
		}
		if _, err = safeDestination(root, name); err != nil {
			return written, err
		}
		if err = atomicWrite(dest, contents[name]); err != nil {
			return written, err
		}
		written = append(written, name)
		if desiredMode == snapshot.ExecutableMode {
			if err = os.Chmod(dest, 0755); err != nil {
				return written, fmt.Errorf("set executable artifact mode for %s: %w", name, err)
			}
		}
	}
	final, err := source.Load(root, "")
	if err != nil {
		return written, err
	}
	for _, name := range names {
		if !bytes.Equal(final.Files[name], contents[name]) || final.Modes[name] != projectionOutputMode(modes, name) {
			return written, fmt.Errorf("post-write artifact observation does not match reviewed bytes/mode: %s", name)
		}
		delete(final.Files, name)
		delete(final.Modes, name)
		delete(fresh.Files, name)
		delete(fresh.Modes, name)
	}
	if final.Digest() != fresh.Digest() {
		return written, errors.New("non-target inputs changed during apply; materialization is provisional")
	}
	if branch != "" {
		if err := ensureWriteBranch(root, branch); err != nil {
			return written, err
		}
	}
	return written, nil
}

// validateProjectionWriteModes rejects a known unsupported materialization
// before either adopter bytes or an external ledger are created. Preparation
// remains platform-neutral and can describe executable artifacts on Windows.
func validateProjectionWriteModes(contents map[string][]byte, modes map[string]string) error {
	if err := validateProjectionOutputModes(contents, modes); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return nil
	}
	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if projectionOutputMode(modes, name) == snapshot.ExecutableMode {
			return fmt.Errorf("executable artifact mode is unsupported on Windows working trees: %s", name)
		}
	}
	return nil
}

func validateProjectionOutputModes(contents map[string][]byte, modes map[string]string) error {
	for name, mode := range modes {
		if _, exists := contents[name]; !exists {
			return fmt.Errorf("artifact mode is bound to absent output %q", name)
		}
		if mode != snapshot.RegularMode && mode != snapshot.ExecutableMode {
			return fmt.Errorf("artifact mode for %q must be 100644 or 100755", name)
		}
	}
	return nil
}

func projectionOutputMode(modes map[string]string, name string) string {
	if mode := modes[name]; mode != "" {
		return mode
	}
	return snapshot.RegularMode
}
