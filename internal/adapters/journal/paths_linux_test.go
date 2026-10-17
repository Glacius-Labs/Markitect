package journal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxReplacementAndPrivateDirectoryRefusals(t *testing.T) {
	for _, name := range []string{"journal.lock", "journal.sqlite", "state directory"} {
		t.Run(name, func(t *testing.T) {
			r := request(t)
			s := acquire(t, r)
			path := filepath.Join(r.Repository, ".markitect", "state", "journal.lock")
			if name == "journal.sqlite" {
				path = filepath.Join(r.StateDirectory, name)
			}
			if name == "state directory" {
				path = r.StateDirectory
			}
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			if name == "state directory" {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			assertFenced(t, s)
		})
	}
	r := request(t)
	if err := os.Mkdir(r.StateDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if got, err := New().Acquire(context.Background(), r); err == nil {
		got.Close()
		t.Fatal("insecure directory accepted")
	}
	r = request(t)
	if err := os.Symlink(filepath.Dir(r.StateDirectory), r.StateDirectory); err != nil {
		t.Fatal(err)
	}
	if got, err := New().Acquire(context.Background(), r); err == nil {
		got.Close()
		t.Fatal("symlink state accepted")
	}
}
