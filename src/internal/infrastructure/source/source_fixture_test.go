package source

import (
	"os"
	"path/filepath"
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
