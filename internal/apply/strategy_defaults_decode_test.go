package apply

import (
	"crypto/sha256"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestProductStrategyDefaultsDocumentBindsAcceptedEffects(t *testing.T) {
	input := ProductStrategyDefaults()
	decoded := decodeStrategyDefaults(input)
	if decoded.status != model.Passed || len(decoded.diagnostics) != 0 || decoded.rejectionLimit.Value != 3 || len(decoded.manager.Value) != 0 || len(decoded.reviewer.Value) != 0 || decoded.batch.Value || len(decoded.approval.Value) != 0 {
		t.Fatalf("actual embedded product defaults failed: %+v", decoded)
	}
	id := snapshot.ContentID(sha256.Sum256(input.Bytes))
	for _, field := range []struct {
		source organization.SettingSource
		path   string
	}{
		{decoded.rejectionLimit.Source, "/defaults/rejectionLimit"},
		{decoded.manager.Source, "/defaults/instructions/manager"},
		{decoded.reviewer.Source, "/defaults/instructions/reviewer"},
		{decoded.batch.Source, "/defaults/batchNoChangeReviews"},
		{decoded.approval.Source, "/acceptance/ownerApproval"},
	} {
		if field.source.File != "product:recursive-apply-v1/defaults.yaml" || field.source.ContentID != id || field.source.Field != field.path {
			t.Fatalf("product field lost original byte digest or actual pointer: %+v", field.source)
		}
	}
}
