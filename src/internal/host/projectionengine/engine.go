// Package projectionengine evaluates explicit, local representation contracts. It
// is deliberately pure: the Host acquires files, invokes renderers/checks and
// performs any writes after reviewing a plan.
package projectionengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	ConfigAPIVersion = "markitect.example.org/projections/v1alpha1"
	PlanAPIVersion   = "markitect.example.org/projection-plan/v1alpha1"
	ConfigVersion    = "1"

	ModeDeterministic = "deterministic"
	ModeAI            = "ai"

	StatusConverged  = "converged"
	StatusDrift      = "drift"
	StatusIncomplete = "incomplete"
	StatusBlocked    = "blocked"

	EvidencePassed     = "passed"
	EvidenceFailed     = "failed"
	EvidenceIncomplete = "incomplete"

	TargetMatched    = "matched"
	TargetVerified   = "verified"
	TargetMissing    = "missing"
	TargetDrifted    = "drifted"
	TargetIncomplete = "incomplete"

	maxConfigBytes = 1 << 20
	maxContracts   = 128
	maxTargets     = 512
	maxChecks      = 128
	maxInputFiles  = 10000
	maxPathBytes   = 1024
)

var stableID = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")

// Config is the closed, versioned local projection contract owned by a
// Project. ConfigDigest in Input binds the exact source bytes parsed by Host.
type Config struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Version    string     `yaml:"version" json:"version"`
	Contracts  []Contract `yaml:"contracts" json:"contracts"`
}

type Contract struct {
	ID                 string       `yaml:"id" json:"id"`
	Sources            []string     `yaml:"sources" json:"sources"`
	Representation     string       `yaml:"representation" json:"representation"`
	Materializer       Materializer `yaml:"materializer" json:"materializer"`
	Targets            []TargetPath `yaml:"targets" json:"targets"`
	VerificationChecks []string     `yaml:"verificationChecks,omitempty" json:"verificationChecks,omitempty"`
	Freedom            []string     `yaml:"freedom,omitempty" json:"freedom,omitempty"`
	DependsOn          []string     `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

type Materializer struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	Mode    string `yaml:"mode" json:"mode"`
}

// TargetPath intentionally models one explicit file path, not a glob or
// selector. The Host supplies renderer bytes for deterministic targets.
type TargetPath struct {
	Path string `yaml:"path" json:"path"`
}

// CheckEvidence is an untrusted Host-supplied report. Build checks that it is
// bound to this fixed model snapshot, intent, contract and exact target bytes;
// this package cannot authenticate who ran the check or how.
type CheckEvidence struct {
	ContractID     string            `yaml:"contractId" json:"contractId"`
	Check          string            `yaml:"check" json:"check"`
	Status         string            `yaml:"status" json:"status"`
	SnapshotDigest string            `yaml:"snapshotDigest" json:"snapshotDigest"`
	IntentDigest   string            `yaml:"intentDigest" json:"intentDigest"`
	ContractDigest string            `yaml:"contractDigest" json:"contractDigest"`
	TargetDigests  map[string]string `yaml:"targetDigests" json:"targetDigests"`
}

// Input contains only explicit Host-owned facts and bytes. Files is an
// opaque observed snapshot keyed by exact project-relative paths. Desired
// contains registered deterministic renderer outputs; it is never accepted
// for AI materializers.
type SourceInfo struct {
	Path   string `yaml:"path" json:"path"`
	Line   int    `yaml:"line,omitempty" json:"line,omitempty"`
	Digest string `yaml:"digest" json:"digest"`
}
type SourceProvenance struct {
	Key            string     `yaml:"key" json:"key"`
	Kind           string     `yaml:"kind" json:"kind"`
	Package        string     `yaml:"package,omitempty" json:"package,omitempty"`
	PackageVersion string     `yaml:"packageVersion,omitempty" json:"packageVersion,omitempty"`
	Source         SourceInfo `yaml:"source" json:"source"`
}

const (
	GovernancePassed        = "passed"
	GovernanceFailed        = "failed"
	GovernanceWaived        = "waived"
	GovernanceNotConfigured = "not-configured"
)

type Governance struct {
	Status     string
	HasFailure bool
}
type Input struct {
	ModelDigest    string
	SnapshotDigest string
	Config         Config
	ConfigDigest   string
	IntentDigest   string
	ToolName       string
	ToolVersion    string
	ToolDigest     string
	Sources        []SourceProvenance
	Files          map[string][]byte
	FileModes      map[string]string
	Desired        map[string][]byte
	DesiredModes   map[string]string
	ProtectedPaths []string
	Evidence       []CheckEvidence
	Governance     Governance
	OwnershipPaths []string
}

type TargetPlan struct {
	Path           string `yaml:"path" json:"path"`
	ObservedDigest string `yaml:"observedDigest,omitempty" json:"observedDigest,omitempty"`
	ObservedMode   string `yaml:"observedMode,omitempty" json:"observedMode,omitempty"`
	DesiredDigest  string `yaml:"desiredDigest,omitempty" json:"desiredDigest,omitempty"`
	DesiredMode    string `yaml:"desiredMode,omitempty" json:"desiredMode,omitempty"`
	Status         string `yaml:"status" json:"status"`
}

type CheckResult struct {
	Check          string `yaml:"check" json:"check"`
	Status         string `yaml:"status" json:"status"`
	EvidenceDigest string `yaml:"evidenceDigest,omitempty" json:"evidenceDigest,omitempty"`
	Reason         string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type ContractPlan struct {
	ID                 string             `yaml:"id" json:"id"`
	ContractDigest     string             `yaml:"contractDigest" json:"contractDigest"`
	Sources            []SourceProvenance `yaml:"sources" json:"sources"`
	Representation     string             `yaml:"representation" json:"representation"`
	Materializer       Materializer       `yaml:"materializer" json:"materializer"`
	Targets            []TargetPlan       `yaml:"targets" json:"targets"`
	VerificationChecks []CheckResult      `yaml:"verificationChecks" json:"verificationChecks"`
	Freedom            []string           `yaml:"freedom,omitempty" json:"freedom,omitempty"`
	DependsOn          []string           `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	Status             string             `yaml:"status" json:"status"`
}

type Diagnostic struct {
	Code       string `yaml:"code" json:"code"`
	Severity   string `yaml:"severity" json:"severity"`
	ContractID string `yaml:"contractId,omitempty" json:"contractId,omitempty"`
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`
	Check      string `yaml:"check,omitempty" json:"check,omitempty"`
	Message    string `yaml:"message" json:"message"`
}

// Plan summarizes exact desired/observed identities and projection state. It
// deliberately does not embed the full Core model or target file contents.
type Plan struct {
	APIVersion       string         `yaml:"apiVersion" json:"apiVersion"`
	Status           string         `yaml:"status" json:"status"`
	PlanDigest       string         `yaml:"planDigest" json:"planDigest"`
	ModelDigest      string         `yaml:"modelDigest" json:"modelDigest"`
	SnapshotDigest   string         `yaml:"snapshotDigest" json:"snapshotDigest"`
	ConfigDigest     string         `yaml:"configDigest" json:"configDigest"`
	IntentDigest     string         `yaml:"intentDigest" json:"intentDigest"`
	ToolName         string         `yaml:"toolName" json:"toolName"`
	ToolVersion      string         `yaml:"toolVersion" json:"toolVersion"`
	ToolDigest       string         `yaml:"toolDigest" json:"toolDigest"`
	FilesDigest      string         `yaml:"filesDigest" json:"filesDigest"`
	ProtectedDigest  string         `yaml:"protectedDigest" json:"protectedDigest"`
	Contracts        []ContractPlan `yaml:"contracts" json:"contracts"`
	GovernanceStatus string         `yaml:"governanceStatus" json:"governanceStatus"`
	GovernanceFailed bool           `yaml:"governanceFailed" json:"governanceFailed"`
	Diagnostics      []Diagnostic   `yaml:"diagnostics" json:"diagnostics"`
}

// ParseConfig decodes exactly one closed YAML document and validates its
// finite local contract graph.
func ParseConfig(data []byte) (Config, error) {
	if len(data) == 0 || len(data) > maxConfigBytes {
		return Config{}, fmt.Errorf("projection config must contain 1..%d bytes", maxConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode projection config: %w", err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("projection config must contain exactly one YAML document")
		}
		return Config{}, fmt.Errorf("decode trailing projection config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return normalizeConfig(config), nil
}

// Build validates a projection contract and evaluates its current snapshot.
// Structural/model/configuration errors return an error. Ordinary failed
// policy results are preserved in a deterministic blocked Plan so callers can
// explain why materialization is unavailable.
func Build(input Input) (Plan, error) {
	config := normalizeConfig(input.Config)
	if input.Governance.Status != GovernancePassed && input.Governance.Status != GovernanceFailed && input.Governance.Status != GovernanceWaived && input.Governance.Status != GovernanceNotConfigured {
		return Plan{}, fmt.Errorf("projection input has unsupported governance status %q", input.Governance.Status)
	}
	if err := validateConfig(config); err != nil {
		return Plan{}, err
	}
	if err := validateInputBindings(input); err != nil {
		return Plan{}, err
	}
	sources, err := SourceIndex(input.Sources)
	if err != nil {
		return Plan{}, err
	}
	for _, c := range config.Contracts {
		for _, key := range c.Sources {
			if _, ok := sources[key]; !ok {
				return Plan{}, fmt.Errorf("contract %q references unknown source identity %q", c.ID, key)
			}
		}
	}
	if err := validateTargetOwnership(config, ownershipPaths(input)); err != nil {
		return Plan{}, err
	}
	if err := validateTargetsAgainstFiles(config, input.Files); err != nil {
		return Plan{}, err
	}
	if err := validateDesiredTargets(config, input.Desired); err != nil {
		return Plan{}, err
	}
	if err := validateModeTargets(config, input.DesiredModes, "desired artifact mode"); err != nil {
		return Plan{}, err
	}
	evidence, err := indexEvidence(config, input.Evidence)
	if err != nil {
		return Plan{}, err
	}
	filesDigest, err := digestFiles(input.Files, input.FileModes)
	if err != nil {
		return Plan{}, err
	}
	protectedDigest, err := digestStringSet(input.ProtectedPaths)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{APIVersion: PlanAPIVersion, Status: StatusConverged, ModelDigest: input.ModelDigest, SnapshotDigest: input.SnapshotDigest, ConfigDigest: input.ConfigDigest, IntentDigest: input.IntentDigest, ToolName: input.ToolName, ToolVersion: input.ToolVersion, ToolDigest: input.ToolDigest, FilesDigest: filesDigest, ProtectedDigest: protectedDigest, GovernanceStatus: input.Governance.Status, GovernanceFailed: input.Governance.HasFailure, Contracts: make([]ContractPlan, 0, len(config.Contracts)), Diagnostics: []Diagnostic{}}
	if input.Governance.Status == "failed" || input.Governance.HasFailure {
		plan.Status = StatusBlocked
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Code: "governance.failed", Severity: "error", Message: "governance failure blocks projection materialization"})
	}
	for _, c := range config.Contracts {
		cp, ds := evaluateContract(input, c, sources, evidence)
		plan.Contracts = append(plan.Contracts, cp)
		plan.Diagnostics = append(plan.Diagnostics, ds...)
	}
	plan.Diagnostics = append(plan.Diagnostics, propagateDependencyStatus(plan.Contracts)...)
	if plan.Status != StatusBlocked {
		plan.Status = aggregateStatus(plan.Contracts)
	}
	sortDiagnostics(plan.Diagnostics)
	plan.PlanDigest = ""
	encoded, err := json.Marshal(plan)
	if err != nil {
		return Plan{}, fmt.Errorf("encode projection plan digest: %w", err)
	}
	plan.PlanDigest = sha256Hex(encoded)
	return plan, nil
}
func validateInputBindings(input Input) error {
	for _, v := range []struct{ name, value string }{{"model digest", input.ModelDigest}, {"snapshot digest", input.SnapshotDigest}, {"config digest", input.ConfigDigest}, {"intent digest", input.IntentDigest}, {"tool name", input.ToolName}, {"tool version", input.ToolVersion}, {"tool digest", input.ToolDigest}} {
		if strings.TrimSpace(v.value) == "" {
			return fmt.Errorf("projection input requires %s", v.name)
		}
	}
	if len(input.Files) > maxInputFiles {
		return fmt.Errorf("projection input exceeds %d explicit files", maxInputFiles)
	}
	if err := validatePathMap(input.Files, "observed file"); err != nil {
		return err
	}
	return validateModeMap(input.Files, input.FileModes, "observed artifact mode")
}

func validateConfig(config Config) error {
	if config.APIVersion != ConfigAPIVersion {
		return fmt.Errorf("projection config apiVersion must be %q", ConfigAPIVersion)
	}
	if config.Version != ConfigVersion {
		return fmt.Errorf("projection config version must be %q", ConfigVersion)
	}
	if len(config.Contracts) == 0 || len(config.Contracts) > maxContracts {
		return fmt.Errorf("projection config must contain 1..%d contracts", maxContracts)
	}
	contractIDs := map[string]bool{}
	byID := make(map[string]Contract, len(config.Contracts))
	targets := make([]string, 0)
	targetCount := 0
	for _, contract := range config.Contracts {
		if !stableID.MatchString(contract.ID) {
			return fmt.Errorf("invalid projection contract id %q", contract.ID)
		}
		foldedID := strings.ToLower(contract.ID)
		if contractIDs[foldedID] {
			return fmt.Errorf("duplicate or case-aliased projection contract id %q", contract.ID)
		}
		contractIDs[foldedID] = true
		byID[contract.ID] = contract
		if strings.TrimSpace(contract.Representation) == "" {
			return fmt.Errorf("contract %q requires a representation", contract.ID)
		}
		if strings.TrimSpace(contract.Materializer.Name) == "" || strings.TrimSpace(contract.Materializer.Version) == "" {
			return fmt.Errorf("contract %q requires materializer name and version", contract.ID)
		}
		switch contract.Materializer.Mode {
		case ModeDeterministic:
		case ModeAI:
			if len(contract.VerificationChecks) == 0 {
				return fmt.Errorf("AI contract %q requires at least one named verification check", contract.ID)
			}
		default:
			return fmt.Errorf("contract %q has unsupported materializer mode %q", contract.ID, contract.Materializer.Mode)
		}
		if len(contract.Sources) == 0 {
			return fmt.Errorf("contract %q requires at least one exact source identity", contract.ID)
		}
		if err := uniqueStrings(contract.Sources, "source identity", contract.ID); err != nil {
			return err
		}
		for _, key := range contract.Sources {
			if strings.TrimSpace(key) != key || key == "" {
				return fmt.Errorf("contract %q has invalid exact source identity %q", contract.ID, key)
			}
		}
		if len(contract.Targets) == 0 {
			return fmt.Errorf("contract %q requires at least one target path", contract.ID)
		}
		targetCount += len(contract.Targets)
		if targetCount > maxTargets {
			return fmt.Errorf("projection config exceeds %d targets", maxTargets)
		}
		for _, target := range contract.Targets {
			if err := validateRelativePath(target.Path); err != nil {
				return fmt.Errorf("contract %q target: %w", contract.ID, err)
			}
			targets = append(targets, target.Path)
		}
		for _, item := range []struct {
			name   string
			values []string
		}{
			{"verification check", contract.VerificationChecks}, {"freedom statement", contract.Freedom}, {"dependency", contract.DependsOn},
		} {
			if len(item.values) > maxChecks {
				return fmt.Errorf("contract %q exceeds %d %ss", contract.ID, maxChecks, item.name)
			}
			if err := uniqueStrings(item.values, item.name, contract.ID); err != nil {
				return err
			}
		}
		for _, check := range contract.VerificationChecks {
			if strings.TrimSpace(check) != check || check == "" {
				return fmt.Errorf("contract %q has empty or padded verification check", contract.ID)
			}
		}
		for _, freedom := range contract.Freedom {
			if strings.TrimSpace(freedom) == "" {
				return fmt.Errorf("contract %q has empty freedom statement", contract.ID)
			}
		}
	}
	if err := validateNoPathOverlaps(targets, "projection targets"); err != nil {
		return err
	}
	for _, contract := range config.Contracts {
		for _, dependency := range contract.DependsOn {
			if _, exists := byID[dependency]; !exists {
				return fmt.Errorf("contract %q depends on unknown contract %q", contract.ID, dependency)
			}
			if dependency == contract.ID {
				return fmt.Errorf("contract %q cannot depend on itself", contract.ID)
			}
		}
	}
	return validateAcyclic(byID)
}

func normalizeConfig(config Config) Config {
	config.Contracts = append([]Contract(nil), config.Contracts...)
	for i := range config.Contracts {
		contract := &config.Contracts[i]
		contract.Sources = sortedUniqueCopy(contract.Sources)
		contract.VerificationChecks = sortedUniqueCopy(contract.VerificationChecks)
		contract.Freedom = sortedUniqueCopy(contract.Freedom)
		contract.DependsOn = sortedUniqueCopy(contract.DependsOn)
		contract.Targets = append([]TargetPath(nil), contract.Targets...)
		sort.Slice(contract.Targets, func(i, j int) bool { return contract.Targets[i].Path < contract.Targets[j].Path })
	}
	sort.Slice(config.Contracts, func(i, j int) bool { return config.Contracts[i].ID < config.Contracts[j].ID })
	return config
}

func validateAcyclic(contracts map[string]Contract) error {
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("projection contract dependency cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		dependencies := append([]string(nil), contracts[id].DependsOn...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	ids := make([]string, 0, len(contracts))
	for id := range contracts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func evaluateContract(input Input, contract Contract, sources map[string]SourceProvenance, evidence map[string]CheckEvidence) (ContractPlan, []Diagnostic) {
	planned := ContractPlan{
		ID: contract.ID, ContractDigest: contractDigest(contract, input.ConfigDigest),
		Representation: contract.Representation, Materializer: contract.Materializer,
		Targets: []TargetPlan{}, VerificationChecks: []CheckResult{},
		Freedom: append([]string(nil), contract.Freedom...), DependsOn: append([]string(nil), contract.DependsOn...),
		Status: StatusConverged,
	}
	for _, key := range contract.Sources {
		planned.Sources = append(planned.Sources, sources[key])
	}
	var diagnostics []Diagnostic
	for _, target := range contract.Targets {
		targetPlan := TargetPlan{Path: target.Path}
		observed, exists := input.Files[target.Path]
		if exists {
			targetPlan.ObservedDigest = sha256Hex(observed)
			targetPlan.ObservedMode = artifactMode(input.FileModes, target.Path)
		}
		targetPlan.DesiredMode = artifactMode(input.DesiredModes, target.Path)
		if contract.Materializer.Mode == ModeDeterministic {
			desired, desiredExists := input.Desired[target.Path]
			if !desiredExists {
				targetPlan.Status = TargetIncomplete
				diagnostics = append(diagnostics, Diagnostic{Code: "projection.desired-missing", Severity: "warning", ContractID: contract.ID, Path: target.Path, Message: "registered deterministic renderer did not supply desired bytes"})
			} else {
				targetPlan.DesiredDigest = sha256Hex(desired)
				if targetPlan.DesiredMode == "" {
					targetPlan.DesiredMode = "100644"
				}
				switch {
				case !exists:
					targetPlan.Status = TargetMissing
					diagnostics = append(diagnostics, Diagnostic{Code: "projection.target-missing", Severity: "warning", ContractID: contract.ID, Path: target.Path, Message: "required projection target is absent"})
				case !bytes.Equal(observed, desired) || targetPlan.ObservedMode != targetPlan.DesiredMode:
					targetPlan.Status = TargetDrifted
					diagnostics = append(diagnostics, Diagnostic{Code: "projection.target-drift", Severity: "warning", ContractID: contract.ID, Path: target.Path, Message: "observed target bytes differ from registered deterministic renderer output"})
				default:
					targetPlan.Status = TargetMatched
				}
			}
		} else {
			targetPlan.Status = TargetIncomplete
			if targetPlan.DesiredMode == "" {
				targetPlan.DesiredMode = "100644"
			}
			if !exists {
				diagnostics = append(diagnostics, Diagnostic{Code: "projection.candidate-missing", Severity: "warning", ContractID: contract.ID, Path: target.Path, Message: "AI materialization candidate is absent"})
			}
		}
		planned.Targets = append(planned.Targets, targetPlan)
	}
	for _, check := range contract.VerificationChecks {
		result := CheckResult{Check: check, Status: EvidenceIncomplete, Reason: "no fixed-snapshot evidence supplied"}
		if item, ok := evidence[evidenceKey(contract.ID, check)]; ok {
			result.EvidenceDigest = checkEvidenceDigest(item)
			result.Status, result.Reason = evaluateEvidence(item, input, planned)
		}
		planned.VerificationChecks = append(planned.VerificationChecks, result)
		switch result.Status {
		case EvidenceFailed:
			diagnostics = append(diagnostics, Diagnostic{Code: "projection.check-failed", Severity: "error", ContractID: contract.ID, Check: check, Message: "named project verification check failed for this exact snapshot and target set"})
		case EvidenceIncomplete:
			diagnostics = append(diagnostics, Diagnostic{Code: "projection.check-incomplete", Severity: "warning", ContractID: contract.ID, Check: check, Message: result.Reason})
		}
	}
	if contract.Materializer.Mode == ModeAI {
		allChecksPassed := len(planned.VerificationChecks) == len(contract.VerificationChecks)
		for _, check := range planned.VerificationChecks {
			if check.Status != EvidencePassed {
				allChecksPassed = false
			}
		}
		if allChecksPassed {
			for i := range planned.Targets {
				if planned.Targets[i].ObservedDigest != "" {
					planned.Targets[i].Status = TargetVerified
				}
			}
		} else {
			for _, target := range planned.Targets {
				if target.ObservedDigest != "" {
					diagnostics = append(diagnostics, Diagnostic{Code: "projection.evidence-required", Severity: "warning", ContractID: contract.ID, Path: target.Path, Message: "candidate presence does not prove the required project checks"})
				}
			}
		}
	}
	planned.Status = aggregateContractStatus(planned.Targets, planned.VerificationChecks)
	return planned, diagnostics
}

func evaluateEvidence(evidence CheckEvidence, input Input, contract ContractPlan) (string, string) {
	if evidence.SnapshotDigest != input.SnapshotDigest {
		return EvidenceIncomplete, "evidence snapshot digest does not match the supplied model snapshot"
	}
	if evidence.IntentDigest != input.IntentDigest {
		return EvidenceIncomplete, "evidence intent digest does not match the supplied canonical intent"
	}
	if evidence.ContractDigest != contract.ContractDigest {
		return EvidenceIncomplete, "evidence contract digest does not match the active contract"
	}
	if len(evidence.TargetDigests) != len(contract.Targets) {
		return EvidenceIncomplete, "evidence does not bind every exact contract target"
	}
	for _, target := range contract.Targets {
		if target.ObservedDigest == "" {
			return EvidenceIncomplete, "a contract target is absent from the supplied snapshot"
		}
		if evidence.TargetDigests[target.Path] != target.ObservedDigest {
			return EvidenceIncomplete, "evidence target digest does not match the supplied snapshot"
		}
	}
	switch evidence.Status {
	case EvidencePassed:
		return EvidencePassed, ""
	case EvidenceFailed:
		return EvidenceFailed, "named project verification check failed"
	case EvidenceIncomplete:
		return EvidenceIncomplete, "named project verification check is incomplete"
	default:
		return EvidenceIncomplete, "evidence has unsupported status"
	}
}

func aggregateContractStatus(targets []TargetPlan, checks []CheckResult) string {
	status := StatusConverged
	for _, target := range targets {
		switch target.Status {
		case TargetMissing, TargetDrifted:
			status = StatusDrift
		case TargetIncomplete:
			if status != StatusDrift {
				status = StatusIncomplete
			}
		}
	}
	for _, check := range checks {
		switch check.Status {
		case EvidenceFailed:
			status = StatusDrift
		case EvidenceIncomplete:
			if status != StatusDrift {
				status = StatusIncomplete
			}
		}
	}
	return status
}

// propagateDependencyStatus makes an explicit prerequisite part of the
// dependent contract's evidence-relative status. The validated acyclic
// contract graph bounds this traversal; it does not schedule or execute work.
func propagateDependencyStatus(contracts []ContractPlan) []Diagnostic {
	byID := make(map[string]*ContractPlan, len(contracts))
	for i := range contracts {
		byID[contracts[i].ID] = &contracts[i]
	}
	state := make(map[string]uint8, len(contracts))
	var diagnostics []Diagnostic
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			// Config validation rejects cycles before this function is called.
			return
		}
		state[id] = 1
		contract := byID[id]
		for _, dependencyID := range contract.DependsOn {
			dependency := byID[dependencyID]
			visit(dependencyID)
			switch dependency.Status {
			case StatusDrift:
				if contract.Status != StatusDrift {
					contract.Status = StatusDrift
				}
				diagnostics = append(diagnostics, Diagnostic{Code: "projection.dependency-drift", Severity: "warning", ContractID: id, Message: fmt.Sprintf("prerequisite contract %q has drift", dependencyID)})
			case StatusIncomplete:
				if contract.Status != StatusDrift {
					contract.Status = StatusIncomplete
				}
				diagnostics = append(diagnostics, Diagnostic{Code: "projection.dependency-incomplete", Severity: "warning", ContractID: id, Message: fmt.Sprintf("prerequisite contract %q is incomplete", dependencyID)})
			}
		}
		state[id] = 2
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id)
	}
	return diagnostics
}

func aggregateStatus(contracts []ContractPlan) string {
	status := StatusConverged
	for _, c := range contracts {
		if c.Status == StatusDrift {
			return StatusDrift
		}
		if c.Status == StatusIncomplete {
			status = StatusIncomplete
		}
	}
	return status
}
func SourceIndex(sources []SourceProvenance) (map[string]SourceProvenance, error) {
	result := make(map[string]SourceProvenance, len(sources))
	for _, v := range sources {
		if v.Key == "" || strings.TrimSpace(v.Key) != v.Key {
			return nil, errors.New("projection input contains source without an exact identity key")
		}
		if strings.TrimSpace(v.Kind) == "" || strings.TrimSpace(v.Kind) != v.Kind {
			return nil, fmt.Errorf("projection source %q has invalid kind", v.Key)
		}
		if prior, ok := result[v.Key]; ok {
			if prior != v {
				return nil, fmt.Errorf("projection input contains conflicting source provenance for %q", v.Key)
			}
			continue
		}
		result[v.Key] = v
	}
	return result, nil
}

func validateTargetOwnership(config Config, protected []string) error {
	targets := []string{}
	for _, contract := range config.Contracts {
		for _, target := range contract.Targets {
			targets = append(targets, target.Path)
		}
	}
	protectedAll := append([]string(nil), protected...)
	for _, path := range protectedAll {
		if err := validateRelativePath(path); err != nil {
			return fmt.Errorf("protected path: %w", err)
		}
	}
	for _, target := range targets {
		for _, path := range protectedAll {
			if pathsOverlap(target, path) {
				return fmt.Errorf("projection target %q overlaps protected owner path %q", target, path)
			}
		}
	}
	return nil
}

func validateTargetsAgainstFiles(config Config, files map[string][]byte) error {
	for _, contract := range config.Contracts {
		for _, target := range contract.Targets {
			for path := range files {
				if pathsOverlap(target.Path, path) && target.Path != path {
					return fmt.Errorf("projection target %q case-aliases or overlaps observed file %q", target.Path, path)
				}
			}
		}
	}
	return nil
}

func isProjectPath(value string) bool {
	return validateRelativePath(value) == nil
}

func validateDesiredTargets(config Config, desired map[string][]byte) error {
	owners := map[string]Contract{}
	for _, contract := range config.Contracts {
		for _, target := range contract.Targets {
			owners[target.Path] = contract
		}
	}
	if err := validatePathMap(desired, "desired output"); err != nil {
		return err
	}
	for path := range desired {
		contract, exists := owners[path]
		if !exists {
			return fmt.Errorf("desired output %q is not a configured target", path)
		}
		if contract.Materializer.Mode != ModeDeterministic {
			return fmt.Errorf("AI target %q cannot be supplied as deterministic desired bytes", path)
		}
	}
	return nil
}

func indexEvidence(config Config, evidence []CheckEvidence) (map[string]CheckEvidence, error) {
	contracts := map[string]Contract{}
	for _, contract := range config.Contracts {
		contracts[contract.ID] = contract
	}
	result := make(map[string]CheckEvidence, len(evidence))
	for _, item := range evidence {
		contract, exists := contracts[item.ContractID]
		if !exists {
			return nil, fmt.Errorf("verification evidence references unknown contract %q", item.ContractID)
		}
		if !contains(contract.VerificationChecks, item.Check) {
			return nil, fmt.Errorf("verification evidence references undeclared check %q for contract %q", item.Check, item.ContractID)
		}
		if item.Status != EvidencePassed && item.Status != EvidenceFailed && item.Status != EvidenceIncomplete {
			return nil, fmt.Errorf("verification evidence has unsupported status %q", item.Status)
		}
		key := evidenceKey(item.ContractID, item.Check)
		if _, duplicate := result[key]; duplicate {
			return nil, fmt.Errorf("duplicate verification evidence for contract %q check %q", item.ContractID, item.Check)
		}
		result[key] = item
	}
	return result, nil
}

func evidenceKey(contractID, check string) string { return contractID + "\x00" + check }

func checkEvidenceDigest(evidence CheckEvidence) string {
	encoded, _ := json.Marshal(evidence)
	return sha256Hex(encoded)
}

func contractDigest(contract Contract, configDigest string) string {
	encoded, _ := json.Marshal(struct {
		ConfigDigest string   `json:"configDigest"`
		Contract     Contract `json:"contract"`
	}{ConfigDigest: configDigest, Contract: contract})
	return sha256Hex(encoded)
}

func digestFiles(files map[string][]byte, modes map[string]string) (string, error) {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	type fileDigest struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
		Mode   string `json:"mode"`
	}
	rows := make([]fileDigest, 0, len(paths))
	for _, path := range paths {
		rows = append(rows, fileDigest{Path: path, Digest: sha256Hex(files[path]), Mode: artifactMode(modes, path)})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return sha256Hex(encoded), nil
}

func validateModeMap(files map[string][]byte, modes map[string]string, label string) error {
	for name, mode := range modes {
		if _, exists := files[name]; !exists {
			return fmt.Errorf("%s is bound to absent file %q", label, name)
		}
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("%s for %q must be 100644 or 100755", label, name)
		}
	}
	return nil
}

func validateModeTargets(config Config, modes map[string]string, label string) error {
	owners := map[string]bool{}
	for _, contract := range config.Contracts {
		for _, target := range contract.Targets {
			owners[target.Path] = true
		}
	}
	for name, mode := range modes {
		if !owners[name] {
			return fmt.Errorf("%s %q is not a configured target", label, name)
		}
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("%s for %q must be 100644 or 100755", label, name)
		}
	}
	return nil
}

func artifactMode(modes map[string]string, name string) string {
	if mode := modes[name]; mode != "" {
		return mode
	}
	return "100644"
}

func digestStringSet(values []string) (string, error) {
	copyOf := sortedUniqueCopy(values)
	encoded, err := json.Marshal(copyOf)
	if err != nil {
		return "", err
	}
	return sha256Hex(encoded), nil
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func validatePathMap(values map[string][]byte, label string) error {
	paths := make([]string, 0, len(values))
	for path := range values {
		if err := validateRelativePath(path); err != nil {
			return fmt.Errorf("%s path: %w", label, err)
		}
		paths = append(paths, path)
	}
	return validateNoPathOverlaps(paths, label+" paths")
}

func validatePathSet(values []string, label string) error {
	for _, path := range values {
		if err := validateRelativePath(path); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return validateNoPathOverlaps(values, label+"s")
}

func validateRelativePath(path string) error {
	if path == "" || len(path) > maxPathBytes || strings.TrimSpace(path) != path || strings.Contains(path, "\\") || strings.ContainsAny(path, ":*?<>|\"") || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("%q is not a portable exact relative file path", path)
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.TrimSpace(part) != part || strings.HasSuffix(part, ".") {
			return fmt.Errorf("%q contains an invalid path component", path)
		}
		if strings.EqualFold(part, ".git") {
			return fmt.Errorf("%q targets Git metadata", path)
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || reservedDevice(base) {
			return fmt.Errorf("%q uses a Windows-reserved path component", path)
		}
		for _, r := range part {
			if r < 32 {
				return fmt.Errorf("%q contains a control character", path)
			}
		}
	}
	return nil
}

func reservedDevice(base string) bool {
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}

func validateNoPathOverlaps(paths []string, label string) error {
	copyOf := append([]string(nil), paths...)
	sort.Strings(copyOf)
	for i := range copyOf {
		for j := i + 1; j < len(copyOf); j++ {
			if pathsOverlap(copyOf[i], copyOf[j]) {
				return fmt.Errorf("%s overlap or case alias: %q and %q", label, copyOf[i], copyOf[j])
			}
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	a, b := strings.Split(left, "/"), strings.Split(right, "/")
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func uniqueStrings(values []string, label, contract string) error {
	seen := map[string]string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("contract %q has empty %s", contract, label)
		}
		folded := strings.ToLower(value)
		if previous, exists := seen[folded]; exists {
			return fmt.Errorf("contract %q has duplicate/case-aliased %s %q and %q", contract, label, previous, value)
		}
		seen[folded] = value
	}
	return nil
}

func sortedUniqueCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortDiagnostics(values []Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		return strings.Join([]string{a.Code, a.ContractID, a.Path, a.Check, a.Message}, "\x00") < strings.Join([]string{b.Code, b.ContractID, b.Path, b.Check, b.Message}, "\x00")
	})
}

// ValidateRelativePath exposes the engine's portable exact-path validator to adapters.
func ValidateRelativePath(path string) error { return validateRelativePath(path) }

// PropagateDependencyStatus applies validated prerequisite status to plan facts.
func PropagateDependencyStatus(contracts []ContractPlan) []Diagnostic {
	return propagateDependencyStatus(contracts)
}
func ownershipPaths(input Input) []string {
	if input.OwnershipPaths != nil {
		return input.OwnershipPaths
	}
	return input.ProtectedPaths
}
