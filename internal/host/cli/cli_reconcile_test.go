package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func TestCommandAdapterPlanExitReflectsResult(t *testing.T) {
	cases := []struct {
		name   string
		result app.AdapterResult
		want   int
	}{
		{name: "complete", result: app.AdapterResult{Status: "complete"}, want: 0},
		{name: "incomplete", result: app.AdapterResult{Status: "incomplete"}, want: 2},
		{name: "failed", result: app.AdapterResult{Status: "failed"}, want: 1},
		{name: "error finding", result: app.AdapterResult{Status: "complete", Findings: []app.AdapterFinding{{Code: "policy", Severity: "error", Message: "drift"}}}, want: 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			emitted := false
			code := emitCommandAdapterPlan(app.CommandAdapterPlan{Result: test.result}, func(value any) int {
				emitted = true
				if _, ok := value.(app.CommandAdapterPlan); !ok {
					t.Fatalf("emitted %T, want full command adapter plan", value)
				}
				return 0
			})
			if !emitted {
				t.Fatal("plan was not emitted")
			}
			if code != test.want {
				t.Fatalf("exit code = %d, want %d", code, test.want)
			}
		})
	}
}
