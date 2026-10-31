package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTheStoreKeepsEachRepositoryOnceByName(t *testing.T) {
	dir := t.TempDir()
	s := &Store{File: filepath.Join(dir, projectsFile)}
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("empty store: %v %v", list, err)
	}
	b, err := s.Add(filepath.Join(dir, "beta"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(filepath.Join(dir, "Alpha")); err != nil {
		t.Fatal(err)
	}
	again, err := s.Add(filepath.Join(dir, "beta"))
	if err != nil || again != b {
		t.Fatalf("second add %v %v", again, err)
	}
	list, _ := s.List()
	if len(list) != 2 || list[0].Name != "Alpha" || list[1].Name != "beta" {
		t.Fatalf("list %v", list)
	}
	if err := s.Remove(b.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.List(); len(list) != 1 || list[0].Name != "Alpha" {
		t.Fatalf("after remove %v", list)
	}
}

func TestProjectIDsFollowTheFileSystemsCase(t *testing.T) {
	a, b := projectID(`C:\Work\Library`), projectID(`c:\work\library`)
	if (a == b) != (runtime.GOOS == "windows" || runtime.GOOS == "darwin") {
		t.Errorf("ids %s and %s on %s", a, b, runtime.GOOS)
	}
}

func TestTheStoreRemembersTheLastParentBesideTheProjects(t *testing.T) {
	s := &Store{File: filepath.Join(t.TempDir(), projectsFile)}
	if parent, err := s.Parent(); err != nil || parent != "" {
		t.Fatalf("empty store: %q %v", parent, err)
	}
	if _, err := s.Add("/work/library"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetParent("/work"); err != nil {
		t.Fatal(err)
	}
	if parent, _ := s.Parent(); parent != "/work" {
		t.Errorf("parent %q", parent)
	}
	if list, _ := s.List(); len(list) != 1 {
		t.Errorf("the parent cost a project: %v", list)
	}
}

func TestABrokenStoreFileSaysWhichFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), projectsFile)
	if err := os.WriteFile(file, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Store{File: file}).List(); err == nil || !strings.Contains(err.Error(), file) {
		t.Errorf("error %v", err)
	}
}
