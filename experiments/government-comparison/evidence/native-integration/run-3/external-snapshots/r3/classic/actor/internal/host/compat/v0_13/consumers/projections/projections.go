// Package projections retains the published v0.13 projection API. Planning is
// delegated to the model-independent projectionengine; this package binds the
// historical semantic model and typed governance results.
package projections

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	engine "github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
)

const (
	ConfigAPIVersion   = engine.ConfigAPIVersion
	PlanAPIVersion     = engine.PlanAPIVersion
	ConfigVersion      = engine.ConfigVersion
	ModeDeterministic  = engine.ModeDeterministic
	ModeAI             = engine.ModeAI
	StatusConverged    = engine.StatusConverged
	StatusDrift        = engine.StatusDrift
	StatusIncomplete   = engine.StatusIncomplete
	StatusBlocked      = engine.StatusBlocked
	EvidencePassed     = engine.EvidencePassed
	EvidenceFailed     = engine.EvidenceFailed
	EvidenceIncomplete = engine.EvidenceIncomplete
	TargetMatched      = engine.TargetMatched
	TargetVerified     = engine.TargetVerified
	TargetMissing      = engine.TargetMissing
	TargetDrifted      = engine.TargetDrifted
	TargetIncomplete   = engine.TargetIncomplete
)

type Config = engine.Config
type Contract = engine.Contract
type Materializer = engine.Materializer
type TargetPath = engine.TargetPath
type CheckEvidence = engine.CheckEvidence
type TargetPlan = engine.TargetPlan
type CheckResult = engine.CheckResult
type Diagnostic = engine.Diagnostic

type Input struct {
	Model          core.SemanticModel
	Config         Config
	ConfigDigest   string
	IntentDigest   string
	ToolName       string
	ToolVersion    string
	ToolDigest     string
	Files          map[string][]byte
	Desired        map[string][]byte
	ProtectedPaths []string
	Evidence       []CheckEvidence
}
type SourceProvenance struct {
	Key            string           `yaml:"key" json:"key"`
	Kind           string           `yaml:"kind" json:"kind"`
	Package        string           `yaml:"package,omitempty" json:"package,omitempty"`
	PackageVersion string           `yaml:"packageVersion,omitempty" json:"packageVersion,omitempty"`
	Source         core.ModelSource `yaml:"source" json:"source"`
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
type Plan struct {
	APIVersion      string                  `yaml:"apiVersion" json:"apiVersion"`
	Status          string                  `yaml:"status" json:"status"`
	PlanDigest      string                  `yaml:"planDigest" json:"planDigest"`
	ModelDigest     string                  `yaml:"modelDigest" json:"modelDigest"`
	SnapshotDigest  string                  `yaml:"snapshotDigest" json:"snapshotDigest"`
	ConfigDigest    string                  `yaml:"configDigest" json:"configDigest"`
	IntentDigest    string                  `yaml:"intentDigest" json:"intentDigest"`
	ToolName        string                  `yaml:"toolName" json:"toolName"`
	ToolVersion     string                  `yaml:"toolVersion" json:"toolVersion"`
	ToolDigest      string                  `yaml:"toolDigest" json:"toolDigest"`
	FilesDigest     string                  `yaml:"filesDigest" json:"filesDigest"`
	ProtectedDigest string                  `yaml:"protectedDigest" json:"protectedDigest"`
	Contracts       []ContractPlan          `yaml:"contracts" json:"contracts"`
	PolicyStatus    string                  `yaml:"policyStatus" json:"policyStatus"`
	PolicyResults   []core.PolicyResult     `yaml:"policyResults" json:"policyResults"`
	PolicySources   []core.ModelDomainInput `yaml:"policySources" json:"policySources"`
	Diagnostics     []Diagnostic            `yaml:"diagnostics" json:"diagnostics"`
}

func ParseConfig(data []byte) (Config, error) {
	config, err := engine.ParseConfig(data)
	if err != nil {
		return Config{}, err
	}
	for _, c := range config.Contracts {
		for _, key := range c.Sources {
			if strings.ContainsAny(key, "*?[]") {
				return Config{}, fmt.Errorf("contract %q has invalid exact source GraphKey %q", c.ID, key)
			}
		}
	}
	return config, nil
}

func Build(input Input) (Plan, error) {
	if err := validateModel(input.Model); err != nil {
		return Plan{}, err
	}
	sources, err := SourceIndex(input.Model)
	if err != nil {
		return Plan{}, err
	}
	facts := make([]engine.SourceProvenance, 0, len(sources))
	for _, source := range sources {
		facts = append(facts, engine.SourceProvenance{Key: source.Key, Kind: source.Kind, Package: source.Package, PackageVersion: source.PackageVersion, Source: engine.SourceInfo{Path: source.Source.Path, Line: source.Source.Line, Digest: source.Source.Digest}})
	}
	protectedOwners := append([]string(nil), input.ProtectedPaths...)
	for _, r := range input.Model.Resources {
		if isProjectPath(r.Source.Path) {
			protectedOwners = append(protectedOwners, r.Source.Path)
		}
	}
	for _, d := range input.Model.DomainInputs {
		if isProjectPath(d.Path) {
			protectedOwners = append(protectedOwners, d.Path)
		}
	}
	failed := input.Model.PolicyStatus == core.PolicyFailed || hasFailedPolicy(input.Model.PolicyResults)
	generic, err := engine.Build(engine.Input{
		ModelDigest: input.Model.ModelDigest, SnapshotDigest: input.Model.Snapshot.Digest,
		Config: input.Config, ConfigDigest: input.ConfigDigest, IntentDigest: input.IntentDigest,
		ToolName: input.ToolName, ToolVersion: input.ToolVersion, ToolDigest: input.ToolDigest,
		Sources: facts, Files: input.Files, Desired: input.Desired, ProtectedPaths: input.ProtectedPaths,
		OwnershipPaths: protectedOwners, Evidence: input.Evidence,
		Governance: engine.Governance{Status: string(input.Model.PolicyStatus), HasFailure: failed},
	})
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		APIVersion: generic.APIVersion, Status: generic.Status, ModelDigest: generic.ModelDigest, SnapshotDigest: generic.SnapshotDigest,
		ConfigDigest: generic.ConfigDigest, IntentDigest: generic.IntentDigest, ToolName: generic.ToolName, ToolVersion: generic.ToolVersion, ToolDigest: generic.ToolDigest,
		FilesDigest: generic.FilesDigest, ProtectedDigest: generic.ProtectedDigest, Contracts: make([]ContractPlan, 0, len(generic.Contracts)),
		PolicyStatus: string(input.Model.PolicyStatus), PolicyResults: sortedPolicyResults(input.Model.PolicyResults), PolicySources: policySourceInputs(input.Model),
		Diagnostics: append([]Diagnostic(nil), generic.Diagnostics...),
	}
	if failed {
		plan.Diagnostics = nil
		if len(plan.PolicyResults) == 0 {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Code: "policy.failed", Severity: "error", Message: "model policy status is failed; projection materialization is blocked"})
		}
		for _, result := range plan.PolicyResults {
			if result.Status != core.PolicyFailed {
				continue
			}
			message := result.Message
			if message == "" {
				message = "policy result is failed"
			}
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Code: "policy.failed", Severity: "error", Message: fmt.Sprintf("%s constraint %s for %s: %s", result.APIVersion, result.Constraint, result.Subject, message)})
		}
	}
	for _, c := range generic.Contracts {
		cp := ContractPlan{ID: c.ID, ContractDigest: c.ContractDigest, Representation: c.Representation, Materializer: c.Materializer, Targets: c.Targets, VerificationChecks: c.VerificationChecks, Freedom: c.Freedom, DependsOn: c.DependsOn, Status: c.Status}
		for _, source := range c.Sources {
			cp.Sources = append(cp.Sources, SourceProvenance{Key: source.Key, Kind: source.Kind, Package: source.Package, PackageVersion: source.PackageVersion, Source: core.ModelSource{Path: source.Source.Path, Line: source.Source.Line, Digest: source.Source.Digest}})
		}
		plan.Contracts = append(plan.Contracts, cp)
	}
	sortDiagnostics(plan.Diagnostics)
	encoded, err := json.Marshal(plan)
	if err != nil {
		return Plan{}, fmt.Errorf("encode projection plan digest: %w", err)
	}
	plan.PlanDigest = sha256Hex(encoded)
	return plan, nil
}

func validateModel(model core.SemanticModel) error {
	if model.APIVersion != core.SemanticModelVersion {
		return fmt.Errorf("projection input requires semantic model %q", core.SemanticModelVersion)
	}
	if model.StructuralStatus != "passed" {
		return fmt.Errorf("projection input requires structurally valid model; status is %q", model.StructuralStatus)
	}
	if strings.TrimSpace(model.ModelDigest) == "" || strings.TrimSpace(model.Snapshot.Digest) == "" {
		return errors.New("projection input requires model and snapshot digests")
	}
	switch model.PolicyStatus {
	case core.PolicyPassed, core.PolicyFailed, core.PolicyWaived:
	default:
		return fmt.Errorf("projection input has unsupported policy status %q", model.PolicyStatus)
	}
	for _, r := range model.PolicyResults {
		if r.Status != core.PolicyPassed && r.Status != core.PolicyFailed && r.Status != core.PolicyWaived {
			return fmt.Errorf("projection input has unsupported PolicyResult status %q", r.Status)
		}
	}
	return nil
}
func SourceIndex(model core.SemanticModel) (map[string]SourceProvenance, error) {
	result := make(map[string]SourceProvenance, len(model.Resources))
	add := func(v SourceProvenance) error {
		if v.Key == "" || strings.TrimSpace(v.Key) != v.Key {
			return errors.New("semantic model contains source without an exact identity key")
		}
		if old, ok := result[v.Key]; ok {
			if old != v {
				return fmt.Errorf("semantic model contains conflicting source provenance for %q", v.Key)
			}
			return nil
		}
		result[v.Key] = v
		return nil
	}
	for _, r := range model.Resources {
		if r.Identity.Kind == "" || strings.TrimSpace(r.Identity.Kind) != r.Identity.Kind {
			return nil, fmt.Errorf("semantic model resource %q has invalid kind", r.Identity.Key)
		}
		if err := add(SourceProvenance{Key: r.Identity.Key, Kind: r.Identity.Kind, Package: r.Identity.Package, Source: r.Source}); err != nil {
			return nil, err
		}
	}
	for _, d := range model.DomainInputs {
		if strings.TrimSpace(d.APIVersion) == "" || strings.TrimSpace(d.Name) == "" || strings.TrimSpace(d.APIVersion) != d.APIVersion || strings.TrimSpace(d.Name) != d.Name {
			return nil, errors.New("semantic model contains Domain input without an exact API version and name")
		}
		key := "domain:" + d.APIVersion + "/" + d.Name
		v := SourceProvenance{Key: key, Kind: "Domain", Package: d.Package, PackageVersion: d.PackageVersion, Source: core.ModelSource{Path: d.Path, Digest: d.Digest}}
		if err := add(v); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func policySourceInputs(model core.SemanticModel) []core.ModelDomainInput {
	out := append([]core.ModelDomainInput(nil), model.DomainInputs...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return strings.Join([]string{a.APIVersion, a.Name, a.Package, a.PackageVersion, a.Path, a.Digest}, "\x00") < strings.Join([]string{b.APIVersion, b.Name, b.Package, b.PackageVersion, b.Path, b.Digest}, "\x00")
	})
	return out
}
func sortedPolicyResults(results []core.PolicyResult) []core.PolicyResult {
	out := append([]core.PolicyResult(nil), results...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return strings.Join([]string{a.APIVersion, a.Constraint, a.Subject, a.Status, a.ConstraintDigest, a.SubjectDigest, a.Message, a.ExceptionName}, "\x00") < strings.Join([]string{b.APIVersion, b.Constraint, b.Subject, b.Status, b.ConstraintDigest, b.SubjectDigest, b.Message, b.ExceptionName}, "\x00")
	})
	return out
}
func hasFailedPolicy(results []core.PolicyResult) bool {
	for _, r := range results {
		if r.Status == core.PolicyFailed {
			return true
		}
	}
	return false
}
func hasWaivedPolicy(results []core.PolicyResult) bool {
	for _, r := range results {
		if r.Status == core.PolicyWaived {
			return true
		}
	}
	return false
}
func isProjectPath(p string) bool { return engine.ValidateRelativePath(p) == nil }
func propagateDependencyStatus(contracts []ContractPlan) []Diagnostic {
	generic := make([]engine.ContractPlan, len(contracts))
	for i, c := range contracts {
		generic[i] = engine.ContractPlan{ID: c.ID, DependsOn: c.DependsOn, Status: c.Status}
	}
	ds := engine.PropagateDependencyStatus(generic)
	for i := range contracts {
		contracts[i].Status = generic[i].Status
	}
	out := make([]Diagnostic, len(ds))
	copy(out, ds)
	return out
}
func sortDiagnostics(values []Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		return strings.Join([]string{a.Code, a.ContractID, a.Path, a.Check, a.Message}, "\x00") < strings.Join([]string{b.Code, b.ContractID, b.Path, b.Check, b.Message}, "\x00")
	})
}
func sha256Hex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
