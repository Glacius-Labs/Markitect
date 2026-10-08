package projectrun

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeTaskResponseRejectsDuplicateObligations(t *testing.T) {
	valid := `{"status":"partial","summary":"needs decision","delegations":[],"integrated":false,"questions":["which API?","which API?"],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":"parent"}`
	if _, err := decodeTaskResponse(json.RawMessage(valid), "work", nil); err == nil || !strings.Contains(err.Error(), "duplicate obligation") {
		t.Fatalf("expected duplicate obligation rejection, got %v", err)
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
