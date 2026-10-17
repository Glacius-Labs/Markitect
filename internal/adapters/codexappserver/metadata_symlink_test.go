package codexappserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func makeTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		// ERROR_PRIVILEGE_NOT_HELD (1314); unexpected filesystem errors fail.
		if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314)) {
			t.Skipf("Windows symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

func TestRedirectedExecutableBoundaryAncestorAndUnusedSchemaAreRefused(t *testing.T) {
	for _, which := range []string{"executable", "inventory boundary", "unused member", "ancestor"} {
		t.Run(which, func(t *testing.T) {
			f := fixture(t)
			inline := f.inline()
			switch which {
			case "executable":
				link := f.exe + "-link"
				makeTestSymlink(t, f.exe, link)
				inline = strings.Replace(inline, f.exe, link, 1)
			case "inventory boundary":
				link := f.stable + "-link"
				makeTestSymlink(t, f.stable, link)
				inline = strings.Replace(inline, f.stable, link, 1)
			case "unused member":
				target := filepath.Join(f.stable, "unused.json")
				link := filepath.Join(f.stable, "unselected-link.json")
				makeTestSymlink(t, target, link)
			case "ancestor":
				bin := filepath.Dir(f.exe)
				link := bin + "-link"
				makeTestSymlink(t, bin, link)
				inline = strings.Replace(inline, f.exe, filepath.Join(link, filepath.Base(f.exe)), 1)
			}
			out, _ := prepareMetadata(context.Background(), profileInput(t, inline, nil), f.pin)
			if out.Status != model.Failed {
				t.Fatalf("redirect accepted: %+v", out)
			}
		})
	}
}
