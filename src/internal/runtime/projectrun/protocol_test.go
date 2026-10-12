package projectrun

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeTaskResponseRejectsDuplicateObligations(t *testing.T) {
	valid := `{"status":"partial","summary":"needs decision","delegations":[],"reworkRequests":[],"integrated":false,"questions":["which API?","which API?"],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":"parent"}`
	if _, err := decodeTaskResponse(json.RawMessage(valid), "work", nil); err == nil || !strings.Contains(err.Error(), "duplicate obligation") {
		t.Fatalf("expected duplicate obligation rejection, got %v", err)
	}
}

func TestTaskResponseSchemaExplainsActionableObligationSemantics(t *testing.T) {
	for _, phase := range []string{"work", "integrate"} {
		t.Run(phase, func(t *testing.T) {
			raw, err := TaskResponseSchema(phase)
			if err != nil {
				t.Fatal(err)
			}
			var schema struct {
				Properties map[string]struct {
					Description string `json:"description"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatalf("decode task response schema: %v", err)
			}
			containsAll := func(field string, want ...string) {
				t.Helper()
				description := schema.Properties[field].Description
				for _, phrase := range want {
					if !strings.Contains(description, phrase) {
						t.Errorf("%s description %q omits %q", field, description, phrase)
					}
				}
			}
			containsAll("status", "partial", "actionable question or risk", "complete", "no questions or risks remain")
			containsAll("summary", "supporting evidence", "successful command", "not unresolved risks by themselves")
			containsAll("questions", "Actionable", "unresolved", "empty array", "partial status")
			containsAll("risks", "Actionable", "unresolved", "empty array", "successful work belong in summary")
			containsAll("resolvedQuestions", "exact original question", "supplied current evidence", "leave it unresolved")
			containsAll("resolvedRisks", "exact original risk", "supplied current evidence", "leave it unresolved")
		})
	}
}

func TestDecodeTaskResponseBoundsManagerDirectedReworkToDirectChildren(t *testing.T) {
	valid := `{"status":"complete","summary":"integration complete","delegations":[],"reworkRequests":[{"managerId":"orders","goal":"Correct the one owned artifact.","reason":"The integrated candidate misses its declared output."}],"integrated":true,"questions":[],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":""}`
	response, err := decodeTaskResponse(json.RawMessage(valid), "integrate", []string{"orders"})
	if err != nil || len(response.ReworkRequests) != 1 {
		t.Fatalf("valid direct-child rework rejected: response=%+v err=%v", response, err)
	}
	if _, err := decodeTaskResponse(json.RawMessage(valid), "integrate", []string{"inventory"}); err == nil || !strings.Contains(err.Error(), "outside its active direct-child plan") {
		t.Fatalf("rework outside direct children was accepted: %v", err)
	}
	if _, err := decodeTaskResponse(json.RawMessage(valid), "work", []string{"orders"}); err == nil || !strings.Contains(err.Error(), "may not request") {
		t.Fatalf("work-phase rework request was accepted: %v", err)
	}
}

func TestManagerObligationsRejectAmbiguousDuplicateAcrossManagers(t *testing.T) {
	parent := ManagerTask{ManagerID: "root", Questions: []string{"which API?"}}
	children := []ManagerTask{{ManagerID: "orders", Questions: []string{"which API?"}}}
	_, _, err := managerObligations(parent, children, []string{"orders"})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected manager-scoped duplicate rejection, got %v", err)
	}
}
