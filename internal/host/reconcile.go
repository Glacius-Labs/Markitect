package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/render"
	"go.yaml.in/yaml/v3"
)

const ReconcilePlanVersion = "markitect.example.org/reconcile-plan/v1alpha1"
const reconcileEvidenceRoot = ".artifacts/markitect/reconcile/"

// ReconcileOperation is a concrete local projection write. Content remains
// UTF-8 text because generated Markitect projections are text resources.
type ReconcileOperation struct {
	Action       string `yaml:"action"`
	Path         string `yaml:"path"`
	ExpectedHash string `yaml:"expectedHash,omitempty"`
	ContentHash  string `yaml:"contentHash"`
	Content      string `yaml:"content"`
}

// ReconcilePlan binds an explicit set of output writes to the exact source,
// adapter configuration, and current destination bytes observed during plan.
type ReconcilePlan struct {
	APIVersion    string               `yaml:"apiVersion"`
	Adapter       string               `yaml:"adapter"`
	SourceDigest  string               `yaml:"sourceDigest"`
	ConfigDigest  string               `yaml:"configDigest"`
	ModelDigest   string               `yaml:"modelDigest"`
	DesiredDigest string               `yaml:"desiredDigest"`
	AdapterDigest string               `yaml:"adapterDigest"`
	ToolVersion   string               `yaml:"toolVersion"`
	ToolDigest    string               `yaml:"toolDigest"`
	Status        string               `yaml:"status"`
	Stale         []string             `yaml:"stale,omitempty"`
	Conflicts     []string             `yaml:"conflicts,omitempty"`
	Operations    []ReconcileOperation `yaml:"operations"`
}

type ReconcileObservation struct {
	APIVersion   string            `yaml:"apiVersion"`
	Adapter      string            `yaml:"adapter"`
	SourceDigest string            `yaml:"sourceDigest"`
	ConfigDigest string            `yaml:"configDigest"`
	Desired      map[string]string `yaml:"desired"`
	Observed     map[string]string `yaml:"observed"`
	Drift        []string          `yaml:"drift,omitempty"`
	Stale        []string          `yaml:"stale,omitempty"`
	Conflicts    []string          `yaml:"conflicts,omitempty"`
	ToolVersion  string            `yaml:"toolVersion"`
	ToolDigest   string            `yaml:"toolDigest"`
}

// ObserveProjection compares deterministic Markitect outputs with the
// selected snapshot. It never writes files.
func ObserveProjection(p *Project, toolVersion, toolDigest string) (ReconcileObservation, error) {
	if p == nil || p.Snapshot == nil || p.Graph == nil {
		return ReconcileObservation{}, errors.New("a loaded project snapshot is required")
	}
	outputs, err := render.Generate(p.Graph, p.Snapshot.Files)
	if err != nil {
		return ReconcileObservation{}, err
	}
	configBytes, err := YAML(p.Graph.Project.Spec)
	if err != nil {
		return ReconcileObservation{}, err
	}
	desired := map[string]string{}
	observed := map[string]string{}
	var drift, conflicts []string
	for _, name := range sortedFiles(outputs) {
		desired[name] = hashBytes(outputs[name])
		if current, ok := p.Snapshot.Files[name]; ok {
			observed[name] = hashBytes(current)
			if !bytes.Equal(normalize(current), normalize(outputs[name])) {
				drift = append(drift, name)
				if !Generated(current) {
					conflicts = append(conflicts, name)
				}
			}
		} else {
			observed[name] = "missing"
			drift = append(drift, name)
		}
	}
	stale := staleProjectionPaths(p, outputs)
	return ReconcileObservation{
		APIVersion: ReconcilePlanVersion, Adapter: "markitect-render",
		SourceDigest: reconcileInputDigest(p), ConfigDigest: hashBytes(configBytes),
		Desired: desired, Observed: observed, Drift: drift, Stale: stale, Conflicts: conflicts, ToolVersion: toolVersion, ToolDigest: toolDigest,
	}, nil
}

// PlanProjection returns the complete set of writes needed to bring current
// generated outputs to the declared projection. Generated files with no
// current owner are reported as stale and deliberately require a separate
// reviewed removal because this writer never deletes them.
func PlanProjection(p *Project, toolVersion, toolDigest string) (ReconcilePlan, error) {
	observation, err := ObserveProjection(p, toolVersion, toolDigest)
	if err != nil {
		return ReconcilePlan{}, err
	}
	outputs, err := render.Generate(p.Graph, p.Snapshot.Files)
	if err != nil {
		return ReconcilePlan{}, err
	}
	model, err := CompileModel(p)
	if err != nil {
		return ReconcilePlan{}, err
	}
	status := "complete"
	if len(observation.Stale) > 0 || len(observation.Conflicts) > 0 {
		status = "incomplete"
	}
	plan := ReconcilePlan{APIVersion: ReconcilePlanVersion, Adapter: observation.Adapter,
		SourceDigest: observation.SourceDigest, ConfigDigest: observation.ConfigDigest,
		ModelDigest: model.ModelDigest, DesiredDigest: digestStringMap(observation.Desired),
		AdapterDigest: hashBytes([]byte("markitect-render/v1alpha1")), ToolVersion: toolVersion, ToolDigest: toolDigest, Status: status, Stale: append([]string(nil), observation.Stale...), Conflicts: append([]string(nil), observation.Conflicts...)}
	for _, name := range sortedFiles(outputs) {
		current, exists := p.Snapshot.Files[name]
		if exists && bytes.Equal(normalize(current), normalize(outputs[name])) {
			continue
		}
		if exists && !Generated(current) {
			continue // Unmanaged collisions require an ownership decision, not a write operation.
		}
		action := "create"
		expected := "missing"
		if exists {
			action = "update"
			expected = hashBytes(current)
		}
		plan.Operations = append(plan.Operations, ReconcileOperation{Action: action, Path: name,
			ExpectedHash: expected, ContentHash: hashBytes(outputs[name]), Content: string(outputs[name])})
	}
	return plan, nil
}

// ValidateProjectionPlan rejects modified, stale, or incomplete plans before
// any write. ApplyProjection then delegates to the guarded single-owner writer.
func ValidateProjectionPlan(p *Project, plan ReconcilePlan, toolVersion, toolDigest string) error {
	if plan.APIVersion != ReconcilePlanVersion || plan.Adapter != "markitect-render" {
		return errors.New("unsupported reconciliation plan identity")
	}
	expected, err := PlanProjection(p, toolVersion, toolDigest)
	if err != nil {
		return err
	}
	a, err := YAML(plan)
	if err != nil {
		return err
	}
	b, err := YAML(expected)
	if err != nil {
		return err
	}
	if !bytes.Equal(a, b) {
		return errors.New("reconciliation plan is stale or was modified; observe and plan again from the current working tree")
	}
	if plan.Status != "complete" || len(plan.Stale) > 0 {
		return errors.New("incomplete projection plan cannot be applied")
	}
	return nil
}

// VerifyProjectionPlan checks that every planned destination now contains the
// exact planned bytes. Generated outputs and the reserved evidence directory
// are excluded from the source digest so apply does not invalidate its own
// evidence; all canonical source and ordinary input bytes remain covered.
func VerifyProjectionPlan(p *Project, plan ReconcilePlan, toolVersion, toolDigest string) error {
	if plan.APIVersion != ReconcilePlanVersion || plan.Adapter != "markitect-render" {
		return errors.New("unsupported reconciliation plan identity")
	}
	model, err := CompileModel(p)
	if err != nil {
		return err
	}
	if reconcileInputDigest(p) != plan.SourceDigest || model.ModelDigest != plan.ModelDigest || plan.ToolVersion != toolVersion || plan.ToolDigest != toolDigest || plan.AdapterDigest != hashBytes([]byte("markitect-render/v1alpha1")) {
		return errors.New("reconciliation source inputs changed since planning")
	}
	if len(plan.Stale) > 0 {
		return fmt.Errorf("stale generated outputs require deliberate removal: %s", strings.Join(plan.Stale, ", "))
	}
	observation, err := ObserveProjection(p, toolVersion, toolDigest)
	if err != nil {
		return err
	}
	if len(observation.Drift) > 0 || len(observation.Stale) > 0 || digestStringMap(observation.Desired) != plan.DesiredDigest {
		return errors.New("projection outputs do not match the planned desired state")
	}
	for _, operation := range plan.Operations {
		current, ok := p.Snapshot.Files[operation.Path]
		if !ok || hashBytes(current) != operation.ContentHash {
			return fmt.Errorf("planned output %s has not been verified at its planned content", operation.Path)
		}
	}
	return nil
}

func ApplyProjection(root string, p *Project, plan ReconcilePlan, toolVersion, toolDigest string) ([]string, error) {
	if err := ValidateProjectionPlan(p, plan, toolVersion, toolDigest); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		paths = append(paths, operation.Path)
	}
	return writeOutputPaths(root, p, paths)
}

func ParseReconcilePlan(data []byte) (ReconcilePlan, error) {
	var plan ReconcilePlan
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&plan); err != nil {
		return plan, fmt.Errorf("parse reconciliation plan: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return plan, errors.New("reconciliation plan must contain exactly one YAML document")
	}
	seen := map[string]bool{}
	for _, op := range plan.Operations {
		if op.Path == "" || seen[strings.ToLower(op.Path)] {
			return plan, errors.New("reconciliation plan paths must be nonempty and unique")
		}
		seen[strings.ToLower(op.Path)] = true
		if op.Action != "create" && op.Action != "update" {
			return plan, fmt.Errorf("unsupported local operation %q", op.Action)
		}
		if hashBytes([]byte(op.Content)) != op.ContentHash {
			return plan, fmt.Errorf("content hash mismatch for %s", op.Path)
		}
	}
	return plan, nil
}

func reconcileInputDigest(p *Project) string {
	owned := map[string]bool{}
	if outputs, err := render.Generate(p.Graph, p.Snapshot.Files); err == nil {
		for name := range outputs {
			owned[name] = true
		}
	}
	keys := make([]string, 0, len(p.Snapshot.Files))
	for name := range p.Snapshot.Files {
		if strings.HasPrefix(name, reconcileEvidenceRoot) || owned[name] {
			continue
		}
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, name := range keys {
		fmt.Fprintf(&builder, "%s\x00%s\n", name, hashBytes(p.Snapshot.Files[name]))
	}
	return hashBytes([]byte(builder.String()))
}

func hashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func digestStringMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&builder, "%s\x00%s\n", key, values[key])
	}
	return hashBytes([]byte(builder.String()))
}

// ReadPlan reads a local, caller-selected plan file. The source is not trusted;
// full semantic equality is checked against a fresh plan before apply.
func ReadPlan(path string) (ReconcilePlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReconcilePlan{}, err
	}
	return ParseReconcilePlan(data)
}
