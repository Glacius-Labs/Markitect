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

func defaultsAbsentExecution(t *testing.T) string {
	t.Helper()
	content := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n")
	for _, absent := range []string{"  reviewRounds: 3\n", "  batchNoChangeReviews: false\n", "acceptance:\n  ownerApproval: []\n"} {
		if strings.Count(content, absent) != 1 {
			t.Fatal("documented optional-default fixture marker changed")
		}
		content = strings.Replace(content, absent, "", 1)
	}
	return content
}

func TestM1ExecutionPreparesRealProductDefaultsWithoutChangingAuthoredAbsence(t *testing.T) {
	parent := executionResult(t, defaultsAbsentExecution(t))
	before, available := parent.Document()
	if !available || before.Defaults.ReviewRounds != nil || before.Defaults.Manager.Instructions != nil || before.Defaults.Reviewer.Instructions != nil || before.Defaults.BatchNoChangeReviews != nil || before.OwnerApproval != nil {
		t.Fatal("authored absence fixture was not actually decoded")
	}
	product := apply.ProductStrategyDefaults()
	prepared := parent.PrepareDefaults(product)
	values, complete := prepared.Declarations()
	if prepared.Status() != model.Passed || !complete || len(prepared.Diagnostics()) != 0 || values.Defaults.ReviewRounds.Value != 3 || len(values.Defaults.Manager.Instructions.Value) != 0 || len(values.Defaults.Reviewer.Instructions.Value) != 0 || values.Defaults.BatchNoChangeReviews.Value || len(values.OwnerApproval.Value) != 0 {
		t.Fatalf("actual M1 product default application failed: %s %+v", prepared.Status(), prepared.Diagnostics())
	}
	id := snapshot.ContentID(sha256.Sum256(product.Bytes))
	for _, field := range []struct {
		source organization.SettingSource
		path   string
	}{
		{values.Defaults.ReviewRounds.Source, "/defaults/reviewRounds"},
		{values.Defaults.Manager.Instructions.Source, "/defaults/instructions/manager"},
		{values.Defaults.Reviewer.Instructions.Source, "/defaults/instructions/reviewer"},
		{values.Defaults.BatchNoChangeReviews.Source, "/defaults/batchNoChangeReviews"},
		{values.OwnerApproval.Source, "/acceptance/ownerApproval"},
	} {
		if field.source.File != product.Source || field.source.ContentID != id || field.source.Field != field.path {
			t.Fatalf("default acquired invented execution provenance: %+v", field.source)
		}
	}
	after, _ := prepared.Execution().Document()
	if !reflect.DeepEqual(after, before) || !reflect.DeepEqual(prepared.Execution().Model().Inputs(), parent.Model().Inputs()) || prepared.Execution().Model().Inputs().Guard == "" || !reflect.DeepEqual(prepared.ProductInput(), product) || prepared.ProductSource() != (organization.SettingSource{File: product.Source, ContentID: id}) {
		t.Fatal("product bytes or opaque capture/authored parent binding changed")
	}
	if values.Defaults.Manager.Profile != *before.Defaults.Manager.Profile || values.Defaults.Reviewer.Profile != *before.Defaults.Reviewer.Profile || values.Defaults.StepTimeout != before.Defaults.StepTimeout || !reflect.DeepEqual(values.Overrides, before.Packages.Value) || values.Ceilings.ReviewRounds != before.Limits.MaxReviewRounds {
		t.Fatal("other declarations or exact overrides were rewritten")
	}
}
