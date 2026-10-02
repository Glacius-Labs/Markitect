package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const (
	AdapterRequestVersion = "markitect.example.org/adapter-request/v1alpha1"
	AdapterResultVersion  = "markitect.example.org/adapter-result/v1alpha1"
	AdapterPlanVersion    = "markitect.example.org/adapter-plan/v1alpha1"
	adapterOutputLimit    = 1 << 20
	adapterDefaultTimeout = 2 * time.Minute
)

type CommandAdapterConfig struct {
	Inputs           []string       `yaml:"inputs"`
	Parameters       map[string]any `yaml:"parameters,omitempty"`
	Target           string         `yaml:"target,omitempty"`
	Observe          []string       `yaml:"observe"`
	Plan             []string       `yaml:"plan"`
	Apply            []string       `yaml:"apply,omitempty"`
	Verify           []string       `yaml:"verify"`
	AllowApply       bool           `yaml:"allowApply,omitempty"`
	TimeoutSeconds   int            `yaml:"timeoutSeconds,omitempty"`
	OutputLimitBytes int            `yaml:"outputLimitBytes,omitempty"`
}

type AdapterIdentity struct {
	Name       string         `yaml:"name"`
	Type       string         `yaml:"type"`
	Version    string         `yaml:"version"`
	Target     string         `yaml:"target,omitempty"`
	Parameters map[string]any `yaml:"parameters,omitempty"`
}

type AdapterRequest struct {
	APIVersion  string          `yaml:"apiVersion"`
	Action      string          `yaml:"action"`
	Adapter     AdapterIdentity `yaml:"adapter"`
	Model       SemanticModel   `yaml:"model"`
	Observation *AdapterResult  `yaml:"observation,omitempty"`
	Plan        *AdapterResult  `yaml:"plan,omitempty"`
}

type AdapterFinding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}

type AdapterOperation struct {
	ID      string         `yaml:"id"`
	Action  string         `yaml:"action"`
	Target  string         `yaml:"target"`
	Desired map[string]any `yaml:"desired,omitempty"`
}

type AdapterResult struct {
	APIVersion  string             `yaml:"apiVersion"`
	Adapter     string             `yaml:"adapter"`
	Action      string             `yaml:"action"`
	Status      string             `yaml:"status"`
	ModelDigest string             `yaml:"modelDigest"`
	Target      string             `yaml:"target,omitempty"`
	Observed    map[string]any     `yaml:"observed,omitempty"`
	Findings    []AdapterFinding   `yaml:"findings,omitempty"`
	Operations  []AdapterOperation `yaml:"operations,omitempty"`
}

type CommandAdapterPlan struct {
	APIVersion        string              `yaml:"apiVersion"`
	Adapter           AdapterPlanIdentity `yaml:"adapter"`
	SourceDigest      string              `yaml:"sourceDigest"`
	ModelDigest       string              `yaml:"modelDigest"`
	ConfigDigest      string              `yaml:"configDigest"`
	AdapterDigest     string              `yaml:"adapterDigest"`
	ToolVersion       string              `yaml:"toolVersion"`
	ToolDigest        string              `yaml:"toolDigest"`
	ObservationDigest string              `yaml:"observationDigest"`
	Result            AdapterResult       `yaml:"result"`
}

type AdapterPlanIdentity struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
	Target  string `yaml:"target,omitempty"`
}

type commandRun struct {
	name string
	argv []string
}

func FindAdapter(p *Project, name string) (core.AdapterConfig, error) {
	if p == nil || p.Graph == nil || p.Graph.Project == nil {
		return core.AdapterConfig{}, errors.New("parsed Project is required")
	}
	for _, adapter := range p.Graph.Project.Spec.Adapters {
		if adapter.Name == name {
			return adapter, nil
		}
	}
	return core.AdapterConfig{}, fmt.Errorf("adapter %q is not configured in the Project", name)
}

func parseCommandAdapter(adapter core.AdapterConfig) (CommandAdapterConfig, error) {
	if adapter.Type != "command" {
		return CommandAdapterConfig{}, fmt.Errorf("adapter %q has unsupported type %q", adapter.Name, adapter.Type)
	}
	encoded, err := yaml.Marshal(adapter.Config)
	if err != nil {
		return CommandAdapterConfig{}, err
	}
	var config CommandAdapterConfig
	decoder := yaml.NewDecoder(bytes.NewReader(encoded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("adapter %q config: %w", adapter.Name, err)
	}
	if len(config.Inputs) == 0 {
		return config, errors.New("command adapter must declare exact snapshot inputs")
	}
	if len(config.Observe) == 0 || len(config.Plan) == 0 || len(config.Verify) == 0 {
		return config, errors.New("command adapter requires observe, plan, and verify argv")
	}
	if config.AllowApply != (len(config.Apply) > 0) {
		return config, errors.New("command adapter apply argv and allowApply must be declared together")
	}
	if config.AllowApply && strings.TrimSpace(config.Target) == "" {
		return config, errors.New("write-capable command adapter requires an explicit non-secret target identity")
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = int(adapterDefaultTimeout.Seconds())
	}
	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 600 {
		return config, errors.New("timeoutSeconds must be between 1 and 600")
	}
	if config.OutputLimitBytes == 0 {
		config.OutputLimitBytes = adapterOutputLimit
	}
	if config.OutputLimitBytes < 1024 || config.OutputLimitBytes > 10*adapterOutputLimit {
		return config, errors.New("outputLimitBytes must be between 1024 and 10485760")
	}
	seen := map[string]bool{}
	for _, input := range config.Inputs {
		if err := validateAdapterInputPath(input); err != nil {
			return config, err
		}
		if seen[strings.ToLower(input)] {
			return config, fmt.Errorf("duplicate command adapter input %q", input)
		}
		seen[strings.ToLower(input)] = true
	}
	for _, run := range []struct {
		name string
		argv []string
	}{{"observe", config.Observe}, {"plan", config.Plan}, {"apply", config.Apply}, {"verify", config.Verify}} {
		if len(run.argv) == 0 {
			continue
		}
		if strings.TrimSpace(run.argv[0]) == "" || strings.ContainsAny(run.argv[0], `/\\`) || filepath.IsAbs(run.argv[0]) {
			return config, fmt.Errorf("%s executable must be a bare command name", run.name)
		}
	}
	return config, nil
}

func ObserveCommandAdapter(p *Project, name string) (AdapterResult, error) {
	adapter, config, err := loadCommandAdapter(p, name)
	if err != nil {
		return AdapterResult{}, err
	}
	model, err := compileAdapterModel(p)
	if err != nil {
		return AdapterResult{}, err
	}
	result, _, err := invokeCommandAdapter(p, adapter, config, config.Observe, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "observe", Adapter: identity(adapter, config), Model: model})
	return result, err
}

func PlanCommandAdapter(p *Project, name, toolVersion, toolDigest string) (CommandAdapterPlan, error) {
	adapter, config, err := loadCommandAdapter(p, name)
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	model, err := compileAdapterModel(p)
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	observe, adapterDigest, err := invokeCommandAdapter(p, adapter, config, config.Observe, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "observe", Adapter: identity(adapter, config), Model: model})
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	if observe.Status != "complete" {
		return CommandAdapterPlan{}, fmt.Errorf("adapter observation is %s; plan requires complete observation", observe.Status)
	}
	planResult, nextDigest, err := invokeCommandAdapter(p, adapter, config, config.Plan, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "plan", Adapter: identity(adapter, config), Model: model, Observation: &observe})
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	if nextDigest != adapterDigest {
		return CommandAdapterPlan{}, errors.New("adapter executable changed between observe and plan")
	}
	if planResult.Status != "complete" {
		return CommandAdapterPlan{}, fmt.Errorf("adapter plan is %s", planResult.Status)
	}
	configDigest, err := adapterConfigDigest(adapter, config)
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	return CommandAdapterPlan{APIVersion: AdapterPlanVersion, Adapter: planIdentity(adapter, config), SourceDigest: reconcileInputDigest(p), ModelDigest: model.ModelDigest, ConfigDigest: configDigest, AdapterDigest: adapterDigest, ToolVersion: toolVersion, ToolDigest: toolDigest, ObservationDigest: digestYAML(observe), Result: planResult}, nil
}

func ApplyCommandAdapter(p *Project, name, toolVersion, toolDigest string, plan CommandAdapterPlan) (AdapterResult, error) {
	adapter, config, err := loadCommandAdapter(p, name)
	if err != nil {
		return AdapterResult{}, err
	}
	if !config.AllowApply || len(config.Apply) == 0 {
		return AdapterResult{}, errors.New("adapter does not declare apply capability")
	}
	if plan.APIVersion != AdapterPlanVersion || plan.Adapter != planIdentity(adapter, config) {
		return AdapterResult{}, errors.New("unsupported command adapter plan or adapter identity")
	}
	model, err := compileAdapterModel(p)
	if err != nil {
		return AdapterResult{}, err
	}
	configDigest, err := adapterConfigDigest(adapter, config)
	if err != nil {
		return AdapterResult{}, err
	}
	if plan.SourceDigest != reconcileInputDigest(p) || plan.ModelDigest != model.ModelDigest || plan.ConfigDigest != configDigest || plan.ToolVersion != toolVersion || plan.ToolDigest != toolDigest {
		return AdapterResult{}, errors.New("adapter plan is stale; source, model, config, or tool identity changed")
	}
	observation, adapterDigest, err := invokeCommandAdapter(p, adapter, config, config.Observe, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "observe", Adapter: identity(adapter, config), Model: model})
	if err != nil {
		return AdapterResult{}, err
	}
	if adapterDigest != plan.AdapterDigest {
		return AdapterResult{}, errors.New("adapter executable changed since plan")
	}
	if digestYAML(observation) != plan.ObservationDigest {
		return AdapterResult{}, errors.New("external state changed since plan; observe and plan again")
	}
	planned, planDigest, err := invokeCommandAdapter(p, adapter, config, config.Plan, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "plan", Adapter: identity(adapter, config), Model: model, Observation: &observation})
	if err != nil {
		return AdapterResult{}, err
	}
	if planDigest != plan.AdapterDigest {
		return AdapterResult{}, errors.New("adapter executable changed since plan")
	}
	if planned.Status != "complete" {
		return AdapterResult{}, errors.New("fresh adapter plan is incomplete")
	}
	if !yamlEqual(planned, plan.Result) {
		return AdapterResult{}, errors.New("saved adapter operations differ from a fresh plan; refusing apply")
	}
	result, applyDigest, err := invokeCommandAdapter(p, adapter, config, config.Apply, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "apply", Adapter: identity(adapter, config), Model: model, Observation: &observation, Plan: &plan.Result})
	if err != nil {
		return result, err
	}
	if applyDigest != plan.AdapterDigest {
		return result, errors.New("adapter executable changed during apply")
	}
	return result, nil
}

func VerifyCommandAdapter(p *Project, name, toolVersion, toolDigest string, plan CommandAdapterPlan) (AdapterResult, error) {
	adapter, config, err := loadCommandAdapter(p, name)
	if err != nil {
		return AdapterResult{}, err
	}
	model, err := compileAdapterModel(p)
	if err != nil {
		return AdapterResult{}, err
	}
	configDigest, err := adapterConfigDigest(adapter, config)
	if err != nil {
		return AdapterResult{}, err
	}
	if plan.APIVersion != AdapterPlanVersion || plan.Adapter != planIdentity(adapter, config) || plan.SourceDigest != reconcileInputDigest(p) || plan.ModelDigest != model.ModelDigest || plan.ConfigDigest != configDigest || plan.ToolVersion != toolVersion || plan.ToolDigest != toolDigest {
		return AdapterResult{}, errors.New("adapter plan is stale; source, model, config, or tool identity changed")
	}
	result, adapterDigest, err := invokeCommandAdapter(p, adapter, config, config.Verify, AdapterRequest{APIVersion: AdapterRequestVersion, Action: "verify", Adapter: identity(adapter, config), Model: model, Plan: &plan.Result})
	if err != nil {
		return result, err
	}
	if adapterDigest != plan.AdapterDigest {
		return result, errors.New("adapter executable changed since plan")
	}
	return result, nil
}

func ReadCommandAdapterPlan(filename string) (CommandAdapterPlan, error) {
	file, err := os.Open(filename)
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	if !info.Mode().IsRegular() {
		return CommandAdapterPlan{}, errors.New("adapter plan must be a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, (2<<20)+1))
	if err != nil {
		return CommandAdapterPlan{}, err
	}
	if len(data) > 2<<20 {
		return CommandAdapterPlan{}, errors.New("adapter plan exceeds 2 MiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var plan CommandAdapterPlan
	if err = decoder.Decode(&plan); err != nil {
		return plan, fmt.Errorf("parse adapter plan: %w", err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return plan, errors.New("adapter plan must contain exactly one YAML document")
	}
	if plan.APIVersion != AdapterPlanVersion || plan.Adapter.Name == "" || plan.Result.APIVersion != AdapterResultVersion || plan.Result.Action != "plan" || plan.Result.Status != "complete" {
		return plan, errors.New("adapter plan has invalid identity or incomplete plan result")
	}
	if plan.Result.Adapter != plan.Adapter.Name || plan.Result.ModelDigest != plan.ModelDigest || plan.Result.Target != plan.Adapter.Target {
		return plan, errors.New("adapter plan result does not match its bound identity")
	}
	return plan, nil
}

func loadCommandAdapter(p *Project, name string) (core.AdapterConfig, CommandAdapterConfig, error) {
	adapter, err := FindAdapter(p, name)
	if err != nil {
		return adapter, CommandAdapterConfig{}, err
	}
	config, err := parseCommandAdapter(adapter)
	return adapter, config, err
}

func compileAdapterModel(p *Project) (SemanticModel, error) {
	model, err := CompileModel(p)
	if err != nil {
		return SemanticModel{}, err
	}
	if model.ValidationStatus != "passed" {
		return SemanticModel{}, fmt.Errorf("adapter requires a validated model; project has %d diagnostics", len(model.Diagnostics))
	}
	return model, nil
}

func identity(adapter core.AdapterConfig, config CommandAdapterConfig) AdapterIdentity {
	return AdapterIdentity{Name: adapter.Name, Type: adapter.Type, Version: adapter.Version, Target: config.Target, Parameters: config.Parameters}
}
func planIdentity(adapter core.AdapterConfig, config CommandAdapterConfig) AdapterPlanIdentity {
	return AdapterPlanIdentity{Name: adapter.Name, Type: adapter.Type, Version: adapter.Version, Target: config.Target}
}
func yamlEqual(a, b any) bool {
	x, e := YAML(a)
	if e != nil {
		return false
	}
	y, e := YAML(b)
	return e == nil && bytes.Equal(x, y)
}

func invokeCommandAdapter(p *Project, adapter core.AdapterConfig, config CommandAdapterConfig, argv []string, request AdapterRequest) (AdapterResult, string, error) {
	if p == nil || p.Snapshot == nil {
		return AdapterResult{}, "", errors.New("fixed project snapshot is required")
	}
	if len(argv) == 0 {
		return AdapterResult{}, "", errors.New("adapter command is not configured")
	}
	executable, err := findVerifyTool(argv[0])
	if err != nil {
		return AdapterResult{}, "", fmt.Errorf("adapter %s executable unavailable: %w", adapter.Name, err)
	}
	adapterDigest, err := commandAdapterDigest(config)
	if err != nil {
		return AdapterResult{}, "", err
	}
	temporary, err := os.MkdirTemp("", "markitect-adapter-")
	if err != nil {
		return AdapterResult{}, "", err
	}
	defer os.RemoveAll(temporary)
	if err = stageAdapterInputs(p, config.Inputs, temporary); err != nil {
		return AdapterResult{}, "", err
	}
	body, err := YAML(request)
	if err != nil {
		return AdapterResult{}, "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()
	out := &boundedAdapterOutput{limit: config.OutputLimitBytes, cancel: cancel}
	stderr := &boundedAdapterOutput{limit: config.OutputLimitBytes, cancel: cancel}
	cmd := exec.CommandContext(ctx, executable, argv[1:]...)
	cmd.Dir = temporary
	cmd.Env = verifyEnvironment(nil)
	cmd.Stdin = bytes.NewReader(body)
	cmd.Stdout = out
	cmd.Stderr = stderr
	cmd.WaitDelay = verifyWaitDelay
	runErr := cmd.Run()
	if out.exceeded() || stderr.exceeded() {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s exceeded output limit", adapter.Name)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s timed out", adapter.Name)
	}
	if runErr != nil {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s command failed: %w: %s; external side effects may be partial, inspect target state before retrying", adapter.Name, runErr, stderr.String())
	}
	var result AdapterResult
	decoder := yaml.NewDecoder(bytes.NewReader(out.Bytes()))
	decoder.KnownFields(true)
	if err = decoder.Decode(&result); err != nil {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s returned invalid YAML result: %w", adapter.Name, err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return AdapterResult{}, adapterDigest, errors.New("adapter result must contain exactly one YAML document")
	}
	if result.APIVersion != AdapterResultVersion || result.Adapter != adapter.Name || result.Action != request.Action || result.ModelDigest != request.Model.ModelDigest {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s response identity does not match request", adapter.Name)
	}
	if request.Adapter.Target != "" && result.Target != request.Adapter.Target {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s observed target %q, expected explicitly configured target %q", adapter.Name, result.Target, request.Adapter.Target)
	}
	if result.Status != "complete" && result.Status != "incomplete" && result.Status != "failed" {
		return AdapterResult{}, adapterDigest, fmt.Errorf("adapter %s returned unsupported status %q", adapter.Name, result.Status)
	}
	seen := map[string]bool{}
	for _, finding := range result.Findings {
		if finding.Code == "" || finding.Message == "" || (finding.Severity != "info" && finding.Severity != "warning" && finding.Severity != "error") {
			return AdapterResult{}, adapterDigest, errors.New("adapter finding requires code, message and info|warning|error severity")
		}
	}
	for _, operation := range result.Operations {
		if operation.ID == "" || operation.Action == "" || operation.Target == "" || seen[operation.ID] {
			return AdapterResult{}, adapterDigest, errors.New("adapter operations require unique id, action and target")
		}
		seen[operation.ID] = true
	}
	return result, adapterDigest, nil
}

func commandAdapterDigest(config CommandAdapterConfig) (string, error) {
	var builder strings.Builder
	for _, entry := range []struct {
		name string
		argv []string
	}{{"observe", config.Observe}, {"plan", config.Plan}, {"apply", config.Apply}, {"verify", config.Verify}} {
		if len(entry.argv) == 0 {
			continue
		}
		path, err := findVerifyTool(entry.argv[0])
		if err != nil {
			return "", fmt.Errorf("adapter %s executable unavailable: %w", entry.name, err)
		}
		digest, err := fileDigest(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&builder, "%s\x00%s\x00%s\n", entry.name, strings.Join(entry.argv, "\x00"), digest)
	}
	return hashBytes([]byte(builder.String())), nil
}

func stageAdapterInputs(p *Project, inputs []string, root string) error {
	for _, name := range inputs {
		data, ok := p.Snapshot.Files[name]
		if !ok {
			return fmt.Errorf("adapter input %q is absent from selected snapshot", name)
		}
		if err := validateAdapterInputPath(name); err != nil {
			return err
		}
		dest := filepath.Join(root, filepath.FromSlash(name))
		rel, err := filepath.Rel(root, dest)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("adapter input escapes staging directory: %s", name)
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(dest, data, 0600); err != nil {
			return err
		}
	}
	return nil
}

func validateAdapterInputPath(name string) error {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\:\x00") {
		return fmt.Errorf("unsafe adapter input path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("unsafe adapter input path %q", name)
		}
	}
	return nil
}

func adapterConfigDigest(adapter core.AdapterConfig, config CommandAdapterConfig) (string, error) {
	value := struct {
		Adapter AdapterIdentity      `yaml:"adapter"`
		Runtime CommandAdapterConfig `yaml:"runtime"`
	}{identity(adapter, config), config}
	data, err := YAML(value)
	if err != nil {
		return "", err
	}
	return fileDigestBytes(data), nil
}
func digestYAML(value any) string {
	data, err := YAML(value)
	if err != nil {
		return ""
	}
	return fileDigestBytes(data)
}
func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fileDigestBytes(data), nil
}
func fileDigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type boundedAdapterOutput struct {
	bytes.Buffer
	limit    int
	cancel   context.CancelFunc
	tooLarge bool
}

func (w *boundedAdapterOutput) Write(data []byte) (int, error) {
	remaining := w.limit - w.Len()
	if remaining > 0 {
		if len(data) > remaining {
			_, _ = w.Buffer.Write(data[:remaining])
		} else {
			_, _ = w.Buffer.Write(data)
		}
	}
	if len(data) > remaining && !w.tooLarge {
		w.tooLarge = true
		w.cancel()
	}
	return len(data), nil
}
func (w *boundedAdapterOutput) exceeded() bool { return w.tooLarge }
func (w *boundedAdapterOutput) String() string { return w.Buffer.String() }
func (w *boundedAdapterOutput) Bytes() []byte  { return w.Buffer.Bytes() }
