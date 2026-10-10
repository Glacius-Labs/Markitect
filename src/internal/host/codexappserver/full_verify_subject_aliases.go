package codexappserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type fullVerifySubjectContext struct {
	Kind             string   `json:"kind"`
	RequiredSubjects []string `json:"requiredSubjects"`
}

func nativeFullVerifySubjectAliases(inv agentexec.Invocation) (map[string]string, error) {
	if inv.Request.Role != agentexec.RoleExecutor {
		return nil, nil
	}
	var context fullVerifySubjectContext
	if err := json.Unmarshal(inv.Request.Context, &context); err != nil {
		return nil, errors.New("native full-verification request context is invalid")
	}
	if context.Kind != "projectrun-full-verify/v1" {
		return nil, nil
	}
	if context.RequiredSubjects == nil || len(context.RequiredSubjects) == 0 || len(context.RequiredSubjects) > nativeResponseArrayMaxItems {
		return nil, fmt.Errorf("native full-verification required subjects must contain 1 to %d values", nativeResponseArrayMaxItems)
	}
	set := make(map[string]struct{}, len(context.RequiredSubjects))
	for _, subject := range context.RequiredSubjects {
		if subject == "" {
			return nil, errors.New("native full-verification required subject is empty")
		}
		if _, exists := set[subject]; exists {
			return nil, errors.New("native full-verification required subjects contain a duplicate")
		}
		set[subject] = struct{}{}
	}
	canonical := append([]string(nil), context.RequiredSubjects...)
	sort.Strings(canonical)
	for generation := 0; generation <= len(canonical); generation++ {
		prefix := fmt.Sprintf("audit-subject-%06d-", generation)
		aliases := make(map[string]string, len(canonical))
		collision := false
		for index, subject := range canonical {
			alias := fmt.Sprintf("%s%06d", prefix, index)
			if _, exists := set[alias]; exists {
				collision = true
				break
			}
			aliases[alias] = subject
		}
		if !collision {
			return aliases, nil
		}
	}
	return nil, errors.New("could not construct a collision-free full-verification subject alias namespace")
}

func nativeFullVerifyReportSchema(inv agentexec.Invocation, schema json.RawMessage) (json.RawMessage, error) {
	aliases, err := nativeFullVerifySubjectAliases(inv)
	if err != nil || aliases == nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(schema, &root); err != nil {
		return nil, errors.New("native full-verification response schema is invalid")
	}
	properties, ok := root["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("native full-verification response schema has no properties")
	}
	assessments, ok := properties["assessments"].(map[string]any)
	if !ok {
		return nil, errors.New("native full-verification schema omits assessments")
	}
	items, ok := assessments["items"].(map[string]any)
	if !ok {
		return nil, errors.New("native full-verification schema has invalid assessment items")
	}
	assessmentProperties, ok := items["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("native full-verification assessment schema has no properties")
	}
	subject, ok := assessmentProperties["subject"].(map[string]any)
	if !ok {
		return nil, errors.New("native full-verification assessment schema omits subject")
	}
	var context fullVerifySubjectContext
	if err := json.Unmarshal(inv.Request.Context, &context); err != nil {
		return nil, errors.New("native full-verification request context is invalid")
	}
	canonicalValues, ok := subject["enum"].([]any)
	if !ok || len(canonicalValues) != len(context.RequiredSubjects) {
		return nil, errors.New("native full-verification canonical subject schema does not match the request")
	}
	canonicalSet := make(map[string]bool, len(context.RequiredSubjects))
	for _, value := range context.RequiredSubjects {
		canonicalSet[value] = true
	}
	for _, value := range canonicalValues {
		text, ok := value.(string)
		if !ok || !canonicalSet[text] {
			return nil, errors.New("native full-verification canonical subject schema does not match the request")
		}
		delete(canonicalSet, text)
	}
	if len(canonicalSet) != 0 {
		return nil, errors.New("native full-verification canonical subject schema does not match the request")
	}
	values := make([]string, 0, len(aliases))
	for alias := range aliases {
		values = append(values, alias)
	}
	sort.Strings(values)
	subject["enum"] = values
	subject["description"] = "Emit the assigned short alias for one required subject; the Host maps it back to the exact canonical subject."
	wire, err := json.Marshal(root)
	if err != nil {
		return nil, errors.New("native full-verification response schema could not be encoded")
	}
	return wire, nil
}

func decodeNativeFullVerifyReportSubjects(report json.RawMessage, aliases map[string]string) (json.RawMessage, error) {
	if aliases == nil {
		return report, nil
	}
	root, err := decodeNativeSemanticObject(report, map[string]bool{
		"status": true, "summary": true, "assessments": true, "findings": true, "counterexamples": true,
	})
	if err != nil {
		return nil, errors.New("native full-verification report must be one closed JSON object")
	}
	for _, key := range []string{"status", "summary", "assessments", "findings", "counterexamples"} {
		if _, ok := root[key]; !ok {
			return nil, errors.New("native full-verification report omitted a required field")
		}
	}
	var assessments []json.RawMessage
	if err := json.Unmarshal(root["assessments"], &assessments); err != nil || assessments == nil {
		return nil, errors.New("native full-verification assessments must be an array")
	}
	if len(assessments) > nativeResponseArrayMaxItems {
		return nil, fmt.Errorf("native full-verification assessments exceed the supported %d values", nativeResponseArrayMaxItems)
	}
	seenAliases := map[string]bool{}
	seenSubjects := map[string]bool{}
	for index, raw := range assessments {
		assessment, err := decodeNativeSemanticObject(raw, map[string]bool{"subject": true, "outcome": true, "detail": true})
		if err != nil {
			return nil, errors.New("native full-verification assessment must be a closed JSON object")
		}
		for _, key := range []string{"subject", "outcome", "detail"} {
			if _, ok := assessment[key]; !ok {
				return nil, errors.New("native full-verification assessment omitted a required field")
			}
		}
		var alias string
		if err := json.Unmarshal(assessment["subject"], &alias); err != nil || alias == "" {
			return nil, errors.New("native full-verification subject alias is invalid")
		}
		if seenAliases[alias] {
			return nil, errors.New("native full-verification report contains a duplicate subject alias")
		}
		seenAliases[alias] = true
		canonical, ok := aliases[alias]
		if !ok {
			return nil, errors.New("native full-verification report contains an unknown subject alias")
		}
		if seenSubjects[canonical] {
			return nil, errors.New("native full-verification report duplicates a canonical subject")
		}
		seenSubjects[canonical] = true
		assessment["subject"], _ = json.Marshal(canonical)
		assessments[index], err = json.Marshal(assessment)
		if err != nil {
			return nil, errors.New("native full-verification assessment could not be encoded")
		}
	}
	root["assessments"], err = json.Marshal(assessments)
	if err != nil {
		return nil, errors.New("native full-verification assessments could not be encoded")
	}
	wire, err := json.Marshal(root)
	if err != nil {
		return nil, errors.New("native full-verification report could not be encoded")
	}
	return wire, nil
}
