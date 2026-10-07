package inventory

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCaptureClassifiesIgnoredAndUntrackedWithoutReadingIgnoredBytes(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.name", "Inventory Test")
	gitTest(t, root, "config", "user.email", "inventory@example.invalid")
	write(t, root, ".gitignore", "private/\n")
	write(t, root, "tracked.md", "tracked bytes")
	write(t, root, "untracked.md", "new bytes")
	write(t, root, "private/secret.txt", "secret sentinel")
	gitTest(t, root, "add", ".gitignore", "tracked.md")
	gitTest(t, root, "commit", "-qm", "inventory fixture")

	r, err := Capture(root, Options{Roots: []string{"."}, Exclusions: []Boundary{{Path: "excluded", Reason: "fixture boundary"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Provisional || r.Revision == "" || !r.Complete {
		t.Fatalf("unexpected observation status: %#v", r)
	}
	tracked := find(r, "tracked.md")
	untracked := find(r, "untracked.md")
	secret := find(r, "private/secret.txt")
	if tracked.Source != "tracked" || tracked.Digest == "" || tracked.Status != "observed" {
		t.Fatalf("tracked entry = %#v", tracked)
	}
	if untracked.Source != "untracked" || untracked.Digest == "" {
		t.Fatalf("untracked entry = %#v", untracked)
	}
	if secret.Source != "ignored" || secret.Status != "boundary" || secret.Digest != "" || !strings.Contains(secret.Reason, "metadata only") {
		t.Fatalf("ignored secret was not metadata-only: %#v", secret)
	}
	admitted, err := Capture(root, Options{Roots: []string{"."}, AdmitIgnored: []Boundary{{Path: "private", Reason: "owner-approved fixture"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := find(admitted, "private/secret.txt"); got.Source != "ignored" || got.Digest == "" || got.Status != "observed" {
		t.Fatalf("admitted ignored entry = %#v", got)
	}
	if len(admitted.AdmitIgnored) != 1 || admitted.AdmitIgnored[0].Reason != "owner-approved fixture" || admitted.Digest == r.Digest {
		t.Fatalf("report omitted or failed to bind ignored admission: %#v", admitted)
	}
	if find(r, ".git").Status != "boundary" {
		t.Fatal(".git metadata boundary was not visible")
	}
}

func TestCaptureExclusionAndDigestAreDeterministic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "src/a.go", "package a\n")
	write(t, root, "private/marker.txt", "do not hash")
	opts := Options{Roots: []string{"src", "private"}, Exclusions: []Boundary{{Path: "private", Reason: "private scope excluded"}}}
	one, err := Capture(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Capture(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if one.Digest != two.Digest {
		t.Fatalf("same tree produced digests %s and %s", one.Digest, two.Digest)
	}
	if e := find(one, "private"); e.Status != "excluded" || e.Digest != "" {
		t.Fatalf("excluded boundary = %#v", e)
	}
	write(t, root, "src/a.go", "package a // changed\n")
	three, err := Capture(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if one.Digest == three.Digest {
		t.Fatal("content change did not alter inventory digest")
	}
}

func TestCaptureRefusesOverlappingOrEscapingScopes(t *testing.T) {
	root := t.TempDir()
	for _, opts := range []Options{
		{Roots: []string{".", "src"}},
		{Roots: []string{"../outside"}},
		{Roots: []string{"C:/private"}},
		{Roots: []string{"foo\x00bar"}},
		{Roots: []string{".GIT/config"}},
		{Roots: []string{"src", "SRC"}},
		{Roots: []string{"src. "}},
		{Roots: []string{"CON.txt"}},
		{Roots: []string{"src"}, Exclusions: []Boundary{{Path: "outside", Reason: "no"}}},
		{Roots: []string{"src"}, Exclusions: []Boundary{{Path: "src", Reason: "one"}, {Path: "src/private", Reason: "two"}}},
	} {
		if _, err := Capture(root, opts); err == nil {
			t.Fatalf("accepted invalid options: %#v", opts)
		}
	}
}

func TestCaptureRejectsSymlinkAncestorOfExplicitRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write(t, outside, "subdir/file.txt", "outside")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Capture(root, Options{Roots: []string{"linked/subdir"}}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("Capture error = %v, want symlink ancestor refusal", err)
	}
}

func TestCaptureFailsClosedWhenGitClassificationFailsInsideRepository(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	write(t, root, ".gitignore", "private/\n")
	write(t, root, "private/secret.txt", "secret")
	t.Setenv("PATH", t.TempDir())
	if _, err := Capture(root, Options{Roots: []string{"."}}); err == nil || !strings.Contains(err.Error(), "Git classification failed") {
		t.Fatalf("Capture error = %v, want Git classification failure", err)
	}
}

func TestCaptureReportsSymlinkAsBoundary(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	write(t, filepath.Dir(outside), filepath.Base(outside), "outside bytes")
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	r, err := Capture(root, Options{Roots: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	e := find(r, "link.txt")
	if e.Type != "symlink" || e.Status != "boundary" || e.Digest != "" {
		t.Fatalf("symlink entry = %#v", e)
	}
	if _, err := ReadInput(root, "link.txt", 100); err == nil {
		t.Fatal("ReadInput followed a symlink")
	}
}

func TestCaptureTreatsGitMetadataCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	write(t, root, "nested/.GIT/config", "must not be traversed")
	r, err := Capture(root, Options{Roots: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	e := find(r, "nested/.GIT")
	if e.Status != "boundary" || find(r, "nested/.GIT/config").Path != "" {
		t.Fatalf("case-variant Git metadata was traversed: %#v", r.Entries)
	}
}

func TestWindowsCaseVariantRootsPreserveIgnoredAndExcludedClassification(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows filesystem case aliases only")
	}
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.name", "Inventory Test")
	gitTest(t, root, "config", "user.email", "inventory@example.invalid")
	write(t, root, ".gitignore", "private/\n")
	write(t, root, "private/secret.txt", "secret sentinel")
	gitTest(t, root, "add", ".gitignore")
	gitTest(t, root, "commit", "-qm", "inventory fixture")

	byRoot, err := Capture(root, Options{Roots: []string{"PRIVATE"}})
	if err != nil {
		t.Fatal(err)
	}
	secret := find(byRoot, "PRIVATE/secret.txt")
	if secret.Source != "ignored" || secret.Status != "boundary" || secret.Digest != "" {
		t.Fatalf("case-variant root read ignored bytes: %#v", secret)
	}
	byExclusion, err := Capture(root, Options{Roots: []string{"."}, Exclusions: []Boundary{{Path: "PRIVATE", Reason: "case-variant private exclusion"}}})
	if err != nil {
		t.Fatal(err)
	}
	private := find(byExclusion, "private")
	if private.Status != "excluded" || find(byExclusion, "private/secret.txt").Path != "" {
		t.Fatalf("case-variant exclusion did not hide subtree: %#v", byExclusion.Entries)
	}
}

func TestReadInputIsBoundedAndRooted(t *testing.T) {
	root := t.TempDir()
	write(t, root, "model.yaml", "kind: Model\n")
	data, err := ReadInput(root, "model.yaml", 100)
	if err != nil || string(data) != "kind: Model\n" {
		t.Fatalf("ReadInput = %q, %v", data, err)
	}
	for _, name := range []string{"../outside", ".git/config", ".GIT/config", "model.yaml"} {
		if _, err := ReadInput(root, name, 2); err == nil {
			t.Errorf("ReadInput accepted invalid/oversized input %q", name)
		}
	}
}

func find(r Report, name string) Entry {
	for _, e := range r.Entries {
		if e.Path == name {
			return e
		}
	}
	return Entry{}
}

func write(t *testing.T, root, name, text string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = cleanGitEnvironment()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func cleanGitEnvironment() []string {
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			env = append(env, item)
		}
	}
	return env
}
