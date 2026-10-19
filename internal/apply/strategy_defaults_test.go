package apply_test

import (
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// defaultsAbsentExecution authors no instruction list, so the product supplies it.
func defaultsAbsentExecution(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
}

func TestM1ExecutionPreparesRealProductDefaultsWithoutChangingAuthoredAbsence(t *testing.T) {
	parent := executionResult(t, defaultsAbsentExecution(t))
	before, available := parent.Document()
	if !available || before.Defaults.Execute.Instructions != nil || before.Defaults.Verify.Instructions != nil {
		t.Fatal("authored absence fixture was not actually decoded")
	}
	product := apply.ProductStrategyDefaults()
	prepared := parent.PrepareDefaults(product)
	values, complete := prepared.Declarations()
	if prepared.Status() != model.Passed || !complete || len(prepared.Diagnostics()) != 0 || len(values.Defaults.Execute.Instructions.Value) != 0 || len(values.Defaults.Verify.Instructions.Value) != 0 {
		t.Fatalf("actual M1 product default application failed: %s %+v", prepared.Status(), prepared.Diagnostics())
	}
	id := snapshot.ContentID(sha256.Sum256(product.Bytes))
	for _, field := range []struct {
		source organization.SettingSource
		path   string
	}{
		{values.Defaults.Execute.Instructions.Source, "/defaults/instructions/execute"},
		{values.Defaults.Verify.Instructions.Source, "/defaults/instructions/verify"},
	} {
		if field.source.File != product.Source || field.source.ContentID != id || field.source.Field != field.path {
			t.Fatalf("default acquired invented execution provenance: %+v", field.source)
		}
	}
	after, _ := prepared.Execution().Document()
	if !reflect.DeepEqual(after, before) || !reflect.DeepEqual(prepared.Execution().Model().Inputs(), parent.Model().Inputs()) || prepared.Execution().Model().Inputs().Guard == "" || !reflect.DeepEqual(prepared.ProductInput(), product) || prepared.ProductSource() != (organization.SettingSource{File: product.Source, ContentID: id}) {
		t.Fatal("product bytes or opaque capture/authored parent binding changed")
	}
	if values.Defaults.Execute.Profile != *before.Defaults.Execute.Profile || values.Defaults.Verify.Profile != *before.Defaults.Verify.Profile || !reflect.DeepEqual(values.Overrides, before.Packages.Value) {
		t.Fatal("other declarations or exact overrides were rewritten")
	}
}
