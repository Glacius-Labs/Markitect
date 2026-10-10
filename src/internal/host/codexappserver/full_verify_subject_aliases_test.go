package codexappserver

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func fullVerifyAliasTestInvocation(t *testing.T, subjects []string) agentexec.Invocation {
	t.Helper()
	enum, err := json.Marshal(subjects)
	if err != nil {
		t.Fatal(err)
	}
	schema := json.RawMessage(fmt.Sprintf(`{"type":"object","additionalProperties":false,"required":["status","summary","assessments","findings","counterexamples"],"properties":{"status":{"type":"string","enum":["pass","fail","incomplete"]},"summary":{"type":"string"},"assessments":{"type":"array","minItems":%d,"maxItems":%d,"items":{"type":"object","additionalProperties":false,"required":["subject","outcome","detail"],"properties":{"subject":{"type":"string","enum":%s},"outcome":{"type":"string","enum":["pass","fail","incomplete"]},"detail":{"type":"string"}}}},"findings":{"type":"array","items":{"type":"string"}},"counterexamples":{"type":"array","items":{"type":"object"}}}}`, len(subjects), len(subjects), enum))
	context, err := json.Marshal(map[string]any{
		"kind":             "projectrun-full-verify/v1",
		"requiredSubjects": subjects,
		"responseSchema":   schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	inv, _, err := agentexec.PrepareInvocation(agentexec.Request{
		Role:           agentexec.RoleExecutor,
		SourceRevision: strings.Repeat("a", 40),
		ModelDigest:    "sha256:" + strings.Repeat("b", 64),
		ModulePin:      "test@1",
		ProjectionID:   "test",
		ScopeIDs:       []string{"manager"},
		PolicyIDs:      []string{},
		Context:        context,
		Artifacts:      []agentexec.Artifact{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func TestNativeFullVerifySubjectAliasesBindSchemaAndPrompt(t *testing.T) {
	subjects := []string{`project:manager:"quoted"`, "audit-subject-000000-000000", "evidence:line\nbreak"}
	inv := fullVerifyAliasTestInvocation(t, subjects)
	aliases, err := nativeFullVerifySubjectAliases(inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != len(subjects) {
		t.Fatalf("alias count=%d want=%d", len(aliases), len(subjects))
	}
	for alias, canonical := range aliases {
		if strings.ContainsAny(alias, `"\\`) || canonical == "" {
			t.Fatalf("alias is not a short transport identifier: %q -> %q", alias, canonical)
		}
		if _, collision := slicesContains(subjects, alias); collision {
			t.Fatalf("transport alias collides with a canonical subject: %q", alias)
		}
	}
	for alias := range aliases {
		if !strings.HasPrefix(alias, "audit-subject-000001-") {
			t.Fatalf("collision should select next alias generation: %q", alias)
		}
	}

	transformed, err := nativeFullVerifyReportSchema(inv, mustContextSchema(t, inv))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(transformed, &schema); err != nil {
		t.Fatal(err)
	}
	props := schema["properties"].(map[string]any)
	assessment := props["assessments"].(map[string]any)["items"].(map[string]any)
	subjectProp := assessment["properties"].(map[string]any)["subject"].(map[string]any)
	wantAliases := make([]string, 0, len(aliases))
	for alias := range aliases {
		wantAliases = append(wantAliases, alias)
	}
	sort.Strings(wantAliases)
	gotAliases, ok := subjectProp["enum"].([]any)
	if !ok || len(gotAliases) != len(wantAliases) {
		t.Fatalf("transformed schema enum=%#v", subjectProp["enum"])
	}
	for i := range gotAliases {
		if gotAliases[i] != wantAliases[i] {
			t.Fatalf("schema enum=%#v want=%#v", gotAliases, wantAliases)
		}
	}
	if _, ok := props["status"]; !ok {
		t.Fatal("schema transformation unexpectedly removed report properties")
	}
	output := nativeTurnOutputSchema(inv)
	outputProps := output["properties"].(map[string]any)
	outputReport := outputProps["reportJson"].(map[string]any)
	outputAssessments := outputReport["properties"].(map[string]any)["assessments"].(map[string]any)
	if outputAssessments["minItems"] != float64(len(subjects)) {
		t.Fatalf("alias schema weakened required assessment coverage: minItems=%v", outputAssessments["minItems"])
	}
	outputAssessment := outputAssessments["items"].(map[string]any)
	outputEnum := outputAssessment["properties"].(map[string]any)["subject"].(map[string]any)["enum"].([]any)
	if len(outputEnum) != len(wantAliases) {
		t.Fatalf("turn schema did not include transformed report enum: %#v", outputEnum)
	}
	for i := range wantAliases {
		if outputEnum[i] != wantAliases[i] {
			t.Fatalf("turn schema subject enum=%#v want=%#v", outputEnum, wantAliases)
		}
	}

	keys := make([]string, 0, len(aliases))
	for alias := range aliases {
		keys = append(keys, alias)
	}
	sort.Strings(keys)
	pairs := make([][2]string, 0, len(keys))
	for _, alias := range keys {
		pairs = append(pairs, [2]string{alias, aliases[alias]})
	}
	mapping, _ := json.Marshal(pairs)
	prompt := nativeTurnPrompt(inv, []byte(`{}`), `C:\workspace`)
	if !strings.Contains(prompt, string(mapping)) {
		t.Fatalf("prompt omitted exact alias mapping %s", mapping)
	}
	if strings.Contains(prompt, "copy every subject string verbatim into exactly one assessments[].subject") {
		t.Fatal("prompt retains canonical-copy instruction that conflicts with alias transport")
	}
	if !strings.Contains(prompt, "Host maps each alias back to its exact canonical subject") {
		t.Fatal("prompt does not explain Host-side canonicalization")
	}
}

func mustContextSchema(t *testing.T, inv agentexec.Invocation) json.RawMessage {
	t.Helper()
	var context struct {
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}
	if err := json.Unmarshal(inv.Request.Context, &context); err != nil {
		t.Fatal(err)
	}
	return context.ResponseSchema
}

func slicesContains(values []string, target string) (int, bool) {
	for i, value := range values {
		if value == target {
			return i, true
		}
	}
	return 0, false
}

func TestNativeFullVerifySubjectAliasesDecodeStrictlyAndPreserveSemantics(t *testing.T) {
	subjects := []string{`project:manager:"quoted"`, "check:build", "evidence:negative case"}
	inv := fullVerifyAliasTestInvocation(t, subjects)
	aliases, err := nativeFullVerifySubjectAliases(inv)
	if err != nil {
		t.Fatal(err)
	}
	byCanonical := map[string]string{}
	for alias, canonical := range aliases {
		byCanonical[canonical] = alias
	}

	for _, status := range []string{"pass", "fail", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			assessments := make([]map[string]string, 0, len(subjects))
			for i, subject := range subjects {
				assessments = append(assessments, map[string]string{"subject": byCanonical[subject], "outcome": status, "detail": fmt.Sprintf("evidence detail %d", i)})
			}
			report, _ := json.Marshal(map[string]any{"status": status, "summary": "semantic summary", "assessments": assessments, "findings": []string{"finding preserved"}, "counterexamples": []any{}})
			wire, _ := json.Marshal(map[string]any{"outcome": "proposed", "candidateFiles": []any{}, "verifierObservations": []any{}, "uncertainty": []any{}, "reportJson": json.RawMessage(report)})
			decoded, err := decodeNativeFinal(string(wire), inv)
			if err != nil {
				t.Fatalf("valid aliases failed decoding: %v", err)
			}
			var got struct {
				Status      string `json:"status"`
				Summary     string `json:"summary"`
				Assessments []struct {
					Subject string `json:"subject"`
					Outcome string `json:"outcome"`
					Detail  string `json:"detail"`
				} `json:"assessments"`
				Findings []string `json:"findings"`
			}
			if err := json.Unmarshal(decoded.ReportJSON, &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != status || got.Summary != "semantic summary" || len(got.Assessments) != len(subjects) || len(got.Findings) != 1 || got.Findings[0] != "finding preserved" {
				t.Fatalf("semantic report changed during alias decoding: %+v", got)
			}
			for i, subject := range subjects {
				if got.Assessments[i].Subject != subject || got.Assessments[i].Outcome != status || got.Assessments[i].Detail != fmt.Sprintf("evidence detail %d", i) {
					t.Fatalf("assessment %d changed during decode: %+v", i, got.Assessments[i])
				}
			}
		})
	}

	firstAlias := byCanonical[subjects[0]]
	secondAlias := byCanonical[subjects[1]]
	for name, assessmentSubjects := range map[string][]string{
		"canonical": {subjects[0]},
		"unknown":   {"unknown-subject-alias"},
		"duplicate": {firstAlias, firstAlias},
	} {
		t.Run("reject_"+name, func(t *testing.T) {
			report := fullVerifyReport(t, assessmentSubjects)
			wire := fullVerifySemanticResponse(t, report)
			if _, err := decodeNativeFinal(string(wire), inv); err == nil {
				t.Fatalf("invalid subject values accepted: %#v", assessmentSubjects)
			}
		})
	}

	// Missing rows are not filled by the adapter. The existing Host coverage
	// validator remains responsible for rejecting an incomplete audit.
	partialWire := fullVerifySemanticResponse(t, fullVerifyReport(t, []string{firstAlias, secondAlias}))
	partialResponse, err := decodeNativeFinal(string(partialWire), inv)
	if err != nil {
		t.Fatal(err)
	}
	var partialReport struct {
		Assessments []json.RawMessage `json:"assessments"`
	}
	if err := json.Unmarshal(partialResponse.ReportJSON, &partialReport); err != nil || len(partialReport.Assessments) != 2 {
		t.Fatalf("adapter filled missing assessment rows: count=%d err=%v", len(partialReport.Assessments), err)
	}
}

func fullVerifySemanticResponse(t *testing.T, report json.RawMessage) []byte {
	t.Helper()
	wire, err := json.Marshal(map[string]any{"outcome": "proposed", "candidateFiles": []any{}, "verifierObservations": []any{}, "uncertainty": []any{}, "reportJson": report})
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func fullVerifyReport(t *testing.T, subjects []string) json.RawMessage {
	t.Helper()
	assessments := make([]map[string]string, 0, len(subjects))
	for _, subject := range subjects {
		assessments = append(assessments, map[string]string{"subject": subject, "outcome": "incomplete", "detail": "limited evidence"})
	}
	report, err := json.Marshal(map[string]any{"status": "incomplete", "summary": "limited", "assessments": assessments, "findings": []string{}, "counterexamples": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestNativeFullVerifySubjectAliasesRejectInvalidBindingsBeforeRun(t *testing.T) {
	for name, subjects := range map[string][]string{
		"empty":     {""},
		"duplicate": {"same", "same"},
	} {
		t.Run(name, func(t *testing.T) {
			inv := fullVerifyAliasTestInvocation(t, subjects)
			if _, err := nativeFullVerifySubjectAliases(inv); err == nil {
				t.Fatal("invalid required subject binding passed preflight")
			}
		})
	}

	inv := fullVerifyAliasTestInvocation(t, []string{"one"})
	if _, err := nativeFullVerifyReportSchema(inv, json.RawMessage(`{"type":"object","properties":{}}`)); err == nil {
		t.Fatal("mismatched canonical schema passed preflight")
	}
	if schema := nativeTurnOutputSchema(inv); schema == nil {
		t.Fatal("valid full-verification schema was not included in the native turn")
	}
}
