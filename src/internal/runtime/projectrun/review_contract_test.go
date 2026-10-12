package projectrun

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeReviewResponseRequiresClosedUniqueTypedObjects(t *testing.T) {
	valid := `{"status":"pass","summary":"Matches the scoped contract.","findings":[]}`
	if _, err := decodeReviewResponse(json.RawMessage(valid)); err != nil {
		t.Fatalf("valid typed review rejected: %v", err)
	}

	validFinding := `{"path":"src/orders/order.go","expectation":"Preserve the accepted transition.","grounding":"statement:order-transition"}`
	validFail := `{"status":"fail","summary":"One mismatch found.","findings":[` + validFinding + `]}`
	if _, err := decodeReviewResponse(json.RawMessage(validFail)); err != nil {
		t.Fatalf("valid finding report rejected: %v", err)
	}

	cases := map[string]string{
		"malformed JSON":          `{"status":"pass","summary":, "findings":[]}`,
		"missing field":           `{"status":"pass","summary":"ok"}`,
		"extra top-level field":   `{"status":"pass","summary":"ok","findings":[],"transcript":"private"}`,
		"duplicate top-level key": `{"status":"pass","status":"fail","summary":"ok","findings":[]}`,
		"extra finding field":     `{"status":"fail","summary":"bad","findings":[{"path":"src/orders/order.go","expectation":"fix it","grounding":"statement:order-transition","write":true}]}`,
		"duplicate finding field": `{"status":"fail","summary":"bad","findings":[{"path":"src/orders/order.go","path":"src/orders/order.go","expectation":"fix it","grounding":"statement:order-transition"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeReviewResponse(json.RawMessage(raw)); err == nil {
				t.Fatal("invalid typed review was accepted")
			}
		})
	}
}

func TestDecodeReviewResponseRequiresFindingsToMatchVerdict(t *testing.T) {
	finding := `{"path":"src/orders/order.go","expectation":"Preserve the accepted transition.","grounding":"statement:order-transition"}`
	cases := map[string]string{
		"pass with finding":    `{"status":"pass","summary":"Looks good.","findings":[` + finding + `]}`,
		"fail without finding": `{"status":"fail","summary":"Mismatch found.","findings":[]}`,
		"unknown verdict":      `{"status":"incomplete","summary":"Not enough evidence.","findings":[]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeReviewResponse(json.RawMessage(raw)); err == nil {
				t.Fatal("review verdict inconsistent with its typed findings was accepted")
			}
		})
	}
}

func TestDecodeTaskResponseAcceptsOnlyUniqueActiveDirectChildRework(t *testing.T) {
	valid := taskResponseWithRework("integrate", `[{"managerId":"orders","goal":"Correct reservation handling.","reason":"Review found a missing rollback."}]`)
	parsed, err := decodeTaskResponse(json.RawMessage(valid), "integrate", []string{"orders", "inventory"})
	if err != nil {
		t.Fatalf("valid active direct-child rework rejected: %v", err)
	}
	if len(parsed.ReworkRequests) != 1 || parsed.ReworkRequests[0].ManagerID != "orders" || parsed.ReworkRequests[0].Goal != "Correct reservation handling." {
		t.Fatalf("decoded rework requests = %#v", parsed.ReworkRequests)
	}

	cases := map[string]struct {
		phase    string
		children []string
		raw      string
	}{
		"foreign target":   {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"finance","goal":"Change finance behavior.","reason":"Review request."}]`)},
		"duplicate target": {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"orders","goal":"Correct it.","reason":"First finding."},{"managerId":"orders","goal":"Correct it again.","reason":"Second finding."}]`)},
		"missing goal":     {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"orders","reason":"Review request."}]`)},
		"empty goal":       {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"orders","goal":"  ","reason":"Review request."}]`)},
		"missing reason":   {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"orders","goal":"Correct it."}]`)},
		"empty reason":     {phase: "integrate", children: []string{"orders"}, raw: taskResponseWithRework("integrate", `[{"managerId":"orders","goal":"Correct it.","reason":" "}]`)},
		"work phase":       {phase: "work", children: []string{"orders"}, raw: taskResponseWithRework("work", `[{"managerId":"orders","goal":"Correct it.","reason":"Review request."}]`)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeTaskResponse(json.RawMessage(tc.raw), tc.phase, tc.children); err == nil {
				t.Fatal("invalid or unauthorized rework request was accepted")
			}
		})
	}
}

func taskResponseWithRework(phase, reworkRequests string) string {
	integrated := "false"
	delegations := `[]`
	if phase == "integrate" {
		integrated = "true"
	} else {
		delegations = `[{"managerId":"orders","goal":"Implement the orders change."}]`
	}
	return strings.Join([]string{
		`{"status":"complete","summary":"Task complete.","delegations":` + delegations + `,`,
		`"reworkRequests":` + reworkRequests + `,"integrated":` + integrated + `,`,
		`"questions":[],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":""}`,
	}, "")
}
