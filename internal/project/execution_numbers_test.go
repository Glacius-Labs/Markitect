package project_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestExecutionDurationsAreExplicitPositiveStrings(t *testing.T) {
	for _, tc := range []struct {
		literal string
		status  model.Status
		want    time.Duration
	}{
		{"1h2m3.5s", model.Passed, time.Hour + 2*time.Minute + 3500*time.Millisecond},
		{`"+0.5s"`, model.Passed, 500 * time.Millisecond},
		{"2µs", model.Passed, 2 * time.Microsecond},
		{"600", model.Failed, 0}, {"0s", model.Failed, 0}, {"-1m", model.Failed, 0},
		{"forever", model.Failed, 0}, {"1d", model.Failed, 0},
		{"999999999999999999999h", model.NotCompleted, 0},
		{"0.1ns", model.NotCompleted, 0},
		{"1.5ns", model.NotCompleted, 0},
		{"1.0000000001s", model.NotCompleted, 0},
		{"1.000000001s", model.Passed, time.Second + time.Nanosecond},
		{"9223372036854775807ns", model.Passed, time.Duration(1<<63 - 1)},
		{"9223372036854775808ns", model.NotCompleted, 0},
		{"1.s", model.Passed, time.Second},
	} {
		t.Run(tc.literal, func(t *testing.T) {
			selection, raw := executionInputs(t, strings.Replace(documentedExecution(t), "timeout: 2m", "timeout: "+tc.literal, 1))
			result := registryModel(t, selection, raw).DecodeExecution()
			if result.Status() != tc.status || result.Partial().Checks.Value[0].Timeout.Value != tc.want {
				t.Fatalf("duration status/value changed: %s %+v", result.Status(), result.Diagnostics())
			}
			if tc.status == model.NotCompleted && !executionDiagnostic(result.Diagnostics(), "execution.duration_range") {
				t.Fatal("duration tool limit lost")
			}
		})
	}
}
