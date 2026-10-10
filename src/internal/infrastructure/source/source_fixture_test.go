package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

// commitIndex commits the index as it is and returns the commit. Repo.Commit
// stages every change first, which would drop entries without a work tree
// file, such as the symlinks and gitlinks tests add with update-index.
func commitIndex(repo *testkit.Repo, message string) string {
	repo.Git("commit", "--quiet", "--no-verify", "--message", message)
	return repo.Git("rev-parse", "HEAD")
}

// requireSameError calls call repeatedly and requires one error text every
// time, so a diagnostic that depends on map iteration order fails the test.
func requireSameError(t *testing.T, call func() error, want string) {
	t.Helper()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		err := call()
		if err == nil {
			t.Fatal("expected an error")
		}
		seen[err.Error()] = true
	}
	if len(seen) != 1 {
		t.Fatalf("got %d distinct errors: %q", len(seen), mapKeys(seen))
	}
	if got := mapKeys(seen)[0]; !strings.Contains(got, want) {
		t.Fatalf("error = %q, want it to contain %q", got, want)
	}
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
