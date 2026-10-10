package projectrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

var _ candidateWorkspaceService = (*LocalWorkspaceService)(nil)
var _ interface {
	ReopenCandidate(context.Context, projectworkspace.Request, projectworkspace.Handle, []projectworkspace.Change, string, bool) error
} = (*LocalWorkspaceService)(nil)

func TestLocalWorkspaceServiceInitializesOutsideRepositoryOnFirstPrepare(t *testing.T) {
	storage := filepath.Join(t.TempDir(), "cache", "Markitect", "workspaces")
	service := &LocalWorkspaceService{storageRoot: storage}
	if _, err := os.Stat(storage); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("constructor created workspace storage: stat error = %v", err)
	}

	// The invalid request still exercises forwarding after lazy service setup;
	// the underlying service reports its domain validation error.
	if _, err := service.Prepare(context.Background(), projectworkspace.Request{}); err == nil {
		t.Fatal("Prepare accepted an invalid workspace request")
	}
	if info, err := os.Stat(storage); err != nil || !info.IsDir() {
		t.Fatalf("Prepare did not initialize private cache storage: info=%v err=%v", info, err)
	}
}

func TestLocalWorkspaceServiceReturnsInitializationErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	service := &LocalWorkspaceService{storageRoot: filepath.Join(file, "workspaces")}
	if _, err := service.Prepare(context.Background(), projectworkspace.Request{}); err == nil {
		t.Fatal("Prepare hid workspace initialization error")
	}
}
