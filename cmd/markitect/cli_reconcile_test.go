package main

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

func TestCommandAdapterPlanExitReflectsResult(t *testing.T) {
	cases := []struct {
		name   string
		result host.AdapterResult
		want   int
	}{
		{name: "complete", result: host.AdapterResult{Status: "complete"}, want: 0},
		{name: "incomplete", result: host.AdapterResult{Status: "incomplete"}, want: 2},
		{name: "failed", result: host.AdapterResult{Status: "failed"}, want: 1},
		{name: "error finding", result: host.AdapterResult{Status: "complete", Findings: []host.AdapterFinding{{Code: "policy", Severity: "error", Message: "drift"}}}, want: 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			emitted := false
			code := emitCommandAdapterPlan(host.CommandAdapterPlan{Result: test.result}, func(value any) int {
				emitted = true
				if _, ok := value.(host.CommandAdapterPlan); !ok {
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
