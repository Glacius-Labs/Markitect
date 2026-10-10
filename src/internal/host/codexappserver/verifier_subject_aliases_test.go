package codexappserver

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func verifierSubjectAliasTestInvocation(t *testing.T, subjects []string, supply bool) agentexec.Invocation {
	t.Helper()
	context := map[string]any{"kind": "project-verify/v1", "requiredEvidenceRefs": []string{}}
	if supply {
		context["requiredSubjects"] = subjects
	}
	contextJSON, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	inv, _, err := agentexec.PrepareInvocation(agentexec.Request{
		Role:           agentexec.RoleVerifier,
		SourceRevision: strings.Repeat("a", 40),
		ModelDigest:    "sha256:" + strings.Repeat("b", 64),
		ModulePin:      "test@1",
		ProjectionID:   "test",
		ScopeIDs:       []string{},
		PolicyIDs:      []string{},
		Context:        contextJSON,
		Artifacts:      []agentexec.Artifact{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func TestNativeVerifierSubjectAliasesBindSchemaPromptAndDecode(t *testing.T) {
	subjects := []string{`assessment:"quoted"`, "verifier-subject-000000-000000", "check:smoke"}
	inv := verifierSubjectAliasTestInvocation(t, subjects, true)
	aliases, err := nativeVerifierSubjectAliases(inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != len(subjects) {
		t.Fatalf("alias count=%d want=%d", len(aliases), len(subjects))
	}
	byCanonical := map[string]string{}
	for alias, canonical := range aliases {
		if strings.ContainsAny(alias, `"\\`) {
			t.Fatalf("alias is not a short transport identifier: %q", alias)
		}
		for _, canonicalSubject := range subjects {
			if alias == canonicalSubject {
				t.Fatalf("alias collides with canonical subject %q", alias)
			}
		}
		byCanonical[canonical] = alias
	}
	for alias := range aliases {
		if !strings.HasPrefix(alias, "verifier-subject-000001-") {
			t.Fatalf("collision should select next alias generation: %q", alias)
		}
	}

	outputSchema := nativeTurnOutputSchema(inv)
	observations := outputSchema["properties"].(map[string]any)["verifierObservations"].(map[string]any)
	items := observations["items"].(map[string]any)
	subjectSchema := items["properties"].(map[string]any)["subject"].(map[string]any)
	wantAliases := make([]string, 0, len(aliases))
	for alias := range aliases {
		wantAliases = append(wantAliases, alias)
	}
	sort.Strings(wantAliases)
	gotAliases, ok := subjectSchema["enum"].([]string)
	if !ok || len(gotAliases) != len(wantAliases) {
		t.Fatalf("verifier observation subject enum=%#v", subjectSchema["enum"])
	}
	if strings.Join(gotAliases, "\x00") != strings.Join(wantAliases, "\x00") {
		t.Fatalf("schema aliases=%#v want=%#v", gotAliases, wantAliases)
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
	if !strings.Contains(prompt, string(mapping)) || !strings.Contains(prompt, "Use the assigned aliases in verifierObservations[].subject exactly once each") {
		t.Fatalf("verifier prompt omitted scoped subject alias contract %s", mapping)
	}
	if !strings.Contains(prompt, "evidenceRefs aliases") {
		t.Fatal("subject alias guidance obscured the independent evidenceRefs alias contract")
	}

	observationsInput := make([]map[string]string, 0, len(subjects))
	outcomes := []string{"passed", "failed", "incomplete"}
	for i, subject := range subjects {
		observationsInput = append(observationsInput, map[string]string{"subject": byCanonical[subject], "outcome": outcomes[i], "detail": fmt.Sprintf("detail %d", i)})
	}
	decoded, err := decodeNativeFinal(string(verifierAliasSemanticResponse(t, observationsInput)), inv)
	if err != nil {
		t.Fatalf("valid subject aliases failed decoding: %v", err)
	}
	if len(decoded.VerifierObservations) != len(subjects) {
		t.Fatalf("decoded observation count=%d want=%d", len(decoded.VerifierObservations), len(subjects))
	}
	for i, subject := range subjects {
		got := decoded.VerifierObservations[i]
		if got.Subject != subject || got.Outcome != outcomes[i] || got.Detail != fmt.Sprintf("detail %d", i) {
			t.Fatalf("observation semantic content changed: got=%+v want subject=%q outcome=%q detail=%q", got, subject, outcomes[i], fmt.Sprintf("detail %d", i))
		}
	}

	for name, invalidSubjects := range map[string][]string{
		"canonical": {subjects[0]},
		"unknown":   {"unknown-subject-alias"},
		"duplicate": {byCanonical[subjects[0]], byCanonical[subjects[0]]},
	} {
		t.Run("reject_"+name, func(t *testing.T) {
			rows := make([]map[string]string, 0, len(invalidSubjects))
			for _, subject := range invalidSubjects {
				rows = append(rows, map[string]string{"subject": subject, "outcome": "passed", "detail": "claim"})
			}
			if _, err := decodeNativeFinal(string(verifierAliasSemanticResponse(t, rows)), inv); err == nil {
				t.Fatalf("invalid observation subjects were accepted: %#v", invalidSubjects)
			}
		})
	}

	// Keep omitted rows omitted. The Host verifier coverage check remains the
	// authority for rejecting incomplete selection.
	partial, err := decodeNativeFinal(string(verifierAliasSemanticResponse(t, observationsInput[:1])), inv)
	if err != nil || len(partial.VerifierObservations) != 1 || partial.VerifierObservations[0].Subject != subjects[0] {
		t.Fatalf("adapter filled or rejected omitted observations: %#v err=%v", partial.VerifierObservations, err)
	}
}

func TestNativeVerifierSubjectAliasesPreserveGenericCompatibilityAndRejectBadBinding(t *testing.T) {
	generic := verifierSubjectAliasTestInvocation(t, nil, false)
	aliases, err := nativeVerifierSubjectAliases(generic)
	if err != nil || aliases != nil {
		t.Fatalf("generic verifier request changed compatibility: aliases=%#v err=%v", aliases, err)
	}
	schema := nativeTurnOutputSchema(generic)["properties"].(map[string]any)["verifierObservations"].(map[string]any)
	items := schema["items"].(map[string]any)
	subject := items["properties"].(map[string]any)["subject"].(map[string]any)
	if _, constrained := subject["enum"]; constrained {
		t.Fatal("generic request without requiredSubjects was unexpectedly narrowed")
	}
	semantic := verifierAliasSemanticResponse(t, []map[string]string{{"subject": `"exact generic label"`, "outcome": "passed", "detail": "legacy value"}})
	decoded, err := decodeNativeFinal(string(semantic), generic)
	if err != nil || len(decoded.VerifierObservations) != 1 || decoded.VerifierObservations[0].Subject != `"exact generic label"` {
		t.Fatalf("generic verifier subject contract changed: %#v err=%v", decoded.VerifierObservations, err)
	}

	for name, rawContext := range map[string]json.RawMessage{
		"null":      json.RawMessage(`{"requiredSubjects":null}`),
		"empty-id":  json.RawMessage(`{"requiredSubjects":[""]}`),
		"duplicate": json.RawMessage(`{"requiredSubjects":["same","same"]}`),
	} {
		t.Run(name, func(t *testing.T) {
			inv := verifierInvocationWithContext(t, rawContext)
			if _, err := nativeVerifierSubjectAliases(inv); err == nil {
				t.Fatal("invalid required-subject binding passed preflight")
			}
			if schema := nativeTurnOutputSchema(inv); schema != nil {
				t.Fatal("invalid request binding still produced an output schema")
			}
		})
	}
	tooMany := make([]string, nativeResponseArrayMaxItems+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("subject-%03d", i)
	}
	if _, err := nativeVerifierSubjectAliases(verifierSubjectAliasTestInvocation(t, tooMany, true)); err == nil {
		t.Fatal("over-bound subject list passed preflight")
	}

	empty := verifierSubjectAliasTestInvocation(t, []string{}, true)
	emptyAliases, err := nativeVerifierSubjectAliases(empty)
	if err != nil || emptyAliases == nil || len(emptyAliases) != 0 {
		t.Fatalf("explicit empty set not distinguished from absent set: %#v %v", emptyAliases, err)
	}
	emptyObservations := nativeTurnOutputSchema(empty)["properties"].(map[string]any)["verifierObservations"].(map[string]any)
	if emptyObservations["maxItems"] != 0 {
		t.Fatal("empty required set did not constrain observations to an empty array")
	}
}

func verifierInvocationWithContext(t *testing.T, context json.RawMessage) agentexec.Invocation {
	t.Helper()
	inv, _, err := agentexec.PrepareInvocation(agentexec.Request{Role: agentexec.RoleVerifier, SourceRevision: strings.Repeat("a", 40), ModelDigest: "sha256:" + strings.Repeat("b", 64), ModulePin: "test@1", ProjectionID: "test", ScopeIDs: []string{}, PolicyIDs: []string{}, Context: context, Artifacts: []agentexec.Artifact{}})
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func verifierAliasSemanticResponse(t *testing.T, observations []map[string]string) []byte {
	t.Helper()
	wire, err := json.Marshal(map[string]any{"outcome": "incomplete", "candidateFiles": []any{}, "verifierObservations": observations, "uncertainty": []any{}, "evidenceRefs": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
