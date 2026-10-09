package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/adoption/capture"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/adoption/review"
)

func runPrepare(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.scope == "" || o.output == "" || (!o.write && o.expect != "") || (o.write && !capture.ValidHash(o.expect)) {
		return fail(fmt.Errorf("prepare requires --scope and --output; preview first, then --write --expect HANDOFF_DIGEST"))
	}
	data, err := host.ReadAdoptionRecord(o.scope)
	if err != nil {
		return fail(err)
	}
	var scope capture.Scope
	if err = capture.Decode(data, &scope); err != nil {
		return fail(err)
	}
	result, err := host.PrepareAdoption(scope, o.output, o.expect, o.write)
	if err != nil {
		if result != nil {
			if code := emit(result); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	return emit(result)
}

func runCopyMe(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.action == "infer" {
		return runCopyMeInference(o, emit)
	}
	if o.workspace == "" || o.queue == "" {
		return fail(fmt.Errorf("copy-me requires --workspace and --queue; --decision is an optional exact review record"))
	}
	handoff, blobs, err := host.ReadAdoptionWorkspace(o.workspace)
	if err != nil {
		return fail(err)
	}
	queueBytes, err := host.ReadAdoptionRecord(o.queue)
	if err != nil {
		return fail(err)
	}
	var queue review.Queue
	if err = capture.Decode(queueBytes, &queue); err != nil {
		return fail(err)
	}
	candidateBytes := map[string][]byte{}
	for _, c := range queue.Candidates {
		if _, ok := candidateBytes[c.Path]; ok {
			return fail(fmt.Errorf("duplicate candidate path %s", c.Path))
		}
		data, err := host.ReadAdoptionCandidate(filepath.Dir(o.queue), c.Path)
		if err != nil {
			return fail(err)
		}
		candidateBytes[c.Path] = data
	}
	var decisionBytes []byte
	if o.decision != "" {
		decisionBytes, err = host.ReadAdoptionRecord(o.decision)
		if err != nil {
			return fail(err)
		}
	}
	report, err := review.Validate(handoff, blobs, queueBytes, candidateBytes, decisionBytes)
	if err != nil {
		return fail(err)
	}
	return emit(report)
}

const brownfieldInferenceRuntimeAPIVersion = "markitect.brownfield/inference-runtime/v1alpha1"

type brownfieldInferenceRuntime struct {
	APIVersion  string                     `json:"apiVersion"`
	Agent       host.CanonicalRunnerConfig `json:"agent"`
	TempParent  string                     `json:"tempParent"`
	PrivateLogs string                     `json:"privateLogs"`
}

type brownfieldInferenceCLIResult struct {
	host.BrownfieldInferenceResult
	Error string `json:"error,omitempty"`
}

func runCopyMeInference(o commandOptions, emit func(any) int) int {
	result := brownfieldInferenceCLIResult{BrownfieldInferenceResult: host.BrownfieldInferenceResult{Status: "failed", Adopted: false}}
	finishError := func(err error) int {
		result.Error = err.Error()
		if code := emit(result); code != 0 {
			return code
		}
		return 1
	}
	runtimeBytes, err := host.ReadAdoptionRecord(o.runtime)
	if err != nil {
		return finishError(err)
	}
	runtimeConfig, err := decodeBrownfieldInferenceRuntime(runtimeBytes)
	if err != nil {
		return finishError(err)
	}
	handoff, blobs, err := host.ReadAdoptionWorkspace(o.workspace)
	if err != nil {
		return finishError(err)
	}
	queueBytes, err := host.ReadAdoptionRecord(o.queue)
	if err != nil {
		return finishError(err)
	}
	agentConfig := agentexec.Config{
		Command: runtimeConfig.Agent.Command, Args: runtimeConfig.Agent.Args,
		Model: runtimeConfig.Agent.Model, ModelOptions: runtimeConfig.Agent.ModelOptions,
		ProviderVersion: runtimeConfig.Agent.ProviderVersion,
		Timeout:         time.Duration(runtimeConfig.Agent.TimeoutSeconds) * time.Second,
		MaxStdoutBytes:  runtimeConfig.Agent.MaxStdoutBytes, MaxStderrBytes: runtimeConfig.Agent.MaxStderrBytes,
		RuntimeFiles: runtimeConfig.Agent.RuntimeFiles,
	}
	inference, err := host.RunBrownfieldInference(context.Background(), host.BrownfieldInferenceInput{
		HandoffBytes: handoff, Blobs: blobs, QueueBytes: queueBytes,
	}, agentConfig, host.BrownfieldInferenceOptions{
		TempParent: runtimeConfig.TempParent, PrivateLogDirectory: runtimeConfig.PrivateLogs,
	})
	result.BrownfieldInferenceResult = inference
	if err != nil {
		if result.Status == "" || result.Status == "proposed" {
			result.Status = "failed"
		}
		result.Error = err.Error()
		if code := emit(result); code != 0 {
			return code
		}
		if result.Status == "incomplete" {
			return 2
		}
		return 1
	}
	if result.Status == agentexec.OutcomeProposed && result.Candidate != nil && !result.Adopted {
		return emit(result)
	}
	if result.Status == agentexec.OutcomeIncomplete {
		if code := emit(result); code != 0 {
			return code
		}
		return 2
	}
	if result.Status != "failed" && result.Status != agentexec.OutcomeEscalated {
		result.Status = "failed"
		result.Error = "inference did not produce a noncanonical proposed candidate"
	}
	if code := emit(result); code != 0 {
		return code
	}
	return 1
}

func decodeBrownfieldInferenceRuntime(data []byte) (brownfieldInferenceRuntime, error) {
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return brownfieldInferenceRuntime{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return brownfieldInferenceRuntime{}, errors.New("inference runtime must be a JSON object")
	}
	if err := requireJSONFields(fields, "apiVersion", "agent", "tempParent", "privateLogs"); err != nil {
		return brownfieldInferenceRuntime{}, err
	}
	var agentFields map[string]json.RawMessage
	if err := json.Unmarshal(fields["agent"], &agentFields); err != nil || agentFields == nil {
		return brownfieldInferenceRuntime{}, errors.New("inference runtime agent must be a JSON object")
	}
	if err := requireJSONFields(agentFields, "command", "args", "model", "modelOptions", "providerVersion", "timeoutSeconds", "maxStdoutBytes", "maxStderrBytes", "runtimeFiles"); err != nil {
		return brownfieldInferenceRuntime{}, fmt.Errorf("inference runtime agent: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config brownfieldInferenceRuntime
	if err := decoder.Decode(&config); err != nil {
		return brownfieldInferenceRuntime{}, fmt.Errorf("decode inference runtime: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return brownfieldInferenceRuntime{}, err
	}
	if config.APIVersion != brownfieldInferenceRuntimeAPIVersion {
		return brownfieldInferenceRuntime{}, fmt.Errorf("inference runtime apiVersion must be %q", brownfieldInferenceRuntimeAPIVersion)
	}
	if config.Agent.TimeoutSeconds <= 0 || config.Agent.TimeoutSeconds > 600 {
		return brownfieldInferenceRuntime{}, errors.New("inference runtime timeoutSeconds must be between 1 and 600")
	}
	return config, nil
}

func requireJSONFields(fields map[string]json.RawMessage, required ...string) error {
	for _, key := range required {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("missing required field %q", key)
		}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("inference runtime must contain exactly one JSON value")
		}
		return fmt.Errorf("decode inference runtime trailing data: %w", err)
	}
	return nil
}
