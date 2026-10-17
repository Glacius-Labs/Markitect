package checks

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func processFields(executable string, bytes []byte) string {
	return fmt.Sprintf("profileVersion: 1\nexecutable: %s\nsha256: '%x'\n", strconv.Quote(executable), sha256.Sum256(bytes))
}

func TestM1RealPinnedRegularFilePreparesExactCheckWithoutExecution(t *testing.T) {
	bytes := []byte("This file deliberately is not an executable image.\r\n")
	executable := filepath.Join(testkit.TempDir(t), "project-check.exe")
	if err := os.WriteFile(executable, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	files, defaults := m1Parents(t, processFields(executable, bytes))
	recorder := &recordingPreparer{observe: true}
	result, err := files.PrepareSettings(context.Background(), defaults, nil, recorder)
	selections, ok := result.Checks()
	if err != nil || !ok || len(selections) != 1 || result.Status() != model.NotCompleted {
		t.Fatalf("real check metadata: %v %s %+v", err, result.Status(), result.Diagnostics())
	}
	if _, ready := result.Settings(); ready {
		t.Fatal("nil Agent adapter fabricated settings/runtime support")
	}
	profile := selections[0].Profile()
	observation := profile.Observation()
	expected := apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: filepath.Dir(executable), Path: fixturePath(t, filepath.Base(executable))}, ContentID: snapshot.ContentID(sha256.Sum256(bytes)), Size: int64(len(bytes)), Mode: snapshot.Regular}
	if observation.Status != model.Passed || observation.InputBinding != recorder.input.Binding() || observation.Contract != (apply.ProfileContract{Name: "markitect.process-check", Version: 1}) || !reflect.DeepEqual(observation.Files, []apply.ProfileMetadataFile{expected}) || len(observation.Inventories) != 0 || len(observation.Instructions) != 0 || len(observation.RequiredInstructions) != 0 {
		t.Fatalf("actual metadata facts lost: %+v", observation)
	}
	declared := defaults.Execution().Partial().Checks.Value[0]
	if !reflect.DeepEqual(selections[0].Declaration(), declared) || !reflect.DeepEqual(result.FileCheck(), files) {
		t.Fatal("original M1 Check declaration or fixed source parent changed")
	}
	if _, ok := profile.Identity(); !ok {
		t.Fatal("Apply did not derive the comparison identity")
	}
}
