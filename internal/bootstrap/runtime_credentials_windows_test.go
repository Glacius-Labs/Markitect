//go:build windows

package bootstrap

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestRuntimeCredentialRefusesWindowsJunctionAndAliases(t *testing.T) {
	dir := testkit.TempDir(t)
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(realDir, "credential")
	writeFakeCredential(t, file, []byte("fake-secret"))
	junction := filepath.Join(dir, "junction")
	// A directory junction requires no symlink privilege. Its target contains
	// only this test's fake credential and stays inside the temporary directory.
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, realDir).CombinedOutput(); err != nil {
		t.Fatalf("create fake credential junction: %v: %s", err, out)
	}
	resolver, _, err := newCredentialResolvers(map[string]credentialSource{"selected": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: filepath.Join(junction, "credential")}})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := resolver(context.Background(), "selected"); !errors.Is(err, errCredentialPath) || len(got.RuntimeLogin) != 0 {
		t.Fatalf("parent junction was not refused: %v", err)
	}
	for _, path := range []string{`\\server\share\credential`, `\\?\C:\credential`, `C:\NUL`, file + ":stream"} {
		if _, _, err := newCredentialResolvers(map[string]credentialSource{"selected": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: path}}); !errors.Is(err, errCredentialPath) {
			t.Fatalf("native alias or device was not refused: %v", err)
		}
	}
	resolver, _, err = newCredentialResolvers(map[string]credentialSource{"selected": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: filepath.Join(realDir, "CREDENTIAL")}})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := resolver(context.Background(), "selected"); err == nil || len(got.RuntimeLogin) != 0 {
		t.Fatal("inexact native case alias was admitted")
	}
}
