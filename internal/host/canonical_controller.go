package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const CanonicalControllerAPIVersion = "markitect.canonical/controller/v1alpha1"

type CanonicalRunnerConfig struct {
	Command         string                  `json:"command"`
	Args            []string                `json:"args"`
	Model           string                  `json:"model"`
	ModelOptions    json.RawMessage         `json:"modelOptions"`
	ProviderVersion string                  `json:"providerVersion"`
	TimeoutSeconds  int                     `json:"timeoutSeconds"`
	MaxStdoutBytes  int                     `json:"maxStdoutBytes"`
	MaxStderrBytes  int                     `json:"maxStderrBytes"`
	RuntimeFiles    []agentexec.RuntimeFile `json:"runtimeFiles"`
}

func (c CanonicalRunnerConfig) agentConfig() agentexec.Config {
	return agentexec.Config{Command: c.Command, Args: c.Args, Model: c.Model, ModelOptions: c.ModelOptions, ProviderVersion: c.ProviderVersion, Timeout: time.Duration(c.TimeoutSeconds) * time.Second, MaxStdoutBytes: c.MaxStdoutBytes, MaxStderrBytes: c.MaxStderrBytes, RuntimeFiles: c.RuntimeFiles}
}

// Runtime policy is explicit Host configuration, never canonical Core meaning.
type CanonicalAssuranceScope struct {
	ID           string            `json:"id"`
	ProjectionID string            `json:"projectionId"`
	Children     []string          `json:"children"`
	Checks       []authoring.Check `json:"checks"`
	CheckInputs  []string          `json:"checkInputs"`
}
type CanonicalControllerConfig struct {
	APIVersion       string                     `json:"apiVersion"`
	RecordStore      string                     `json:"recordStore"`
	PrivateLogs      string                     `json:"privateLogs"`
	ReferenceDepth   int                        `json:"referenceDepth"`
	AuditAll         bool                       `json:"auditAll"`
	CheckInputs      []string                   `json:"checkInputs"`
	Executor         CanonicalRunnerConfig      `json:"executor"`
	Verifier         CanonicalRunnerConfig      `json:"verifier"`
	AssuranceRoots   []string                   `json:"assuranceRoots"`
	AssuranceScopes  []CanonicalAssuranceScope  `json:"assuranceScopes"`
	TargetExclusions []CanonicalTargetExclusion `json:"targetExclusions,omitempty"`
}
type CanonicalControllerProposal struct {
	APIVersion              string                       `json:"apiVersion"`
	Digest                  string                       `json:"digest"`
	Status                  string                       `json:"status"`
	ConfigDigest            string                       `json:"configDigest"`
	LedgerHead              string                       `json:"ledgerHead"`
	LedgerSelectionDigest   string                       `json:"ledgerSelectionDigest"`
	Plan                    CanonicalScopedReconcilePlan `json:"plan"`
	InputPaths              []string                     `json:"inputPaths"`
	DependencyEvidencePaths []string                     `json:"dependencyEvidencePaths"`
	InputDigest             string                       `json:"inputDigest"`
	fixed                   *CanonicalSource
	observed                *snapshot.Snapshot
	active                  []records.ProjectionRecord
}
type CanonicalControllerWork struct {
	ProjectionID    string                          `json:"projectionId"`
	Candidate       []byte                          `json:"candidate,omitempty"`
	CandidateDigest string                          `json:"candidateDigest"`
	PlanDigest      string                          `json:"planDigest"`
	Outputs         map[string][]byte               `json:"outputs"`
	Executor        *agentexec.Receipt              `json:"executor,omitempty"`
	Escalations     []CanonicalProjectionEscalation `json:"escalations,omitempty"`
}
type CanonicalReviewedRun struct {
	APIVersion           string                      `json:"apiVersion"`
	Digest               string                      `json:"digest"`
	Status               string                      `json:"status"`
	ConfigPath           string                      `json:"configPath"`
	Proposal             CanonicalControllerProposal `json:"proposal"`
	ExecutorDigest       string                      `json:"executorDigest"`
	VerifierDigest       string                      `json:"verifierDigest"`
	HostExecutableDigest string                      `json:"hostExecutableDigest"`
	ToolVersion          string                      `json:"toolVersion"`
	ToolDigest           string                      `json:"toolDigest"`
	Work                 []CanonicalControllerWork   `json:"work"`
}
type CanonicalControllerApply struct {
	Status                  string                     `json:"status"`
	RunDigest               string                     `json:"runDigest"`
	Written                 []string                   `json:"written"`
	Records                 []records.ProjectionRecord `json:"records"`
	LedgerHead              string                     `json:"ledgerHead"`
	EvidenceRevision        string                     `json:"evidenceRevision,omitempty"`
	EvidencePaths           []string                   `json:"evidencePaths,omitempty"`
	EvidenceRefreshRequired []string                   `json:"evidenceRefreshRequired"`
}

func decodeControllerJSON(data []byte, target any, bound int) error {
	if len(data) == 0 || len(data) > bound {
		return errors.New("controller input is empty or exceeds its byte bound")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("controller input must contain exactly one JSON value")
	}
	return nil
}
func DecodeCanonicalControllerConfig(data []byte) (CanonicalControllerConfig, error) {
	var cfg CanonicalControllerConfig
	err := decodeControllerJSON(data, &cfg, 1<<20)
	if err != nil {
		return cfg, err
	}
	return cfg, validateControllerConfig(cfg)
}
func DecodeCanonicalReviewedRun(data []byte) (CanonicalReviewedRun, error) {
	var run CanonicalReviewedRun
	if err := decodeControllerJSON(data, &run, 32<<20); err != nil {
		return run, err
	}
	if run.APIVersion != CanonicalControllerAPIVersion || !validSHA256(run.Digest) || len(run.Work) > 128 {
		return run, errors.New("invalid version, digest or work bound in reviewed controller run")
	}
	actual := run.Digest
	run.Digest = ""
	wanted, err := digestCanonicalValue(run)
	run.Digest = actual
	if err != nil || wanted != actual {
		return run, errors.New("reviewed controller run digest mismatch")
	}
	return run, nil
}
func validateControllerConfig(cfg CanonicalControllerConfig) error {
	if cfg.APIVersion != CanonicalControllerAPIVersion {
		return errors.New("unsupported controller configuration version")
	}
	if err := validateCanonicalTargetExclusionConfig(cfg.TargetExclusions); err != nil {
		return err
	}
	if !filepath.IsAbs(cfg.RecordStore) || !filepath.IsAbs(cfg.PrivateLogs) {
		return errors.New("controller recordStore and privateLogs must be explicit absolute external paths")
	}
	if cfg.ReferenceDepth < 0 || cfg.ReferenceDepth > 2 || len(cfg.CheckInputs) > 128 {
		return errors.New("controller context/input bounds exceeded")
	}
	if len(cfg.AssuranceScopes) > 128 || len(cfg.AssuranceRoots) > 128 {
		return errors.New("assurance scope bound exceeded")
	}
	scopes, projections := map[string]bool{}, map[string]bool{}
	for _, scope := range cfg.AssuranceScopes {
		if scope.ID == "" || scope.ProjectionID == "" || scopes[scope.ID] || projections[scope.ProjectionID] || len(scope.Children) > 128 || len(scope.Checks) == 0 || len(scope.Checks) > 128 || len(scope.CheckInputs) > 128 {
			return errors.New("assurance scopes require unique ids/projections, bounded children and their own checks")
		}
		scopes[scope.ID] = true
		projections[scope.ProjectionID] = true
		for _, check := range scope.Checks {
			if err := authoring.ValidateCheck(check); err != nil {
				return err
			}
		}
	}
	if (len(cfg.AssuranceScopes) == 0) != (len(cfg.AssuranceRoots) == 0) {
		return errors.New("assurance roots and scopes must be supplied together")
	}
	for _, id := range cfg.AssuranceRoots {
		if !scopes[id] {
			return errors.New("assurance root is not declared")
		}
	}
	for _, scope := range cfg.AssuranceScopes {
		for _, child := range scope.Children {
			if !scopes[child] {
				return errors.New("assurance child is not declared")
			}
		}
	}
	seen := map[string]bool{}
	for _, name := range cfg.CheckInputs {
		if err := projectionengine.ValidateRelativePath(name); err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("duplicate check input: %s", name)
		}
		seen[name] = true
	}
	for _, scope := range cfg.AssuranceScopes {
		local := map[string]bool{}
		for _, name := range scope.CheckInputs {
			if err := projectionengine.ValidateRelativePath(name); err != nil {
				return err
			}
			if local[name] {
				return errors.New("duplicate assurance check input")
			}
			local[name] = true
		}
	}
	for _, r := range []CanonicalRunnerConfig{cfg.Executor, cfg.Verifier} {
		if r.Command == "" || r.ProviderVersion == "" || r.TimeoutSeconds < 1 || r.TimeoutSeconds > 600 {
			return errors.New("runner needs literal command, version and timeoutSeconds between 1 and 600")
		}
	}
	if len(canonicalControllerCheckInputs(cfg)) > 128 {
		return errors.New("combined assurance check-input bound exceeded")
	}
	return nil
}
func canonicalControllerForbiddenRoots(identity source.GitIdentity) []string {
	values := []string{identity.Root, identity.GitDir, identity.CommonDir}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) < len(values[j]) })
	roots := []string{}
	for _, v := range values {
		covered := false
		for _, r := range roots {
			if pathsOverlap(v, r) {
				covered = true
				break
			}
		}
		if !covered {
			roots = append(roots, v)
		}
	}
	return roots
}
func validateControllerExternalPath(value string, forbidden []string) error {
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return errors.New("controller external path must be absolute and clean")
	}
	parent, err := realDirectory(filepath.Dir(value))
	if err != nil {
		return err
	}
	if filepath.Join(parent, filepath.Base(value)) != value {
		return errors.New("controller destination uses a path alias")
	}
	if info, err := os.Lstat(value); err == nil {
		if !info.IsDir() || isReparsePoint(info) {
			return errors.New("controller external path is not a real directory")
		}
		if _, err = realDirectory(value); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, r := range forbidden {
		if pathsOverlap(value, r) {
			return fmt.Errorf("controller external path overlaps source/Git root: %s", r)
		}
	}
	return nil
}
func readCanonicalControllerLedger(root string, cfg CanonicalControllerConfig) (*recordstore.Store, recordstore.State, []records.ProjectionRecord, error) {
	id, err := source.IdentifyGit(root)
	if err != nil {
		return nil, recordstore.State{}, nil, err
	}
	forbidden := canonicalControllerForbiddenRoots(id)
	for _, p := range []string{cfg.RecordStore, cfg.PrivateLogs} {
		if err := validateControllerExternalPath(p, forbidden); err != nil {
			return nil, recordstore.State{}, nil, err
		}
	}
	if pathsOverlap(cfg.RecordStore, cfg.PrivateLogs) {
		return nil, recordstore.State{}, nil, errors.New("controller record store and private logs must be disjoint")
	}
	if _, err := os.Lstat(cfg.RecordStore); os.IsNotExist(err) {
		return nil, recordstore.State{}, nil, nil
	} else if err != nil {
		return nil, recordstore.State{}, nil, err
	}
	store, err := recordstore.Open(cfg.RecordStore, forbidden)
	if err != nil {
		return nil, recordstore.State{}, nil, err
	}
	state, err := store.Read()
	if err != nil {
		return nil, state, nil, err
	}
	byID := map[string]records.ProjectionRecord{}
	for _, r := range state.Records {
		byID[r.ID] = r
	}
	active := make([]records.ProjectionRecord, 0, len(state.ActiveSelection.RecordIDs))
	for _, key := range state.ActiveSelection.RecordIDs {
		r, ok := byID[key]
		if !ok {
			return nil, state, nil, errors.New("active record is missing")
		}
		active = append(active, r)
	}
	return store, state, active, nil
}

func ProposeCanonicalController(root, base, revision, configPath string, cfg CanonicalControllerConfig) (CanonicalControllerProposal, error) {
	report := CanonicalControllerProposal{APIVersion: CanonicalControllerAPIVersion}
	if err := validateControllerConfig(cfg); err != nil {
		return report, err
	}
	_, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	report.LedgerHead = state.Head
	report.LedgerSelectionDigest = state.ActiveSelection.Digest
	report.ConfigDigest, err = digestCanonicalValue(cfg)
	if err != nil {
		return report, err
	}
	plan, err := proposeScopedCanonicalReconciliation(root, base, revision, configPath, active, cfg.AuditAll, cfg.TargetExclusions, canonicalControllerCheckInputs(cfg))
	report.Plan = plan
	if err != nil {
		return report, err
	}
	report.Status = plan.Status
	report.fixed = plan.fixed
	if err := validateCanonicalControllerAssurance(cfg, plan.fixed); err != nil {
		return report, err
	}
	report.active = active
	report.DependencyEvidencePaths, err = canonicalControllerDependencyEvidencePaths(cfg, plan, active)
	if err != nil {
		return report, err
	}
	// Dependency bytes are selected evidence for parent work, not child work or
	// canonical context edges. Their digest participates in the reviewed input.
	report.InputPaths = sortedUniquePaths(append(append(append([]string(nil), plan.ObservedPaths...), canonicalControllerCheckInputs(cfg)...), report.DependencyEvidencePaths...))
	// Fixed check source is acquired only from explicit owner-supplied paths.
	if len(canonicalControllerCheckInputs(cfg)) > 0 {
		checks, err := source.LoadSelected(root, revision, canonicalControllerCheckInputs(cfg))
		if err != nil {
			return report, err
		}
		for name, data := range checks.Snapshot.Files {
			report.fixed.Snapshot.Files[name] = data
			report.fixed.Snapshot.Modes[name] = checks.Snapshot.Modes[name]
		}
	}
	observed, err := source.ObserveSelectedWorking(root, report.InputPaths)
	if err != nil {
		return report, err
	}
	if !equalCanonicalValue(observed.Identity, plan.SourceScope.Repository) {
		return report, errors.New("repository identity changed during check-input observation")
	}
	report.observed = observed.Snapshot
	report.InputDigest = sha256Prefix(observed.Snapshot.Digest())
	if err := validateCanonicalControllerDependencyEvidence(report.DependencyEvidencePaths, active, observed.Snapshot); err != nil {
		return report, err
	}
	for _, name := range canonicalControllerCheckInputs(cfg) {
		if !bytes.Equal(observed.Snapshot.Files[name], report.fixed.Snapshot.Files[name]) || observed.Snapshot.Modes[name] != report.fixed.Snapshot.Modes[name] {
			return report, fmt.Errorf("declared check input changed since source revision: %s", name)
		}
	}
	if err := validateCanonicalSourceUnchanged(report.fixed, report.observed); err != nil {
		return report, err
	}
	report.Digest, err = digestCanonicalValue(report)
	return report, err
}

func canonicalControllerCheckInputs(cfg CanonicalControllerConfig) []string {
	paths := append([]string(nil), cfg.CheckInputs...)
	for _, scope := range cfg.AssuranceScopes {
		paths = append(paths, scope.CheckInputs...)
	}
	return sortedUniquePaths(paths)
}
