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
	if decoded.status != model.Passed || len(decoded.diagnostics) != 0 || len(decoded.execute.Value) != 0 || len(decoded.verify.Value) != 0 {
		t.Fatalf("actual embedded product defaults failed: %+v", decoded)
	}
	id := snapshot.ContentID(sha256.Sum256(input.Bytes))
	for _, field := range []struct {
		source organization.SettingSource
		path   string
	}{
		{decoded.execute.Source, "/defaults/instructions/execute"},
		{decoded.verify.Source, "/defaults/instructions/verify"},
	} {
		if field.source.File != "product:recursive-passes-v1/defaults.yaml" || field.source.ContentID != id || field.source.Field != field.path {
			t.Fatalf("product field lost original byte digest or actual pointer: %+v", field.source)
		}
	}
}
