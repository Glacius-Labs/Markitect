package examples

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Retained raw evidence must remain byte-identical after Git checkout on either
// supported platform. This is integrity, not a semantic capability result.
func TestCapabilityProofEvidenceManifest(t *testing.T) {
	root := filepath.Join(harnessRepositoryRoot(t), "experiments", "capability-proof")
	data, err := os.ReadFile(filepath.Join(root, "file-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version int
		Files   []struct {
			Path   string
			SHA256 string
			Bytes  int
		}
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Files) == 0 {
		t.Fatal("missing evidence manifest")
	}
	for _, entry := range manifest.Files {
		if filepath.IsAbs(entry.Path) || strings.Contains(entry.Path, "..") || strings.Contains(entry.Path, "\\") {
			t.Fatal("unsafe evidence path")
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if len(data) != entry.Bytes || hex.EncodeToString(sum[:]) != entry.SHA256 {
			t.Fatalf("retained evidence bytes changed: %s", entry.Path)
		}
	}
}
