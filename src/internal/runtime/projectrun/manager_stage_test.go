package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

func TestManagerStageIdentityCannotEnterOrdinaryRunPorts(t *testing.T) {
	root, spec, invoker, host := stageStoreFixture(t)
	preview, err := PreviewManagerStage(invoker, root, spec)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := RunManagerStage(context.Background(), host, invoker, root, spec, preview.PreviewDigest)
	if err != nil {
		t.Fatal(err)
	}
	// Remove missing-runtime-file as an alternative reason for ordinary rejection.
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath)), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.readLatestState(stage.RunID)
	if err != nil {
		t.Fatal(err)
	}
	calls := []struct {
		name string
		call func() error
	}{
		{"status", func() error { _, err := Status(root, stage.RunID); return err }},
		{"run", func() error { _, err := Run(context.Background(), Host{}, invoker, root, stage.RunID); return err }},
		{"resume", func() error { _, err := Resume(context.Background(), Host{}, invoker, root, stage.RunID); return err }},
		{"repair", func() error { _, err := Repair(context.Background(), Host{}, invoker, root, stage.RunID); return err }},
		{"verify", func() error { _, err := Verify(context.Background(), Host{}, invoker, root, stage.RunID); return err }},
		{"apply", func() error { _, err := Apply(Host{}, invoker, root, ApplyRequest{RunID: stage.RunID}); return err }},
	}
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			if err := c.call(); err == nil {
				t.Fatal("planless stage entered ordinary delivery port")
			}
		})
	}
	after, err := store.readLatestState(stage.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || before.Digest != after.Digest {
		t.Fatal("ordinary port dispatched or rewrote planless authority")
	}
}
