//go:build unix

package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestRuntimeCredentialRefusesSymlinksAndFIFO(t *testing.T) {
	dir := testkit.TempDir(t)
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(realDir, "credential")
	writeFakeCredential(t, file, []byte("fake-secret"))
	leaf, parent, fifo := filepath.Join(dir, "leaf"), filepath.Join(dir, "parent"), filepath.Join(dir, "fifo")
	if err := os.Symlink(file, leaf); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, parent); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path string
		want error
	}{{leaf, errCredentialPath}, {filepath.Join(parent, "credential"), errCredentialPath}, {fifo, errCredentialType}} {
		resolver, _, err := newCredentialResolvers(map[string]credentialSource{"selected": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: tc.path}})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := resolver(context.Background(), "selected"); !errors.Is(err, tc.want) || len(got.RuntimeLogin) != 0 {
			t.Fatalf("redirect/nonregular file was admitted: %v", err)
		}
	}
}
