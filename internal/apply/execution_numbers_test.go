package apply_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestExecutionIntegerRepresentationLimitIsNotALanguageMaximum(t *testing.T) {
	for _, literal := range []string{"9223372036854775808", "184467440737095516160", "999999999999999999999999999999999999999999999999999"} {
		t.Run(literal, func(t *testing.T) {
			content := strings.Replace(documentedExecution(t), "maxRuns: 5", "maxRuns: "+literal, 1)
			selection, raw := executionInputs(t, content)
			result := registryModel(t, selection, raw).DecodeExecution()
			if result.Status() != model.NotCompleted || !executionDiagnostic(result.Diagnostics(), "execution.integer_range") {
				t.Fatalf("representation limit misclassified: %s %+v", result.Status(), result.Diagnostics())
			}
			if _, ok := result.Document(); ok || string(result.Model().Inputs().Raw.Execution[0].File.Bytes()) != content || result.Partial().Limits.MaxRuns.Source.File != "" {
				t.Fatal("unrepresentable integer became usable or original source lost")
			}
			for _, diagnostic := range result.Diagnostics() {
				if diagnostic.Code == "execution.integer_range" && (diagnostic.Source.Field != "/limits/maxRuns" || diagnostic.Source.ContentID != raw.Execution[0].File.ContentID() || diagnostic.Position == nil) {
					t.Fatal("representation diagnostic lost source")
				}
			}
		})
	}
}

func TestExecutionRejectsNonpositiveAndNonintegerCaps(t *testing.T) {
	for _, literal := range []string{"0", "-1", "-184467440737095516160", "-0xffffffffffffffffffffffffffffffff", "1.0", ".inf", ".nan", `"5"`, "true", "!!int nope", "!!int 1.5"} {
		t.Run(literal, func(t *testing.T) {
			selection, raw := executionInputs(t, strings.Replace(documentedExecution(t), "maxRuns: 5", "maxRuns: "+literal, 1))
			result := registryModel(t, selection, raw).DecodeExecution()
			if result.Status() != model.Failed {
				t.Fatalf("invalid integer accepted or treated as tool limit: %s %+v", result.Status(), result.Diagnostics())
			}
		})
	}
	selection, raw := executionInputs(t, strings.Replace(documentedExecution(t), "maxTransientRetries: 2", "maxTransientRetries: 0", 1))
	result := registryModel(t, selection, raw).DecodeExecution()
	if result.Status() != model.Failed || !executionDiagnostic(result.Diagnostics(), "execution.positive") {
		t.Fatal("zero retries invented an off switch")
	}
}

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
			selection, raw := executionInputs(t, strings.Replace(documentedExecution(t), "maxExecutionTime: 16h", "maxExecutionTime: "+tc.literal, 1))
			result := registryModel(t, selection, raw).DecodeExecution()
			if result.Status() != tc.status || result.Partial().Limits.MaxExecutionTime.Value != tc.want {
				t.Fatalf("duration status/value changed: %s %+v", result.Status(), result.Diagnostics())
			}
			if tc.status == model.NotCompleted && !executionDiagnostic(result.Diagnostics(), "execution.duration_range") {
				t.Fatal("duration tool limit lost")
			}
		})
	}
}
