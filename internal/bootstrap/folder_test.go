package bootstrap

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// plainFolder is a folder for a project: Markitect uses no Git (DEC-097), so its
// tests need none.
type plainFolder struct {
	t   testing.TB
	Dir string
}

func newFolder(t testing.TB) *plainFolder {
	t.Helper()
	return &plainFolder{t: t, Dir: testkit.TempDir(t)}
}

// Write writes a file of the folder, creating its folders.
func (f *plainFolder) Write(file, content string) {
	f.t.Helper()
	p := filepath.Join(f.Dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// snapshot reads every file below dir, so that a test sees whether a command
// changed anything.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// changed lists the files that differ between two snapshots.
func changed(before, after map[string]string) []string {
	var out []string
	for f, v := range after {
		if before[f] != v {
			out = append(out, f)
		}
	}
	for f := range before {
		if _, ok := after[f]; !ok {
			out = append(out, f)
		}
	}
	return out
}

// noRepository fails when a command made the folder a Git repository.
func noRepository(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("a repository appeared in %s: %v", dir, err)
	}
}

// newCLI is the command line as Run builds it, with its own standard input.
func newCLI(stdin io.Reader, stdout, stderr io.Writer) cli.CLI {
	return cli.CLI{Folders: folders{}, Work: work(), Stdin: stdin, Stdout: stdout, Stderr: stderr}
}

// certify passes the work of the folder through gate, every item answered ok with
// a pointer to README.md, the way a reviewer does, and returns the certificate.
func certify(t *testing.T, dir string) string {
	t.Helper()
	mustRun(t, dir, "checklist")
	list := mustRun(t, dir, "checklist", "--review")
	var lines []string
	for _, line := range strings.Split(list, "\n") {
		if rest, ok := strings.CutPrefix(line, "[ ] "); ok {
			if i := strings.Index(rest, `: "`); i >= 0 {
				rest = rest[:i]
			}
			lines = append(lines, rest+": ok README.md:1")
		}
	}
	var out, errOut bytes.Buffer
	if code := newCLI(strings.NewReader(strings.Join(lines, "\n")), &out, &errOut).Run(context.Background(), []string{"answer", "--repo", dir}); code != 0 {
		t.Fatalf("answer: %d\n%s%s", code, out.String(), errOut.String())
	}
	passed := mustRun(t, dir, "gate")
	for _, f := range strings.Fields(passed) {
		if f = strings.TrimSuffix(f, ","); strings.HasSuffix(f, ".json") {
			return f
		}
	}
	t.Fatalf("no certificate named:\n%s", passed)
	return ""
}
