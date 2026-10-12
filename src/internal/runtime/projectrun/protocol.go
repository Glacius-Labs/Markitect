package projectrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Delegation struct {
	ManagerID string `json:"managerId"`
	Goal      string `json:"goal"`
}

// TaskResponse is the narrow proposal/report payload carried in agentexec's
// typed ReportJSON field. It contains no transcript or authority grant.
type TaskResponse struct {
	Status            string          `json:"status"`
	Summary           string          `json:"summary"`
	Delegations       []Delegation    `json:"delegations"`
	ReworkRequests    []ReworkRequest `json:"reworkRequests"`
	Integrated        bool            `json:"integrated"`
	Questions         []string        `json:"questions"`
	Risks             []string        `json:"risks"`
	ResolvedQuestions []string        `json:"resolvedQuestions"`
	ResolvedRisks     []string        `json:"resolvedRisks"`
	EscalateTo        string          `json:"escalateTo"`
}

func decodeTaskResponse(raw json.RawMessage, phase string, activeChildren []string) (TaskResponse, error) {
	var response TaskResponse
	if len(raw) == 0 {
		return response, fmt.Errorf("agent response omitted the typed project task report")
	}
	allowed := map[string]bool{"status": true, "summary": true, "delegations": true, "reworkRequests": true, "integrated": true, "questions": true, "risks": true, "resolvedQuestions": true, "resolvedRisks": true, "escalateTo": true}
	if err := validateExactObjectKeys(raw, allowed); err != nil {
		return response, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return response, fmt.Errorf("decode typed project task report: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return response, fmt.Errorf("typed project task report has trailing data")
	}
	switch response.Status {
	case "complete", "partial", "blocked", "failed", "no-op":
	default:
		return response, fmt.Errorf("task report has invalid status %q", response.Status)
	}
	if strings.TrimSpace(response.Summary) == "" {
		return response, fmt.Errorf("task report requires a concise summary")
	}
	if len(response.Summary) > 4096 {
		return response, fmt.Errorf("task summary exceeds 4096 bytes")
	}
	if phase != "integrate" && len(response.ReworkRequests) > 0 {
		return response, fmt.Errorf("work report may not request manager-directed rework")
	}
	if phase == "integrate" && !response.Integrated {
		return response, fmt.Errorf("parent did not report actual child-candidate integration")
	}
	if phase != "integrate" && response.Integrated {
		return response, fmt.Errorf("work report cannot claim child integration")
	}
	if response.Delegations == nil || response.ReworkRequests == nil || response.Questions == nil || response.Risks == nil || response.ResolvedQuestions == nil || response.ResolvedRisks == nil {
		return response, fmt.Errorf("task report requires all list fields as arrays (empty when none)")
	}
	wanted := map[string]bool{}
	for _, id := range activeChildren {
		wanted[id] = true
	}
	seen := map[string]bool{}
	for _, delegation := range response.Delegations {
		if delegation.ManagerID == "" || strings.TrimSpace(delegation.Goal) == "" || len(delegation.Goal) > 4096 {
			return response, fmt.Errorf("delegation requires a Manager and bounded goal")
		}
		if !wanted[delegation.ManagerID] {
			return response, fmt.Errorf("manager delegated outside its active direct-child plan: %s", delegation.ManagerID)
		}
		if seen[delegation.ManagerID] {
			return response, fmt.Errorf("manager delegated twice to %s", delegation.ManagerID)
		}
		seen[delegation.ManagerID] = true
	}
	seenRework := map[string]bool{}
	for _, request := range response.ReworkRequests {
		if phase != "integrate" {
			return response, fmt.Errorf("work report may not request manager-directed rework")
		}
		if !wanted[request.ManagerID] {
			return response, fmt.Errorf("manager requested rework outside its active direct-child plan: %s", request.ManagerID)
		}
		if seenRework[request.ManagerID] || strings.TrimSpace(request.Goal) == "" || len(request.Goal) > 4096 || strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 2048 {
			return response, fmt.Errorf("rework request requires a unique active child and bounded goal and reason")
		}
		seenRework[request.ManagerID] = true
	}
	if phase == "work" {
		if len(response.ResolvedQuestions) > 0 || len(response.ResolvedRisks) > 0 {
			return response, fmt.Errorf("work report may not resolve manager obligations")
		}
		for id := range wanted {
			if !seen[id] {
				return response, fmt.Errorf("manager omitted active child delegation %s", id)
			}
		}
	} else if len(response.Delegations) > 0 {
		return response, fmt.Errorf("integration report may not create new delegations")
	}
	if phase == "work" && response.Integrated {
		return response, fmt.Errorf("work report cannot claim integration")
	}
	for _, question := range response.Questions {
		if strings.TrimSpace(question) == "" || len(question) > 4096 {
			return response, fmt.Errorf("task questions must be nonempty and bounded")
		}
	}
	for _, risk := range response.Risks {
		if strings.TrimSpace(risk) == "" || len(risk) > 4096 {
			return response, fmt.Errorf("task risks must be nonempty and bounded")
		}
	}
	if err := uniqueObligations(response.Questions); err != nil {
		return response, fmt.Errorf("task questions: %w", err)
	}
	if err := uniqueObligations(response.Risks); err != nil {
		return response, fmt.Errorf("task risks: %w", err)
	}
	if err := uniqueObligations(response.ResolvedQuestions); err != nil {
		return response, fmt.Errorf("resolved questions: %w", err)
	}
	if err := uniqueObligations(response.ResolvedRisks); err != nil {
		return response, fmt.Errorf("resolved risks: %w", err)
	}
	return response, nil
}

func uniqueObligations(values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return fmt.Errorf("duplicate obligation %q", value)
		}
		seen[value] = true
	}
	return nil
}

func taskResponseSchema(phase string) json.RawMessage {
	delegations := map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"managerId", "goal"}, "properties": map[string]any{"managerId": map[string]any{"type": "string", "minLength": 1}, "goal": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096}}}}
	textList := map[string]any{"type": "array", "items": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096}}
	reworkItems := map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"managerId", "goal", "reason"}, "properties": map[string]any{"managerId": map[string]any{"type": "string", "minLength": 1, "maxLength": 128}, "goal": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096}, "reason": map[string]any{"type": "string", "minLength": 1, "maxLength": 2048}}}}
	properties := map[string]any{
		"status":            map[string]any{"type": "string", "enum": []string{"complete", "partial", "blocked", "failed", "no-op"}, "description": "Use partial when an actionable question or risk remains unresolved. Use complete only when this phase's obligations are closed and no questions or risks remain."},
		"summary":           map[string]any{"type": "string", "minLength": 1, "maxLength": 4096, "description": "Summarize the outcome and supporting evidence. Put informational warnings or caveats from a successful command here; they are not unresolved risks by themselves."},
		"delegations":       delegations,
		"reworkRequests":    reworkItems,
		"integrated":        map[string]any{"type": "boolean", "enum": []bool{phase == "integrate"}},
		"questions":         map[string]any{"type": "array", "description": "Actionable questions that remain unresolved; use an empty array when none remain. A partial status requires at least one unresolved question or risk.", "items": textList["items"]},
		"risks":             map[string]any{"type": "array", "description": "Actionable risks that remain unresolved; use an empty array when none remain. Informational warnings from successful work belong in summary, not here.", "items": textList["items"]},
		"resolvedQuestions": map[string]any{"type": "array", "description": "Copy the exact original question only when supplied current evidence resolves it; otherwise leave it unresolved in questions.", "items": textList["items"]},
		"resolvedRisks":     map[string]any{"type": "array", "description": "Copy the exact original risk only when supplied current evidence resolves it; otherwise leave it unresolved in risks.", "items": textList["items"]},
		"escalateTo":        map[string]any{"type": "string", "maxLength": 128},
	}
	data, _ := json.Marshal(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"status", "summary", "delegations", "reworkRequests", "integrated", "questions", "risks", "resolvedQuestions", "resolvedRisks", "escalateTo"}, "properties": properties})
	return data
}

// TaskResponseSchema exposes the provider-neutral closed response contract
// for fixtures and adapters that need to inspect the task report shape.
func TaskResponseSchema(phase string) (json.RawMessage, error) {
	if phase != "work" && phase != "integrate" {
		return nil, fmt.Errorf("unknown project task phase %q", phase)
	}
	return append(json.RawMessage(nil), taskResponseSchema(phase)...), nil
}

func validateExactObjectKeys(raw json.RawMessage, allowed map[string]bool) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return fmt.Errorf("task report must be a JSON object")
	}
	seen := map[string]bool{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok || !allowed[key] {
			return fmt.Errorf("task report contains an unknown or noncanonical field %q", key)
		}
		if seen[key] {
			return fmt.Errorf("task report repeats field %q", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		if key == "delegations" {
			if err := validateArrayObjectKeys(value, map[string]bool{"managerId": true, "goal": true}); err != nil {
				return err
			}
		}
		if key == "reworkRequests" {
			if err := validateArrayObjectKeys(value, map[string]bool{"managerId": true, "goal": true, "reason": true}); err != nil {
				return err
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("task report has trailing data")
	}
	for key := range allowed {
		if !seen[key] {
			return fmt.Errorf("task report omitted required field %q", key)
		}
	}
	return nil
}
func validateArrayObjectKeys(raw json.RawMessage, allowed map[string]bool) error {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("delegations must be an array")
	}
	for _, item := range list {
		if err := validateExactObjectKeys(item, allowed); err != nil {
			return err
		}
	}
	return nil
}
