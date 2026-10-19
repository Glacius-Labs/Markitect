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
		{"unknown strategy", "shared-workspace-v1", "future-strategy", "strategy_defaults.profile_unknown", "/strategy", model.NotCompleted},
		{"old strategy", "shared-workspace-v1", "recursive-apply-v1", "strategy_defaults.profile_unknown", "/strategy", model.NotCompleted},
		{"missing defaults", "defaults:\n  instructions:\n    execute: []\n    verify: []\n", "", "strategy_defaults.field_missing", "/defaults", model.Failed},
		{"old rejection setting", "defaults:", "defaults:\n  rejectionLimit: 3", "strategy_defaults.unknown_key", "/defaults/rejectionLimit", model.Failed},
		{"old batch setting", "defaults:", "defaults:\n  batchNoChangeReviews: false", "strategy_defaults.unknown_key", "/defaults/batchNoChangeReviews", model.Failed},
		{"old acceptance section", "defaults:", "acceptance: {ownerApproval: []}\ndefaults:", "strategy_defaults.unknown_key", "/acceptance", model.Failed},
		{"null", "execute: []", "execute: null", "strategy_defaults.yaml_feature", "/defaults/instructions/execute", model.Failed},
		{"duplicate", "    execute: []", "    execute: []\n    execute: []", "strategy_defaults.duplicate_key", "/defaults/instructions/execute", model.Failed},
		{"unknown field", "defaults:", "unlimited: true\ndefaults:", "strategy_defaults.unknown_key", "/unlimited", model.Failed},
		{"old role", "execute: []", "manager: []", "strategy_defaults.unknown_key", "/defaults/instructions/manager", model.Failed},
		{"alias feature", "execute: []", "execute: &instructions []", "strategy_defaults.yaml_feature", "/defaults/instructions/execute", model.Failed},
		{"empty instruction reference", "execute: []", "execute: ['']", "strategy_defaults.instruction_reference", "/defaults/instructions/execute", model.Failed},
		{"missing role list", "    verify: []\n", "", "strategy_defaults.role_instructions", "/defaults/instructions/verify", model.Failed},
		{"extra document", "apiVersion:", "---\n{}\n---\napiVersion:", "strategy_defaults.documents", "", model.Failed},
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
	parent := executionResult(t, strings.Replace(defaultsAbsentExecution(t), "strategy: shared-workspace-v1", "strategy: unsupported", 1))
	result = parent.PrepareDefaults(apply.ProductStrategyDefaults())
	if result.Status() != model.Failed || !reflect.DeepEqual(result.Execution().Diagnostics(), parent.Diagnostics()) || !reflect.DeepEqual(result.Execution().Partial(), parent.Partial()) {
		t.Fatal("failed execution parent was replaced with product defaults")
	}
}
