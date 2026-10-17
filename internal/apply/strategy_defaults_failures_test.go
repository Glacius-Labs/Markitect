package apply_test

import (
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestDefaultsRefuseMissingUnknownAndInvalidProductInputs(t *testing.T) {
	parent := executionResult(t, defaultsAbsentExecution(t))
	product := apply.ProductStrategyDefaults()
	for _, tc := range []struct {
		name, from, to, code, field string
		status                      model.Status
	}{
		{"missing", "", "", "strategy_defaults.unavailable", "", model.NotCompleted},
		{"unknown source", "", "", "strategy_defaults.source_unknown", "", model.NotCompleted},
		{"observed empty", "", "", "strategy_defaults.yaml", "", model.Failed},
		{"unknown version", "markitect.strategy-defaults/v1", "markitect.strategy-defaults/v2", "strategy_defaults.profile_unknown", "/apiVersion", model.NotCompleted},
		{"unknown strategy", "recursive-apply-v1", "future-strategy", "strategy_defaults.profile_unknown", "/strategy", model.NotCompleted},
		{"missing field", "  rejectionLimit: 3\n", "", "strategy_defaults.field_missing", "/defaults/rejectionLimit", model.Failed},
		{"wrong type", "rejectionLimit: 3", "rejectionLimit: '3'", "strategy_defaults.integer", "/defaults/rejectionLimit", model.Failed},
		{"old rejection setting key", "rejectionLimit: 3", "reviewRounds: 3", "strategy_defaults.unknown_key", "/defaults/reviewRounds", model.Failed},
		{"nonpositive", "rejectionLimit: 3", "rejectionLimit: 0", "strategy_defaults.positive", "/defaults/rejectionLimit", model.Failed},
		{"null", "rejectionLimit: 3", "rejectionLimit: null", "strategy_defaults.yaml_feature", "/defaults/rejectionLimit", model.Failed},
		{"duplicate", "  rejectionLimit: 3", "  rejectionLimit: 3\n  rejectionLimit: 2", "strategy_defaults.duplicate_key", "/defaults/rejectionLimit", model.Failed},
		{"unknown field", "defaults:", "unlimited: true\ndefaults:", "strategy_defaults.unknown_key", "/unlimited", model.Failed},
		{"alias feature", "manager: []", "manager: &instructions []", "strategy_defaults.yaml_feature", "/defaults/instructions/manager", model.Failed},
		{"empty instruction reference", "manager: []", "manager: ['']", "strategy_defaults.instruction_reference", "/defaults/instructions/manager", model.Failed},
		{"missing role list", "    reviewer: []\n", "", "strategy_defaults.role_instructions", "/defaults/instructions/reviewer", model.Failed},
		{"quoted boolean", "batchNoChangeReviews: false", "batchNoChangeReviews: 'false'", "strategy_defaults.boolean", "/defaults/batchNoChangeReviews", model.Failed},
		{"extra document", "apiVersion:", "---\n{}\n---\napiVersion:", "strategy_defaults.documents", "", model.Failed},
		{"unknown approval package", "ownerApproval: []", "ownerApproval: [not-a-local-package]", "settings.owner_approval_package", "/acceptance/ownerApproval", model.Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := apply.ProductStrategyDefaults()
			switch tc.name {
			case "missing":
				input.Bytes = nil
			case "unknown source":
				input.Source = "product:unobserved/defaults.yaml"
			case "observed empty":
				input.Bytes = []byte{}
			default:
				if strings.Count(string(input.Bytes), tc.from) != 1 {
					t.Fatal("product fixture mutation marker changed")
				}
				input.Bytes = []byte(strings.Replace(string(input.Bytes), tc.from, tc.to, 1))
			}
			prepared := parent.PrepareDefaults(input)
			if _, usable := prepared.Declarations(); usable || prepared.Status() != tc.status || !reflect.DeepEqual(prepared.ProductInput(), input) || !reflect.DeepEqual(prepared.Execution(), parent) {
				t.Fatalf("refusal lost known source/parent or exposed declarations: %s %+v", prepared.Status(), prepared.Diagnostics())
			}
			var id snapshot.ContentID
			if input.Bytes != nil {
				id = snapshot.ContentID(sha256.Sum256(input.Bytes))
			}
			found := false
			for _, diagnostic := range prepared.Diagnostics() {
				if diagnostic.Code == tc.code {
					found = true
					if diagnostic.Source.File != input.Source || diagnostic.Source.ContentID != id || diagnostic.Source.Field != tc.field {
						t.Fatalf("refusal lost its actual product field: %+v", diagnostic)
					}
				}
			}
			if !found || prepared.ProductSource().ContentID != id {
				t.Fatalf("missing source-bound refusal %s: %+v", tc.code, prepared.Diagnostics())
			}
		})
	}
	if !reflect.DeepEqual(product, apply.ProductStrategyDefaults()) {
		t.Fatal("refusals changed the shipped product document")
	}
}

func TestDefaultsKnownInvalidProductSurvivesIncompleteExecutionParent(t *testing.T) {
	var missing apply.CapturedExecutionDecoding
	valid := missing.PrepareDefaults(apply.ProductStrategyDefaults())
	if _, ok := valid.Declarations(); ok || valid.Status() != model.NotCompleted || valid.Execution().Status() != model.NotCompleted {
		t.Fatal("missing execution parent became usable")
	}
	invalid := apply.ProductStrategyDefaults()
	invalid.Bytes = []byte("apiVersion: [unterminated")
	result := missing.PrepareDefaults(invalid)
	if _, ok := result.Declarations(); ok || result.Status() != model.Failed || len(result.Diagnostics()) != 2 {
		t.Fatalf("known product error was hidden by incomplete parent: %+v", result.Diagnostics())
	}
	parent := executionResult(t, strings.Replace(defaultsAbsentExecution(t), "strategy: recursive-apply-v1", "strategy: unsupported", 1))
	result = parent.PrepareDefaults(apply.ProductStrategyDefaults())
	if result.Status() != model.Failed || !reflect.DeepEqual(result.Execution().Diagnostics(), parent.Diagnostics()) || !reflect.DeepEqual(result.Execution().Partial(), parent.Partial()) {
		t.Fatal("failed execution parent was replaced with product defaults")
	}
}

func TestDefaultsValidateActualProjectCeilingsWithoutClamping(t *testing.T) {
	base := defaultsAbsentExecution(t)
	for _, tc := range []struct{ name, from, to string }{
		{"product default exceeds project ceiling", "maxRejectionLimit: 3", "maxRejectionLimit: 2"},
		{"authored rejection limit exceeds project ceiling", "  stepTimeout: 10m", "  rejectionLimit: 4\n  stepTimeout: 10m"},
		{"override rejection limit exceeds project ceiling", "    stepTimeout: 8m", "    rejectionLimit: 4\n    stepTimeout: 8m"},
		{"default time exceeds project", "  stepTimeout: 10m", "  stepTimeout: 11m"},
		{"override time exceeds project", "    stepTimeout: 8m", "    stepTimeout: 11m"},
		{"default roots exceed project", "  creationRoots: [realization]\npackages:", "  creationRoots: [outside]\npackages:"},
		{"override roots exceed project", "    stepTimeout: 8m", "    creationRoots: [outside]\n    stepTimeout: 8m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Count(base, tc.from) != 1 {
				t.Fatal("project ceiling fixture marker changed")
			}
			parent := executionResult(t, strings.Replace(base, tc.from, tc.to, 1))
			if parent.Status() != model.Passed {
				t.Fatalf("authored declaration did not reach real ceiling validation: %+v", parent.Diagnostics())
			}
			prepared := parent.PrepareDefaults(apply.ProductStrategyDefaults())
			if _, ok := prepared.Declarations(); ok || prepared.Status() != model.Failed || len(prepared.Diagnostics()) != 1 || prepared.Diagnostics()[0].Code != "settings.selections" || !reflect.DeepEqual(prepared.Execution().Partial(), parent.Partial()) {
				t.Fatalf("actual project ceiling was clamped or bypassed: %+v", prepared.Diagnostics())
			}
			if tc.name == "product default exceeds project ceiling" && (prepared.Execution().Partial().Defaults.RejectionLimit != nil || prepared.Execution().Partial().Limits.MaxRejectionLimit.Value != 2) {
				t.Fatal("product default was forged into authored execution or project ceiling was changed")
			}
		})
	}
}
