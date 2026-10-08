package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
)

const maxGoalCLIInputBytes = 8 << 20

type goalInputDocument struct {
	Goal                   string             `json:"goal"`
	DecisionReferenceClaim string             `json:"decisionReferenceClaim"`
	Packages               []goalPackageInput `json:"packages"`
}

type goalPackageInput struct {
	Manifest string            `json:"manifest"`
	Files    map[string]string `json:"files"`
}

type goalSelectionDocument struct {
	SelectedRecommendationIDs []string `json:"selectedRecommendationIds"`
}

// runCanonicalGoal is the I/O adapter for canonical goal-recommend and
// goal-propose. Every read is limited to an explicitly named local input file;
// it does not accept a repository root or write output files.
func runCanonicalGoal(action, goalInputPath, recommendationsPath, selectionPath, runtimePath string, out io.Writer) int {
	emit := func(value any) int {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(value); err != nil {
			return 2
		}
		return 0
	}
	fail := func(err error) int {
		if code := emit(map[string]any{"status": "failed", "error": err.Error(), "adopted": false, "accepted": false}); code != 0 {
			return code
		}
		return 2
	}
	if action != "goal-recommend" && action != "goal-propose" {
		return fail(errors.New("canonical goal action must be goal-recommend or goal-propose"))
	}
	if goalInputPath == "" || runtimePath == "" {
		return fail(errors.New("goal action requires explicit --goal-input and --runtime files"))
	}
	if action == "goal-recommend" && (recommendationsPath != "" || selectionPath != "") {
		return fail(errors.New("goal-recommend does not accept recommendation or selection files"))
	}
	if action == "goal-propose" && (recommendationsPath == "" || selectionPath == "") {
		return fail(errors.New("goal-propose requires explicit --goal-recommendations and --goal-selection files"))
	}
	goalBytes, err := readCanonicalLocalInput(goalInputPath, maxGoalCLIInputBytes)
	if err != nil {
		return fail(fmt.Errorf("read goal input: %w", err))
	}
	var goalWire goalInputDocument
	if err := decodeCanonicalGoalJSON(goalBytes, &goalWire); err != nil {
		return fail(fmt.Errorf("decode closed goal input: %w", err))
	}
	goalInput, err := canonicalGoalInput(goalWire)
	if err != nil {
		return fail(err)
	}
	runtimeBytes, err := readCanonicalLocalInput(runtimePath, 1<<20)
	if err != nil {
		return fail(fmt.Errorf("read inference runtime: %w", err))
	}
	runtimeConfig, err := decodeBrownfieldInferenceRuntime(runtimeBytes)
	if err != nil {
		return fail(fmt.Errorf("decode inference runtime: %w", err))
	}
	config := agentexec.Config{
		Command: runtimeConfig.Agent.Command, Args: runtimeConfig.Agent.Args,
		Model: runtimeConfig.Agent.Model, ModelOptions: runtimeConfig.Agent.ModelOptions,
		ProviderVersion: runtimeConfig.Agent.ProviderVersion,
		Timeout:         time.Duration(runtimeConfig.Agent.TimeoutSeconds) * time.Second,
		MaxStdoutBytes:  runtimeConfig.Agent.MaxStdoutBytes, MaxStderrBytes: runtimeConfig.Agent.MaxStderrBytes,
		RuntimeFiles: runtimeConfig.Agent.RuntimeFiles,
	}
	options := host.GoalModelingOptions{TempParent: runtimeConfig.TempParent, PrivateLogDirectory: runtimeConfig.PrivateLogs}
	if action == "goal-recommend" {
		result, runErr := host.RecommendGoalModules(context.Background(), goalInput, config, options)
		if runErr != nil {
			return emitGoalCLIResult(result, runErr, emit)
		}
		return emitGoalCLIResult(result, nil, emit)
	}
	recommendationBytes, err := readCanonicalLocalInput(recommendationsPath, maxGoalCLIInputBytes)
	if err != nil {
		return fail(fmt.Errorf("read exact goal recommendation result: %w", err))
	}
	var recommendations host.GoalModuleRecommendationResult
	if err := decodeCanonicalGoalJSON(recommendationBytes, &recommendations); err != nil {
		return fail(fmt.Errorf("decode exact goal recommendation result: %w", err))
	}
	selectionBytes, err := readCanonicalLocalInput(selectionPath, 1<<20)
	if err != nil {
		return fail(fmt.Errorf("read goal selection: %w", err))
	}
	var selection goalSelectionDocument
	if err := decodeCanonicalGoalJSON(selectionBytes, &selection); err != nil {
		return fail(fmt.Errorf("decode closed goal selection: %w", err))
	}
	result, runErr := host.ProposeGoalModel(context.Background(), goalInput, recommendations, selection.SelectedRecommendationIDs, config, options)
	if runErr != nil {
		return emitGoalCLIResult(result, runErr, emit)
	}
	return emitGoalCLIResult(result, nil, emit)
}

func canonicalGoalInput(wire goalInputDocument) (host.GoalModelingInput, error) {
	if !utf8.ValidString(wire.Goal) || !utf8.ValidString(wire.DecisionReferenceClaim) {
		return host.GoalModelingInput{}, errors.New("goal and decisionReferenceClaim must contain valid UTF-8")
	}
	packages := make([]canonical.ModulePackage, 0, len(wire.Packages))
	for i, pkg := range wire.Packages {
		if !utf8.ValidString(pkg.Manifest) {
			return host.GoalModelingInput{}, fmt.Errorf("package %d manifest must contain valid UTF-8", i+1)
		}
		if pkg.Files == nil {
			return host.GoalModelingInput{}, fmt.Errorf("package %d must explicitly provide a files object", i+1)
		}
		files := make(map[string][]byte, len(pkg.Files))
		for path, content := range pkg.Files {
			if !utf8.ValidString(content) {
				return host.GoalModelingInput{}, fmt.Errorf("package %d file %q must contain valid UTF-8", i+1, path)
			}
			files[path] = []byte(content)
		}
		packages = append(packages, canonical.ModulePackage{ManifestBytes: []byte(pkg.Manifest), Files: files})
	}
	return host.GoalModelingInput{Goal: wire.Goal, Packages: packages, DecisionReferenceClaim: wire.DecisionReferenceClaim}, nil
}

func decodeCanonicalGoalJSON(data []byte, dst any) error {
	if len(data) == 0 || len(data) > maxGoalCLIInputBytes || !utf8.Valid(data) {
		return errors.New("input must be nonempty UTF-8 JSON within the 8 MiB bound")
	}
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func emitGoalCLIResult(value any, runErr error, emit func(any) int) int {
	data, err := json.Marshal(value)
	if err != nil {
		if code := emit(map[string]any{"status": "failed", "error": err.Error(), "adopted": false, "accepted": false}); code != 0 {
			return code
		}
		return 2
	}
	var flat map[string]any
	if err := json.Unmarshal(data, &flat); err != nil || flat == nil {
		if code := emit(map[string]any{"status": "failed", "error": "goal result was not an object", "adopted": false, "accepted": false}); code != 0 {
			return code
		}
		return 2
	}
	flat["adopted"] = false
	flat["accepted"] = false
	if runErr != nil {
		flat["error"] = runErr.Error()
	}
	code := emit(flat)
	if code != 0 {
		return code
	}
	if runErr != nil {
		if flat["status"] == "incomplete" {
			return 2
		}
		return 1
	}
	if flat["status"] == "proposed" {
		return 0
	}
	if flat["status"] == "incomplete" {
		return 2
	}
	return 1
}

func isCanonicalGoalAction(action string) bool {
	return action == "goal-recommend" || action == "goal-propose"
}
