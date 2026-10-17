package recordstore

import (
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"os"
	"testing"
)

func TestMain(m *testing.M) { testkit.Main(m) }
func openFixture(t *testing.T) (*Store, string) {
	t.Helper()
	root := testkit.TempDir(t)
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err = s.CheckWrite(Run); err != nil {
		t.Fatal(err)
	}
	return s, root
}
func fixtureName() Name {
	return Name{Kind: Run, ID: "0123456789abcdef0123456789abcdef", Role: "plan.json"}
}
func writeFixture(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := root + string(os.PathSeparator) + rel
	dir := p[:len(p)-len(baseLeaf(rel))]
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func baseLeaf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
